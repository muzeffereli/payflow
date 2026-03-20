package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payment-platform/internal/gateway/handler"
	"payment-platform/internal/gateway/middleware"
	"payment-platform/pkg/config"
	"payment-platform/pkg/logger"
	"payment-platform/pkg/metrics"
	"payment-platform/pkg/telemetry"

	_ "payment-platform/cmd/api-gateway/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg.LogLevel)

	cfg.OTel.Service = "api-gateway"
	otelShutdown, err := telemetry.Init(context.Background(), cfg.OTel)
	if err != nil {
		log.Warn("telemetry init failed â€” tracing disabled", "err", err)
	} else {
		defer otelShutdown(context.Background())
	}

	log.Info("api gateway starting", "port", cfg.Port)

	authAddr := cfg.Services.AuthAddr
	orderAddr := cfg.Services.OrderAddr
	paymentAddr := cfg.Services.PaymentAddr
	walletAddr := cfg.Services.WalletAddr
	productAddr := cfg.Services.ProductAddr
	cartAddr := cfg.Services.CartAddr
	fraudAddr := cfg.Services.FraudAddr
	notifAddr := cfg.Services.NotificationAddr
	storeAddr := cfg.Services.StoreAddr

	authProxy, err := handler.NewServiceProxy(authAddr)
	if err != nil {
		log.Error("auth proxy init failed", "err", err)
		os.Exit(1)
	}

	orderProxy, err := handler.NewServiceProxy(orderAddr)
	if err != nil {
		log.Error("order proxy init failed", "err", err)
		os.Exit(1)
	}

	paymentProxy, err := handler.NewServiceProxy(paymentAddr)
	if err != nil {
		log.Error("payment proxy init failed", "err", err)
		os.Exit(1)
	}

	walletProxy, err := handler.NewServiceProxy(walletAddr)
	if err != nil {
		log.Error("wallet proxy init failed", "err", err)
		os.Exit(1)
	}

	authDocProxy, err := handler.NewDocProxy(authAddr, "/docs/auth")
	if err != nil {
		log.Error("auth doc proxy init failed", "err", err)
		os.Exit(1)
	}
	orderDocProxy, err := handler.NewDocProxy(orderAddr, "/docs/orders")
	if err != nil {
		log.Error("order doc proxy init failed", "err", err)
		os.Exit(1)
	}
	paymentDocProxy, err := handler.NewDocProxy(paymentAddr, "/docs/payments")
	if err != nil {
		log.Error("payment doc proxy init failed", "err", err)
		os.Exit(1)
	}
	walletDocProxy, err := handler.NewDocProxy(walletAddr, "/docs/wallet")
	if err != nil {
		log.Error("wallet doc proxy init failed", "err", err)
		os.Exit(1)
	}

	productProxy, err := handler.NewServiceProxy(productAddr)
	if err != nil {
		log.Error("product proxy init failed", "err", err)
		os.Exit(1)
	}
	productDocProxy, err := handler.NewDocProxy(productAddr, "/docs/products")
	if err != nil {
		log.Error("product doc proxy init failed", "err", err)
		os.Exit(1)
	}

	cartProxy, err := handler.NewServiceProxy(cartAddr)
	if err != nil {
		log.Error("cart proxy init failed", "err", err)
		os.Exit(1)
	}

	fraudProxy, err := handler.NewServiceProxy(fraudAddr)
	if err != nil {
		log.Error("fraud proxy init failed", "err", err)
		os.Exit(1)
	}

	notifProxy, err := handler.NewServiceProxy(notifAddr)
	if err != nil {
		log.Error("notification proxy init failed", "err", err)
		os.Exit(1)
	}

	storeProxy, err := handler.NewServiceProxy(storeAddr)
	if err != nil {
		log.Error("store proxy init failed", "err", err)
		os.Exit(1)
	}
	storeDocProxy, err := handler.NewDocProxy(storeAddr, "/docs/stores")
	if err != nil {
		log.Error("store doc proxy init failed", "err", err)
		os.Exit(1)
	}

	router := gin.New()

	router.Use(
		otelgin.Middleware("api-gateway"),
		middleware.RequestID(),
		middleware.Logger(log),
		gin.Recovery(),
		metrics.HTTPMiddleware("api-gateway"),
		middleware.RateLimit(cfg.Redis, 100, time.Minute),
		middleware.Auth(cfg.JWT),
	)

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "api-gateway"})
	})
	router.GET("/metrics", metrics.Handler())

	authRateLimit := middleware.AuthRateLimit(cfg.Redis, 10, time.Minute)

	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", authRateLimit, authProxy.Forward)
		authGroup.POST("/login", authRateLimit, authProxy.Forward)
		authGroup.POST("/refresh", authProxy.Forward)
		authGroup.POST("/logout", authProxy.Forward)
		authGroup.GET("/me", authProxy.Forward)
		authGroup.POST("/change-password", authProxy.Forward)
	}

	idem := middleware.Idempotency(cfg.Redis)

	v1 := router.Group("/api/v1")
	{
		orders := v1.Group("/orders")
		{
			orders.POST("", idem, orderProxy.Forward)
			orders.GET("", orderProxy.Forward)
			orders.GET("/:id", orderProxy.Forward)
			orders.DELETE("/:id", orderProxy.Forward)
		}

		payments := v1.Group("/payments")
		{
			payments.GET("/:id", paymentProxy.Forward)
			payments.GET("/order/:order_id", paymentProxy.Forward)
			payments.POST("/:id/refund", paymentProxy.Forward)
		}

		wallet := v1.Group("/wallet")
		{
			wallet.POST("", idem, walletProxy.Forward)
			wallet.GET("", walletProxy.Forward)
			wallet.POST("/topup", walletProxy.Forward)
			wallet.GET("/transactions", walletProxy.Forward)
		}

		products := v1.Group("/products")
		{
			products.POST("", productProxy.Forward)
			products.GET("", productProxy.Forward)
			products.POST("/upload-image", productProxy.Forward)
			products.GET("/:id", productProxy.Forward)
			products.PATCH("/:id", productProxy.Forward)
			products.DELETE("/:id", productProxy.Forward)
			products.GET("/:id/variants", productProxy.Forward)
			products.POST("/:id/variants", productProxy.Forward)
			products.PATCH("/:id/variants/:vid", productProxy.Forward)
			products.DELETE("/:id/variants/:vid", productProxy.Forward)
		}

		adminAttrs := v1.Group("/admin/attributes")
		{
			adminAttrs.POST("", productProxy.Forward)
			adminAttrs.GET("", productProxy.Forward)
			adminAttrs.PATCH("/:id", productProxy.Forward)
			adminAttrs.DELETE("/:id", productProxy.Forward)
		}
		adminCategories := v1.Group("/admin/categories")
		{
			adminCategories.POST("", productProxy.Forward)
			adminCategories.GET("", productProxy.Forward)
			adminCategories.PATCH("/:id", productProxy.Forward)
			adminCategories.DELETE("/:id", productProxy.Forward)
		}
		adminSubcategories := v1.Group("/admin/subcategories")
		{
			adminSubcategories.POST("", productProxy.Forward)
			adminSubcategories.PATCH("/:id", productProxy.Forward)
			adminSubcategories.DELETE("/:id", productProxy.Forward)
		}
		v1.GET("/attributes", productProxy.Forward)
		v1.GET("/categories", productProxy.Forward)
		v1.GET("/categories/:id/subcategories", productProxy.Forward)

		cart := v1.Group("/cart")
		{
			cart.POST("/items", cartProxy.Forward)
			cart.PATCH("/items/:product_id", cartProxy.Forward)
			cart.DELETE("/items/:product_id", cartProxy.Forward)
			cart.GET("", cartProxy.Forward)
			cart.POST("/checkout", idem, cartProxy.Forward)
		}

		notifs := v1.Group("/notifications")
		{
			notifs.GET("", notifProxy.Forward)
			notifs.GET("/unread-count", notifProxy.Forward)
			notifs.PATCH("/:id/read", notifProxy.Forward)
			notifs.POST("/read-all", notifProxy.Forward)
		}

		v1.GET("/seller/orders", orderProxy.Forward)
		v1.GET("/seller/analytics", orderProxy.Forward)

		v1.POST("/seller/withdrawals", walletProxy.Forward)
		v1.GET("/seller/withdrawals", walletProxy.Forward)

		adminWithdrawals := v1.Group("/admin/withdrawals")
		{
			adminWithdrawals.GET("", walletProxy.Forward)
			adminWithdrawals.POST("/:id/approve", walletProxy.Forward)
			adminWithdrawals.POST("/:id/reject", walletProxy.Forward)
		}

		adminUsers := v1.Group("/admin/users")
		{
			adminUsers.GET("", authProxy.Forward)
			adminUsers.PATCH("/:id/role", authProxy.Forward)
		}

		adminFraud := v1.Group("/admin/fraud-checks")
		{
			adminFraud.GET("", fraudProxy.Forward)
			adminFraud.GET("/:id", fraudProxy.Forward)
		}

		stores := v1.Group("/stores")
		{
			stores.POST("", storeProxy.Forward)
			stores.GET("", storeProxy.Forward)
			stores.GET("/me", storeProxy.Forward)
			stores.GET("/:id", storeProxy.Forward)
			stores.PATCH("/:id", storeProxy.Forward)
			stores.DELETE("/:id", storeProxy.Forward)
			stores.POST("/:id/approve", storeProxy.Forward)
			stores.POST("/:id/suspend", storeProxy.Forward)
			stores.POST("/:id/reactivate", storeProxy.Forward)
			stores.GET("/:id/products", func(c *gin.Context) {
				q := c.Request.URL.Query()
				q.Set("store_id", c.Param("id"))
				c.Request.URL.Path = "/api/v1/products"
				c.Request.URL.RawQuery = q.Encode()
				productProxy.Forward(c)
			})
		}
	}

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.GET("/docs", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"swagger_docs": gin.H{
				"gateway":  "http://localhost:8080/swagger/index.html",
				"auth":     "http://localhost:8080/docs/auth/index.html",
				"orders":   "http://localhost:8080/docs/orders/index.html",
				"payments": "http://localhost:8080/docs/payments/index.html",
				"wallet":   "http://localhost:8080/docs/wallet/index.html",
				"products": "http://localhost:8080/docs/products/index.html",
				"stores":   "http://localhost:8080/docs/stores/index.html",
			},
		})
	})
	router.GET("/docs/auth/*any", authDocProxy.Forward)
	router.GET("/docs/orders/*any", orderDocProxy.Forward)
	router.GET("/docs/payments/*any", paymentDocProxy.Forward)
	router.GET("/docs/wallet/*any", walletDocProxy.Forward)
	router.GET("/docs/products/*any", productDocProxy.Forward)
	router.GET("/docs/stores/*any", storeDocProxy.Forward)

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

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("api gateway shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", "err", err)
	}

	log.Info("api gateway stopped")
}

