package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"payment-platform/internal/auth/domain"
	"payment-platform/internal/auth/port"

	"github.com/lib/pq"
)

var _ port.UserRepository = (*postgresRepo)(nil)

type postgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) port.UserRepository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) CreateUser(ctx context.Context, user *domain.User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, role, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID, user.Email, user.Name, user.Role, user.PasswordHash, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailTaken
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *postgresRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, name, role, password_hash, created_at, updated_at
		FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return u, nil
}

func (r *postgresRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, name, role, password_hash, created_at, updated_at
		FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return u, nil
}

func (r *postgresRepo) UpdatePasswordHash(ctx context.Context, userID, hash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		hash, userID,
	)
	return err
}

func (r *postgresRepo) ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, name, role, password_hash, created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u := &domain.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *postgresRepo) UpdateRole(ctx context.Context, userID, role string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`, role, userID)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *postgresRepo) SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

func (r *postgresRepo) FindRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	rt := &domain.RefreshToken{}
	var revokedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens WHERE token_hash = $1`, tokenHash,
	).Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &revokedAt, &rt.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // not found = nil (caller checks)
	}
	if err != nil {
		return nil, fmt.Errorf("find refresh token: %w", err)
	}
	if revokedAt.Valid {
		t := revokedAt.Time
		rt.RevokedAt = &t
	}
	return rt, nil
}

func (r *postgresRepo) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE token_hash = $2 AND revoked_at IS NULL`,
		now, tokenHash,
	)
	return err
}

func (r *postgresRepo) RevokeAllUserTokens(ctx context.Context, userID string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`,
		now, userID,
	)
	return err
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
