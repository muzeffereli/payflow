package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"payment-platform/internal/store/domain"
	"payment-platform/internal/store/port"
)

var _ port.StoreRepository = (*postgresRepo)(nil)

type postgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) port.StoreRepository {
	return &postgresRepo{db: db}
}

const storeColumns = `id, owner_id, name, description, email, commission, status, created_at, updated_at`

func scanStore(row interface {
	Scan(dest ...interface{}) error
}) (*domain.Store, error) {
	s := &domain.Store{}
	err := row.Scan(&s.ID, &s.OwnerID, &s.Name, &s.Description, &s.Email,
		&s.Commission, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrStoreNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan store: %w", err)
	}
	return s, nil
}

func (r *postgresRepo) Create(ctx context.Context, s *domain.Store) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO stores (id, owner_id, name, description, email, commission, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		s.ID, s.OwnerID, s.Name, s.Description, s.Email,
		s.Commission, s.Status, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return domain.ErrOwnerAlreadyHasStore
		}
		return fmt.Errorf("insert store: %w", err)
	}
	return nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*domain.Store, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+storeColumns+` FROM stores WHERE id = $1`, id)
	return scanStore(row)
}

func (r *postgresRepo) GetByOwnerID(ctx context.Context, ownerID string) (*domain.Store, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+storeColumns+` FROM stores WHERE owner_id = $1`, ownerID)
	return scanStore(row)
}

func (r *postgresRepo) List(ctx context.Context, f port.ListFilter) ([]*domain.Store, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if f.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", idx))
		args = append(args, f.Status)
		idx++
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM stores WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count stores: %w", err)
	}

	args = append(args, f.Limit, f.Offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+storeColumns+` FROM stores WHERE `+whereClause+
			fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, idx, idx+1),
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list stores: %w", err)
	}
	defer rows.Close()

	var stores []*domain.Store
	for rows.Next() {
		s, err := scanStore(rows)
		if err != nil {
			return nil, 0, err
		}
		stores = append(stores, s)
	}
	return stores, total, rows.Err()
}

func (r *postgresRepo) Update(ctx context.Context, s *domain.Store) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE stores
		SET name=$1, description=$2, email=$3, commission=$4, status=$5, updated_at=$6
		WHERE id=$7`,
		s.Name, s.Description, s.Email, s.Commission, s.Status, s.UpdatedAt, s.ID,
	)
	return err
}
