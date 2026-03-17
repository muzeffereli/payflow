package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentPending    PaymentStatus = "pending"
	PaymentProcessing PaymentStatus = "processing"
	PaymentSucceeded  PaymentStatus = "succeeded"
	PaymentFailed     PaymentStatus = "failed"
	PaymentRefunded   PaymentStatus = "refunded"
)

type Payment struct {
	ID            string        `json:"id" db:"id"`
	OrderID       string        `json:"order_id" db:"order_id"`
	UserID        string        `json:"user_id" db:"user_id"`
	Amount        int64         `json:"amount" db:"amount"` // cents
	Currency      string        `json:"currency" db:"currency"`
	Status        PaymentStatus `json:"status" db:"status"`
	Method        string        `json:"method" db:"method"` // "card", "wallet"
	TransactionID string        `json:"transaction_id,omitempty" db:"transaction_id"`
	FailureReason string        `json:"failure_reason,omitempty" db:"failure_reason"`
	StoreID       *string       `json:"store_id,omitempty" db:"store_id"`
	StoreOwnerID  string        `json:"store_owner_id,omitempty" db:"store_owner_id"`
	Commission    int           `json:"commission,omitempty" db:"commission"` // platform % e.g. 10
	CreatedAt     time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at" db:"updated_at"`
}

func NewPayment(orderID, userID, currency, method string, amount int64) *Payment {
	now := time.Now().UTC()
	return &Payment{
		ID:        uuid.New().String(),
		OrderID:   orderID,
		UserID:    userID,
		Amount:    amount,
		Currency:  currency,
		Status:    PaymentPending,
		Method:    method,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (p *Payment) Succeed(transactionID string) {
	p.Status = PaymentSucceeded
	p.TransactionID = transactionID
	p.UpdatedAt = time.Now().UTC()
}

func (p *Payment) Fail(reason string) {
	p.Status = PaymentFailed
	p.FailureReason = reason
	p.UpdatedAt = time.Now().UTC()
}

func (p *Payment) Refund() error {
	if p.Status != PaymentSucceeded {
		return fmt.Errorf("cannot refund payment in status %s", p.Status)
	}
	p.Status = PaymentRefunded
	p.UpdatedAt = time.Now().UTC()
	return nil
}
