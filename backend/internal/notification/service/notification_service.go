package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"payment-platform/internal/notification/domain"
	"payment-platform/internal/notification/port"
	"payment-platform/pkg/eventbus"
)

type Sender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
	SendWebhook(ctx context.Context, url string, payload []byte) error
}

type NotificationService struct {
	sender Sender
	repo   port.NotificationRepository // optional â€” nil if no DB
	log    *slog.Logger
}

func New(sender Sender, log *slog.Logger) *NotificationService {
	return &NotificationService{sender: sender, log: log}
}

func NewWithRepo(sender Sender, repo port.NotificationRepository, log *slog.Logger) *NotificationService {
	return &NotificationService{sender: sender, repo: repo, log: log}
}

func (s *NotificationService) persist(ctx context.Context, userID, notifType, title, body string, metadata map[string]interface{}) {
	if s.repo == nil {
		return
	}
	n := domain.NewNotification(userID, notifType, title, body, metadata)
	if err := s.repo.Save(ctx, n); err != nil {
		s.log.Error("failed to persist notification", "err", err)
	}
}

func (s *NotificationService) HandleEvent(ctx context.Context, event eventbus.Event) error {
	s.log.Debug("received event for notification",
		"type", event.Type,
		"aggregate_id", event.AggregateID,
		"user_id", event.Metadata.UserID,
	)

	switch event.Type {
	case eventbus.SubjectUserRegistered:
		return s.onUserRegistered(ctx, event)
	case "order.confirmed":
		return s.onOrderConfirmed(ctx, event)
	case "payment.succeeded":
		return s.onPaymentSucceeded(ctx, event)
	case "payment.failed":
		return s.onPaymentFailed(ctx, event)
	case "payment.refunded":
		return s.onPaymentRefunded(ctx, event)
	case "wallet.credited":
		return s.onWalletCredited(ctx, event)
	case "fraud.rejected":
		return s.onFraudRejected(ctx, event)
	default:
		s.log.Debug("unhandled event type â€” skipping", "type", event.Type)
		return nil
	}
}

func (s *NotificationService) onUserRegistered(ctx context.Context, event eventbus.Event) error {
	var data eventbus.UserRegisteredData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return err
	}

	title := "Welcome to PayFlow!"
	body := formatf("Hi %s, your account has been created successfully.", data.Name)
	s.persist(ctx, data.UserID, "welcome", title, body, map[string]interface{}{"email": data.Email})
	return s.sender.SendEmail(ctx, data.Email, title, body)
}

func (s *NotificationService) onOrderConfirmed(ctx context.Context, event eventbus.Event) error {
	userID := event.Metadata.UserID
	email := s.emailOrFallback(event)
	title := "Order Confirmed"
	body := "Your order " + event.AggregateID + " has been confirmed and is being processed."
	s.persist(ctx, userID, "order", title, body, map[string]interface{}{"order_id": event.AggregateID})
	return s.sender.SendEmail(ctx, email, title, body)
}

func (s *NotificationService) onPaymentSucceeded(ctx context.Context, event eventbus.Event) error {
	var data eventbus.PaymentSucceededData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return err
	}

	userID := event.Metadata.UserID
	email := s.emailOrFallback(event)
	title := "Payment Received"
	body := formatf("Your payment of %s has been processed successfully.", formatCents(data.Amount))
	s.persist(ctx, userID, "payment", title, body, map[string]interface{}{
		"payment_id": data.PaymentID, "order_id": data.OrderID, "amount": data.Amount,
	})
	return s.sender.SendEmail(ctx, email, title, body)
}

func (s *NotificationService) onPaymentFailed(ctx context.Context, event eventbus.Event) error {
	var data eventbus.PaymentFailedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return err
	}

	userID := event.Metadata.UserID
	email := s.emailOrFallback(event)
	title := "Payment Failed"
	body := formatf("Payment for order %s failed: %s. Please update your payment method.", data.OrderID, data.Reason)
	s.persist(ctx, userID, "payment", title, body, map[string]interface{}{
		"order_id": data.OrderID, "reason": data.Reason,
	})
	return s.sender.SendEmail(ctx, email, title, body)
}

func (s *NotificationService) onPaymentRefunded(ctx context.Context, event eventbus.Event) error {
	var data eventbus.PaymentRefundedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return err
	}

	userID := event.Metadata.UserID
	email := s.emailOrFallback(event)
	title := "Refund Processed"
	body := formatf("Your refund of %s for order %s has been processed.", formatCents(data.Amount), data.OrderID)
	s.persist(ctx, userID, "refund", title, body, map[string]interface{}{
		"order_id": data.OrderID, "amount": data.Amount,
	})
	return s.sender.SendEmail(ctx, email, title, body)
}

func (s *NotificationService) onWalletCredited(ctx context.Context, event eventbus.Event) error {
	var data eventbus.WalletCreditedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return err
	}

	userID := event.Metadata.UserID
	email := s.emailOrFallback(event)
	title := "Wallet Balance Updated"
	body := formatf("%s has been added to your wallet. Source: %s.", formatCents(data.Amount), data.Source)
	s.persist(ctx, userID, "wallet", title, body, map[string]interface{}{
		"source": data.Source, "amount": data.Amount,
	})
	return s.sender.SendEmail(ctx, email, title, body)
}

func (s *NotificationService) onFraudRejected(ctx context.Context, event eventbus.Event) error {
	var data eventbus.FraudCheckResultData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return err
	}

	userID := event.Metadata.UserID
	email := s.emailOrFallback(event)
	title := "Order Flagged for Review"
	body := formatf("Your order %s has been flagged for security review. Our team will contact you.", data.OrderID)
	s.persist(ctx, userID, "fraud", title, body, map[string]interface{}{
		"order_id": data.OrderID, "risk_score": data.RiskScore,
	})
	return s.sender.SendEmail(ctx, email, title, body)
}

func (s *NotificationService) emailOrFallback(event eventbus.Event) string {
	if event.Metadata.Email != "" {
		return event.Metadata.Email
	}
	s.log.Warn("event missing email in metadata, falling back to userID", "user_id", event.Metadata.UserID)
	return event.Metadata.UserID
}

func formatCents(cents int64) string {
	dollars := cents / 100
	pennies := cents % 100
	return formatf("$%d.%02d", dollars, pennies)
}

func formatf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
