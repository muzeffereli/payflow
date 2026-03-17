package port

import (
	"context"

	"payment-platform/internal/auth/domain"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
	UpdatePasswordHash(ctx context.Context, userID, hash string) error

	ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, int, error)
	UpdateRole(ctx context.Context, userID, role string) error

	SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error
	FindRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserTokens(ctx context.Context, userID string) error
}
