package adapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"payment-platform/internal/notification/domain"
	"payment-platform/internal/notification/port"
)

var _ port.NotificationRepository = (*postgresNotifRepo)(nil)

type postgresNotifRepo struct {
	db *sql.DB
}

func NewPostgresNotifRepo(db *sql.DB) port.NotificationRepository {
	return &postgresNotifRepo{db: db}
}

func (r *postgresNotifRepo) Save(ctx context.Context, n *domain.Notification) error {
	metaJSON, _ := json.Marshal(n.Metadata)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, type, title, body, read, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		n.ID, n.UserID, n.Type, n.Title, n.Body, n.Read, metaJSON, n.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save notification: %w", err)
	}
	return nil
}

func (r *postgresNotifRepo) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1`, userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, type, title, body, read, metadata, created_at
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var notifs []*domain.Notification
	for rows.Next() {
		n := &domain.Notification{}
		var metaJSON []byte
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Read, &metaJSON, &n.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan notification: %w", err)
		}
		if err := json.Unmarshal(metaJSON, &n.Metadata); err != nil {
			n.Metadata = map[string]interface{}{}
		}
		notifs = append(notifs, n)
	}
	return notifs, total, rows.Err()
}

func (r *postgresNotifRepo) MarkRead(ctx context.Context, id, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *postgresNotifRepo) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET read = true WHERE user_id = $1 AND read = false`, userID)
	return err
}

func (r *postgresNotifRepo) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = false`, userID,
	).Scan(&count)
	return count, err
}
