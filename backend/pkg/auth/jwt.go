package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"

	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
)

type Claims struct {
	UserID    string    `json:"user_id"`
	Role      string    `json:"role,omitempty"` // "customer" | "seller" | "admin"
	TokenType TokenType `json:"token_type,omitempty"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secret []byte
	ttl    time.Duration // used only by the backward-compatible Sign method
	issuer string
}

func NewJWTManager(secret, issuer string, ttl time.Duration) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
		ttl:    ttl,
		issuer: issuer,
	}
}

func (m *JWTManager) Sign(userID string) (string, error) {
	return m.sign(userID, "customer", TokenTypeAccess, m.ttl)
}

func (m *JWTManager) SignAccess(userID, role string) (string, error) {
	return m.sign(userID, role, TokenTypeAccess, AccessTokenTTL)
}

func (m *JWTManager) SignRefresh(userID, role string) (string, error) {
	return m.sign(userID, role, TokenTypeRefresh, RefreshTokenTTL)
}

func (m *JWTManager) sign(userID, role string, tokenType TokenType, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *JWTManager) Verify(tokenStr string) (*Claims, error) {
	return m.parseToken(tokenStr)
}

func (m *JWTManager) VerifyAccess(tokenStr string) (*Claims, error) {
	claims, err := m.parseToken(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "" && claims.TokenType != TokenTypeAccess {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (m *JWTManager) parseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return m.secret, nil
		},
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
)
