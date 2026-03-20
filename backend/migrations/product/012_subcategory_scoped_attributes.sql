-- Replace category-scoped global attributes with subcategory-scoped

ALTER TABLE global_attributes
    ADD COLUMN IF NOT EXISTS subcategory_id TEXT REFERENCES subcategories(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS subcategory VARCHAR(150) NOT NULL DEFAULT '';

-- Drop old category-based unique indexes
DROP INDEX IF EXISTS idx_global_attributes_category_id_name;
DROP INDEX IF EXISTS idx_global_attributes_category_name;

-- New unique constraint: one attribute name per subcategory
CREATE UNIQUE INDEX IF NOT EXISTS idx_global_attributes_subcategory_id_name
    ON global_attributes (subcategory_id, LOWER(name))
    WHERE subcategory_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_global_attributes_subcategory_id
    ON global_attributes (subcategory_id);
