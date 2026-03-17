package service

import (
	"context"
	"fmt"
	"time"

	"payment-platform/internal/fraud/domain"

	"github.com/redis/go-redis/v9"
)

type HighAmountRule struct {
	ThresholdCents int64 // e.g. 50000 = $500.00
}

func (r *HighAmountRule) Name() string { return "high_amount" }

func (r *HighAmountRule) Evaluate(_ context.Context, req domain.FraudCheckRequest) domain.RuleResult {
	if req.Amount > r.ThresholdCents {
		return domain.RuleResult{Score: 0.4, Triggered: true}
	}
	return domain.RuleResult{}
}

type VelocityRule struct {
	rdb             *redis.Client
	WindowSeconds   int64 // time window size (e.g. 60)
	MaxTransactions int64 // max allowed in window (e.g. 3)
}

func NewVelocityRule(rdb *redis.Client, windowSeconds, maxTransactions int64) *VelocityRule {
	return &VelocityRule{
		rdb:             rdb,
		WindowSeconds:   windowSeconds,
		MaxTransactions: maxTransactions,
	}
}

func (r *VelocityRule) Name() string { return "velocity" }

func (r *VelocityRule) Evaluate(ctx context.Context, req domain.FraudCheckRequest) domain.RuleResult {
	bucket := time.Now().Unix() / r.WindowSeconds
	key := fmt.Sprintf("fraud:velocity:%s:%d", req.UserID, bucket)

	count, err := r.rdb.Incr(ctx, key).Result()
	if err != nil {
		return domain.RuleResult{}
	}

	if count == 1 {
		r.rdb.Expire(ctx, key, time.Duration(r.WindowSeconds)*time.Second)
	}

	if count > r.MaxTransactions {
		return domain.RuleResult{Score: 0.5, Triggered: true}
	}

	return domain.RuleResult{}
}

type SuspiciousCountryRule struct {
	blocklist map[string]bool
}

func NewSuspiciousCountryRule(blockedCountries []string) *SuspiciousCountryRule {
	m := make(map[string]bool, len(blockedCountries))
	for _, c := range blockedCountries {
		m[c] = true
	}
	return &SuspiciousCountryRule{blocklist: m}
}

func (r *SuspiciousCountryRule) Name() string { return "suspicious_country" }

func (r *SuspiciousCountryRule) Evaluate(_ context.Context, req domain.FraudCheckRequest) domain.RuleResult {
	if req.Country != "" && r.blocklist[req.Country] {
		return domain.RuleResult{Score: 0.3, Triggered: true}
	}
	return domain.RuleResult{}
}

type RoundAmountRule struct{}

func (r *RoundAmountRule) Name() string { return "round_amount" }

func (r *RoundAmountRule) Evaluate(_ context.Context, req domain.FraudCheckRequest) domain.RuleResult {
	if req.Amount > 0 && req.Amount%10000 == 0 {
		return domain.RuleResult{Score: 0.1, Triggered: true}
	}
	return domain.RuleResult{}
}
