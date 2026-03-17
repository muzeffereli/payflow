CREATE TABLE IF NOT EXISTS wallets (
    id         TEXT        PRIMARY KEY,
    user_id    TEXT        NOT NULL UNIQUE, -- one wallet per user
    balance    BIGINT      NOT NULL DEFAULT 0 CHECK (balance >= 0), -- DB-level guard too
    currency   TEXT        NOT NULL DEFAULT 'USD',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS wallet_transactions (
    id             TEXT        PRIMARY KEY,
    wallet_id      TEXT        NOT NULL REFERENCES wallets(id),
    type           TEXT        NOT NULL CHECK (type IN ('credit', 'debit')),
    amount         BIGINT      NOT NULL CHECK (amount > 0),
    source         TEXT        NOT NULL, -- "refund", "deposit", "payment"
    reference_id   TEXT        NOT NULL, -- payment_id, refund_id, etc.
    balance_before BIGINT      NOT NULL,
    balance_after  BIGINT      NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_wallet_transactions_wallet_id ON wallet_transactions(wallet_id, created_at DESC);
