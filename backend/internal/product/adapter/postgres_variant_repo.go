package adapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"payment-platform/internal/product/domain"
	"payment-platform/internal/product/port"
)

var _ port.VariantRepository = (*postgresVariantRepo)(nil)

type postgresVariantRepo struct {
	db *sql.DB
}

func NewPostgresVariantRepo(db *sql.DB) port.VariantRepository {
	return &postgresVariantRepo{db: db}
}

func (r *postgresVariantRepo) Create(ctx context.Context, v *domain.Variant) error {
	attrJSON, _ := json.Marshal(v.AttributeValues)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO product_variants (id, product_id, sku, price, stock, attribute_values, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		v.ID, v.ProductID, v.SKU, v.Price, v.Stock, attrJSON, v.Status, v.CreatedAt, v.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return domain.ErrVariantSKUConflict
		}
		return fmt.Errorf("insert variant: %w", err)
	}
	return nil
}

func (r *postgresVariantRepo) Update(ctx context.Context, v *domain.Variant) error {
	attrJSON, _ := json.Marshal(v.AttributeValues)
	_, err := r.db.ExecContext(ctx, `
		UPDATE product_variants
		SET sku=$1, price=$2, stock=$3, attribute_values=$4, status=$5, updated_at=$6
		WHERE id=$7`,
		v.SKU, v.Price, v.Stock, attrJSON, v.Status, v.UpdatedAt, v.ID,
	)
	return err
}

func (r *postgresVariantRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM product_variants WHERE id = $1`, id)
	return err
}

func (r *postgresVariantRepo) GetByID(ctx context.Context, id string) (*domain.Variant, error) {
	v := &domain.Variant{}
	var attrJSON []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT id, product_id, sku, price, stock, attribute_values, status, created_at, updated_at
		FROM product_variants WHERE id = $1`, id,
	).Scan(&v.ID, &v.ProductID, &v.SKU, &v.Price, &v.Stock, &attrJSON, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrVariantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get variant: %w", err)
	}
	if err := json.Unmarshal(attrJSON, &v.AttributeValues); err != nil {
		return nil, fmt.Errorf("unmarshal variant attributes: %w", err)
	}
	return v, nil
}

func (r *postgresVariantRepo) ListByProduct(ctx context.Context, productID string) ([]*domain.Variant, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, product_id, sku, price, stock, attribute_values, status, created_at, updated_at
		FROM product_variants WHERE product_id = $1 ORDER BY created_at`, productID)
	if err != nil {
		return nil, fmt.Errorf("list variants: %w", err)
	}
	defer rows.Close()

	var variants []*domain.Variant
	for rows.Next() {
		v := &domain.Variant{}
		var attrJSON []byte
		if err := rows.Scan(&v.ID, &v.ProductID, &v.SKU, &v.Price, &v.Stock, &attrJSON, &v.Status, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan variant: %w", err)
		}
		if err := json.Unmarshal(attrJSON, &v.AttributeValues); err != nil {
			return nil, fmt.Errorf("unmarshal variant attributes: %w", err)
		}
		variants = append(variants, v)
	}
	return variants, rows.Err()
}

func (r *postgresVariantRepo) GetBySKU(ctx context.Context, sku string) (*domain.Variant, error) {
	v := &domain.Variant{}
	var attrJSON []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT id, product_id, sku, price, stock, attribute_values, status, created_at, updated_at
		FROM product_variants WHERE sku = $1`, sku,
	).Scan(&v.ID, &v.ProductID, &v.SKU, &v.Price, &v.Stock, &attrJSON, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrVariantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get variant by sku: %w", err)
	}
	if err := json.Unmarshal(attrJSON, &v.AttributeValues); err != nil {
		return nil, fmt.Errorf("unmarshal variant attributes: %w", err)
	}
	return v, nil
}
