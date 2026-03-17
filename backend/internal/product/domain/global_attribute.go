package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type GlobalAttribute struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Values    []string  `json:"values"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewGlobalAttribute(name string, values []string) (*GlobalAttribute, error) {
	if name == "" {
		return nil, errors.New("attribute name is required")
	}
	if len(values) == 0 {
		return nil, errors.New("attribute must have at least one value")
	}
	now := time.Now().UTC()
	return &GlobalAttribute{
		ID:        uuid.New().String(),
		Name:      name,
		Values:    values,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

var (
	ErrGlobalAttributeNotFound = errors.New("global attribute not found")
	ErrGlobalAttributeConflict = errors.New("global attribute name already exists")
)
