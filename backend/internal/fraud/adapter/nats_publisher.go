package adapter

import (
	"context"

	"payment-platform/pkg/eventbus"
)

type NATSPublisher struct{ pub *eventbus.Publisher }

func NewNATSPublisher(pub *eventbus.Publisher) *NATSPublisher {
	return &NATSPublisher{pub: pub}
}

func (n *NATSPublisher) Publish(ctx context.Context, subject string, event eventbus.Event) error {
	return n.pub.Publish(ctx, subject, event)
}
