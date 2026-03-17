package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"payment-platform/internal/product/domain"
	"payment-platform/internal/product/port"
	"payment-platform/internal/product/service"
	"payment-platform/pkg/middleware"

	"github.com/gin-gonic/gin"
)

type AttributeInput struct {
	Name   string   `json:"name"   binding:"required" example:"Color"`
	Values []string `json:"values" binding:"required" example:"Red,Blue,Green"`
}

type CreateProductRequest struct {
	Name        string           `json:"name"        binding:"required" example:"Wireless Headphones"`
	Description string           `json:"description"                    example:"Noise-cancelling over-ear headphones"`
	SKU         string           `json:"sku"         binding:"required" example:"HDPH-001"`
	Price       int64            `json:"price"       binding:"required" example:"9999"` // cents
	Currency    string           `json:"currency"                       example:"USD"`
	Category    string           `json:"category"                       example:"electronics"`
	Stock       int              `json:"stock"       binding:"min=0"    example:"100"`
	StoreID     *string          `json:"store_id,omitempty"             example:"store-uuid"`
	Images      []string         `json:"images,omitempty"               example:"[\"http://minio:9000/products/img.jpg\"]"`
	Attributes  []AttributeInput `json:"attributes,omitempty"`
}

type UpdateProductRequest struct {
	Name        *string          `json:"name"        example:"Wireless Headphones Pro"`
	Description *string          `json:"description" example:"Updated description"`
	Price       *int64           `json:"price"       example:"11999"`
	Stock       *int             `json:"stock"       example:"200"`
	Category    *string          `json:"category"    example:"electronics"`
	Images      []string         `json:"images,omitempty"` // nil = no change; [] = clear; [...] = replace
	Attributes  []AttributeInput `json:"attributes,omitempty"`
}

type CreateVariantRequest struct {
	SKU             string            `json:"sku"              binding:"required" example:"HDPH-001-RED-M"`
	Price           *int64            `json:"price,omitempty"                     example:"10999"`
	Stock           int               `json:"stock"            binding:"min=0"    example:"50"`
	AttributeValues map[string]string `json:"attribute_values" binding:"required"`
}

type UpdateVariantRequest struct {
	SKU             *string           `json:"sku,omitempty"`
	Price           *int64            `json:"price,omitempty"`
	Stock           *int              `json:"stock,omitempty"`
	AttributeValues map[string]string `json:"attribute_values,omitempty"`
}

type ListProductsResponse struct {
	Products []*domain.Product `json:"products"`
	Total    int               `json:"total"  example:"42"`
	Limit    int               `json:"limit"  example:"20"`
	Offset   int               `json:"offset" example:"0"`
}

type ProductErrorResponse struct {
	Error string `json:"error" example:"product not found"`
}

type ProductHandler struct {
	svc *service.ProductService
}

func New(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

func (h *ProductHandler) RegisterRoutes(rg *gin.RouterGroup) {
	products := rg.Group("/products")

	products.GET("", h.ListProducts)
	products.GET("/:id", h.GetProduct)

	write := products.Group("", middleware.RequireRole("seller"))
	write.POST("", h.CreateProduct)
	write.PATCH("/:id", h.UpdateProduct)
	write.DELETE("/:id", h.DeleteProduct)

	write.GET("/:id/variants", h.ListVariants)
	write.POST("/:id/variants", h.CreateVariant)
	write.PATCH("/:id/variants/:vid", h.UpdateVariant)
	write.DELETE("/:id/variants/:vid", h.DeleteVariant)
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var body CreateProductRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	attrs := make([]service.AttributeInput, len(body.Attributes))
	for i, a := range body.Attributes {
		attrs[i] = service.AttributeInput{Name: a.Name, Values: a.Values}
	}

	p, err := h.svc.Create(c.Request.Context(), service.CreateRequest{
		Name:        body.Name,
		Description: body.Description,
		SKU:         body.SKU,
		Price:       body.Price,
		Currency:    body.Currency,
		Category:    body.Category,
		Stock:       body.Stock,
		StoreID:     body.StoreID,
		Images:      body.Images,
		Attributes:  attrs,
		CallerID:    c.GetString("user_id"),
		CallerRole:  c.GetString("user_role"),
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSKUConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "SKU already exists"})
		case errors.Is(err, service.ErrNotStoreOwner), errors.Is(err, service.ErrSellerMustProvideStore), errors.Is(err, service.ErrSellerOnly):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *ProductHandler) ListProducts(c *gin.Context) {
	if rawIDs := c.Query("ids"); rawIDs != "" {
		ids := strings.Split(rawIDs, ",")
		products, err := h.svc.GetByIDs(c.Request.Context(), ids)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch products"})
			return
		}
		c.JSON(http.StatusOK, ListProductsResponse{Products: products, Total: len(products), Limit: len(products), Offset: 0})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	products, total, err := h.svc.List(c.Request.Context(), port.ListFilter{
		Category: c.Query("category"),
		Status:   c.Query("status"),
		StoreID:  c.Query("store_id"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}

	c.JSON(http.StatusOK, ListProductsResponse{
		Products: products,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	})
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	p, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get product"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	var body UpdateProductRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var svcAttrs []service.AttributeInput
	if body.Attributes != nil {
		svcAttrs = make([]service.AttributeInput, len(body.Attributes))
		for i, a := range body.Attributes {
			svcAttrs[i] = service.AttributeInput{Name: a.Name, Values: a.Values}
		}
	}

	var images *[]string
	if body.Images != nil {
		imgs := body.Images
		images = &imgs
	}

	p, err := h.svc.Update(c.Request.Context(), c.Param("id"), service.UpdateRequest{
		Name:        body.Name,
		Description: body.Description,
		Price:       body.Price,
		Stock:       body.Stock,
		Category:    body.Category,
		Images:      images,
		Attributes:  svcAttrs,
		CallerID:    c.GetString("user_id"),
		CallerRole:  c.GetString("user_role"),
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrProductNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		case errors.Is(err, service.ErrNotStoreOwner), errors.Is(err, service.ErrSellerOnly):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	err := h.svc.Deactivate(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate product"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ProductHandler) ListVariants(c *gin.Context) {
	variants, err := h.svc.ListVariants(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list variants"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"variants": variants})
}

func (h *ProductHandler) CreateVariant(c *gin.Context) {
	var body CreateVariantRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	v, err := h.svc.CreateVariant(c.Request.Context(), c.Param("id"), service.CreateVariantRequest{
		SKU:             body.SKU,
		Price:           body.Price,
		Stock:           body.Stock,
		AttributeValues: body.AttributeValues,
		CallerID:        c.GetString("user_id"),
		CallerRole:      c.GetString("user_role"),
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrProductNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		case errors.Is(err, domain.ErrVariantSKUConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "variant SKU already exists"})
		case errors.Is(err, service.ErrNotStoreOwner), errors.Is(err, service.ErrSellerOnly):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *ProductHandler) UpdateVariant(c *gin.Context) {
	var body UpdateVariantRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	v, err := h.svc.UpdateVariant(c.Request.Context(), c.Param("id"), c.Param("vid"), service.UpdateVariantRequest{
		SKU:             body.SKU,
		Price:           body.Price,
		Stock:           body.Stock,
		AttributeValues: body.AttributeValues,
		CallerID:        c.GetString("user_id"),
		CallerRole:      c.GetString("user_role"),
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrVariantNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "variant not found"})
		case errors.Is(err, domain.ErrProductNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		case errors.Is(err, service.ErrNotStoreOwner), errors.Is(err, service.ErrSellerOnly):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *ProductHandler) DeleteVariant(c *gin.Context) {
	err := h.svc.DeleteVariant(c.Request.Context(), c.Param("id"), c.Param("vid"),
		c.GetString("user_id"), c.GetString("user_role"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrVariantNotFound), errors.Is(err, domain.ErrProductNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		case errors.Is(err, service.ErrNotStoreOwner), errors.Is(err, service.ErrSellerOnly):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.Status(http.StatusNoContent)
}
