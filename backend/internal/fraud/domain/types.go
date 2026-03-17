package domain

type FraudCheckRequest struct {
	PaymentID string
	OrderID   string
	UserID    string
	Amount    int64 // cents
	Currency  string
	IP        string
	Country   string
}

type RuleResult struct {
	Score     float64 // contribution to total risk (0.0 = no risk, 1.0 = maximum risk)
	Triggered bool    // true if this rule fired
}

type FraudDecision struct {
	RiskScore float64  `json:"risk_score"`
	Decision  string   `json:"decision"` // "approved", "review", "rejected"
	Rules     []string `json:"rules_triggered"`
}

type FraudCheck struct {
	ID        string   `json:"id"         db:"id"`
	PaymentID string   `json:"payment_id" db:"payment_id"`
	OrderID   string   `json:"order_id"   db:"order_id"`
	UserID    string   `json:"user_id"    db:"user_id"`
	Amount    int64    `json:"amount"     db:"amount"`
	Currency  string   `json:"currency"   db:"currency"`
	RiskScore float64  `json:"risk_score" db:"risk_score"`
	Decision  string   `json:"decision"   db:"decision"`
	Rules     []string `json:"rules"      db:"rules"`
	CreatedAt string   `json:"created_at" db:"created_at"`
}
