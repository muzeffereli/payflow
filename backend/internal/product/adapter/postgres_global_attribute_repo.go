package adapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

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

func (r *postgresGlobalAttributeRepo) Create(ctx context.Context, a *domain.GlobalAttribute) error {
	valuesJSON, err := json.Marshal(a.Values)
	if err != nil {
		return fmt.Errorf("marshal values: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO global_attributes (id, name, values, position, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		a.ID, a.Name, valuesJSON, a.Position, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert global attribute: %w", err)
	}
	return nil
}

func (r *postgresGlobalAttributeRepo) GetByID(ctx context.Context, id string) (*domain.GlobalAttribute, error) {
	a := &domain.GlobalAttribute{}
	var valuesJSON []byte
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, values, position, created_at, updated_at
		 FROM global_attributes WHERE id = $1`, id).
		Scan(&a.ID, &a.Name, &valuesJSON, &a.Position, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrGlobalAttributeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get global attribute: %w", err)
	}
	if err := json.Unmarshal(valuesJSON, &a.Values); err != nil {
		return nil, fmt.Errorf("unmarshal values: %w", err)
	}
	return a, nil
}

func (r *postgresGlobalAttributeRepo) List(ctx context.Context) ([]*domain.GlobalAttribute, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, values, position, created_at, updated_at
		 FROM global_attributes ORDER BY position, name`)
	if err != nil {
		return nil, fmt.Errorf("list global attributes: %w", err)
	}
	defer rows.Close()

	var attrs []*domain.GlobalAttribute
	for rows.Next() {
		a := &domain.GlobalAttribute{}
		var valuesJSON []byte
		if err := rows.Scan(&a.ID, &a.Name, &valuesJSON, &a.Position, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if err := json.Unmarshal(valuesJSON, &a.Values); err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}
		attrs = append(attrs, a)
	}
	return attrs, rows.Err()
}

func (r *postgresGlobalAttributeRepo) Update(ctx context.Context, a *domain.GlobalAttribute) error {
	valuesJSON, err := json.Marshal(a.Values)
	if err != nil {
		return fmt.Errorf("marshal values: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE global_attributes SET name = $1, values = $2, position = $3, updated_at = $4
		 WHERE id = $5`,
		a.Name, valuesJSON, a.Position, a.UpdatedAt, a.ID)
	if err != nil {
		return fmt.Errorf("update global attribute: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrGlobalAttributeNotFound
	}
	return nil
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
