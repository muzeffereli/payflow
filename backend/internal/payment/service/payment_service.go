package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"payment-platform/internal/payment/domain"
	"payment-platform/internal/payment/port"
	"payment-platform/pkg/eventbus"
)

type PaymentService struct {
	repo      port.PaymentRepository
	publisher port.EventPublisher
	stores    port.StoreClient
	wallets   port.WalletClient
	log       *slog.Logger
	workers   chan struct{}
}

func New(repo port.PaymentRepository, pub port.EventPublisher, stores port.StoreClient, wallets port.WalletClient, log *slog.Logger) *PaymentService {
	return &PaymentService{
		repo:      repo,
		publisher: pub,
		stores:    stores,
		wallets:   wallets,
		log:       log,
		workers:   make(chan struct{}, 10),
	}
}

func (s *PaymentService) HandleOrderCreated(ctx context.Context, data eventbus.OrderCreatedData) error {
	existing, err := s.repo.GetByOrderID(ctx, data.OrderID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("idempotency check: %w", err)
	}
	if existing != nil {
		s.log.Info("payment already exists for order, skipping",
			"order_id", data.OrderID,
			"payment_id", existing.ID,
		)
		return nil
	}

	payment := domain.NewPayment(
		data.OrderID,
		data.UserID,
		data.Currency,
		normalizePaymentMethod(data.PaymentMethod),
		data.TotalAmount,
	)

	if data.StoreID != nil {
		ownerID, commission, err := s.stores.GetStoreOwner(ctx, *data.StoreID)
		if err != nil {
			s.log.Warn("could not fetch store owner - settlement will be skipped",
				"store_id", *data.StoreID, "err", err)
		} else if ownerID != "" {
			payment.StoreID = data.StoreID
			payment.StoreOwnerID = ownerID
			payment.Commission = commission
		}
	}

	if err := s.repo.Save(ctx, payment); err != nil {
		return fmt.Errorf("save payment: %w", err)
	}

	s.log.Info("payment created - awaiting fraud verdict",
		"payment_id", payment.ID,
		"order_id", payment.OrderID,
		"amount", payment.Amount,
	)

	return s.publishPaymentInitiated(ctx, payment, data.UserID)
}

func (s *PaymentService) HandleFraudApproved(ctx context.Context, data eventbus.FraudCheckResultData) error {
	payment, err := s.repo.GetByID(ctx, data.PaymentID)
	if err != nil {
		return fmt.Errorf("get payment for fraud approval: %w", err)
	}

	if payment.Status != domain.PaymentPending {
		s.log.Warn("fraud.approved received but payment is not pending - skipping",
			"payment_id", payment.ID,
			"status", payment.Status,
		)
		return nil
	}

	s.log.Info("fraud approved - processing payment",
		"payment_id", payment.ID,
		"risk_score", data.RiskScore,
	)

	select {
	case s.workers <- struct{}{}:
		go func() {
			defer func() { <-s.workers }()
			s.processPayment(context.Background(), payment)
		}()
	default:
		s.log.Warn("worker pool full, processing inline", "payment_id", payment.ID)
		s.processPayment(ctx, payment)
	}

	return nil
}

func (s *PaymentService) HandleFraudRejected(ctx context.Context, data eventbus.FraudCheckResultData) error {
	payment, err := s.repo.GetByID(ctx, data.PaymentID)
	if err != nil {
		return fmt.Errorf("get payment for fraud rejection: %w", err)
	}

	if payment.Status != domain.PaymentPending {
		s.log.Warn("fraud.rejected received but payment is not pending - skipping",
			"payment_id", payment.ID,
			"status", payment.Status,
		)
		return nil
	}

	payment.Fail("fraud_rejected")
	if err := s.repo.UpdateStatus(ctx, payment); err != nil {
		return fmt.Errorf("save fraud-rejected payment: %w", err)
	}

	s.log.Info("payment rejected by fraud service",
		"payment_id", payment.ID,
		"risk_score", data.RiskScore,
		"rules", data.Rules,
	)

	return s.publishResult(ctx, payment)
}

func (s *PaymentService) processPayment(ctx context.Context, payment *domain.Payment) {
	payment.Status = domain.PaymentProcessing
	if err := s.repo.UpdateStatus(ctx, payment); err != nil {
		s.log.Error("failed to update payment to processing", "err", err, "payment_id", payment.ID)
		return
	}

	if payment.Method == "wallet" {
		transactionID, err := s.wallets.Debit(ctx, payment.UserID, payment.Amount, payment.ID)
		if err == nil {
			payment.Succeed(transactionID)
			s.log.Info("wallet payment succeeded",
				"payment_id", payment.ID,
				"transaction_id", transactionID,
			)
		} else {
			payment.Fail(err.Error())
			s.log.Info("wallet payment failed",
				"payment_id", payment.ID,
				"reason", payment.FailureReason,
			)
		}
	} else {
		time.Sleep(time.Duration(100+rand.Intn(400)) * time.Millisecond)

		if rand.Float64() < 0.95 {
			transactionID := "txn_" + payment.ID[:8]
			payment.Succeed(transactionID)
			s.log.Info("payment succeeded",
				"payment_id", payment.ID,
				"transaction_id", transactionID,
			)
		} else {
			payment.Fail("card_declined")
			s.log.Info("payment failed",
				"payment_id", payment.ID,
				"reason", payment.FailureReason,
			)
		}
	}

	if err := s.repo.UpdateStatus(ctx, payment); err != nil {
		s.log.Error("failed to save payment result", "err", err, "payment_id", payment.ID)
		return
	}

	if err := s.publishResult(ctx, payment); err != nil {
		s.log.Error("failed to publish payment result",
			"err", err,
			"payment_id", payment.ID,
			"status", payment.Status,
		)
	}
}

func (s *PaymentService) publishPaymentInitiated(ctx context.Context, p *domain.Payment, userID string) error {
	data, _ := json.Marshal(eventbus.PaymentInitiatedData{
		PaymentID: p.ID,
		OrderID:   p.OrderID,
		UserID:    userID,
		Amount:    p.Amount,
		Currency:  p.Currency,
		Method:    p.Method,
	})
	meta := eventbus.Metadata{CorrelationID: p.OrderID, UserID: userID}
	event := eventbus.NewEvent("payment.initiated", p.ID, "payment", data, meta)
	return s.publisher.Publish(ctx, eventbus.SubjectPaymentInitiated, event)
}

func (s *PaymentService) publishResult(ctx context.Context, p *domain.Payment) error {
	meta := eventbus.Metadata{
		CorrelationID: p.OrderID,
		UserID:        p.UserID,
	}

	switch p.Status {
	case domain.PaymentSucceeded:
		data, _ := json.Marshal(eventbus.PaymentSucceededData{
			PaymentID:     p.ID,
			OrderID:       p.OrderID,
			UserID:        p.UserID,
			TransactionID: p.TransactionID,
			Amount:        p.Amount,
			Currency:      p.Currency,
			Method:        p.Method,
			StoreID:       p.StoreID,
			StoreOwnerID:  p.StoreOwnerID,
			Commission:    p.Commission,
		})
		event := eventbus.NewEvent("payment.succeeded", p.ID, "payment", data, meta)
		return s.publisher.Publish(ctx, eventbus.SubjectPaymentSucceeded, event)

	case domain.PaymentFailed:
		data, _ := json.Marshal(eventbus.PaymentFailedData{
			PaymentID: p.ID,
			OrderID:   p.OrderID,
			Reason:    p.FailureReason,
		})
		event := eventbus.NewEvent("payment.failed", p.ID, "payment", data, meta)
		return s.publisher.Publish(ctx, eventbus.SubjectPaymentFailed, event)

	default:
		return fmt.Errorf("unexpected payment status to publish: %s", p.Status)
	}
}

func (s *PaymentService) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PaymentService) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	return s.repo.GetByOrderID(ctx, orderID)
}

func (s *PaymentService) RefundPayment(ctx context.Context, paymentID, userID string) (*domain.Payment, error) {
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	if payment.UserID != userID {
		return nil, ErrUnauthorized
	}

	if err := payment.Refund(); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateStatus(ctx, payment); err != nil {
		return nil, fmt.Errorf("save refunded payment: %w", err)
	}

	s.log.Info("payment refunded", "payment_id", payment.ID, "order_id", payment.OrderID)

	if err := s.publishRefunded(ctx, payment); err != nil {
		s.log.Error("failed to publish payments.refunded - refund saved, event will be lost",
			"payment_id", payment.ID, "err", err)
	}

	return payment, nil
}

func (s *PaymentService) publishRefunded(ctx context.Context, p *domain.Payment) error {
	refundID := uuid.New().String()
	data, _ := json.Marshal(eventbus.PaymentRefundedData{
		PaymentID: p.ID,
		RefundID:  refundID,
		OrderID:   p.OrderID,
		UserID:    p.UserID,
		Amount:    p.Amount,
		Reason:    "requested",
	})
	meta := eventbus.Metadata{CorrelationID: p.OrderID, UserID: p.UserID}
	event := eventbus.NewEvent("payment.refunded", p.ID, "payment", data, meta)
	return s.publisher.Publish(ctx, eventbus.SubjectPaymentRefunded, event)
}

var (
	ErrNotFound     = errors.New("payment not found")
	ErrUnauthorized = errors.New("user does not own this payment")
)

func normalizePaymentMethod(method string) string {
	if method == "wallet" {
		return "wallet"
	}
	return "card"
}
