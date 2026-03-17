package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payment-platform/internal/product/adapter"
	producthandler "payment-platform/internal/product/handler"
	"payment-platform/internal/product/service"
	"payment-platform/pkg/config"
	"payment-platform/pkg/database"
	"payment-platform/pkg/eventbus"
	"payment-platform/pkg/logger"
	"payment-platform/pkg/metrics"
	"payment-platform/pkg/middleware"
	"payment-platform/pkg/telemetry"

	miniogo "github.com/minio/minio-go/v7"
	miniocreds "github.com/minio/minio-go/v7/pkg/credentials"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	_ "payment-platform/cmd/product-service/docs"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg.LogLevel)

	cfg.OTel.Service = "product-service"
	otelShutdown, err := telemetry.Init(context.Background(), cfg.OTel)
	if err != nil {
		log.Warn("telemetry init failed â€” tracing disabled", "err", err)
	} else {
		defer otelShutdown(context.Background())
	}

	log.Info("product service starting", "port", cfg.Port)

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

	storeAddr := envOrDefault("STORE_SERVICE_ADDR", "http://localhost:8089")

	minioEndpoint := envOrDefault("MINIO_ENDPOINT", "localhost:9000")
	minioAccess := envOrDefault("MINIO_ACCESS_KEY", "minioadmin")
	minioSecret := envOrDefault("MINIO_SECRET_KEY", "minioadmin")
	minioBucket := envOrDefault("MINIO_BUCKET", "products")
	minioPublicURL := envOrDefault("MINIO_PUBLIC_URL", "http://localhost:9000")
	minioUseSSL := envOrDefault("MINIO_USE_SSL", "false") == "true"

	minioClient, err := miniogo.New(minioEndpoint, &miniogo.Options{
		Creds:  miniocreds.NewStaticV4(minioAccess, minioSecret, ""),
		Secure: minioUseSSL,
	})
	if err != nil {
		log.Warn("minio client init failed â€” image uploads disabled", "err", err)
		minioClient = nil
	} else {
		log.Info("connected to minio", "endpoint", minioEndpoint)
	}

	repo := adapter.NewPostgresRepo(db)
	reservationRepo := adapter.NewPostgresReservationRepo(db)
	attrRepo := adapter.NewPostgresAttributeRepo(db)
	variantRepo := adapter.NewPostgresVariantRepo(db)
	globalAttrRepo := adapter.NewPostgresGlobalAttributeRepo(db)
	imageRepo := adapter.NewPostgresImageRepo(db)
	pub := adapter.NewNATSPublisher(natsPublisher)
	storeClient := adapter.NewHTTPStoreClient(storeAddr)
	svc := service.New(repo, reservationRepo, attrRepo, variantRepo, imageRepo, pub, storeClient, log)

	sub, err := eventbus.NewSubscriber(nc, log)
	if err != nil {
		log.Error("failed to create subscriber", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sub.Subscribe(ctx, eventbus.SubscribeConfig{
		Stream:        "ORDERS",
		Subjects:      []string{"orders.*"},
		Consumer:      "product-service-orders",
		FilterSubject: eventbus.SubjectOrderCreated,
	}, func(ctx context.Context, event eventbus.Event) error {
		var data eventbus.OrderCreatedData
		if err := eventbus.DecodeData(event.Data, &data); err != nil {
			return err
		}
		items := make([]service.StockItem, len(data.Items))
		for i, it := range data.Items {
			items[i] = service.StockItem{ProductID: it.ProductID, VariantID: it.VariantID, Quantity: it.Quantity}
		}
		if err := svc.ReserveStock(ctx, data.OrderID, items); err != nil {
			log.Error("stock reservation failed", "order_id", data.OrderID, "err", err)
			return err
		}
		return nil
	}); err != nil {
		log.Error("failed to subscribe to orders.created", "err", err)
		os.Exit(1)
	}

	if err := sub.Subscribe(ctx, eventbus.SubscribeConfig{
		Stream:        "PAYMENTS",
		Subjects:      []string{"payments.*"},
		Consumer:      "product-service-payment-succeeded",
		FilterSubject: eventbus.SubjectPaymentSucceeded,
	}, func(ctx context.Context, event eventbus.Event) error {
		var data eventbus.PaymentSucceededData
		if err := eventbus.DecodeData(event.Data, &data); err != nil {
			return err
		}
		return svc.CommitReservation(ctx, data.OrderID)
	}); err != nil {
		log.Error("failed to subscribe to payments.succeeded", "err", err)
		os.Exit(1)
	}

	if err := sub.Subscribe(ctx, eventbus.SubscribeConfig{
		Stream:        "PAYMENTS",
		Subjects:      []string{"payments.*"},
		Consumer:      "product-service-payment-failed",
		FilterSubject: eventbus.SubjectPaymentFailed,
	}, func(ctx context.Context, event eventbus.Event) error {
		var data eventbus.PaymentFailedData
		if err := eventbus.DecodeData(event.Data, &data); err != nil {
			return err
		}
		log.Info("payment failed â€” releasing reserved stock", "order_id", data.OrderID)
		return svc.ReleaseReservation(ctx, data.OrderID, "payment_failed")
	}); err != nil {
		log.Error("failed to subscribe to payments.failed", "err", err)
		os.Exit(1)
	}

	if err := sub.Subscribe(ctx, eventbus.SubscribeConfig{
		Stream:        "ORDERS",
		Subjects:      []string{"orders.*"},
		Consumer:      "product-service-cancellations",
		FilterSubject: eventbus.SubjectOrderCancelled,
	}, func(ctx context.Context, event eventbus.Event) error {
		var data eventbus.OrderCancelledData
		if err := eventbus.DecodeData(event.Data, &data); err != nil {
			return err
		}
		log.Info("order cancelled â€” releasing reserved stock", "order_id", data.OrderID)
		return svc.ReleaseReservation(ctx, data.OrderID, "order_cancelled")
	}); err != nil {
		log.Error("failed to subscribe to orders.cancelled", "err", err)
		os.Exit(1)
	}

	router := gin.New()
	router.Use(otelgin.Middleware("product-service"))
	router.Use(gin.Recovery())
	router.Use(middleware.UserID())
	router.Use(middleware.Role())
	router.Use(middleware.RequestLogger(log))
	router.Use(metrics.HTTPMiddleware("product-service"))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "product-service"})
	})
	router.GET("/metrics", metrics.Handler())

	v1 := router.Group("/api/v1")
	producthandler.New(svc).RegisterRoutes(v1)

	attrHandler := producthandler.NewAttributeHandler(globalAttrRepo)
	adminAttrs := v1.Group("/admin/attributes", middleware.RequireRole("admin"))
	publicAttrs := v1.Group("/attributes")
	attrHandler.RegisterRoutes(adminAttrs, publicAttrs)

	if minioClient != nil {
		uploadHandler := producthandler.NewUploadHandler(minioClient, minioBucket, minioPublicURL)
		sellerGroup := v1.Group("", middleware.RequireRole("seller"))
		uploadHandler.RegisterRoutes(sellerGroup)
	}

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

	log.Info("product service shutting down...")
	cancel()

	shutdownCtx, done := context.WithTimeout(context.Background(), 15*time.Second)
	defer done()
	srv.Shutdown(shutdownCtx)
	log.Info("product service stopped")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
