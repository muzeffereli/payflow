package domain

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string // SHA-256 hex of the raw JWT string
	ExpiresAt time.Time
	RevokedAt *time.Time // nil = active; non-nil = revoked
	CreatedAt time.Time
}

func NewRefreshToken(userID, rawToken string, ttl time.Duration) *RefreshToken {
	return &RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		TokenHash: HashToken(rawToken),
		ExpiresAt: time.Now().UTC().Add(ttl),
		CreatedAt: time.Now().UTC(),
	}
}

func HashToken(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return fmt.Sprintf("%x", h)
}

func (rt *RefreshToken) IsValid() bool {
	return rt.RevokedAt == nil && time.Now().UTC().Before(rt.ExpiresAt)
}
