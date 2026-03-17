
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS store_id VARCHAR(36);

CREATE INDEX IF NOT EXISTS idx_products_store_id ON products (store_id);
