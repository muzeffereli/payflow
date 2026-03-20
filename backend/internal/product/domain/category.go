package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID        string    `json:"id"         db:"id"`
	Name      string    `json:"name"       db:"name"`
	Slug      string    `json:"slug"       db:"slug"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Subcategory struct {
	ID         string    `json:"id"          db:"id"`
	CategoryID string    `json:"category_id" db:"category_id"`
	Name       string    `json:"name"        db:"name"`
	Slug       string    `json:"slug"        db:"slug"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"  db:"updated_at"`
}

func NewCategory(name string) (*Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("category name is required")
	}

	now := time.Now().UTC()
	return &Category{
		ID:        uuid.New().String(),
		Name:      name,
		Slug:      slugify(name),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func NewSubcategory(categoryID, name string) (*Subcategory, error) {
	categoryID = strings.TrimSpace(categoryID)
	name = strings.TrimSpace(name)
	if categoryID == "" {
		return nil, errors.New("category_id is required")
	}
	if name == "" {
		return nil, errors.New("subcategory name is required")
	}

	now := time.Now().UTC()
	return &Subcategory{
		ID:         uuid.New().String(),
		CategoryID: categoryID,
		Name:       name,
		Slug:       slugify(name),
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func slugify(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}

	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_' || r == '/':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	result := strings.Trim(b.String(), "-")
	if result == "" {
		return value
	}
	return result
}

var (
	ErrCategoryNotFound    = errors.New("category not found")
	ErrCategoryConflict    = errors.New("category already exists")
	ErrSubcategoryNotFound = errors.New("subcategory not found")
	ErrSubcategoryConflict = errors.New("subcategory already exists")
)
