ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS store_id VARCHAR(36);

CREATE INDEX IF NOT EXISTS idx_orders_store_id ON orders (store_id)
    WHERE store_id IS NOT NULL;
