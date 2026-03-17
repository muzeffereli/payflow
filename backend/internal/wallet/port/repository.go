package port

import (
	"context"
	"database/sql"

	"payment-platform/internal/wallet/domain"
)

type WithdrawalRepository interface {
	Create(ctx context.Context, w *domain.Withdrawal) error
	GetByID(ctx context.Context, id string) (*domain.Withdrawal, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Withdrawal, error)
	ListByStatus(ctx context.Context, status domain.WithdrawalStatus, limit, offset int) ([]*domain.Withdrawal, error)
	UpdateStatus(ctx context.Context, w *domain.Withdrawal) error
}

type WalletRepository interface {
	Create(ctx context.Context, wallet *domain.Wallet) error
	GetByUserID(ctx context.Context, userID string) (*domain.Wallet, error)

	GetByUserIDForUpdate(ctx context.Context, tx *sql.Tx, userID string) (*domain.Wallet, error)

	UpdateBalance(ctx context.Context, tx *sql.Tx, wallet *domain.Wallet) error

	SaveTransaction(ctx context.Context, tx *sql.Tx, t *domain.Transaction) error

	BeginTx(ctx context.Context) (*sql.Tx, error)

	ListTransactions(ctx context.Context, walletID string, limit, offset int) ([]*domain.Transaction, int, error)
}
