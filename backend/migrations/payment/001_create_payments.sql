CREATE TABLE IF NOT EXISTS payments (
    id             TEXT        PRIMARY KEY,
    order_id       TEXT        NOT NULL UNIQUE, -- one payment per order
    user_id        TEXT        NOT NULL,
    amount         BIGINT      NOT NULL,
    currency       TEXT        NOT NULL DEFAULT 'USD',
    status         TEXT        NOT NULL DEFAULT 'pending',
    method         TEXT        NOT NULL DEFAULT 'card',
    transaction_id TEXT,                        -- set on success
    failure_reason TEXT,                        -- set on failure
    created_at     TIMESTAMPTZ NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_payments_order_id  ON payments(order_id);
CREATE INDEX IF NOT EXISTS idx_payments_user_id   ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_status    ON payments(status);
