package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"payment-platform/internal/wallet/domain"
	"payment-platform/internal/wallet/service"
	"payment-platform/pkg/middleware"
)

type WithdrawalHandler struct {
	svc *service.WithdrawalService
}

func NewWithdrawalHandler(svc *service.WithdrawalService) *WithdrawalHandler {
	return &WithdrawalHandler{svc: svc}
}

func (h *WithdrawalHandler) RegisterRoutes(rg *gin.RouterGroup) {
	seller := rg.Group("/seller/withdrawals", middleware.RequireRole("seller", "admin"))
	{
		seller.POST("", h.RequestWithdrawal)
		seller.GET("", h.ListMyWithdrawals)
	}

	admin := rg.Group("/admin/withdrawals", middleware.RequireRole("admin"))
	{
		admin.GET("", h.ListPendingWithdrawals)
		admin.POST("/:id/approve", h.ApproveWithdrawal)
		admin.POST("/:id/reject", h.RejectWithdrawal)
	}
}

type WithdrawalRequest struct {
	StoreID  string `json:"store_id"  binding:"required" example:"store-uuid"`
	Amount   int64  `json:"amount"    binding:"required,min=1" example:"50000"` // cents
	Currency string `json:"currency"  binding:"required,len=3" example:"USD"`
	Method   string `json:"method"    example:"bank_transfer"`
}

type RejectRequest struct {
	Reason string `json:"reason" binding:"required" example:"Insufficient documentation"`
}

func (h *WithdrawalHandler) RequestWithdrawal(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var body WithdrawalRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	w, err := h.svc.RequestWithdrawal(c.Request.Context(),
		userID, body.StoreID, body.Currency, body.Method, body.Amount)
	if err != nil {
		if errors.Is(err, service.ErrInsufficientBalance) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, w)
}

func (h *WithdrawalHandler) ListMyWithdrawals(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	list, err := h.svc.ListMyWithdrawals(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list withdrawals"})
		return
	}
	if list == nil {
		list = []*domain.Withdrawal{}
	}
	c.JSON(http.StatusOK, gin.H{"withdrawals": list, "limit": limit, "offset": offset})
}

func (h *WithdrawalHandler) ListPendingWithdrawals(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	list, err := h.svc.ListPendingWithdrawals(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list withdrawals"})
		return
	}
	if list == nil {
		list = []*domain.Withdrawal{}
	}
	c.JSON(http.StatusOK, gin.H{"withdrawals": list, "limit": limit, "offset": offset})
}

func (h *WithdrawalHandler) ApproveWithdrawal(c *gin.Context) {
	w, err := h.svc.ApproveWithdrawal(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrWithdrawalNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "withdrawal not found"})
		case errors.Is(err, service.ErrInsufficientBalance):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, w)
}

func (h *WithdrawalHandler) RejectWithdrawal(c *gin.Context) {
	var body RejectRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	w, err := h.svc.RejectWithdrawal(c.Request.Context(), c.Param("id"), body.Reason)
	if err != nil {
		if errors.Is(err, domain.ErrWithdrawalNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "withdrawal not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, w)
}
