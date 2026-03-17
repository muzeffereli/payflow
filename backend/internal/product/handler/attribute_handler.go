package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"payment-platform/internal/product/domain"
	"payment-platform/internal/product/port"
)

type AttributeHandler struct {
	repo port.GlobalAttributeRepository
}

func NewAttributeHandler(repo port.GlobalAttributeRepository) *AttributeHandler {
	return &AttributeHandler{repo: repo}
}

func (h *AttributeHandler) RegisterRoutes(adminGroup, publicGroup gin.IRouter) {
	adminGroup.POST("", h.Create)
	adminGroup.GET("", h.List)
	adminGroup.PATCH("/:id", h.Update)
	adminGroup.DELETE("/:id", h.Delete)

	publicGroup.GET("", h.List)
}

type createGlobalAttributeRequest struct {
	Name   string   `json:"name"   binding:"required"`
	Values []string `json:"values" binding:"required,min=1"`
}

type updateGlobalAttributeRequest struct {
	Name   *string  `json:"name"`
	Values []string `json:"values"`
}

func (h *AttributeHandler) Create(c *gin.Context) {
	var req createGlobalAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	attr, err := domain.NewGlobalAttribute(req.Name, req.Values)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(c.Request.Context(), attr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create attribute"})
		return
	}

	c.JSON(http.StatusCreated, attr)
}

func (h *AttributeHandler) List(c *gin.Context) {
	attrs, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list attributes"})
		return
	}
	if attrs == nil {
		attrs = []*domain.GlobalAttribute{}
	}
	c.JSON(http.StatusOK, gin.H{"attributes": attrs})
}

func (h *AttributeHandler) Update(c *gin.Context) {
	id := c.Param("id")

	attr, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrGlobalAttributeNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "attribute not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch attribute"})
		return
	}

	var req updateGlobalAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		attr.Name = *req.Name
	}
	if req.Values != nil {
		attr.Values = req.Values
	}
	attr.UpdatedAt = time.Now().UTC()

	if err := h.repo.Update(c.Request.Context(), attr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update attribute"})
		return
	}

	c.JSON(http.StatusOK, attr)
}

func (h *AttributeHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrGlobalAttributeNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "attribute not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete attribute"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
