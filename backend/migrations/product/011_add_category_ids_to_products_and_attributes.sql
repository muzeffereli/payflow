ALTER TABLE products
    ADD COLUMN IF NOT EXISTS category_id TEXT REFERENCES categories(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS subcategory_id TEXT REFERENCES subcategories(id) ON DELETE SET NULL;

UPDATE products AS p
SET category_id = c.id
FROM categories AS c
WHERE p.category_id IS NULL
  AND trim(p.category) <> ''
  AND lower(trim(p.category)) = lower(trim(c.name));

CREATE INDEX IF NOT EXISTS idx_products_category_id ON products(category_id);
CREATE INDEX IF NOT EXISTS idx_products_subcategory_id ON products(subcategory_id);

ALTER TABLE global_attributes
    ADD COLUMN IF NOT EXISTS category_id TEXT REFERENCES categories(id) ON DELETE CASCADE;

UPDATE global_attributes AS ga
SET category_id = c.id
FROM categories AS c
WHERE ga.category_id IS NULL
  AND trim(ga.category) <> ''
  AND lower(trim(ga.category)) = lower(trim(c.name));

DROP INDEX IF EXISTS idx_global_attributes_category_name;

CREATE UNIQUE INDEX IF NOT EXISTS idx_global_attributes_category_id_name
    ON global_attributes (category_id, LOWER(name));

CREATE INDEX IF NOT EXISTS idx_global_attributes_category_id
    ON global_attributes (category_id);
