package port

import (
	"context"

	"payment-platform/internal/notification/domain"
)

type NotificationRepository interface {
	Save(ctx context.Context, n *domain.Notification) error
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, int, error)
	MarkRead(ctx context.Context, id, userID string) error
	MarkAllRead(ctx context.Context, userID string) error
	UnreadCount(ctx context.Context, userID string) (int, error)
}
