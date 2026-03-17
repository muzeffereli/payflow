
CREATE TABLE IF NOT EXISTS stock_reservations (
    id          TEXT        PRIMARY KEY,
    order_id    TEXT        NOT NULL,
    product_id  TEXT        NOT NULL REFERENCES products(id),
    quantity    INT         NOT NULL CHECK (quantity > 0),
    status      TEXT        NOT NULL DEFAULT 'reserved', -- 'reserved' | 'committed' | 'released'
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    UNIQUE (order_id, product_id) -- one reservation row per product per order
);

CREATE INDEX IF NOT EXISTS idx_reservations_order_id   ON stock_reservations(order_id);
CREATE INDEX IF NOT EXISTS idx_reservations_product_id ON stock_reservations(product_id);
CREATE INDEX IF NOT EXISTS idx_reservations_status     ON stock_reservations(status);
