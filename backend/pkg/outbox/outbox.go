package outbox

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type Publisher interface {
	PublishRaw(ctx context.Context, subject string, data []byte) error
}

type Record struct {
	ID      string
	Subject string
	Payload []byte
}

func Write(ctx context.Context, tx *sql.Tx, subject string, payload []byte) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO outbox (id, subject, payload, created_at)
		VALUES ($1, $2, $3, NOW())`,
		uuid.New().String(), subject, payload,
	)
	return err
}

type Relay struct {
	db        *sql.DB
	publisher Publisher
	poll      time.Duration
	batch     int
	log       *slog.Logger
}

func NewRelay(db *sql.DB, publisher Publisher, log *slog.Logger) *Relay {
	return &Relay{
		db:        db,
		publisher: publisher,
		poll:      time.Second,
		batch:     50,
		log:       log,
	}
}

func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.poll)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.flush(ctx); err != nil {
				r.log.Error("outbox flush error", "err", err)
			}
		}
	}
}

func (r *Relay) flush(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, subject, payload
		FROM   outbox
		WHERE  published_at IS NULL
		ORDER  BY created_at
		LIMIT  $1`,
		r.batch,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ID, &rec.Subject, &rec.Payload); err != nil {
			return err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, rec := range records {
		if err := r.publisher.PublishRaw(ctx, rec.Subject, rec.Payload); err != nil {
			r.log.Error("outbox publish failed", "id", rec.ID, "subject", rec.Subject, "err", err)
			continue // leave row pending â€” will retry next tick
		}

		if _, err := r.db.ExecContext(ctx,
			`UPDATE outbox SET published_at = NOW() WHERE id = $1`, rec.ID,
		); err != nil {
			r.log.Error("outbox mark-published failed", "id", rec.ID, "err", err)
		}
	}

	return nil
}
