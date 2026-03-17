package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"payment-platform/internal/auth/domain"
	"payment-platform/internal/auth/port"
	"payment-platform/pkg/auth"
	"payment-platform/pkg/eventbus"
)

var ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // access token TTL in seconds
}

type AuthService struct {
	repo      port.UserRepository
	publisher port.EventPublisher
	jwt       *auth.JWTManager
	log       *slog.Logger
}

func New(repo port.UserRepository, publisher port.EventPublisher, jwt *auth.JWTManager, log *slog.Logger) *AuthService {
	return &AuthService{repo: repo, publisher: publisher, jwt: jwt, log: log}
}

func (s *AuthService) Register(ctx context.Context, email, name, password string) (*TokenPair, error) {
	user, err := domain.NewUser(email, name, password)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err // ErrEmailTaken bubbles up here
	}

	s.log.Info("user registered", "user_id", user.ID, "email", user.Email)
	s.publishUserRegistered(ctx, user)

	return s.issueTokenPair(ctx, user.ID, user.Role)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := user.CheckPassword(password); err != nil {
		return nil, err // ErrInvalidCredentials
	}

	s.log.Info("user logged in", "user_id", user.ID)
	return s.issueTokenPair(ctx, user.ID, user.Role)
}

func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (*TokenPair, error) {
	claims, err := s.jwt.Verify(rawRefreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}
	if claims.TokenType != auth.TokenTypeRefresh {
		return nil, ErrInvalidRefreshToken
	}

	tokenHash := domain.HashToken(rawRefreshToken)
	stored, err := s.repo.FindRefreshToken(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if stored == nil || !stored.IsValid() {
		return nil, ErrInvalidRefreshToken
	}

	if err := s.repo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return nil, err
	}

	s.log.Info("token refreshed", "user_id", claims.UserID)
	role := claims.Role
	if role == "" {
		role = "customer" // backward compat: old tokens without role claim
	}
	return s.issueTokenPair(ctx, claims.UserID, role)
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	tokenHash := domain.HashToken(rawRefreshToken)
	if err := s.repo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return err
	}
	s.log.Info("user logged out")
	return nil
}

func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	return s.repo.RevokeAllUserTokens(ctx, userID)
}

func (s *AuthService) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	return s.repo.FindByID(ctx, userID)
}

func (s *AuthService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := user.CheckPassword(oldPassword); err != nil {
		return err // ErrInvalidCredentials
	}

	if err := user.UpdatePassword(newPassword); err != nil {
		return err
	}

	if err := s.repo.UpdatePasswordHash(ctx, user.ID, user.PasswordHash); err != nil {
		return err
	}

	return s.repo.RevokeAllUserTokens(ctx, userID)
}

func (s *AuthService) ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, int, error) {
	return s.repo.ListUsers(ctx, limit, offset)
}

func (s *AuthService) UpdateUserRole(ctx context.Context, userID, role string) error {
	validRoles := map[string]bool{domain.RoleCustomer: true, domain.RoleAdmin: true, domain.RoleSeller: true}
	if !validRoles[role] {
		return errors.New("invalid role")
	}
	return s.repo.UpdateRole(ctx, userID, role)
}

func (s *AuthService) issueTokenPair(ctx context.Context, userID, role string) (*TokenPair, error) {
	accessToken, err := s.jwt.SignAccess(userID, role)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwt.SignRefresh(userID, role)
	if err != nil {
		return nil, err
	}

	rt := domain.NewRefreshToken(userID, refreshToken, auth.RefreshTokenTTL)
	if err := s.repo.SaveRefreshToken(ctx, rt); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(auth.AccessTokenTTL.Seconds()),
	}, nil
}

func (s *AuthService) publishUserRegistered(ctx context.Context, user *domain.User) {
	payload := eventbus.UserRegisteredData{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		s.log.Error("failed to marshal user registered payload", "err", err)
		return
	}

	event := eventbus.NewEvent(
		eventbus.SubjectUserRegistered,
		user.ID,
		"user",
		data,
		eventbus.Metadata{UserID: user.ID},
	)

	if err := s.publisher.Publish(ctx, eventbus.SubjectUserRegistered, event); err != nil {
		s.log.Error("failed to publish user.registered", "err", err)
	}
}
