package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"payment-platform/internal/product/domain"
	"payment-platform/internal/product/port"
)

type CategoryHandler struct {
	repo port.CategoryRepository
}

func NewCategoryHandler(repo port.CategoryRepository) *CategoryHandler {
	return &CategoryHandler{repo: repo}
}

func (h *CategoryHandler) RegisterRoutes(adminCategories, publicCategories, adminSubcategories gin.IRouter) {
	adminCategories.POST("", h.CreateCategory)
	adminCategories.GET("", h.ListCategories)
	adminCategories.PATCH("/:id", h.UpdateCategory)
	adminCategories.DELETE("/:id", h.DeleteCategory)

	publicCategories.GET("", h.ListCategories)
	publicCategories.GET("/:id/subcategories", h.ListSubcategories)

	adminSubcategories.POST("", h.CreateSubcategory)
	adminSubcategories.PATCH("/:id", h.UpdateSubcategory)
	adminSubcategories.DELETE("/:id", h.DeleteSubcategory)
}

type createCategoryRequest struct {
	Name string `json:"name" binding:"required"`
}

type updateCategoryRequest struct {
	Name *string `json:"name"`
}

type createSubcategoryRequest struct {
	CategoryID string `json:"category_id" binding:"required"`
	Name       string `json:"name"        binding:"required"`
}

type updateSubcategoryRequest struct {
	CategoryID *string `json:"category_id"`
	Name       *string `json:"name"`
}

func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := domain.NewCategory(req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.CreateCategory(c.Request.Context(), category); err != nil {
		if errors.Is(err, domain.ErrCategoryConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create category"})
		return
	}

	c.JSON(http.StatusCreated, category)
}

func (h *CategoryHandler) ListCategories(c *gin.Context) {
	categories, err := h.repo.ListCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list categories"})
		return
	}
	if categories == nil {
		categories = []*domain.Category{}
	}
	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	category, err := h.repo.GetCategoryByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrCategoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch category"})
		return
	}

	var req updateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		updated, err := domain.NewCategory(*req.Name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		category.Name = updated.Name
		category.Slug = updated.Slug
	}
	category.UpdatedAt = time.Now().UTC()

	if err := h.repo.UpdateCategory(c.Request.Context(), category); err != nil {
		if errors.Is(err, domain.ErrCategoryConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update category"})
		return
	}

	c.JSON(http.StatusOK, category)
}

func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	if err := h.repo.DeleteCategory(c.Request.Context(), c.Param("id")); err != nil {
		if errors.Is(err, domain.ErrCategoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete category"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *CategoryHandler) CreateSubcategory(c *gin.Context) {
	var req createSubcategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subcategory, err := domain.NewSubcategory(req.CategoryID, req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.repo.GetCategoryByID(c.Request.Context(), req.CategoryID); err != nil {
		if errors.Is(err, domain.ErrCategoryNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate category"})
		return
	}

	if err := h.repo.CreateSubcategory(c.Request.Context(), subcategory); err != nil {
		if errors.Is(err, domain.ErrSubcategoryConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create subcategory"})
		return
	}

	c.JSON(http.StatusCreated, subcategory)
}

func (h *CategoryHandler) ListSubcategories(c *gin.Context) {
	subcategories, err := h.repo.ListSubcategories(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list subcategories"})
		return
	}
	if subcategories == nil {
		subcategories = []*domain.Subcategory{}
	}
	c.JSON(http.StatusOK, gin.H{"subcategories": subcategories})
}

func (h *CategoryHandler) UpdateSubcategory(c *gin.Context) {
	subcategory, err := h.repo.GetSubcategoryByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrSubcategoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "subcategory not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch subcategory"})
		return
	}

	var req updateSubcategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.CategoryID != nil {
		if _, err := h.repo.GetCategoryByID(c.Request.Context(), *req.CategoryID); err != nil {
			if errors.Is(err, domain.ErrCategoryNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "category not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate category"})
			return
		}
		subcategory.CategoryID = *req.CategoryID
	}
	if req.Name != nil {
		updated, err := domain.NewSubcategory(subcategory.CategoryID, *req.Name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		subcategory.Name = updated.Name
		subcategory.Slug = updated.Slug
	}
	subcategory.UpdatedAt = time.Now().UTC()

	if err := h.repo.UpdateSubcategory(c.Request.Context(), subcategory); err != nil {
		if errors.Is(err, domain.ErrSubcategoryConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update subcategory"})
		return
	}

	c.JSON(http.StatusOK, subcategory)
}

func (h *CategoryHandler) DeleteSubcategory(c *gin.Context) {
	if err := h.repo.DeleteSubcategory(c.Request.Context(), c.Param("id")); err != nil {
		if errors.Is(err, domain.ErrSubcategoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "subcategory not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete subcategory"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
