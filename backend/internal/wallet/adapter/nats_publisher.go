package adapter

import (
	"context"

	"payment-platform/internal/wallet/port"
	"payment-platform/pkg/eventbus"
)

var _ port.EventPublisher = (*natsPublisher)(nil)

type natsPublisher struct{ pub *eventbus.Publisher }

func NewNATSPublisher(pub *eventbus.Publisher) port.EventPublisher {
	return &natsPublisher{pub: pub}
}

func (n *natsPublisher) Publish(ctx context.Context, subject string, event eventbus.Event) error {
	return n.pub.Publish(ctx, subject, event)
}
