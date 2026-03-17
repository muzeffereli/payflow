package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"payment-platform/internal/fraud/domain"
	"payment-platform/internal/fraud/port"
	"payment-platform/pkg/eventbus"
)

type Rule interface {
	Name() string
	Evaluate(ctx context.Context, req domain.FraudCheckRequest) domain.RuleResult
}

type EventPublisher interface {
	Publish(ctx context.Context, subject string, event eventbus.Event) error
}

type FraudService struct {
	rules     []Rule
	publisher EventPublisher
	repo      port.FraudCheckRepository // optional â€” nil if no DB
	log       *slog.Logger
}

func New(publisher EventPublisher, log *slog.Logger, rules ...Rule) *FraudService {
	return &FraudService{
		rules:     rules,
		publisher: publisher,
		log:       log,
	}
}

func NewWithRepo(publisher EventPublisher, repo port.FraudCheckRepository, log *slog.Logger, rules ...Rule) *FraudService {
	return &FraudService{
		rules:     rules,
		publisher: publisher,
		repo:      repo,
		log:       log,
	}
}

func (s *FraudService) HandlePaymentInitiated(ctx context.Context, data eventbus.PaymentInitiatedData) error {
	req := domain.FraudCheckRequest{
		PaymentID: data.PaymentID,
		OrderID:   data.OrderID,
		UserID:    data.UserID,
		Amount:    data.Amount,
		Currency:  data.Currency,
	}

	decision := s.Check(ctx, req)

	s.log.Info("fraud check complete",
		"payment_id", req.PaymentID,
		"risk_score", fmt.Sprintf("%.2f", decision.RiskScore),
		"decision", decision.Decision,
		"rules_triggered", decision.Rules,
	)

	if s.repo != nil {
		fc := &domain.FraudCheck{
			ID:        req.PaymentID + "-check",
			PaymentID: req.PaymentID,
			OrderID:   req.OrderID,
			UserID:    req.UserID,
			Amount:    req.Amount,
			Currency:  req.Currency,
			RiskScore: decision.RiskScore,
			Decision:  decision.Decision,
			Rules:     decision.Rules,
		}
		if err := s.repo.Save(ctx, fc); err != nil {
			s.log.Error("failed to persist fraud check", "err", err)
		}
	}

	return s.publishDecision(ctx, req, decision)
}

func (s *FraudService) Check(ctx context.Context, req domain.FraudCheckRequest) domain.FraudDecision {
	var totalScore float64
	var triggered []string

	for _, rule := range s.rules {
		result := rule.Evaluate(ctx, req)

		if result.Triggered {
			triggered = append(triggered, rule.Name())
			s.log.Debug("fraud rule triggered",
				"rule", rule.Name(),
				"score", result.Score,
				"payment_id", req.PaymentID,
			)
		}

		totalScore += result.Score
	}

	if totalScore > 1.0 {
		totalScore = 1.0
	}

	decision := "approved"
	switch {
	case totalScore >= 0.8:
		decision = "rejected"
	case totalScore >= 0.5:
		decision = "review"
	}

	return domain.FraudDecision{
		RiskScore: totalScore,
		Decision:  decision,
		Rules:     triggered,
	}
}

func (s *FraudService) publishDecision(ctx context.Context, req domain.FraudCheckRequest, d domain.FraudDecision) error {
	payload := eventbus.FraudCheckResultData{
		CheckID:   req.PaymentID + "-check",
		PaymentID: req.PaymentID,
		OrderID:   req.OrderID,
		RiskScore: d.RiskScore,
		Decision:  d.Decision,
		Rules:     d.Rules,
	}

	data, _ := json.Marshal(payload)
	meta := eventbus.Metadata{CorrelationID: req.OrderID, UserID: req.UserID}

	var subject string
	var eventType string

	if d.Decision == "rejected" {
		subject = eventbus.SubjectFraudRejected
		eventType = "fraud.rejected"
	} else {
		subject = eventbus.SubjectFraudApproved
		eventType = "fraud.approved"
	}

	event := eventbus.NewEvent(eventType, req.PaymentID, "fraud", data, meta)
	return s.publisher.Publish(ctx, subject, event)
}
