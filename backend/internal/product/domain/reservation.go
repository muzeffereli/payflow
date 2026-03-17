package domain

import (
	"time"

	"github.com/google/uuid"
)

type ReservationStatus string

const (
	ReservationReserved  ReservationStatus = "reserved"
	ReservationCommitted ReservationStatus = "committed" // payment succeeded â€” stock stays decremented
	ReservationReleased  ReservationStatus = "released"  // payment failed / order cancelled â€” stock returned
)

type Reservation struct {
	ID        string            `db:"id"`
	OrderID   string            `db:"order_id"`
	ProductID string            `db:"product_id"`
	VariantID *string           `db:"variant_id"`
	Quantity  int               `db:"quantity"`
	Status    ReservationStatus `db:"status"`
	CreatedAt time.Time         `db:"created_at"`
	UpdatedAt time.Time         `db:"updated_at"`
}

func NewReservation(orderID, productID string, variantID *string, quantity int) *Reservation {
	now := time.Now().UTC()
	return &Reservation{
		ID:        uuid.New().String(),
		OrderID:   orderID,
		ProductID: productID,
		VariantID: variantID,
		Quantity:  quantity,
		Status:    ReservationReserved,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
