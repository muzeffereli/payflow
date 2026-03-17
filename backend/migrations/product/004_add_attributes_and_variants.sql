CREATE TABLE IF NOT EXISTS product_attributes (
    id         TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    values     JSONB NOT NULL DEFAULT '[]',
    position   INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_product_attr_name ON product_attributes(product_id, name);

CREATE TABLE IF NOT EXISTS product_variants (
    id               TEXT PRIMARY KEY,
    product_id       TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sku              TEXT NOT NULL UNIQUE,
    price            BIGINT,           -- NULL = use parent product base price
    stock            INT NOT NULL DEFAULT 0 CHECK (stock >= 0),
    attribute_values JSONB NOT NULL DEFAULT '{}',
    status           VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_variants_product ON product_variants(product_id);
