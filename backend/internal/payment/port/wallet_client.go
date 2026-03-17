package port

import "context"

type WalletClient interface {
	Debit(ctx context.Context, userID string, amount int64, referenceID string) (string, error)
}
