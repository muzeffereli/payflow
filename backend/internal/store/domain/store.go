package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type StoreStatus string

const (
	StatusPending   StoreStatus = "pending"   // awaiting admin approval
	StatusActive    StoreStatus = "active"    // open for business
	StatusSuspended StoreStatus = "suspended" // temporarily disabled by admin
	StatusClosed    StoreStatus = "closed"    // permanently closed
)

type Store struct {
	ID          string      `json:"id"`
	OwnerID     string      `json:"owner_id"` // user ID of the seller
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Email       string      `json:"email"`      // public contact email
	Commission  int         `json:"commission"` // percentage 0â€“100, e.g. 10 = 10%
	Status      StoreStatus `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

var (
	ErrStoreNotFound        = errors.New("store not found")
	ErrStoreNameRequired    = errors.New("store name is required")
	ErrInvalidCommission    = errors.New("commission must be between 0 and 100")
	ErrStoreSuspended       = errors.New("store is suspended")
	ErrStoreClosed          = errors.New("store is closed")
	ErrNotStoreOwner        = errors.New("not the store owner")
	ErrInvalidTransition    = errors.New("invalid status transition")
	ErrOwnerAlreadyHasStore = errors.New("seller already owns a store")
)

func NewStore(ownerID, name, description, email string, commission int) (*Store, error) {
	if name == "" {
		return nil, ErrStoreNameRequired
	}
	if commission < 0 || commission > 100 {
		return nil, ErrInvalidCommission
	}
	if commission == 0 {
		commission = 10
	}

	now := time.Now().UTC()
	return &Store{
		ID:          uuid.New().String(),
		OwnerID:     ownerID,
		Name:        name,
		Description: description,
		Email:       email,
		Commission:  commission,
		Status:      StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (s *Store) Approve() error {
	if s.Status != StatusPending {
		return ErrInvalidTransition
	}
	s.Status = StatusActive
	s.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Store) Suspend() error {
	if s.Status == StatusClosed {
		return ErrInvalidTransition
	}
	s.Status = StatusSuspended
	s.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Store) Reactivate() error {
	if s.Status != StatusSuspended {
		return ErrInvalidTransition
	}
	s.Status = StatusActive
	s.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Store) Close() error {
	if s.Status == StatusClosed {
		return ErrInvalidTransition
	}
	s.Status = StatusClosed
	s.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Store) UpdateDetails(name, description, email string) error {
	if name == "" {
		return ErrStoreNameRequired
	}
	s.Name = name
	s.Description = description
	s.Email = email
	s.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Store) IsActive() bool {
	return s.Status == StatusActive
}
