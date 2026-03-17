package handler

import (
	"errors"
	"net/http"
	"strconv"

	"payment-platform/internal/store/domain"
	"payment-platform/internal/store/service"
	"payment-platform/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func userID(c *gin.Context) string   { return c.GetString("user_id") }
func userRole(c *gin.Context) string { return c.GetString("user_role") }

type CreateStoreRequest struct {
	Name        string `json:"name"        binding:"required" example:"Alice's Gadgets"`
	Description string `json:"description"                    example:"The best gadgets online"`
	Email       string `json:"email"                          example:"contact@alices.shop"`
	Commission  int    `json:"commission"                     example:"10"` // percentage, 0 = default 10%
}

type UpdateStoreRequest struct {
	Name        string `json:"name"        binding:"required" example:"Alice's Gadgets Pro"`
	Description string `json:"description"                    example:"Updated description"`
	Email       string `json:"email"                          example:"hello@alices.shop"`
}

type ListStoresResponse struct {
	Stores []*domain.Store `json:"stores"`
	Total  int             `json:"total"  example:"5"`
	Limit  int             `json:"limit"  example:"20"`
	Offset int             `json:"offset" example:"0"`
}

type StoreErrorResponse struct {
	Error string `json:"error" example:"store not found"`
}

type StoreHandler struct {
	svc *service.StoreService
}

func New(svc *service.StoreService) *StoreHandler {
	return &StoreHandler{svc: svc}
}

func (h *StoreHandler) RegisterRoutes(rg *gin.RouterGroup) {
	stores := rg.Group("/stores")

	stores.GET("", h.ListStores)
	stores.GET("/me", middleware.RequireRole("seller"), h.GetMyStore)
	stores.GET("/:id", h.GetStore)

	seller := stores.Group("", middleware.RequireRole("seller"))
	seller.POST("", h.CreateStore)
	seller.PATCH("/:id", h.UpdateStore)
	seller.DELETE("/:id", h.CloseStore)

	admin := stores.Group("", middleware.RequireRole("admin"))
	admin.POST("/:id/approve", h.ApproveStore)
	admin.POST("/:id/suspend", h.SuspendStore)
	admin.POST("/:id/reactivate", h.ReactivateStore)
}

func (h *StoreHandler) CreateStore(c *gin.Context) {
	var req CreateStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	callerID := userID(c)
	store, err := h.svc.CreateStore(c.Request.Context(), callerID, req.Name, req.Description, req.Email, req.Commission)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrOwnerAlreadyHasStore):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, store)
}

func (h *StoreHandler) GetStore(c *gin.Context) {
	store, err := h.svc.GetStore(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrStoreNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, store)
}

func (h *StoreHandler) GetMyStore(c *gin.Context) {
	callerID := userID(c)
	store, err := h.svc.GetMyStore(c.Request.Context(), callerID)
	if err != nil {
		if errors.Is(err, domain.ErrStoreNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "you do not have a store yet"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, store)
}

func (h *StoreHandler) ListStores(c *gin.Context) {
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	stores, total, err := h.svc.ListStores(c.Request.Context(), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, ListStoresResponse{
		Stores: stores,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *StoreHandler) UpdateStore(c *gin.Context) {
	var req UpdateStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	callerID := userID(c)
	store, err := h.svc.UpdateStore(c.Request.Context(), c.Param("id"), callerID, req.Name, req.Description, req.Email)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrStoreNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrNotStoreOwner):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, store)
}

func (h *StoreHandler) CloseStore(c *gin.Context) {
	callerID := userID(c)
	callerRole := userRole(c)
	store, err := h.svc.CloseStore(c.Request.Context(), c.Param("id"), callerID, callerRole)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrStoreNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrNotStoreOwner):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, store)
}

func (h *StoreHandler) ApproveStore(c *gin.Context) {
	store, err := h.svc.ApproveStore(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrStoreNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, store)
}

func (h *StoreHandler) SuspendStore(c *gin.Context) {
	store, err := h.svc.SuspendStore(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrStoreNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, store)
}

func (h *StoreHandler) ReactivateStore(c *gin.Context) {
	store, err := h.svc.ReactivateStore(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrStoreNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, store)
}
