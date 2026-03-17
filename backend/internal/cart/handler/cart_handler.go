package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"payment-platform/internal/cart/domain"
	"payment-platform/internal/cart/service"
)

type CartHandler struct {
	svc *service.CartService
}

func New(svc *service.CartService) *CartHandler {
	return &CartHandler{svc: svc}
}

func (h *CartHandler) RegisterRoutes(r gin.IRouter) {
	r.POST("/items", h.AddItem)
	r.PATCH("/items/:product_id", h.UpdateQuantity)
	r.DELETE("/items/:product_id", h.RemoveItem)
	r.GET("", h.GetCart)
	r.POST("/checkout", h.Checkout)
}

type AddItemRequest struct {
	ProductID string  `json:"product_id" binding:"required"`
	VariantID *string `json:"variant_id,omitempty"`
	Quantity  int     `json:"quantity"   binding:"required,min=1"`
}

type UpdateQuantityRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1"`
}

type CheckoutRequest struct {
	Currency       string `json:"currency"        binding:"omitempty,len=3"`
	PaymentMethod  string `json:"payment_method"  binding:"omitempty,oneof=card wallet"`
	IdempotencyKey string `json:"idempotency_key"`
}

type CheckoutResponse struct {
	OrderIDs []string `json:"order_ids"`
}

func (h *CartHandler) AddItem(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
		return
	}

	var req AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cart, err := h.svc.AddItem(c.Request.Context(), service.AddItemRequest{
		UserID:    userID,
		ProductID: req.ProductID,
		VariantID: normalizeVariantID(req.VariantID),
		Quantity:  req.Quantity,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidProduct) || errors.Is(err, service.ErrVariantRequired) || errors.Is(err, service.ErrInsufficientStock) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cart)
}

func (h *CartHandler) RemoveItem(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
		return
	}

	productID := c.Param("product_id")
	cart, err := h.svc.RemoveItem(c.Request.Context(), userID, productID, normalizeVariantID(optionalQueryParam(c, "variant_id")))
	if err != nil {
		if errors.Is(err, domain.ErrItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cart)
}

func (h *CartHandler) UpdateQuantity(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
		return
	}

	productID := c.Param("product_id")
	variantID := normalizeVariantID(optionalQueryParam(c, "variant_id"))
	var req UpdateQuantityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cart, err := h.svc.SetQuantity(c.Request.Context(), userID, productID, variantID, req.Quantity)
	if err != nil {
		if errors.Is(err, domain.ErrItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrInvalidProduct) || errors.Is(err, service.ErrVariantRequired) || errors.Is(err, service.ErrInsufficientStock) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cart)
}

func (h *CartHandler) GetCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
		return
	}

	view, err := h.svc.GetCart(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *CartHandler) Checkout(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
		return
	}

	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Currency == "" {
		req.Currency = "USD"
	}

	if req.IdempotencyKey == "" {
		req.IdempotencyKey = c.GetHeader("Idempotency-Key")
	}

	result, err := h.svc.Checkout(c.Request.Context(), service.CheckoutRequest{
		UserID:         userID,
		Currency:       req.Currency,
		PaymentMethod:  cartPaymentMethod(req.PaymentMethod),
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCartEmpty):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "cart is empty"})
		case errors.Is(err, service.ErrInvalidProduct), errors.Is(err, service.ErrInsufficientStock):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, CheckoutResponse{OrderIDs: result.OrderIDs})
}

func optionalQueryParam(c *gin.Context, key string) *string {
	value := c.Query(key)
	if value == "" {
		return nil
	}
	return &value
}

func normalizeVariantID(variantID *string) *string {
	if variantID == nil {
		return nil
	}
	if *variantID == "" {
		return nil
	}
	return variantID
}

func cartPaymentMethod(method string) string {
	if method == "" {
		return "card"
	}
	return method
}
