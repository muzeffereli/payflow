package handler

import (
	"net/http"
	"strconv"

	"payment-platform/internal/fraud/port"
	"payment-platform/pkg/middleware"

	"github.com/gin-gonic/gin"
)

type FraudHandler struct {
	repo port.FraudCheckRepository
}

func New(repo port.FraudCheckRepository) *FraudHandler {
	return &FraudHandler{repo: repo}
}

func (h *FraudHandler) RegisterRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/admin/fraud-checks", middleware.RequireRole("admin"))
	admin.GET("", h.List)
	admin.GET("/:id", h.GetByID)
}

func (h *FraudHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	decision := c.Query("decision")

	checks, total, err := h.repo.List(c.Request.Context(), decision, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list fraud checks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fraud_checks": checks,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}

func (h *FraudHandler) GetByID(c *gin.Context) {
	fc, err := h.repo.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "fraud check not found"})
		return
	}
	c.JSON(http.StatusOK, fc)
}
