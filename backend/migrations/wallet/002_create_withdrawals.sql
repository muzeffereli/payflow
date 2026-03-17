CREATE TABLE IF NOT EXISTS withdrawals (
    id          VARCHAR(36)  PRIMARY KEY,
    user_id     VARCHAR(36)  NOT NULL,
    store_id    VARCHAR(36)  NOT NULL,
    amount      BIGINT       NOT NULL CHECK (amount > 0),
    currency    VARCHAR(3)   NOT NULL DEFAULT 'USD',
    method      VARCHAR(50)  NOT NULL DEFAULT 'bank_transfer',
    status      VARCHAR(20)  NOT NULL DEFAULT 'pending'
                             CHECK (status IN ('pending', 'approved', 'rejected')),
    notes       TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals (user_id);
CREATE INDEX IF NOT EXISTS idx_withdrawals_status  ON withdrawals (status) WHERE status = 'pending';
