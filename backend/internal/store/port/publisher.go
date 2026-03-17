package port

import "context"

type EventPublisher interface {
	Publish(ctx context.Context, subject string, data interface{}) error
}
