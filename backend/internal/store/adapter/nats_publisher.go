package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"payment-platform/internal/store/port"
	"payment-platform/pkg/eventbus"
)

var _ port.EventPublisher = (*natsPublisher)(nil)

type natsPublisher struct {
	pub *eventbus.Publisher
}

func NewNATSPublisher(pub *eventbus.Publisher) port.EventPublisher {
	return &natsPublisher{pub: pub}
}

func (n *natsPublisher) Publish(ctx context.Context, subject string, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}
	return n.pub.PublishRaw(ctx, subject, b)
}
