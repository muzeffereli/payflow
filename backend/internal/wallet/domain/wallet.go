package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrNegativeAmount    = errors.New("amount must be positive")
)

type Wallet struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Balance   int64     `json:"balance" db:"balance"` // cents
	Currency  string    `json:"currency" db:"currency"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func NewWallet(userID, currency string) *Wallet {
	now := time.Now().UTC()
	return &Wallet{
		ID:        uuid.New().String(),
		UserID:    userID,
		Balance:   0,
		Currency:  currency,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (w *Wallet) Credit(amount int64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}
	w.Balance += amount
	w.UpdatedAt = time.Now().UTC()
	return nil
}

func (w *Wallet) Debit(amount int64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}
	if w.Balance < amount {
		return ErrInsufficientFunds
	}
	w.Balance -= amount
	w.UpdatedAt = time.Now().UTC()
	return nil
}

type Transaction struct {
	ID            string    `json:"id" db:"id"`
	WalletID      string    `json:"wallet_id" db:"wallet_id"`
	Type          string    `json:"type" db:"type"` // "credit" or "debit"
	Amount        int64     `json:"amount" db:"amount"`
	Source        string    `json:"source" db:"source"`             // "refund", "deposit", "payment"
	ReferenceID   string    `json:"reference_id" db:"reference_id"` // payment ID, refund ID, etc.
	BalanceBefore int64     `json:"balance_before" db:"balance_before"`
	BalanceAfter  int64     `json:"balance_after" db:"balance_after"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

func NewTransaction(walletID, txType, source, referenceID string, amount, balanceBefore, balanceAfter int64) *Transaction {
	return &Transaction{
		ID:            uuid.New().String(),
		WalletID:      walletID,
		Type:          txType,
		Amount:        amount,
		Source:        source,
		ReferenceID:   referenceID,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		CreatedAt:     time.Now().UTC(),
	}
}
