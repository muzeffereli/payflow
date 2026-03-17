package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"payment-platform/internal/payment/domain"
	"payment-platform/internal/payment/port"
	"payment-platform/internal/payment/service"
)

var _ port.PaymentRepository = (*postgresRepo)(nil)

type postgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) port.PaymentRepository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) Save(ctx context.Context, p *domain.Payment) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO payments
			(id, order_id, user_id, amount, currency, status, method,
			 transaction_id, failure_reason, store_id, store_owner_id, commission,
			 created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			status         = EXCLUDED.status,
			transaction_id = EXCLUDED.transaction_id,
			failure_reason = EXCLUDED.failure_reason,
			updated_at     = EXCLUDED.updated_at`,
		p.ID, p.OrderID, p.UserID, p.Amount, p.Currency,
		p.Status, p.Method, p.TransactionID, p.FailureReason,
		p.StoreID, nullStr(p.StoreOwnerID), p.Commission,
		p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	p := &domain.Payment{}
	var ownerID sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT id, order_id, user_id, amount, currency, status, method,
		       COALESCE(transaction_id, ''), COALESCE(failure_reason, ''),
		       store_id, COALESCE(store_owner_id, ''), COALESCE(commission, 0),
		       created_at, updated_at
		FROM payments WHERE id = $1`, id,
	).Scan(
		&p.ID, &p.OrderID, &p.UserID, &p.Amount, &p.Currency, &p.Status,
		&p.Method, &p.TransactionID, &p.FailureReason,
		&p.StoreID, &ownerID, &p.Commission,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if ownerID.Valid {
		p.StoreOwnerID = ownerID.String
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get payment by id: %w", err)
	}
	return p, nil
}

func (r *postgresRepo) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	p := &domain.Payment{}
	var ownerID sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT id, order_id, user_id, amount, currency, status, method,
		       COALESCE(transaction_id, ''), COALESCE(failure_reason, ''),
		       store_id, COALESCE(store_owner_id, ''), COALESCE(commission, 0),
		       created_at, updated_at
		FROM payments WHERE order_id = $1`, orderID,
	).Scan(
		&p.ID, &p.OrderID, &p.UserID, &p.Amount, &p.Currency, &p.Status,
		&p.Method, &p.TransactionID, &p.FailureReason,
		&p.StoreID, &ownerID, &p.Commission,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if ownerID.Valid {
		p.StoreOwnerID = ownerID.String
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get payment by order_id: %w", err)
	}
	return p, nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (r *postgresRepo) UpdateStatus(ctx context.Context, p *domain.Payment) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE payments
		SET status = $1, transaction_id = $2, failure_reason = $3, updated_at = $4
		WHERE id = $5`,
		p.Status, p.TransactionID, p.FailureReason, p.UpdatedAt, p.ID,
	)
	return err
}
