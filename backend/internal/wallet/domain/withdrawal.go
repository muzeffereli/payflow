package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type WithdrawalStatus string

const (
	WithdrawalPending  WithdrawalStatus = "pending"
	WithdrawalApproved WithdrawalStatus = "approved"
	WithdrawalRejected WithdrawalStatus = "rejected"
)

type Withdrawal struct {
	ID        string           `json:"id"`
	UserID    string           `json:"user_id"`
	StoreID   string           `json:"store_id"`
	Amount    int64            `json:"amount"` // cents
	Currency  string           `json:"currency"`
	Method    string           `json:"method"` // e.g. "bank_transfer"
	Status    WithdrawalStatus `json:"status"`
	Notes     string           `json:"notes,omitempty"` // admin note on rejection
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

func NewWithdrawal(userID, storeID, currency, method string, amount int64) (*Withdrawal, error) {
	if amount <= 0 {
		return nil, errors.New("withdrawal amount must be positive")
	}
	if method == "" {
		method = "bank_transfer"
	}
	now := time.Now().UTC()
	return &Withdrawal{
		ID:        uuid.New().String(),
		UserID:    userID,
		StoreID:   storeID,
		Amount:    amount,
		Currency:  currency,
		Method:    method,
		Status:    WithdrawalPending,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (w *Withdrawal) Approve() error {
	if w.Status != WithdrawalPending {
		return fmt.Errorf("cannot approve withdrawal in status %s", w.Status)
	}
	w.Status = WithdrawalApproved
	w.UpdatedAt = time.Now().UTC()
	return nil
}

func (w *Withdrawal) Reject(reason string) error {
	if w.Status != WithdrawalPending {
		return fmt.Errorf("cannot reject withdrawal in status %s", w.Status)
	}
	w.Status = WithdrawalRejected
	w.Notes = reason
	w.UpdatedAt = time.Now().UTC()
	return nil
}

var ErrWithdrawalNotFound = errors.New("withdrawal not found")
