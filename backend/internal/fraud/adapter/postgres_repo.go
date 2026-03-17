package adapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"payment-platform/internal/fraud/domain"
	"payment-platform/internal/fraud/port"
)

var _ port.FraudCheckRepository = (*postgresFraudRepo)(nil)

type postgresFraudRepo struct {
	db *sql.DB
}

func NewPostgresFraudRepo(db *sql.DB) port.FraudCheckRepository {
	return &postgresFraudRepo{db: db}
}

func (r *postgresFraudRepo) Save(ctx context.Context, fc *domain.FraudCheck) error {
	rulesJSON, _ := json.Marshal(fc.Rules)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO fraud_checks (id, payment_id, order_id, user_id, amount, currency, risk_score, decision, rules, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (id) DO NOTHING`,
		fc.ID, fc.PaymentID, fc.OrderID, fc.UserID, fc.Amount, fc.Currency, fc.RiskScore, fc.Decision, rulesJSON,
	)
	if err != nil {
		return fmt.Errorf("save fraud check: %w", err)
	}
	return nil
}

func (r *postgresFraudRepo) GetByID(ctx context.Context, id string) (*domain.FraudCheck, error) {
	fc := &domain.FraudCheck{}
	var rulesJSON []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT id, payment_id, order_id, user_id, amount, currency, risk_score, decision, rules, created_at
		FROM fraud_checks WHERE id = $1`, id,
	).Scan(&fc.ID, &fc.PaymentID, &fc.OrderID, &fc.UserID, &fc.Amount, &fc.Currency, &fc.RiskScore, &fc.Decision, &rulesJSON, &fc.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("fraud check not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get fraud check: %w", err)
	}
	json.Unmarshal(rulesJSON, &fc.Rules)
	return fc, nil
}

func (r *postgresFraudRepo) List(ctx context.Context, decision string, limit, offset int) ([]*domain.FraudCheck, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if decision != "" {
		where = append(where, fmt.Sprintf("decision = $%d", idx))
		args = append(args, decision)
		idx++
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM fraud_checks WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count fraud checks: %w", err)
	}

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, payment_id, order_id, user_id, amount, currency, risk_score, decision, rules, created_at
		 FROM fraud_checks WHERE `+whereClause+
			fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, idx, idx+1),
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list fraud checks: %w", err)
	}
	defer rows.Close()

	var checks []*domain.FraudCheck
	for rows.Next() {
		fc := &domain.FraudCheck{}
		var rulesJSON []byte
		if err := rows.Scan(&fc.ID, &fc.PaymentID, &fc.OrderID, &fc.UserID, &fc.Amount, &fc.Currency, &fc.RiskScore, &fc.Decision, &rulesJSON, &fc.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan fraud check: %w", err)
		}
		json.Unmarshal(rulesJSON, &fc.Rules)
		checks = append(checks, fc)
	}
	return checks, total, rows.Err()
}
