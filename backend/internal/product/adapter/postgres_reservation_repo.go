package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"payment-platform/internal/product/domain"
	"payment-platform/internal/product/port"
)

var _ port.ReservationRepository = (*postgresReservationRepo)(nil)

type postgresReservationRepo struct {
	db *sql.DB
}

func NewPostgresReservationRepo(db *sql.DB) port.ReservationRepository {
	return &postgresReservationRepo{db: db}
}

func (r *postgresReservationRepo) Save(ctx context.Context, res *domain.Reservation) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO stock_reservations (id, order_id, product_id, variant_id, quantity, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		res.ID, res.OrderID, res.ProductID, res.VariantID, res.Quantity, res.Status, res.CreatedAt, res.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("save reservation: %w", err)
	}
	return nil
}

func (r *postgresReservationRepo) GetByOrderID(ctx context.Context, orderID string) ([]*domain.Reservation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, order_id, product_id, variant_id, quantity, status, created_at, updated_at
		FROM stock_reservations WHERE order_id = $1`, orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("query reservations: %w", err)
	}
	defer rows.Close()

	var reservations []*domain.Reservation
	for rows.Next() {
		res := &domain.Reservation{}
		if err := rows.Scan(&res.ID, &res.OrderID, &res.ProductID, &res.VariantID, &res.Quantity,
			&res.Status, &res.CreatedAt, &res.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan reservation: %w", err)
		}
		reservations = append(reservations, res)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(reservations) == 0 {
		return nil, errors.New("no reservations found for order")
	}
	return reservations, nil
}

func (r *postgresReservationRepo) UpdateStatus(ctx context.Context, orderID string, status domain.ReservationStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE stock_reservations SET status = $1, updated_at = NOW() WHERE order_id = $2`,
		status, orderID,
	)
	return err
}
