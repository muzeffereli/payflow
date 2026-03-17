
CREATE TABLE IF NOT EXISTS orders (
    id              TEXT        PRIMARY KEY,
    user_id         TEXT        NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'pending',
    total_amount    BIGINT      NOT NULL,           -- stored in cents, never FLOAT for money
    currency        TEXT        NOT NULL DEFAULT 'USD',
    idempotency_key TEXT        NOT NULL UNIQUE,    -- UNIQUE enforces idempotency at DB level too
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS order_items (
    id          TEXT    PRIMARY KEY,
    order_id    TEXT    NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id  TEXT    NOT NULL,
    variant_id  TEXT,
    variant_sku TEXT    NOT NULL DEFAULT '',
    variant_label TEXT  NOT NULL DEFAULT '',
    quantity    INT     NOT NULL CHECK (quantity > 0),
    price       BIGINT  NOT NULL CHECK (price > 0)   -- unit price in cents
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
