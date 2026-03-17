package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type EventHandler func(ctx context.Context, event Event) error

const (
	maxDeliver = 5

	DLQStream = "DLQ"

	DLQSubjectPrefix = "dlq."
)

type Subscriber struct {
	js  jetstream.JetStream
	pub *Publisher // used to forward exhausted messages to the DLQ stream
	log *slog.Logger
}

func NewSubscriber(nc *nats.Conn, log *slog.Logger) (*Subscriber, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("init jetstream: %w", err)
	}

	pub, err := NewPublisher(nc)
	if err != nil {
		return nil, fmt.Errorf("init dlq publisher: %w", err)
	}

	s := &Subscriber{js: js, pub: pub, log: log}

	if err := s.ensureDLQStream(context.Background()); err != nil {
		log.Warn("could not ensure DLQ stream â€” dead-letter forwarding may not work", "err", err)
	}

	return s, nil
}

func (s *Subscriber) ensureDLQStream(ctx context.Context) error {
	_, err := s.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      DLQStream,
		Subjects:  []string{DLQSubjectPrefix + ">"},
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.FileStorage,
		MaxAge:    7 * 24 * time.Hour,
	})
	return err
}

type SubscribeConfig struct {
	Stream string

	Subjects []string

	Consumer string

	FilterSubject string
}

func (s *Subscriber) Subscribe(ctx context.Context, cfg SubscribeConfig, handler EventHandler) error {
	_, err := s.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      cfg.Stream,
		Subjects:  cfg.Subjects,
		Retention: jetstream.LimitsPolicy, // keep messages up to storage limit
		Storage:   jetstream.FileStorage,  // persisted to disk (survives NATS restarts)
	})
	if err != nil {
		return fmt.Errorf("create stream %s: %w", cfg.Stream, err)
	}

	consumerCfg := jetstream.ConsumerConfig{
		Durable:    cfg.Consumer,
		AckPolicy:  jetstream.AckExplicitPolicy,
		AckWait:    30 * time.Second, // if no Ack within 30s, redeliver
		MaxDeliver: maxDeliver,       // after maxDeliver failures, NATS stops delivering
	}
	if cfg.FilterSubject != "" {
		consumerCfg.FilterSubject = cfg.FilterSubject
	}

	cons, err := s.js.CreateOrUpdateConsumer(ctx, cfg.Stream, consumerCfg)
	if err != nil {
		return fmt.Errorf("create consumer %s: %w", cfg.Consumer, err)
	}

	go s.consume(ctx, cons, handler, cfg.Consumer)

	s.log.Info("subscribed to stream",
		"stream", cfg.Stream,
		"consumer", cfg.Consumer,
		"subjects", cfg.Subjects,
	)

	return nil
}

func (s *Subscriber) consume(ctx context.Context, cons jetstream.Consumer, handler EventHandler, consumerName string) {
	iter, err := cons.Messages()
	if err != nil {
		s.log.Error("failed to get message iterator", "consumer", consumerName, "err", err)
		return
	}
	defer iter.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("subscriber stopping", "consumer", consumerName)
			return
		default:
		}

		msg, err := iter.Next()
		if err != nil {
			s.log.Error("message iterator error", "consumer", consumerName, "err", err)
			return
		}

		meta, err := msg.Metadata()
		if err == nil && int(meta.NumDelivered) >= maxDeliver {
			s.forwardToDLQ(ctx, msg, consumerName)
			continue
		}

		var event Event
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			s.log.Error("unmarshal event failed â€” forwarding to DLQ",
				"consumer", consumerName,
				"err", err,
			)
			s.forwardToDLQ(ctx, msg, consumerName)
			continue
		}

		s.log.Debug("received event",
			"type", event.Type,
			"aggregate_id", event.AggregateID,
			"consumer", consumerName,
		)

		if err := handler(ctx, event); err != nil {
			s.log.Error("event handler failed â€” will retry",
				"type", event.Type,
				"aggregate_id", event.AggregateID,
				"err", err,
			)
			msg.NakWithDelay(5 * time.Second)
			continue
		}

		msg.Ack()
	}
}

func (s *Subscriber) forwardToDLQ(ctx context.Context, msg jetstream.Msg, consumerName string) {
	dlqSubject := DLQSubjectPrefix + msg.Subject()

	if err := s.pub.PublishRaw(ctx, dlqSubject, msg.Data()); err != nil {
		s.log.Error("failed to forward message to DLQ â€” message will be lost",
			"consumer", consumerName,
			"subject", msg.Subject(),
			"dlq_subject", dlqSubject,
			"err", err,
		)
	} else {
		s.log.Warn("message forwarded to DLQ after max delivery attempts",
			"consumer", consumerName,
			"subject", msg.Subject(),
			"dlq_subject", dlqSubject,
		)
	}

	msg.Ack()
}
