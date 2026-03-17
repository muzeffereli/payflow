package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	RoleCustomer = "customer"
	RoleAdmin    = "admin"
	RoleSeller   = "seller"
)

type User struct {
	ID           string
	Email        string
	Name         string
	Role         string // "customer" | "admin"
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

var (
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrWeakPassword       = errors.New("password must be at least 8 characters")
)

func NewUser(email, name, password string) (*User, error) {
	if len(password) < 8 {
		return nil, ErrWeakPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &User{
		ID:           uuid.New().String(),
		Email:        strings.ToLower(strings.TrimSpace(email)),
		Name:         strings.TrimSpace(name),
		Role:         RoleCustomer, // new users are always customers
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (u *User) CheckPassword(password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}

func (u *User) UpdatePassword(newPassword string) error {
	if len(newPassword) < 8 {
		return ErrWeakPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	u.UpdatedAt = time.Now().UTC()
	return nil
}
