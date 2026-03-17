package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"payment-platform/internal/wallet/domain"
	"payment-platform/internal/wallet/port"
)

var _ port.WithdrawalRepository = (*postgresWithdrawalRepo)(nil)

type postgresWithdrawalRepo struct {
	db *sql.DB
}

func NewPostgresWithdrawalRepo(db *sql.DB) port.WithdrawalRepository {
	return &postgresWithdrawalRepo{db: db}
}

func (r *postgresWithdrawalRepo) Create(ctx context.Context, w *domain.Withdrawal) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO withdrawals (id, user_id, store_id, amount, currency, method, status, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		w.ID, w.UserID, w.StoreID, w.Amount, w.Currency,
		w.Method, w.Status, w.Notes, w.CreatedAt, w.UpdatedAt,
	)
	return err
}

func (r *postgresWithdrawalRepo) GetByID(ctx context.Context, id string) (*domain.Withdrawal, error) {
	w := &domain.Withdrawal{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, store_id, amount, currency, method, status,
		       COALESCE(notes, ''), created_at, updated_at
		FROM withdrawals WHERE id = $1`, id,
	).Scan(
		&w.ID, &w.UserID, &w.StoreID, &w.Amount, &w.Currency,
		&w.Method, &w.Status, &w.Notes, &w.CreatedAt, &w.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrWithdrawalNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get withdrawal: %w", err)
	}
	return w, nil
}

func (r *postgresWithdrawalRepo) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Withdrawal, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, store_id, amount, currency, method, status,
		       COALESCE(notes, ''), created_at, updated_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list withdrawals by user: %w", err)
	}
	defer rows.Close()
	return scanWithdrawals(rows)
}

func (r *postgresWithdrawalRepo) ListByStatus(ctx context.Context, status domain.WithdrawalStatus, limit, offset int) ([]*domain.Withdrawal, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, store_id, amount, currency, method, status,
		       COALESCE(notes, ''), created_at, updated_at
		FROM withdrawals
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3`,
		status, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list withdrawals by status: %w", err)
	}
	defer rows.Close()
	return scanWithdrawals(rows)
}

func (r *postgresWithdrawalRepo) UpdateStatus(ctx context.Context, w *domain.Withdrawal) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE withdrawals SET status = $1, notes = $2, updated_at = $3
		WHERE id = $4`,
		w.Status, w.Notes, w.UpdatedAt, w.ID,
	)
	return err
}

func scanWithdrawals(rows *sql.Rows) ([]*domain.Withdrawal, error) {
	var out []*domain.Withdrawal
	for rows.Next() {
		w := &domain.Withdrawal{}
		if err := rows.Scan(
			&w.ID, &w.UserID, &w.StoreID, &w.Amount, &w.Currency,
			&w.Method, &w.Status, &w.Notes, &w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan withdrawal: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
