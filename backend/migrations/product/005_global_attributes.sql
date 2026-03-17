CREATE TABLE IF NOT EXISTS global_attributes (
    id         TEXT PRIMARY KEY,
    name       VARCHAR(100) NOT NULL UNIQUE,
    values     JSONB NOT NULL DEFAULT '[]',
    position   INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE product_attributes
    ADD COLUMN IF NOT EXISTS global_attribute_id TEXT REFERENCES global_attributes(id),
    ADD COLUMN IF NOT EXISTS selected_values JSONB NOT NULL DEFAULT '[]';

CREATE INDEX IF NOT EXISTS idx_product_attr_global ON product_attributes(global_attribute_id);
