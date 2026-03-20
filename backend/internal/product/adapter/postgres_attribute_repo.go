package adapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"payment-platform/internal/product/domain"
	"payment-platform/internal/product/port"
)

var _ port.AttributeRepository = (*postgresAttributeRepo)(nil)

type postgresAttributeRepo struct {
	db *sql.DB
}

func NewPostgresAttributeRepo(db *sql.DB) port.AttributeRepository {
	return &postgresAttributeRepo{db: db}
}

func (r *postgresAttributeRepo) SaveBatch(ctx context.Context, attrs []*domain.Attribute) error {
	if len(attrs) == 0 {
		return nil
	}

	productID := attrs[0].ProductID
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM product_attributes WHERE product_id = $1`, productID); err != nil {
		return fmt.Errorf("delete old attributes: %w", err)
	}

	valueStrings := make([]string, 0, len(attrs))
	args := make([]interface{}, 0, len(attrs)*8)
	for i, a := range attrs {
		valuesJSON, _ := json.Marshal(a.Values)
		base := i * 8
		valueStrings = append(valueStrings, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)", base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8))
		args = append(args, a.ID, a.ProductID, a.GlobalAttributeID, a.Name, valuesJSON, valuesJSON, a.Position, a.CreatedAt)
	}

	query := `INSERT INTO product_attributes (id, product_id, global_attribute_id, name, values, selected_values, position, created_at) VALUES ` + strings.Join(valueStrings, ",")
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("insert attributes: %w", err)
	}

	return tx.Commit()
}

func (r *postgresAttributeRepo) ListByProduct(ctx context.Context, productID string) ([]*domain.Attribute, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, product_id, global_attribute_id, name, values, position, created_at
		FROM product_attributes WHERE product_id = $1 ORDER BY position`, productID)
	if err != nil {
		return nil, fmt.Errorf("list attributes: %w", err)
	}
	defer rows.Close()

	var attrs []*domain.Attribute
	for rows.Next() {
		a := &domain.Attribute{}
		var valuesJSON []byte
		if err := rows.Scan(&a.ID, &a.ProductID, &a.GlobalAttributeID, &a.Name, &valuesJSON, &a.Position, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan attribute: %w", err)
		}
		if err := json.Unmarshal(valuesJSON, &a.Values); err != nil {
			return nil, fmt.Errorf("unmarshal attribute values: %w", err)
		}
		attrs = append(attrs, a)
	}
	return attrs, rows.Err()
}

func (r *postgresAttributeRepo) DeleteByProduct(ctx context.Context, productID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM product_attributes WHERE product_id = $1`, productID)
	return err
}
