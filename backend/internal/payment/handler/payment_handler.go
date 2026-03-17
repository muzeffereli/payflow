package handler

import (
	"errors"
	"net/http"

	"payment-platform/internal/payment/domain"
	"payment-platform/internal/payment/service"

	"github.com/gin-gonic/gin"
)

type PaymentResponse struct {
	ID            string               `json:"id"                       example:"pay-uuid"`
	OrderID       string               `json:"order_id"                 example:"order-uuid"`
	UserID        string               `json:"user_id"                  example:"user-uuid"`
	Amount        int64                `json:"amount"                   example:"5000"`
	Currency      string               `json:"currency"                 example:"USD"`
	Status        domain.PaymentStatus `json:"status"                   example:"succeeded"`
	Method        string               `json:"method"                   example:"card"`
	TransactionID string               `json:"transaction_id,omitempty" example:"txn_abc123"`
	FailureReason string               `json:"failure_reason,omitempty" example:"card_declined"`
	CreatedAt     string               `json:"created_at"               example:"2026-01-01T00:00:00Z"`
	UpdatedAt     string               `json:"updated_at"               example:"2026-01-01T00:01:00Z"`
}

type PaymentErrorResponse struct {
	Error string `json:"error" example:"payment not found"`
}

type PaymentHandler struct {
	svc *service.PaymentService
}

func New(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

type RefundResponse struct {
	Message   string `json:"message"    example:"payment refunded"`
	PaymentID string `json:"payment_id" example:"pay-uuid"`
	OrderID   string `json:"order_id"   example:"ord-uuid"`
}

func (h *PaymentHandler) RegisterRoutes(rg *gin.RouterGroup) {
	payments := rg.Group("/payments")
	payments.GET("/:id", h.GetPayment)
	payments.GET("/order/:order_id", h.GetPaymentByOrder)
	payments.POST("/:id/refund", h.RefundPayment)
}

func (h *PaymentHandler) GetPayment(c *gin.Context) {
	p, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get payment"})
		return
	}
	c.JSON(http.StatusOK, toResponse(p))
}

func (h *PaymentHandler) GetPaymentByOrder(c *gin.Context) {
	p, err := h.svc.GetByOrderID(c.Request.Context(), c.Param("order_id"))
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found for this order"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get payment"})
		return
	}
	c.JSON(http.StatusOK, toResponse(p))
}

func (h *PaymentHandler) RefundPayment(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	p, err := h.svc.RefundPayment(c.Request.Context(), c.Param("id"), userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, RefundResponse{
		Message:   "payment refunded",
		PaymentID: p.ID,
		OrderID:   p.OrderID,
	})
}

func toResponse(p *domain.Payment) PaymentResponse {
	return PaymentResponse{
		ID:            p.ID,
		OrderID:       p.OrderID,
		UserID:        p.UserID,
		Amount:        p.Amount,
		Currency:      p.Currency,
		Status:        p.Status,
		Method:        p.Method,
		TransactionID: p.TransactionID,
		FailureReason: p.FailureReason,
		CreatedAt:     p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
