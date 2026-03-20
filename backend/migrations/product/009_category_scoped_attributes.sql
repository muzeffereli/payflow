ALTER TABLE global_attributes
    ADD COLUMN IF NOT EXISTS category VARCHAR(100) NOT NULL DEFAULT 'general';

ALTER TABLE global_attributes
    DROP CONSTRAINT IF EXISTS global_attributes_name_key;

DROP INDEX IF EXISTS idx_global_attributes_category_name;

CREATE UNIQUE INDEX IF NOT EXISTS idx_global_attributes_category_name
    ON global_attributes (LOWER(category), LOWER(name));

CREATE INDEX IF NOT EXISTS idx_global_attributes_category
    ON global_attributes (LOWER(category));
