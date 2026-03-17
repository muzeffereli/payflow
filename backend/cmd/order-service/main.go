package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payment-platform/internal/order/adapter"
	orderhandler "payment-platform/internal/order/handler"
	"payment-platform/internal/order/service"
	"payment-platform/pkg/config"
	"payment-platform/pkg/database"
	"payment-platform/pkg/eventbus"
	"payment-platform/pkg/logger"
	"payment-platform/pkg/metrics"
	"payment-platform/pkg/middleware"
	"payment-platform/pkg/outbox"
	"payment-platform/pkg/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	_ "payment-platform/cmd/order-service/docs"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg.LogLevel)

	cfg.OTel.Service = "order-service"
	otelShutdown, err := telemetry.Init(context.Background(), cfg.OTel)
	if err != nil {
		log.Warn("telemetry init failed â€” tracing disabled", "err", err)
	} else {
		defer otelShutdown(context.Background())
	}

	log.Info("order service starting", "port", cfg.Port)

	db, err := database.Connect(cfg.DB)
	if err != nil {
		log.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("connected to postgres")

	nc, err := nats.Connect(cfg.NATS.URL,
		nats.RetryOnFailedConnect(true), // retry connection if NATS isn't ready yet
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		log.Error("failed to connect to NATS", "err", err)
		os.Exit(1)
	}
	defer nc.Drain() // Drain flushes pending messages before closing
	log.Info("connected to NATS")

	natsPublisher, err := eventbus.NewPublisher(nc)
	if err != nil {
		log.Error("failed to create event publisher", "err", err)
		os.Exit(1)
	}

	repo := adapter.NewPostgresRepo(db)
	pub := adapter.NewNATSPublisher(natsPublisher)
	productAddr := envOrDefault("PRODUCT_SERVICE_ADDR", "http://localhost:8087")
	productClient := adapter.NewHTTPProductClient(productAddr)
	svc := service.New(repo, pub, productClient, log)

	sub, err := eventbus.NewSubscriber(nc, log)
	if err != nil {
		log.Error("failed to create subscriber", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	relay := outbox.NewRelay(db, natsPublisher, log)
	go relay.Run(ctx)
	log.Info("outbox relay started")

	if err := sub.Subscribe(ctx, eventbus.SubscribeConfig{
		Stream:        "PAYMENTS",
		Subjects:      []string{"payments.*"},
		Consumer:      "order-service-payments",
		FilterSubject: eventbus.SubjectPaymentSucceeded,
	}, func(ctx context.Context, event eventbus.Event) error {
		var data eventbus.PaymentSucceededData
		if err := eventbus.DecodeData(event.Data, &data); err != nil {
			return err
		}
		return svc.HandlePaymentSucceeded(ctx, data.OrderID)
	}); err != nil {
		log.Error("failed to subscribe to payments.succeeded", "err", err)
		os.Exit(1)
	}

	if err := sub.Subscribe(ctx, eventbus.SubscribeConfig{
		Stream:        "PAYMENTS",
		Subjects:      []string{"payments.*"},
		Consumer:      "order-service-payment-failed",
		FilterSubject: eventbus.SubjectPaymentFailed,
	}, func(ctx context.Context, event eventbus.Event) error {
		var data eventbus.PaymentFailedData
		if err := eventbus.DecodeData(event.Data, &data); err != nil {
			return err
		}
		return svc.HandlePaymentFailed(ctx, data.OrderID)
	}); err != nil {
		log.Error("failed to subscribe to payments.failed", "err", err)
		os.Exit(1)
	}

	if err := sub.Subscribe(ctx, eventbus.SubscribeConfig{
		Stream:        "PAYMENTS",
		Subjects:      []string{"payments.*"},
		Consumer:      "order-service-payment-refunded",
		FilterSubject: eventbus.SubjectPaymentRefunded,
	}, func(ctx context.Context, event eventbus.Event) error {
		var data eventbus.PaymentRefundedData
		if err := eventbus.DecodeData(event.Data, &data); err != nil {
			return err
		}
		return svc.HandlePaymentRefunded(ctx, data.OrderID)
	}); err != nil {
		log.Error("failed to subscribe to payments.refunded", "err", err)
		os.Exit(1)
	}

	router := gin.New()
	router.Use(otelgin.Middleware("order-service"))
	router.Use(gin.Recovery())
	router.Use(middleware.UserID())
	router.Use(middleware.RequestLogger(log))
	router.Use(metrics.HTTPMiddleware("order-service"))

	v1 := router.Group("/api/v1")
	orderhandler.New(svc).RegisterRoutes(v1)

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/metrics", metrics.Handler())

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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

	log.Info("shutting down gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("forced shutdown", "err", err)
	}

	log.Info("order service stopped")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
