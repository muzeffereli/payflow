package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"payment-platform/internal/wallet/domain"
	"payment-platform/internal/wallet/port"
	"payment-platform/internal/wallet/service"
)

var _ port.WalletRepository = (*postgresRepo)(nil)

type postgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) port.WalletRepository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *postgresRepo) Create(ctx context.Context, w *domain.Wallet) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO wallets (id, user_id, balance, currency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		w.ID, w.UserID, w.Balance, w.Currency, w.CreatedAt, w.UpdatedAt,
	)
	return err
}

func (r *postgresRepo) GetByUserID(ctx context.Context, userID string) (*domain.Wallet, error) {
	return r.scanWallet(r.db.QueryRowContext(ctx, `
		SELECT id, user_id, balance, currency, created_at, updated_at
		FROM wallets WHERE user_id = $1`, userID,
	))
}

func (r *postgresRepo) GetByUserIDForUpdate(ctx context.Context, tx *sql.Tx, userID string) (*domain.Wallet, error) {
	return r.scanWallet(tx.QueryRowContext(ctx, `
		SELECT id, user_id, balance, currency, created_at, updated_at
		FROM wallets WHERE user_id = $1 FOR UPDATE`, userID,
	))
}

func (r *postgresRepo) UpdateBalance(ctx context.Context, tx *sql.Tx, w *domain.Wallet) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE wallets SET balance = $1, updated_at = $2 WHERE id = $3`,
		w.Balance, w.UpdatedAt, w.ID,
	)
	return err
}

func (r *postgresRepo) SaveTransaction(ctx context.Context, tx *sql.Tx, t *domain.Transaction) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO wallet_transactions
			(id, wallet_id, type, amount, source, reference_id, balance_before, balance_after, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		t.ID, t.WalletID, t.Type, t.Amount, t.Source,
		t.ReferenceID, t.BalanceBefore, t.BalanceAfter, t.CreatedAt,
	)
	return err
}

func (r *postgresRepo) ListTransactions(ctx context.Context, walletID string, limit, offset int) ([]*domain.Transaction, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM wallet_transactions WHERE wallet_id = $1`, walletID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count transactions: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, wallet_id, type, amount, source, reference_id, balance_before, balance_after, created_at
		FROM wallet_transactions WHERE wallet_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, walletID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var txns []*domain.Transaction
	for rows.Next() {
		t := &domain.Transaction{}
		if err := rows.Scan(&t.ID, &t.WalletID, &t.Type, &t.Amount, &t.Source, &t.ReferenceID,
			&t.BalanceBefore, &t.BalanceAfter, &t.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan transaction: %w", err)
		}
		txns = append(txns, t)
	}
	return txns, total, rows.Err()
}

func (r *postgresRepo) scanWallet(row *sql.Row) (*domain.Wallet, error) {
	w := &domain.Wallet{}
	err := row.Scan(&w.ID, &w.UserID, &w.Balance, &w.Currency, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan wallet: %w", err)
	}
	return w, nil
}
