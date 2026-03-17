package port

import (
	"context"

	"payment-platform/internal/product/domain"
)

type ProductRepository interface {
	Create(ctx context.Context, p *domain.Product) error
	GetByID(ctx context.Context, id string) (*domain.Product, error)
	GetByIDs(ctx context.Context, ids []string) ([]*domain.Product, error) // batch fetch for order validation
	GetBySKU(ctx context.Context, sku string) (*domain.Product, error)
	List(ctx context.Context, filter ListFilter) ([]*domain.Product, int, error)
	Update(ctx context.Context, p *domain.Product) error
}

type ListFilter struct {
	Category string
	Status   string
	StoreID  string // filter to a specific store's products
	Limit    int
	Offset   int
}

type StoreClient interface {
	GetStoreIDByOwner(ctx context.Context, ownerID string) (string, error)
}

type ReservationRepository interface {
	Save(ctx context.Context, r *domain.Reservation) error
	GetByOrderID(ctx context.Context, orderID string) ([]*domain.Reservation, error)
	UpdateStatus(ctx context.Context, orderID string, status domain.ReservationStatus) error
}

type AttributeRepository interface {
	SaveBatch(ctx context.Context, attrs []*domain.Attribute) error
	ListByProduct(ctx context.Context, productID string) ([]*domain.Attribute, error)
	DeleteByProduct(ctx context.Context, productID string) error
}

type GlobalAttributeRepository interface {
	Create(ctx context.Context, a *domain.GlobalAttribute) error
	GetByID(ctx context.Context, id string) (*domain.GlobalAttribute, error)
	List(ctx context.Context) ([]*domain.GlobalAttribute, error)
	Update(ctx context.Context, a *domain.GlobalAttribute) error
	Delete(ctx context.Context, id string) error
}

type ImageRepository interface {
	SetImages(ctx context.Context, productID string, urls []string) ([]*domain.ProductImage, error)
	GetByProductID(ctx context.Context, productID string) ([]*domain.ProductImage, error)
	GetByProductIDs(ctx context.Context, productIDs []string) (map[string][]*domain.ProductImage, error)
}

type VariantRepository interface {
	Create(ctx context.Context, v *domain.Variant) error
	Update(ctx context.Context, v *domain.Variant) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*domain.Variant, error)
	ListByProduct(ctx context.Context, productID string) ([]*domain.Variant, error)
	GetBySKU(ctx context.Context, sku string) (*domain.Variant, error)
}
