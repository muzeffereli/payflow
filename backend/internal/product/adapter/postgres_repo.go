package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"payment-platform/internal/product/domain"
	"payment-platform/internal/product/port"
)

var _ port.ProductRepository = (*postgresRepo)(nil)

type postgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) port.ProductRepository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) Create(ctx context.Context, p *domain.Product) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO products (id, name, description, sku, price, currency, stock, category_id, category, subcategory_id, status, store_id, image_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		p.ID, p.Name, p.Description, p.SKU, p.Price, p.Currency,
		p.Stock, p.CategoryID, p.Category, p.SubcategoryID, p.Status, p.StoreID, p.ImageURL, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return domain.ErrSKUConflict
		}
		return fmt.Errorf("insert product: %w", err)
	}
	return nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	p := &domain.Product{}
	err := r.db.QueryRowContext(ctx, `
		SELECT p.id, p.name, p.description, p.sku, p.price, p.currency, p.stock, p.category_id,
		       COALESCE(c.name, p.category, ''), p.subcategory_id, COALESCE(sc.name, ''), p.status,
		       p.store_id, p.image_url, p.created_at, p.updated_at
		FROM products p
		LEFT JOIN categories c ON c.id = p.category_id
		LEFT JOIN subcategories sc ON sc.id = p.subcategory_id
		WHERE p.id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.Currency,
		&p.Stock, &p.CategoryID, &p.Category, &p.SubcategoryID, &p.Subcategory, &p.Status, &p.StoreID, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrProductNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	return p, nil
}

func (r *postgresRepo) GetByIDs(ctx context.Context, ids []string) ([]*domain.Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT p.id, p.name, p.description, p.sku, p.price, p.currency, p.stock, p.category_id,
		        COALESCE(c.name, p.category, ''), p.subcategory_id, COALESCE(sc.name, ''), p.status,
		        p.store_id, p.image_url, p.created_at, p.updated_at
		   FROM products p
		   LEFT JOIN categories c ON c.id = p.category_id
		   LEFT JOIN subcategories sc ON sc.id = p.subcategory_id
		  WHERE p.id IN (`+strings.Join(placeholders, ",")+`)`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("get products by ids: %w", err)
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		p := &domain.Product{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.Currency,
			&p.Stock, &p.CategoryID, &p.Category, &p.SubcategoryID, &p.Subcategory, &p.Status, &p.StoreID, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *postgresRepo) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	p := &domain.Product{}
	err := r.db.QueryRowContext(ctx, `
		SELECT p.id, p.name, p.description, p.sku, p.price, p.currency, p.stock, p.category_id,
		       COALESCE(c.name, p.category, ''), p.subcategory_id, COALESCE(sc.name, ''), p.status,
		       p.store_id, p.image_url, p.created_at, p.updated_at
		  FROM products p
		  LEFT JOIN categories c ON c.id = p.category_id
		  LEFT JOIN subcategories sc ON sc.id = p.subcategory_id
		 WHERE p.sku = $1`, sku,
	).Scan(&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.Currency,
		&p.Stock, &p.CategoryID, &p.Category, &p.SubcategoryID, &p.Subcategory, &p.Status, &p.StoreID, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrProductNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get product by sku: %w", err)
	}
	return p, nil
}

func (r *postgresRepo) List(ctx context.Context, f port.ListFilter) ([]*domain.Product, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if f.Category != "" {
		where = append(where, fmt.Sprintf("LOWER(p.category) = LOWER($%d)", idx))
		args = append(args, f.Category)
		idx++
	}
	if f.CategoryID != "" {
		where = append(where, fmt.Sprintf("p.category_id = $%d", idx))
		args = append(args, f.CategoryID)
		idx++
	}
	if f.SubcategoryID != "" {
		where = append(where, fmt.Sprintf("p.subcategory_id = $%d", idx))
		args = append(args, f.SubcategoryID)
		idx++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("p.status = $%d", idx))
		args = append(args, f.Status)
		idx++
	}
	if f.StoreID != "" {
		where = append(where, fmt.Sprintf("p.store_id = $%d", idx))
		args = append(args, f.StoreID)
		idx++
	}
	if strings.TrimSpace(f.Search) != "" {
		where = append(where, fmt.Sprintf("(LOWER(p.name) LIKE LOWER($%d) OR LOWER(p.description) LIKE LOWER($%d))", idx, idx))
		args = append(args, "%"+strings.TrimSpace(f.Search)+"%")
		idx++
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM products p WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	query := `SELECT p.id, p.name, p.description, p.sku, p.price, p.currency, p.stock, p.category_id,
	                 COALESCE(c.name, p.category, ''), p.subcategory_id, COALESCE(sc.name, ''), p.status,
	                 p.store_id, p.image_url, p.created_at, p.updated_at
	            FROM products p
	            LEFT JOIN categories c ON c.id = p.category_id
	            LEFT JOIN subcategories sc ON sc.id = p.subcategory_id
	           WHERE ` + whereClause + ` ORDER BY p.created_at DESC`
	if f.Limit > 0 {
		args = append(args, f.Limit, f.Offset)
		query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, idx, idx+1)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		p := &domain.Product{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.Currency,
			&p.Stock, &p.CategoryID, &p.Category, &p.SubcategoryID, &p.Subcategory, &p.Status, &p.StoreID, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	return products, total, rows.Err()
}

func (r *postgresRepo) Update(ctx context.Context, p *domain.Product) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE products
		SET name=$1, description=$2, price=$3, stock=$4, category_id=$5, category=$6, subcategory_id=$7, status=$8, image_url=$9, updated_at=$10
		WHERE id=$11`,
		p.Name, p.Description, p.Price, p.Stock, p.CategoryID, p.Category, p.SubcategoryID, p.Status, p.ImageURL, p.UpdatedAt, p.ID,
	)
	return err
}
