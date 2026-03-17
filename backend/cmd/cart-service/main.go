package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payment-platform/internal/cart/adapter"
	carthandler "payment-platform/internal/cart/handler"
	"payment-platform/internal/cart/service"
	"payment-platform/pkg/config"
	"payment-platform/pkg/logger"
	"payment-platform/pkg/metrics"
	"payment-platform/pkg/middleware"
	"payment-platform/pkg/telemetry"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg.LogLevel)

	cfg.OTel.Service = "cart-service"
	otelShutdown, err := telemetry.Init(context.Background(), cfg.OTel)
	if err != nil {
		log.Warn("telemetry init failed â€” tracing disabled", "err", err)
	} else {
		defer otelShutdown(context.Background())
	}

	log.Info("cart service starting", "port", cfg.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Error("failed to connect to redis", "err", err)
		os.Exit(1)
	}
	log.Info("connected to redis")

	orderAddr := envOrDefault("ORDER_SERVICE_ADDR", "http://localhost:8081")
	productAddr := envOrDefault("PRODUCT_SERVICE_ADDR", "http://localhost:8087")

	repo := adapter.NewRedisRepo(rdb)
	productClient := adapter.NewHTTPProductClient(productAddr)
	orderClient := adapter.NewHTTPOrderClient(orderAddr)
	svc := service.New(repo, productClient, orderClient, log)

	router := gin.New()
	router.Use(otelgin.Middleware("cart-service"))
	router.Use(gin.Recovery())
	router.Use(middleware.UserID())
	router.Use(middleware.RequestLogger(log))
	router.Use(metrics.HTTPMiddleware("cart-service"))

	v1 := router.Group("/api/v1")
	carthandler.New(svc).RegisterRoutes(v1.Group("/cart"))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "cart-service"})
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

	log.Info("cart service shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("forced shutdown", "err", err)
	}

	log.Info("cart service stopped")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
