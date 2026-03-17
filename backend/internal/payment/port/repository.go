package port

import (
	"context"

	"payment-platform/internal/payment/domain"
)

type PaymentRepository interface {
	Save(ctx context.Context, payment *domain.Payment) error // upsert â€” insert or update
	GetByID(ctx context.Context, id string) (*domain.Payment, error)
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	UpdateStatus(ctx context.Context, payment *domain.Payment) error
}
