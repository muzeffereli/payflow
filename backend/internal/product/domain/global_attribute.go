package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type GlobalAttribute struct {
	ID            string    `json:"id"`
	SubcategoryID string    `json:"subcategory_id"`
	Subcategory   string    `json:"subcategory"`
	CategoryID    string    `json:"category_id"` // parent category, populated via JOIN
	Category      string    `json:"category"`    // parent category name, populated via JOIN
	Name          string    `json:"name"`
	Values        []string  `json:"values"`
	Position      int       `json:"position"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func NewGlobalAttribute(subcategoryID, subcategoryName, name string, values []string) (*GlobalAttribute, error) {
	if subcategoryID == "" {
		return nil, errors.New("subcategory_id is required")
	}
	if name == "" {
		return nil, errors.New("attribute name is required")
	}
	if len(values) == 0 {
		return nil, errors.New("attribute must have at least one value")
	}
	now := time.Now().UTC()
	return &GlobalAttribute{
		ID:            uuid.New().String(),
		SubcategoryID: subcategoryID,
		Subcategory:   subcategoryName,
		Name:          name,
		Values:        values,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

var (
	ErrGlobalAttributeNotFound = errors.New("global attribute not found")
	ErrGlobalAttributeConflict = errors.New("global attribute name already exists for this subcategory")
)
