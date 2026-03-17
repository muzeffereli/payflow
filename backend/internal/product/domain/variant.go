package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Attribute struct {
	ID        string    `json:"id"         db:"id"`
	ProductID string    `json:"product_id" db:"product_id"`
	Name      string    `json:"name"       db:"name"`
	Values    []string  `json:"values"     db:"values"`
	Position  int       `json:"position"   db:"position"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func NewAttribute(productID, name string, values []string, position int) (*Attribute, error) {
	if name == "" {
		return nil, errors.New("attribute name is required")
	}
	if len(values) == 0 {
		return nil, errors.New("attribute must have at least one value")
	}
	return &Attribute{
		ID:        uuid.New().String(),
		ProductID: productID,
		Name:      name,
		Values:    values,
		Position:  position,
		CreatedAt: time.Now().UTC(),
	}, nil
}

type Variant struct {
	ID              string            `json:"id"               db:"id"`
	ProductID       string            `json:"product_id"       db:"product_id"`
	SKU             string            `json:"sku"              db:"sku"`
	Price           *int64            `json:"price"            db:"price"` // nil = use parent product base price
	Stock           int               `json:"stock"            db:"stock"`
	AttributeValues map[string]string `json:"attribute_values" db:"attribute_values"` // {"Color":"Red","Size":"M"}
	Status          ProductStatus     `json:"status"           db:"status"`
	CreatedAt       time.Time         `json:"created_at"       db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"       db:"updated_at"`
}

func NewVariant(productID, sku string, price *int64, stock int, attrValues map[string]string) (*Variant, error) {
	if sku == "" {
		return nil, errors.New("variant SKU is required")
	}
	if stock < 0 {
		return nil, errors.New("variant stock cannot be negative")
	}
	if price != nil && *price <= 0 {
		return nil, errors.New("variant price must be greater than zero")
	}

	status := StatusActive
	if stock == 0 {
		status = StatusOutOfStock
	}

	now := time.Now().UTC()
	return &Variant{
		ID:              uuid.New().String(),
		ProductID:       productID,
		SKU:             sku,
		Price:           price,
		Stock:           stock,
		AttributeValues: attrValues,
		Status:          status,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func (v *Variant) ReserveStock(qty int) error {
	if qty <= 0 {
		return errors.New("quantity must be positive")
	}
	if v.Status == StatusInactive {
		return ErrProductInactive
	}
	if v.Stock < qty {
		return fmt.Errorf("%w: have %d, need %d", ErrInsufficientStock, v.Stock, qty)
	}
	v.Stock -= qty
	if v.Stock == 0 {
		v.Status = StatusOutOfStock
	}
	v.UpdatedAt = time.Now().UTC()
	return nil
}

func (v *Variant) ReleaseStock(qty int) {
	v.Stock += qty
	if v.Status == StatusOutOfStock && v.Stock > 0 {
		v.Status = StatusActive
	}
	v.UpdatedAt = time.Now().UTC()
}

func (v *Variant) EffectivePrice(basePrice int64) int64 {
	if v.Price != nil {
		return *v.Price
	}
	return basePrice
}

var (
	ErrVariantNotFound    = errors.New("variant not found")
	ErrVariantSKUConflict = errors.New("variant SKU already exists")
)
