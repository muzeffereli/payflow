package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusConfirmed OrderStatus = "confirmed"
	StatusPaid      OrderStatus = "paid"
	StatusCancelled OrderStatus = "cancelled"
	StatusRefunded  OrderStatus = "refunded"
)

type ShippingAddress struct {
	Name       string `json:"name"`
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"` // ISO 3166-1 alpha-2, e.g. "US"
}

type Order struct {
	ID              string           `json:"id" db:"id"`
	UserID          string           `json:"user_id" db:"user_id"`
	StoreID         *string          `json:"store_id,omitempty" db:"store_id"` // nil = platform order
	Status          OrderStatus      `json:"status" db:"status"`
	Items           []OrderItem      `json:"items"`
	TotalAmount     int64            `json:"total_amount" db:"total_amount"` // cents
	Currency        string           `json:"currency" db:"currency"`
	IdempotencyKey  string           `json:"idempotency_key" db:"idempotency_key"`
	ShippingAddress *ShippingAddress `json:"shipping_address,omitempty" db:"shipping_address"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at" db:"updated_at"`
}

type OrderItem struct {
	ID           string  `json:"id" db:"id"`
	OrderID      string  `json:"order_id" db:"order_id"`
	ProductID    string  `json:"product_id" db:"product_id"`
	VariantID    *string `json:"variant_id,omitempty" db:"variant_id"`
	VariantSKU   string  `json:"variant_sku,omitempty" db:"variant_sku"`
	VariantLabel string  `json:"variant_label,omitempty" db:"variant_label"`
	Quantity     int     `json:"quantity" db:"quantity"`
	Price        int64   `json:"price" db:"price"` // unit price in cents
}

func NewOrder(userID, currency, idempotencyKey string, items []OrderItem, addr *ShippingAddress) *Order {
	id := uuid.New().String()
	now := time.Now().UTC()

	var total int64
	for i := range items {
		items[i].ID = uuid.New().String()
		items[i].OrderID = id
		total += items[i].Price * int64(items[i].Quantity)
	}

	return &Order{
		ID:              id,
		UserID:          userID,
		Status:          StatusPending,
		Items:           items,
		TotalAmount:     total,
		Currency:        currency,
		IdempotencyKey:  idempotencyKey,
		ShippingAddress: addr,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func (o *Order) Transition(to OrderStatus) error {
	allowed := map[OrderStatus][]OrderStatus{
		StatusPending:   {StatusConfirmed, StatusCancelled},
		StatusConfirmed: {StatusPaid, StatusCancelled},
		StatusPaid:      {StatusRefunded},
	}

	for _, next := range allowed[o.Status] {
		if to == next {
			o.Status = to
			o.UpdatedAt = time.Now().UTC()
			return nil
		}
	}

	return &InvalidTransitionError{From: o.Status, To: to}
}

type InvalidTransitionError struct {
	From, To OrderStatus
}

func (e *InvalidTransitionError) Error() string {
	return fmt.Sprintf("invalid transition: %s â†’ %s", e.From, e.To)
}
