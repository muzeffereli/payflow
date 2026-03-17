package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	fraudadapter "payment-platform/internal/fraud/adapter"
	fraudhandler "payment-platform/internal/fraud/handler"
	"payment-platform/internal/fraud/service"
	"payment-platform/pkg/config"
	"payment-platform/pkg/database"
	"payment-platform/pkg/eventbus"
	"payment-platform/pkg/logger"
	"payment-platform/pkg/metrics"
	"payment-platform/pkg/middleware"
	"payment-platform/pkg/telemetry"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg.LogLevel)

	cfg.OTel.Service = "fraud-service"
	otelShutdown, err := telemetry.Init(context.Background(), cfg.OTel)
	if err != nil {
		log.Warn("telemetry init failed â€” tracing disabled", "err", err)
	} else {
		defer otelShutdown(context.Background())
	}

	log.Info("fraud service starting")

	db, err := database.Connect(cfg.DB)
	if err != nil {
		log.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("connected to postgres")

	fraudRepo := fraudadapter.NewPostgresFraudRepo(db)

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

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})
	defer rdb.Close()

	rules := []service.Rule{
		&service.HighAmountRule{ThresholdCents: 50_000},
		service.NewVelocityRule(rdb, 60, 3),
		service.NewSuspiciousCountryRule([]string{"KP", "IR", "CU", "SY"}),
		&service.RoundAmountRule{},
	}

	natsPublisher, _ := eventbus.NewPublisher(nc)
	pub := fraudadapter.NewNATSPublisher(natsPublisher)
	svc := service.NewWithRepo(pub, fraudRepo, log, rules...)

	sub, _ := eventbus.NewSubscriber(nc, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sub.Subscribe(ctx, eventbus.SubscribeConfig{
		Stream:        "PAYMENTS",
		Subjects:      []string{"payments.*"},
		Consumer:      "fraud-service-payments",
		FilterSubject: eventbus.SubjectPaymentInitiated,
	}, func(ctx context.Context, event eventbus.Event) error {
		var data eventbus.PaymentInitiatedData
		if err := eventbus.DecodeData(event.Data, &data); err != nil {
			return err
		}
		return svc.HandlePaymentInitiated(ctx, data)
	}); err != nil {
		log.Error("subscribe failed", "err", err)
		os.Exit(1)
	}

	log.Info("fraud service ready", "rules", len(rules))

	router := gin.New()
	router.Use(otelgin.Middleware("fraud-service"))
	router.Use(gin.Recovery())
	router.Use(middleware.UserID())
	router.Use(middleware.Role())
	router.Use(middleware.RequestLogger(log))
	router.Use(metrics.HTTPMiddleware("fraud-service"))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "fraud-service"})
	})
	router.GET("/metrics", metrics.Handler())

	v1 := router.Group("/api/v1")
	fraudhandler.New(fraudRepo).RegisterRoutes(v1)

	srv := &http.Server{Addr: ":8085", Handler: router}
	go func() {
		log.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("fraud service shutting down...")
	cancel()
	shutdownCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	srv.Shutdown(shutdownCtx)
}
