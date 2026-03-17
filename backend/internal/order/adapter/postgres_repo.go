package adapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"payment-platform/internal/order/domain"
	"payment-platform/internal/order/port"
	"payment-platform/internal/order/service"
	"payment-platform/pkg/outbox"
)

var _ port.OrderRepository = (*postgresRepo)(nil)

type postgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) port.OrderRepository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) CreateWithOutbox(ctx context.Context, order *domain.Order, subject string, payload []byte) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.insertOrder(ctx, tx, order); err != nil {
		return err
	}
	if err := outbox.Write(ctx, tx, subject, payload); err != nil {
		return fmt.Errorf("write outbox: %w", err)
	}
	return tx.Commit()
}

func (r *postgresRepo) Create(ctx context.Context, order *domain.Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.insertOrder(ctx, tx, order); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *postgresRepo) insertOrder(ctx context.Context, tx *sql.Tx, order *domain.Order) error {
	var addrJSON []byte
	if order.ShippingAddress != nil {
		addrJSON, _ = json.Marshal(order.ShippingAddress)
	}

	var storeID *string
	if order.StoreID != nil {
		storeID = order.StoreID
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO orders (id, user_id, store_id, status, total_amount, currency, idempotency_key, shipping_address, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		order.ID, order.UserID, storeID, order.Status, order.TotalAmount,
		order.Currency, order.IdempotencyKey, nullJSON(addrJSON), order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	for _, item := range order.Items {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO order_items (id, order_id, product_id, variant_id, variant_sku, variant_label, quantity, price)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			item.ID, item.OrderID, item.ProductID, item.VariantID, item.VariantSKU, item.VariantLabel, item.Quantity, item.Price,
		)
		if err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}
	}
	return nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	order := &domain.Order{}
	var addrJSON sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, store_id, status, total_amount, currency, idempotency_key, shipping_address, created_at, updated_at
		FROM orders WHERE id = $1`, id,
	).Scan(
		&order.ID, &order.UserID, &order.StoreID, &order.Status, &order.TotalAmount,
		&order.Currency, &order.IdempotencyKey, &addrJSON, &order.CreatedAt, &order.UpdatedAt,
	)
	if addrJSON.Valid {
		var addr domain.ShippingAddress
		if err := json.Unmarshal([]byte(addrJSON.String), &addr); err == nil {
			order.ShippingAddress = &addr
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query order: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, order_id, product_id, variant_id, variant_sku, variant_label, quantity, price
		FROM order_items WHERE order_id = $1`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close() // always close rows to release the DB connection back to the pool

	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.VariantID, &item.VariantSKU, &item.VariantLabel, &item.Quantity, &item.Price); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		order.Items = append(order.Items, item)
	}

	return order, rows.Err() // rows.Err() catches errors that occurred during iteration
}

func (r *postgresRepo) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error) {
	order := &domain.Order{}
	var addrJSON sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, store_id, status, total_amount, currency, idempotency_key, shipping_address, created_at, updated_at
		FROM orders WHERE idempotency_key = $1`, key,
	).Scan(
		&order.ID, &order.UserID, &order.StoreID, &order.Status, &order.TotalAmount,
		&order.Currency, &order.IdempotencyKey, &addrJSON, &order.CreatedAt, &order.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // not found = nil, nil (by design)
	}
	if err != nil {
		return nil, fmt.Errorf("query by idempotency key: %w", err)
	}
	if addrJSON.Valid {
		var addr domain.ShippingAddress
		if err := json.Unmarshal([]byte(addrJSON.String), &addr); err == nil {
			order.ShippingAddress = &addr
		}
	}
	return order, nil
}

func (r *postgresRepo) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	return err
}

func (r *postgresRepo) ListByUser(ctx context.Context, userID string, limit, offset int) ([]domain.Order, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, store_id, status, total_amount, currency, idempotency_key, shipping_address, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	return r.scanOrders(ctx, rows)
}

func (r *postgresRepo) ListByStore(ctx context.Context, storeID string, limit, offset int) ([]domain.Order, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, store_id, status, total_amount, currency, idempotency_key, shipping_address, created_at, updated_at
		FROM orders
		WHERE store_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		storeID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list store orders: %w", err)
	}
	defer rows.Close()

	return r.scanOrders(ctx, rows)
}

func (r *postgresRepo) GetStoreAnalytics(ctx context.Context, storeID string) (*port.StoreAnalytics, error) {
	a := &port.StoreAnalytics{}
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(total_amount), 0),
			COUNT(CASE WHEN status = 'paid' THEN 1 END),
			COUNT(CASE WHEN status = 'pending' THEN 1 END)
		FROM orders
		WHERE store_id = $1`, storeID).
		Scan(&a.TotalOrders, &a.TotalRevenue, &a.PaidOrders, &a.PendingOrders)
	if err != nil {
		return nil, fmt.Errorf("get store analytics: %w", err)
	}
	return a, nil
}

func (r *postgresRepo) scanOrders(ctx context.Context, rows *sql.Rows) ([]domain.Order, error) {
	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		var addrJSON sql.NullString
		if err := rows.Scan(&o.ID, &o.UserID, &o.StoreID, &o.Status, &o.TotalAmount,
			&o.Currency, &o.IdempotencyKey, &addrJSON, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		if addrJSON.Valid {
			var addr domain.ShippingAddress
			if err := json.Unmarshal([]byte(addrJSON.String), &addr); err == nil {
				o.ShippingAddress = &addr
			}
		}
		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range orders {
		items, err := r.listOrderItems(ctx, orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].Items = items
	}

	return orders, nil
}

func (r *postgresRepo) listOrderItems(ctx context.Context, orderID string) ([]domain.OrderItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, order_id, product_id, variant_id, variant_sku, variant_label, quantity, price
		FROM order_items
		WHERE order_id = $1
		ORDER BY id`, orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	var items []domain.OrderItem
	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.VariantID, &item.VariantSKU, &item.VariantLabel, &item.Quantity, &item.Price); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate items: %w", err)
	}

	return items, nil
}

func nullJSON(data []byte) interface{} {
	if len(data) == 0 {
		return nil
	}
	return string(data)
}
