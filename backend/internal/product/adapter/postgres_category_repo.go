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

var _ port.CategoryRepository = (*postgresCategoryRepo)(nil)

type postgresCategoryRepo struct {
	db *sql.DB
}

func NewPostgresCategoryRepo(db *sql.DB) port.CategoryRepository {
	return &postgresCategoryRepo{db: db}
}

func (r *postgresCategoryRepo) CreateCategory(ctx context.Context, category *domain.Category) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO categories (id, name, slug, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		category.ID, category.Name, category.Slug, category.CreatedAt, category.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrCategoryConflict
		}
		return fmt.Errorf("create category: %w", err)
	}
	return nil
}

func (r *postgresCategoryRepo) GetCategoryByID(ctx context.Context, id string) (*domain.Category, error) {
	category := &domain.Category{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, slug, created_at, updated_at FROM categories WHERE id = $1`,
		id,
	).Scan(&category.ID, &category.Name, &category.Slug, &category.CreatedAt, &category.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrCategoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}
	return category, nil
}

func (r *postgresCategoryRepo) ListCategories(ctx context.Context) ([]*domain.Category, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, slug, created_at, updated_at FROM categories ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var categories []*domain.Category
	for rows.Next() {
		category := &domain.Category{}
		if err := rows.Scan(&category.ID, &category.Name, &category.Slug, &category.CreatedAt, &category.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func (r *postgresCategoryRepo) UpdateCategory(ctx context.Context, category *domain.Category) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE categories SET name = $1, slug = $2, updated_at = $3 WHERE id = $4`,
		category.Name, category.Slug, category.UpdatedAt, category.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrCategoryConflict
		}
		return fmt.Errorf("update category: %w", err)
	}
	return nil
}

func (r *postgresCategoryRepo) DeleteCategory(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

func (r *postgresCategoryRepo) CreateSubcategory(ctx context.Context, subcategory *domain.Subcategory) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO subcategories (id, category_id, name, slug, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		subcategory.ID, subcategory.CategoryID, subcategory.Name, subcategory.Slug, subcategory.CreatedAt, subcategory.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrSubcategoryConflict
		}
		return fmt.Errorf("create subcategory: %w", err)
	}
	return nil
}

func (r *postgresCategoryRepo) GetSubcategoryByID(ctx context.Context, id string) (*domain.Subcategory, error) {
	subcategory := &domain.Subcategory{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, category_id, name, slug, created_at, updated_at FROM subcategories WHERE id = $1`,
		id,
	).Scan(&subcategory.ID, &subcategory.CategoryID, &subcategory.Name, &subcategory.Slug, &subcategory.CreatedAt, &subcategory.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrSubcategoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get subcategory: %w", err)
	}
	return subcategory, nil
}

func (r *postgresCategoryRepo) ListSubcategories(ctx context.Context, categoryID string) ([]*domain.Subcategory, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, category_id, name, slug, created_at, updated_at
		 FROM subcategories
		 WHERE ($1 = '' OR category_id = $1)
		 ORDER BY name`,
		categoryID,
	)
	if err != nil {
		return nil, fmt.Errorf("list subcategories: %w", err)
	}
	defer rows.Close()

	var subcategories []*domain.Subcategory
	for rows.Next() {
		subcategory := &domain.Subcategory{}
		if err := rows.Scan(&subcategory.ID, &subcategory.CategoryID, &subcategory.Name, &subcategory.Slug, &subcategory.CreatedAt, &subcategory.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan subcategory: %w", err)
		}
		subcategories = append(subcategories, subcategory)
	}
	return subcategories, rows.Err()
}

func (r *postgresCategoryRepo) UpdateSubcategory(ctx context.Context, subcategory *domain.Subcategory) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE subcategories
		 SET category_id = $1, name = $2, slug = $3, updated_at = $4
		 WHERE id = $5`,
		subcategory.CategoryID, subcategory.Name, subcategory.Slug, subcategory.UpdatedAt, subcategory.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrSubcategoryConflict
		}
		return fmt.Errorf("update subcategory: %w", err)
	}
	return nil
}

func (r *postgresCategoryRepo) DeleteSubcategory(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM subcategories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete subcategory: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return domain.ErrSubcategoryNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") || strings.Contains(message, "duplicate")
}
