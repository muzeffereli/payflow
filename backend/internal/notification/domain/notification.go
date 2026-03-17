package domain

import (
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	TypeEmail   NotificationType = "email"
	TypeWebhook NotificationType = "webhook"
)

type Notification struct {
	ID        string                 `json:"id"         db:"id"`
	UserID    string                 `json:"user_id"    db:"user_id"`
	Type      string                 `json:"type"       db:"type"`
	Title     string                 `json:"title"      db:"title"`
	Body      string                 `json:"body"       db:"body"`
	Read      bool                   `json:"read"       db:"read"`
	Metadata  map[string]interface{} `json:"metadata"   db:"metadata"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}

func NewNotification(userID, notifType, title, body string, metadata map[string]interface{}) *Notification {
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	return &Notification{
		ID:        uuid.New().String(),
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Body:      body,
		Read:      false,
		Metadata:  metadata,
		CreatedAt: time.Now().UTC(),
	}
}
