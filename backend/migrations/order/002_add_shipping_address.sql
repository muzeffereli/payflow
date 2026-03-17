
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS shipping_address JSONB;
