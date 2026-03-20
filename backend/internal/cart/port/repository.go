package port

import (
	"context"

	"payment-platform/internal/cart/domain"
	"payment-platform/pkg/catalog"
)

type CartRepository interface {
	Get(ctx context.Context, userID string) (*domain.Cart, error)

	Save(ctx context.Context, cart *domain.Cart) error

	Delete(ctx context.Context, userID string) error
}

type ProductClient interface {
	GetProducts(ctx context.Context, ids []string) ([]ProductInfo, error)
}

// Type aliases so existing code using port.ProductInfo compiles unchanged.
type ProductInfo = catalog.ProductInfo
type AttributeInfo = catalog.AttributeInfo
type VariantInfo = catalog.VariantInfo

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
