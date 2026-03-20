package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ProductStatus string

const (
	StatusActive     ProductStatus = "active"
	StatusInactive   ProductStatus = "inactive"
	StatusOutOfStock ProductStatus = "out_of_stock"
)

type ProductImage struct {
	ID        string    `json:"id"         db:"id"`
	ProductID string    `json:"product_id" db:"product_id"`
	URL       string    `json:"url"        db:"url"`
	Position  int       `json:"position"   db:"position"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Product struct {
	ID            string        `json:"id"                         db:"id"`
	Name          string        `json:"name"                       db:"name"`
	Description   string        `json:"description"                db:"description"`
	SKU           string        `json:"sku"                        db:"sku"`
	Price         int64         `json:"price"                      db:"price"` // unit price in cents
	Currency      string        `json:"currency"                   db:"currency"`
	Stock         int           `json:"stock"                      db:"stock"`
	CategoryID    string        `json:"category_id"                db:"category_id"`
	Category      string        `json:"category"                   db:"category"`
	SubcategoryID *string       `json:"subcategory_id,omitempty"   db:"subcategory_id"`
	Subcategory   string        `json:"subcategory,omitempty"      db:"subcategory"`
	Status        ProductStatus `json:"status"                     db:"status"`
	StoreID       *string       `json:"store_id,omitempty"         db:"store_id"`  // nil = platform product
	ImageURL      string        `json:"image_url"                  db:"image_url"` // first image (thumbnail)
	CreatedAt     time.Time     `json:"created_at"                 db:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"                 db:"updated_at"`

	Attributes []*Attribute    `json:"attributes,omitempty"`
	Variants   []*Variant      `json:"variants,omitempty"`
	Images     []*ProductImage `json:"images,omitempty"`
}

func NewProduct(name, description, sku string, price int64, currency, category, imageURL string, stock int) (*Product, error) {
	if name == "" {
		return nil, errors.New("product name is required")
	}
	if sku == "" {
		return nil, errors.New("product SKU is required")
	}
	if price <= 0 {
		return nil, errors.New("product price must be greater than zero")
	}
	if stock < 0 {
		return nil, errors.New("stock cannot be negative")
	}
	if currency == "" {
		currency = "USD"
	}

	now := time.Now().UTC()
	status := StatusActive
	if stock == 0 {
		status = StatusOutOfStock
	}

	return &Product{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		SKU:         sku,
		Price:       price,
		Currency:    currency,
		Stock:       stock,
		Category:    category,
		ImageURL:    imageURL,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (p *Product) ReserveStock(qty int) error {
	if qty <= 0 {
		return errors.New("quantity must be positive")
	}
	if p.Status == StatusInactive {
		return ErrProductInactive
	}
	if p.Stock < qty {
		return fmt.Errorf("%w: have %d, need %d", ErrInsufficientStock, p.Stock, qty)
	}
	p.Stock -= qty
	if p.Stock == 0 {
		p.Status = StatusOutOfStock
	}
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (p *Product) ReleaseStock(qty int) {
	p.Stock += qty
	if p.Status == StatusOutOfStock && p.Stock > 0 {
		p.Status = StatusActive
	}
	p.UpdatedAt = time.Now().UTC()
}

func (p *Product) UpdatePrice(cents int64) error {
	if cents <= 0 {
		return errors.New("price must be greater than zero")
	}
	p.Price = cents
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (p *Product) Deactivate() {
	p.Status = StatusInactive
	p.UpdatedAt = time.Now().UTC()
}

var (
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrProductInactive   = errors.New("product is inactive")
	ErrProductNotFound   = errors.New("product not found")
	ErrSKUConflict       = errors.New("SKU already exists")
)
