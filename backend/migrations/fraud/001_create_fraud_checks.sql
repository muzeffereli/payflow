CREATE TABLE IF NOT EXISTS fraud_checks (
    id          TEXT PRIMARY KEY,
    payment_id  TEXT NOT NULL,
    order_id    TEXT NOT NULL,
    user_id     TEXT NOT NULL,
    amount      BIGINT NOT NULL,
    currency    VARCHAR(10) NOT NULL DEFAULT 'USD',
    risk_score  DOUBLE PRECISION NOT NULL,
    decision    VARCHAR(20) NOT NULL,
    rules       JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_fraud_decision ON fraud_checks(decision, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_fraud_payment ON fraud_checks(payment_id);
