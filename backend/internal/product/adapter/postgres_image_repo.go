package adapter

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"payment-platform/internal/product/domain"
	"payment-platform/internal/product/port"
)

var _ port.ImageRepository = (*postgresImageRepo)(nil)

type postgresImageRepo struct {
	db *sql.DB
}

func NewPostgresImageRepo(db *sql.DB) port.ImageRepository {
	return &postgresImageRepo{db: db}
}

func (r *postgresImageRepo) SetImages(ctx context.Context, productID string, urls []string) ([]*domain.ProductImage, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM product_images WHERE product_id = $1`, productID); err != nil {
		return nil, fmt.Errorf("delete images: %w", err)
	}

	images := make([]*domain.ProductImage, 0, len(urls))
	for i, url := range urls {
		img := &domain.ProductImage{
			ID:        uuid.New().String(),
			ProductID: productID,
			URL:       url,
			Position:  i,
			CreatedAt: time.Now().UTC(),
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO product_images (id, product_id, url, position, created_at) VALUES ($1, $2, $3, $4, $5)`,
			img.ID, img.ProductID, img.URL, img.Position, img.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("insert image: %w", err)
		}
		images = append(images, img)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return images, nil
}

func (r *postgresImageRepo) GetByProductID(ctx context.Context, productID string) ([]*domain.ProductImage, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, product_id, url, position, created_at FROM product_images WHERE product_id = $1 ORDER BY position ASC`,
		productID,
	)
	if err != nil {
		return nil, fmt.Errorf("query images: %w", err)
	}
	defer rows.Close()

	var images []*domain.ProductImage
	for rows.Next() {
		img := &domain.ProductImage{}
		if err := rows.Scan(&img.ID, &img.ProductID, &img.URL, &img.Position, &img.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan image: %w", err)
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

func (r *postgresImageRepo) GetByProductIDs(ctx context.Context, productIDs []string) (map[string][]*domain.ProductImage, error) {
	if len(productIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(productIDs))
	args := make([]interface{}, len(productIDs))
	for i, id := range productIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, product_id, url, position, created_at FROM product_images
		 WHERE product_id IN (`+strings.Join(placeholders, ",")+`)
		 ORDER BY product_id, position ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("batch query images: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]*domain.ProductImage)
	for rows.Next() {
		img := &domain.ProductImage{}
		if err := rows.Scan(&img.ID, &img.ProductID, &img.URL, &img.Position, &img.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan image: %w", err)
		}
		result[img.ProductID] = append(result[img.ProductID], img)
	}
	return result, rows.Err()
}

func (r *postgresImageRepo) setSQLDB() *sql.DB { return r.db }
