package port

import (
	"context"

	"payment-platform/internal/store/domain"
)

type StoreRepository interface {
	Create(ctx context.Context, s *domain.Store) error
	GetByID(ctx context.Context, id string) (*domain.Store, error)
	GetByOwnerID(ctx context.Context, ownerID string) (*domain.Store, error)
	List(ctx context.Context, f ListFilter) ([]*domain.Store, int, error)
	Update(ctx context.Context, s *domain.Store) error
}

type ListFilter struct {
	Status string
	Limit  int
	Offset int
}
