package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payment-platform/internal/notification/adapter"
	notifhandler "payment-platform/internal/notification/handler"
	"payment-platform/internal/notification/service"
	"payment-platform/pkg/config"
	"payment-platform/pkg/database"
	"payment-platform/pkg/eventbus"
	"payment-platform/pkg/logger"
	"payment-platform/pkg/metrics"
	"payment-platform/pkg/middleware"
	"payment-platform/pkg/telemetry"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg.LogLevel)

	cfg.OTel.Service = "notification-service"
	otelShutdown, err := telemetry.Init(context.Background(), cfg.OTel)
	if err != nil {
		log.Warn("telemetry init failed â€” tracing disabled", "err", err)
	} else {
		defer otelShutdown(context.Background())
	}

	log.Info("notification service starting")

	db, err := database.Connect(cfg.DB)
	if err != nil {
		log.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("connected to postgres")

	notifRepo := adapter.NewPostgresNotifRepo(db)

	nc, err := nats.Connect(cfg.NATS.URL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		log.Error("nats connect failed", "err", err)
		os.Exit(1)
	}
	defer nc.Drain()

	sender := adapter.NewLogSender(log)
	svc := service.NewWithRepo(sender, notifRepo, log)

	sub, err := eventbus.NewSubscriber(nc, log)
	if err != nil {
		log.Error("subscriber init failed", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subscriptions := []eventbus.SubscribeConfig{
		{Stream: "ORDERS", Subjects: []string{"orders.*"}, Consumer: "notification-service-orders"},
		{Stream: "PAYMENTS", Subjects: []string{"payments.*"}, Consumer: "notification-service-payments"},
		{Stream: "WALLETS", Subjects: []string{"wallets.*"}, Consumer: "notification-service-wallets"},
		{Stream: "FRAUD", Subjects: []string{"fraud.*"}, Consumer: "notification-service-fraud"},
	}

	for _, cfg := range subscriptions {
		if err := sub.Subscribe(ctx, cfg, svc.HandleEvent); err != nil {
			log.Error("failed to subscribe", "stream", cfg.Stream, "err", err)
			os.Exit(1)
		}
	}

	log.Info("notification service ready â€” listening for events")

	router := gin.New()
	router.Use(otelgin.Middleware("notification-service"))
	router.Use(gin.Recovery())
	router.Use(middleware.UserID())
	router.Use(middleware.Role())
	router.Use(middleware.RequestLogger(log))
	router.Use(metrics.HTTPMiddleware("notification-service"))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "notification-service"})
	})
	router.GET("/metrics", metrics.Handler())

	v1 := router.Group("/api/v1")
	notifhandler.New(notifRepo).RegisterRoutes(v1)

	srv := &http.Server{Addr: ":8084", Handler: router}
	go func() {
		log.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("notification service shutting down...")
	cancel()
	shutdownCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	srv.Shutdown(shutdownCtx)
}
