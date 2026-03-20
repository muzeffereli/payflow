package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"payment-platform/internal/product/domain"
	"payment-platform/internal/product/port"
)

type AttributeHandler struct {
	repo       port.GlobalAttributeRepository
	categories port.CategoryRepository
}

func NewAttributeHandler(repo port.GlobalAttributeRepository, categories port.CategoryRepository) *AttributeHandler {
	return &AttributeHandler{repo: repo, categories: categories}
}

func (h *AttributeHandler) RegisterRoutes(adminGroup, publicGroup gin.IRouter) {
	adminGroup.POST("", h.Create)
	adminGroup.GET("", h.List)
	adminGroup.GET("/categories", h.ListCategories)
	adminGroup.PATCH("/:id", h.Update)
	adminGroup.DELETE("/:id", h.Delete)

	publicGroup.GET("", h.List)
	publicGroup.GET("/categories", h.ListCategories)
}

type createGlobalAttributeRequest struct {
	SubcategoryID string   `json:"subcategory_id" binding:"required"`
	Name          string   `json:"name"           binding:"required"`
	Values        []string `json:"values"         binding:"required,min=1"`
}

type updateGlobalAttributeRequest struct {
	SubcategoryID *string  `json:"subcategory_id"`
	Name          *string  `json:"name"`
	Values        []string `json:"values"`
}

func (h *AttributeHandler) Create(c *gin.Context) {
	var req createGlobalAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subcategory, err := h.resolveSubcategory(c.Request.Context(), req.SubcategoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	attr, err := domain.NewGlobalAttribute(subcategory.ID, subcategory.Name, req.Name, req.Values)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(c.Request.Context(), attr); err != nil {
		if errors.Is(err, domain.ErrGlobalAttributeConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create attribute"})
		return
	}

	// Populate parent category for response
	if cat, err := h.categories.GetCategoryByID(c.Request.Context(), subcategory.CategoryID); err == nil {
		attr.CategoryID = cat.ID
		attr.Category = cat.Name
	}

	c.JSON(http.StatusCreated, attr)
}

func (h *AttributeHandler) List(c *gin.Context) {
	attrs, err := h.repo.List(c.Request.Context(), port.GlobalAttributeFilter{
		SubcategoryID: c.Query("subcategory_id"),
		CategoryID:    c.Query("category_id"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list attributes"})
		return
	}
	if attrs == nil {
		attrs = []*domain.GlobalAttribute{}
	}
	c.JSON(http.StatusOK, gin.H{"attributes": attrs})
}

func (h *AttributeHandler) ListCategories(c *gin.Context) {
	categories, err := h.categories.ListCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list categories"})
		return
	}
	if categories == nil {
		categories = []*domain.Category{}
	}
	c.JSON(http.StatusOK, gin.H{"categories": categories})
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
	if req.SubcategoryID != nil {
		subcategoryID := strings.TrimSpace(*req.SubcategoryID)
		if subcategoryID != "" {
			subcategory, err := h.resolveSubcategory(c.Request.Context(), subcategoryID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			attr.SubcategoryID = subcategory.ID
			attr.Subcategory = subcategory.Name
		}
	}
	if req.Values != nil {
		attr.Values = req.Values
	}
	attr.UpdatedAt = time.Now().UTC()

	if err := h.repo.Update(c.Request.Context(), attr); err != nil {
		if errors.Is(err, domain.ErrGlobalAttributeConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update attribute"})
		return
	}

	c.JSON(http.StatusOK, attr)
}

func (h *AttributeHandler) resolveSubcategory(ctx context.Context, subcategoryID string) (*domain.Subcategory, error) {
	subcategoryID = strings.TrimSpace(subcategoryID)
	if subcategoryID == "" {
		return nil, errors.New("subcategory_id is required")
	}
	subcategory, err := h.categories.GetSubcategoryByID(ctx, subcategoryID)
	if err != nil {
		if errors.Is(err, domain.ErrSubcategoryNotFound) {
			return nil, errors.New("subcategory not found")
		}
		return nil, err
	}
	return subcategory, nil
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
