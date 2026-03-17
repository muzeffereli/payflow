package handler

import (
	"errors"
	"net/http"
	"strconv"

	"payment-platform/internal/order/domain"
	"payment-platform/internal/order/service"

	"github.com/gin-gonic/gin"
)

type CancelOrderResponse struct {
	Message string `json:"message" example:"order cancelled"`
	OrderID string `json:"order_id" example:"ord-abc-123"`
}

type ShippingAddressBody struct {
	Name       string `json:"name"        binding:"required" example:"Alice Smith"`
	Street     string `json:"street"      binding:"required" example:"123 Main St"`
	City       string `json:"city"        binding:"required" example:"New York"`
	State      string `json:"state"       example:"NY"`
	PostalCode string `json:"postal_code" binding:"required" example:"10001"`
	Country    string `json:"country"     binding:"required" example:"US"`
}

type CreateOrderBody struct {
	Currency        string                `json:"currency"          binding:"required" example:"USD"`
	PaymentMethod   string                `json:"payment_method,omitempty" binding:"omitempty,oneof=card wallet" example:"card"`
	Items           []CreateOrderItemBody `json:"items"            binding:"required,min=1"`
	ShippingAddress *ShippingAddressBody  `json:"shipping_address"`
	StoreID         *string               `json:"store_id,omitempty" example:"store-uuid"` // set by cart service for store sub-orders
}

type CreateOrderItemBody struct {
	ProductID string  `json:"product_id" binding:"required" example:"prod-abc-123"`
	VariantID *string `json:"variant_id,omitempty" example:"variant-abc-123"`
	Quantity  int     `json:"quantity"   binding:"required,min=1" example:"2"`
}

type ListOrdersResponse struct {
	Orders []domain.Order `json:"orders"`
	Limit  int            `json:"limit"  example:"20"`
	Offset int            `json:"offset" example:"0"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"something went wrong"`
}

type OrderHandler struct {
	svc *service.OrderService
}

func New(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) RegisterRoutes(rg *gin.RouterGroup) {
	orders := rg.Group("/orders")
	orders.POST("", h.CreateOrder)
	orders.GET("/:id", h.GetOrder)
	orders.GET("", h.ListOrders)
	orders.DELETE("/:id", h.CancelOrder)

	rg.GET("/seller/orders", h.ListSellerOrders)
	rg.GET("/seller/analytics", h.GetSellerAnalytics)
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var body CreateOrderBody

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Idempotency-Key header is required"})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	items := make([]service.OrderItemInput, len(body.Items))
	for i, it := range body.Items {
		items[i] = service.OrderItemInput{ProductID: it.ProductID, VariantID: it.VariantID, Quantity: it.Quantity}
	}

	var addr *domain.ShippingAddress
	if body.ShippingAddress != nil {
		addr = &domain.ShippingAddress{
			Name:       body.ShippingAddress.Name,
			Street:     body.ShippingAddress.Street,
			City:       body.ShippingAddress.City,
			State:      body.ShippingAddress.State,
			PostalCode: body.ShippingAddress.PostalCode,
			Country:    body.ShippingAddress.Country,
		}
	}

	order, err := h.svc.CreateOrder(c.Request.Context(), service.CreateOrderRequest{
		UserID:          userID,
		Currency:        body.Currency,
		PaymentMethod:   defaultPaymentMethod(body.PaymentMethod),
		IdempotencyKey:  idempotencyKey,
		Items:           items,
		ShippingAddress: addr,
		StoreID:         body.StoreID,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidProduct),
			errors.Is(err, service.ErrInsufficientStock),
			errors.Is(err, service.ErrCurrencyMismatch):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
		}
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetString("user_id")

	order, err := h.svc.GetOrder(c.Request.Context(), orderID, userID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		if errors.Is(err, service.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get order"})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	userID := c.GetString("user_id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	orders, err := h.svc.ListOrders(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders, "limit": limit, "offset": offset})
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.svc.CancelOrder(c.Request.Context(), orderID, userID)
	if err != nil {
		var transErr *domain.InvalidTransitionError
		switch {
		case errors.Is(err, service.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.As(err, &transErr):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel order"})
		}
		return
	}

	c.JSON(http.StatusOK, CancelOrderResponse{Message: "order cancelled", OrderID: orderID})
}

func (h *OrderHandler) ListSellerOrders(c *gin.Context) {
	storeID := c.Query("store_id")
	if storeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "store_id query parameter is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit > 100 {
		limit = 100
	}

	orders, err := h.svc.ListStoreOrders(c.Request.Context(), storeID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list store orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders, "limit": limit, "offset": offset})
}

func (h *OrderHandler) GetSellerAnalytics(c *gin.Context) {
	storeID := c.Query("store_id")
	if storeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "store_id query parameter is required"})
		return
	}

	analytics, err := h.svc.GetStoreAnalytics(c.Request.Context(), storeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get analytics"})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func defaultPaymentMethod(method string) string {
	if method == "" {
		return "card"
	}
	return method
}
