package port

import (
	"context"

	"payment-platform/pkg/eventbus"
)

type EventPublisher interface {
	Publish(ctx context.Context, subject string, event eventbus.Event) error
}
