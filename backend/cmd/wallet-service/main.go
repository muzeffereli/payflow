package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payment-platform/internal/wallet/adapter"
	wallethandler "payment-platform/internal/wallet/handler"
	"payment-platform/internal/wallet/service"
	"payment-platform/pkg/config"
	"payment-platform/pkg/database"
	"payment-platform/pkg/eventbus"
	"payment-platform/pkg/logger"
	"payment-platform/pkg/metrics"
	"payment-platform/pkg/middleware"
	"payment-platform/pkg/outbox"
	"payment-platform/pkg/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	_ "payment-platform/cmd/wallet-service/docs"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg.LogLevel)

	cfg.OTel.Service = "wallet-service"
	otelShutdown, err := telemetry.Init(context.Background(), cfg.OTel)
	if err != nil {
		log.Warn("telemetry init failed â€” tracing disabled", "err", err)
	} else {
		defer otelShutdown(context.Background())
	}

	log.Info("wallet service starting")

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

	natsPublisher, _ := eventbus.NewPublisher(nc)
	repo := adapter.NewPostgresRepo(db)
	withdrawalRepo := adapter.NewPostgresWithdrawalRepo(db)
	pub := adapter.NewNATSPublisher(natsPublisher)
	svc := service.New(repo, pub, log, cfg.Platform.WalletUserID)
	withdrawalSvc := service.NewWithdrawalService(withdrawalRepo, repo, pub, log)

	sub, _ := eventbus.NewSubscriber(nc, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go outbox.NewRelay(db, natsPublisher, log).Run(ctx)

	sub.Subscribe(ctx, eventbus.SubscribeConfig{
		Stream:        "PAYMENTS",
		Subjects:      []string{"payments.*"},
		Consumer:      "wallet-service-payments",
		FilterSubject: eventbus.SubjectPaymentSucceeded,
	}, func(ctx context.Context, event eventbus.Event) error {
		var data eventbus.PaymentSucceededData
		if err := eventbus.DecodeData(event.Data, &data); err != nil {
			return err
		}
		return svc.HandlePaymentSucceeded(ctx, data)
	})

	sub.Subscribe(ctx, eventbus.SubscribeConfig{
		Stream:        "PAYMENTS",
		Subjects:      []string{"payments.*"},
		Consumer:      "wallet-service-refunds",
		FilterSubject: eventbus.SubjectPaymentRefunded,
	}, func(ctx context.Context, event eventbus.Event) error {
		var data eventbus.PaymentRefundedData
		if err := eventbus.DecodeData(event.Data, &data); err != nil {
			return err
		}
		log.Info("crediting wallet for refund",
			"order_id", data.OrderID,
			"amount", data.Amount,
		)
		return svc.HandlePaymentRefunded(ctx, data)
	})

	sub.Subscribe(ctx, eventbus.SubscribeConfig{
		Stream:        "ORDERS",
		Subjects:      []string{"orders.*"},
		Consumer:      "wallet-service-cancellations",
		FilterSubject: eventbus.SubjectOrderCancelled,
	}, func(ctx context.Context, event eventbus.Event) error {
		var data eventbus.OrderCancelledData
		if err := eventbus.DecodeData(event.Data, &data); err != nil {
			return err
		}
		return svc.HandleOrderCancelled(ctx, data)
	})

	router := gin.New()
	router.Use(otelgin.Middleware("wallet-service"))
	router.Use(gin.Recovery())
	router.Use(middleware.UserID())
	router.Use(middleware.Role())
	router.Use(middleware.RequestLogger(log))
	router.Use(metrics.HTTPMiddleware("wallet-service"))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/metrics", metrics.Handler())

	v1 := router.Group("/api/v1")
	wallethandler.New(svc).RegisterRoutes(v1)
	wallethandler.NewWithdrawalHandler(withdrawalSvc).RegisterRoutes(v1)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: router, ReadTimeout: 5 * time.Second}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http error", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("wallet service shutting down...")
	cancel()
	shutdownCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	srv.Shutdown(shutdownCtx)
}
