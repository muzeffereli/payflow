package port

import (
	"context"

	"payment-platform/internal/order/domain"
)

type OrderRepository interface {
	CreateWithOutbox(ctx context.Context, order *domain.Order, subject string, payload []byte) error

	Create(ctx context.Context, order *domain.Order) error

	GetByID(ctx context.Context, id string) (*domain.Order, error)

	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error)

	UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]domain.Order, error)
	ListByStore(ctx context.Context, storeID string, limit, offset int) ([]domain.Order, error)

	GetStoreAnalytics(ctx context.Context, storeID string) (*StoreAnalytics, error)
}

type StoreAnalytics struct {
	TotalOrders   int   `json:"total_orders"`
	TotalRevenue  int64 `json:"total_revenue"`
	PaidOrders    int   `json:"paid_orders"`
	PendingOrders int   `json:"pending_orders"`
}
