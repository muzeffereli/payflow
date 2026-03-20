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

var _ port.GlobalAttributeRepository = (*postgresGlobalAttributeRepo)(nil)

type postgresGlobalAttributeRepo struct {
	db *sql.DB
}

func NewPostgresGlobalAttributeRepo(db *sql.DB) port.GlobalAttributeRepository {
	return &postgresGlobalAttributeRepo{db: db}
}

const globalAttrSelect = `
	SELECT ga.id,
	       COALESCE(ga.subcategory_id, ''),
	       COALESCE(s.name, ga.subcategory, ''),
	       COALESCE(s.category_id, ''),
	       COALESCE(c.name, ga.category, ''),
	       ga.name, ga.values, ga.position, ga.created_at, ga.updated_at
	  FROM global_attributes ga
	  LEFT JOIN subcategories s ON s.id = ga.subcategory_id
	  LEFT JOIN categories c ON c.id = s.category_id`

func scanGlobalAttr(row interface {
	Scan(...any) error
}) (*domain.GlobalAttribute, error) {
	a := &domain.GlobalAttribute{}
	var valuesJSON []byte
	if err := row.Scan(&a.ID, &a.SubcategoryID, &a.Subcategory, &a.CategoryID, &a.Category,
		&a.Name, &valuesJSON, &a.Position, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(valuesJSON, &a.Values); err != nil {
		return nil, fmt.Errorf("unmarshal values: %w", err)
	}
	return a, nil
}

func (r *postgresGlobalAttributeRepo) Create(ctx context.Context, a *domain.GlobalAttribute) error {
	valuesJSON, err := json.Marshal(a.Values)
	if err != nil {
		return fmt.Errorf("marshal values: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO global_attributes (id, subcategory_id, subcategory, name, values, position, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		a.ID, a.SubcategoryID, a.Subcategory, a.Name, valuesJSON, a.Position, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrGlobalAttributeConflict
		}
		return fmt.Errorf("insert global attribute: %w", err)
	}
	return nil
}

func (r *postgresGlobalAttributeRepo) GetByID(ctx context.Context, id string) (*domain.GlobalAttribute, error) {
	row := r.db.QueryRowContext(ctx, globalAttrSelect+` WHERE ga.id = $1`, id)
	a, err := scanGlobalAttr(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrGlobalAttributeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get global attribute: %w", err)
	}
	return a, nil
}

func (r *postgresGlobalAttributeRepo) List(ctx context.Context, filter port.GlobalAttributeFilter) ([]*domain.GlobalAttribute, error) {
	query := globalAttrSelect
	args := []interface{}{}
	// Only return rows that have been migrated to the subcategory-based model
	where := []string{`ga.subcategory_id IS NOT NULL`}

	if filter.SubcategoryID != "" {
		where = append(where, `ga.subcategory_id = $1`)
		args = append(args, filter.SubcategoryID)
	} else if filter.CategoryID != "" {
		where = append(where, `s.category_id = $1`)
		args = append(args, filter.CategoryID)
	}

	query += ` WHERE ` + strings.Join(where, " AND ")
	query += ` ORDER BY COALESCE(c.name, ''), COALESCE(s.name, ''), ga.position, ga.name`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list global attributes: %w", err)
	}
	defer rows.Close()

	var attrs []*domain.GlobalAttribute
	for rows.Next() {
		a, err := scanGlobalAttr(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		attrs = append(attrs, a)
	}
	return attrs, rows.Err()
}

func (r *postgresGlobalAttributeRepo) ListCategories(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT COALESCE(c.name, '')
		   FROM global_attributes ga
		   LEFT JOIN subcategories s ON s.id = ga.subcategory_id
		   LEFT JOIN categories c ON c.id = s.category_id
		  WHERE COALESCE(c.name, '') <> ''
		  ORDER BY COALESCE(c.name, '')`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func (r *postgresGlobalAttributeRepo) Update(ctx context.Context, a *domain.GlobalAttribute) error {
	valuesJSON, err := json.Marshal(a.Values)
	if err != nil {
		return fmt.Errorf("marshal values: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE global_attributes SET subcategory_id = $1, subcategory = $2, name = $3, values = $4, position = $5, updated_at = $6
		 WHERE id = $7`,
		a.SubcategoryID, a.Subcategory, a.Name, valuesJSON, a.Position, a.UpdatedAt, a.ID)
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrGlobalAttributeConflict
		}
		return fmt.Errorf("update global attribute: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrGlobalAttributeNotFound
	}
	return nil
}

func isUniqueConstraintError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") || strings.Contains(message, "duplicate")
}

func (r *postgresGlobalAttributeRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM global_attributes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete global attribute: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrGlobalAttributeNotFound
	}
	return nil
}
