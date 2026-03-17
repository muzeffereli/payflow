package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payment-platform/internal/store/adapter"
	storehandler "payment-platform/internal/store/handler"
	"payment-platform/internal/store/service"
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

	cfg.OTel.Service = "store-service"
	otelShutdown, err := telemetry.Init(context.Background(), cfg.OTel)
	if err != nil {
		log.Warn("telemetry init failed â€” tracing disabled", "err", err)
	} else {
		defer otelShutdown(context.Background())
	}

	log.Info("store service starting", "port", cfg.Port)

	db, err := database.Connect(cfg.DB)
	if err != nil {
		log.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("connected to postgres")

	nc, err := nats.Connect(cfg.NATS.URL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		log.Error("failed to connect to NATS", "err", err)
		os.Exit(1)
	}
	defer nc.Drain()
	log.Info("connected to NATS")

	natsPublisher, err := eventbus.NewPublisher(nc)
	if err != nil {
		log.Error("failed to create event publisher", "err", err)
		os.Exit(1)
	}

	repo := adapter.NewPostgresRepo(db)
	pub := adapter.NewNATSPublisher(natsPublisher)
	svc := service.New(repo, pub, log)

	router := gin.New()
	router.Use(otelgin.Middleware("store-service"))
	router.Use(gin.Recovery())
	router.Use(middleware.UserID())
	router.Use(middleware.Role())
	router.Use(middleware.RequestLogger(log))
	router.Use(metrics.HTTPMiddleware("store-service"))

	v1 := router.Group("/api/v1")
	storehandler.New(svc).RegisterRoutes(v1)

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "store-service"})
	})
	router.GET("/metrics", metrics.Handler())

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("store service shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("forced shutdown", "err", err)
	}

	log.Info("store service stopped")
}
