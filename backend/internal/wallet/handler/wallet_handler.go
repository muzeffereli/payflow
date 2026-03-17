package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"payment-platform/internal/wallet/domain"
	"payment-platform/internal/wallet/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreateWalletRequest struct {
	Currency string `json:"currency" binding:"required" example:"USD"`
}

type WalletErrorResponse struct {
	Error string `json:"error" example:"wallet not found"`
}

type WalletHandler struct {
	svc *service.WalletService
}

func New(svc *service.WalletService) *WalletHandler {
	return &WalletHandler{svc: svc}
}

type TopUpRequest struct {
	Amount int64 `json:"amount" binding:"required,min=1" example:"5000"`
}

type DebitPaymentRequest struct {
	Amount      int64  `json:"amount" binding:"required,min=1" example:"5000"`
	ReferenceID string `json:"reference_id" binding:"required" example:"pay-uuid"`
}

type DebitPaymentResponse struct {
	TransactionID string `json:"transaction_id" example:"txn-uuid"`
}

func (h *WalletHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/wallet", h.GetWallet)
	rg.POST("/wallet", h.CreateWallet)
	rg.POST("/wallet/topup", h.TopUp)
	rg.POST("/wallet/payments/debit", h.DebitPayment)
	rg.GET("/wallet/transactions", h.ListTransactions)
}

func (h *WalletHandler) GetWallet(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	wallet, err := h.svc.GetWallet(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *WalletHandler) CreateWallet(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var body CreateWalletRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := h.svc.CreateWallet(c.Request.Context(), userID, body.Currency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create wallet"})
		return
	}

	c.JSON(http.StatusCreated, wallet)
}

func (h *WalletHandler) TopUp(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req TopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	refID := fmt.Sprintf("topup-%s", uuid.New().String())
	if err := h.svc.Credit(c.Request.Context(), userID, req.Amount, "topup", refID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "top-up failed"})
		return
	}

	wallet, err := h.svc.GetWallet(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch wallet"})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *WalletHandler) DebitPayment(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req DebitPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := h.svc.DebitForPayment(c.Request.Context(), userID, req.Amount, req.ReferenceID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		case errors.Is(err, domain.ErrInsufficientFunds):
			c.JSON(http.StatusConflict, gin.H{"error": "insufficient funds"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "wallet debit failed"})
		}
		return
	}

	c.JSON(http.StatusOK, DebitPaymentResponse{TransactionID: tx.ID})
}

func (h *WalletHandler) ListTransactions(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	txns, total, err := h.svc.ListTransactions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": txns,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}
