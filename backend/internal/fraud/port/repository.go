package port

import (
	"context"

	"payment-platform/internal/fraud/domain"
)

type FraudCheckRepository interface {
	Save(ctx context.Context, fc *domain.FraudCheck) error
	GetByID(ctx context.Context, id string) (*domain.FraudCheck, error)
	List(ctx context.Context, decision string, limit, offset int) ([]*domain.FraudCheck, int, error)
}
