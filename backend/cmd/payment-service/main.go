package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payment-platform/internal/payment/adapter"
	paymenthandler "payment-platform/internal/payment/handler"
	"payment-platform/internal/payment/service"
	"payment-platform/pkg/config"
	"payment-platform/pkg/database"
	"payment-platform/pkg/eventbus"
	"payment-platform/pkg/logger"
	"payment-platform/pkg/metrics"
	"payment-platform/pkg/middleware"
	"payment-platform/pkg/outbox"
	"payment-platform/pkg/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	_ "payment-platform/cmd/payment-service/docs"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg.LogLevel)

	cfg.OTel.Service = "payment-service"
	otelShutdown, err := telemetry.Init(context.Background(), cfg.OTel)
	if err != nil {
		log.Warn("telemetry init failed â€” tracing disabled", "err", err)
	} else {
		defer otelShutdown(context.Background())
	}

	log.Info("payment service starting", "port", cfg.Port)

	db, err := database.Connect(cfg.DB)
	if err != nil {
		log.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

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

	natsPublisher, err := eventbus.NewPublisher(nc)
	if err != nil {
		log.Error("publisher init failed", "err", err)
		os.Exit(1)
	}

	storeAddr := envOrDefault("STORE_SERVICE_ADDR", "http://localhost:8089")
	walletAddr := envOrDefault("WALLET_SERVICE_ADDR", "http://localhost:8083")

	repo := adapter.NewPostgresRepo(db)
	pub := adapter.NewNATSPublisher(natsPublisher)
	storeClient := adapter.NewHTTPStoreClient(storeAddr)
	walletClient := adapter.NewHTTPWalletClient(walletAddr)
	svc := service.New(repo, pub, storeClient, walletClient, log)

	sub, err := eventbus.NewSubscriber(nc, log)
	if err != nil {
		log.Error("subscriber init failed", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go outbox.NewRelay(db, natsPublisher, log).Run(ctx)

	if err := sub.Subscribe(ctx, eventbus.SubscribeConfig{
		Stream:        "ORDERS",
		Subjects:      []string{"orders.*"},
		Consumer:      "payment-service-orders",
		FilterSubject: eventbus.SubjectOrderCreated,
	}, func(ctx context.Context, event eventbus.Event) error {
		var data eventbus.OrderCreatedData
		if err := eventbus.DecodeData(event.Data, &data); err != nil {
			return err
		}

		log.Info("processing order.created",
			"order_id", data.OrderID,
			"amount", data.TotalAmount,
			"correlation_id", event.Metadata.CorrelationID,
		)

		return svc.HandleOrderCreated(ctx, data)
	}); err != nil {
		log.Error("subscribe to orders.created failed", "err", err)
		os.Exit(1)
	}

	if err := sub.Subscribe(ctx, eventbus.SubscribeConfig{
		Stream:        "FRAUD",
		Subjects:      []string{"fraud.*"},
		Consumer:      "payment-service-fraud-approved",
		FilterSubject: eventbus.SubjectFraudApproved,
	}, func(ctx context.Context, event eventbus.Event) error {
		var data eventbus.FraudCheckResultData
		if err := eventbus.DecodeData(event.Data, &data); err != nil {
			return err
		}
		log.Info("fraud approved â€” proceeding with payment",
			"payment_id", data.PaymentID,
			"risk_score", data.RiskScore,
		)
		return svc.HandleFraudApproved(ctx, data)
	}); err != nil {
		log.Error("subscribe to fraud.approved failed", "err", err)
		os.Exit(1)
	}

	if err := sub.Subscribe(ctx, eventbus.SubscribeConfig{
		Stream:        "FRAUD",
		Subjects:      []string{"fraud.*"},
		Consumer:      "payment-service-fraud-rejected",
		FilterSubject: eventbus.SubjectFraudRejected,
	}, func(ctx context.Context, event eventbus.Event) error {
		var data eventbus.FraudCheckResultData
		if err := eventbus.DecodeData(event.Data, &data); err != nil {
			return err
		}
		log.Info("fraud rejected â€” blocking payment",
			"payment_id", data.PaymentID,
			"risk_score", data.RiskScore,
			"rules", data.Rules,
		)
		return svc.HandleFraudRejected(ctx, data)
	}); err != nil {
		log.Error("subscribe to fraud.rejected failed", "err", err)
		os.Exit(1)
	}

	router := gin.New()
	router.Use(otelgin.Middleware("payment-service"))
	router.Use(gin.Recovery())
	router.Use(middleware.UserID())
	router.Use(middleware.RequestLogger(log))
	router.Use(metrics.HTTPMiddleware("payment-service"))
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "payment-service"})
	})
	router.GET("/metrics", metrics.Handler())

	v1 := router.Group("/api/v1")
	paymenthandler.New(svc).RegisterRoutes(v1)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	srv := &http.Server{
		Addr:        ":" + cfg.Port,
		Handler:     router,
		ReadTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down payment service...")
	cancel() // stop all NATS consumers

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)

	log.Info("payment service stopped")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
