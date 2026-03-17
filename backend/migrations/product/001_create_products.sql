
CREATE TABLE IF NOT EXISTS products (
    id          TEXT        PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    sku         VARCHAR(100) NOT NULL UNIQUE,
    price       BIGINT      NOT NULL CHECK (price > 0),   -- unit price in cents
    currency    VARCHAR(10) NOT NULL DEFAULT 'USD',
    stock       INT         NOT NULL DEFAULT 0 CHECK (stock >= 0),
    category    VARCHAR(100) NOT NULL DEFAULT '',
    status      VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_products_sku      ON products(sku);
CREATE INDEX IF NOT EXISTS idx_products_status   ON products(status);
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);
