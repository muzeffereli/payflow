package port

import (
	"context"

	"payment-platform/internal/cart/domain"
)

type CartRepository interface {
	Get(ctx context.Context, userID string) (*domain.Cart, error)

	Save(ctx context.Context, cart *domain.Cart) error

	Delete(ctx context.Context, userID string) error
}

type ProductClient interface {
	GetProducts(ctx context.Context, ids []string) ([]ProductInfo, error)
}

type ProductInfo struct {
	ID       string
	Name     string
	Price    int64 // unit price in cents
	Currency string
	Stock    int
	Status   string  // "active" | "inactive"
	StoreID  *string // nil = platform product
	Variants []VariantInfo
}

type VariantInfo struct {
	ID              string
	SKU             string
	Price           *int64
	Stock           int
	Status          string
	AttributeValues map[string]string
}

type OrderClient interface {
	CreateOrder(ctx context.Context, req CreateOrderRequest) (string, error)
}

type CreateOrderRequest struct {
	UserID         string
	Currency       string
	PaymentMethod  string
	IdempotencyKey string
	Items          []OrderItemInput
	StoreID        *string // nil = platform order; set when all items come from a single store
}

type OrderItemInput struct {
	ProductID string
	VariantID *string
	Quantity  int
}
