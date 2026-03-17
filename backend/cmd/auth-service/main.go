package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payment-platform/internal/auth/adapter"
	authhandler "payment-platform/internal/auth/handler"
	"payment-platform/internal/auth/service"
	pkgauth "payment-platform/pkg/auth"
	"payment-platform/pkg/config"
	"payment-platform/pkg/database"
	"payment-platform/pkg/eventbus"
	"payment-platform/pkg/logger"
	"payment-platform/pkg/metrics"
	"payment-platform/pkg/outbox"
	"payment-platform/pkg/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	_ "payment-platform/cmd/auth-service/docs"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg.LogLevel)

	cfg.OTel.Service = "auth-service"
	otelShutdown, err := telemetry.Init(context.Background(), cfg.OTel)
	if err != nil {
		log.Warn("telemetry init failed â€” tracing disabled", "err", err)
	} else {
		defer otelShutdown(context.Background())
	}

	log.Info("auth service starting", "port", cfg.Port)

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

	jwtManager := pkgauth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.Issuer, 0)

	repo := adapter.NewPostgresRepo(db)
	pub := adapter.NewNATSPublisher(natsPublisher)
	svc := service.New(repo, pub, jwtManager, log)
	h := authhandler.New(svc)

	router := gin.New()
	router.Use(otelgin.Middleware("auth-service"))
	router.Use(gin.Recovery())
	router.Use(metrics.HTTPMiddleware("auth-service"))

	auth := router.Group("/auth")
	h.RegisterRoutes(auth)

	admin := router.Group("/api/v1/admin")
	h.RegisterAdminRoutes(admin)

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "auth-service"})
	})
	router.GET("/metrics", metrics.Handler())

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	for _, route := range router.Routes() {
		log.Info("route registered", "method", route.Method, "path", route.Path)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go outbox.NewRelay(db, natsPublisher, log).Run(ctx)

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

	log.Info("auth service shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("forced shutdown", "err", err)
	}

	log.Info("auth service stopped")
}
