ALTER TABLE stock_reservations
    ADD COLUMN IF NOT EXISTS variant_id TEXT;

CREATE INDEX IF NOT EXISTS idx_reservations_variant_id ON stock_reservations(variant_id);
