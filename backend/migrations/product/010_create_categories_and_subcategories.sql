CREATE TABLE IF NOT EXISTS categories (
    id         TEXT PRIMARY KEY,
    name       VARCHAR(150) NOT NULL,
    slug       VARCHAR(180) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT categories_name_key UNIQUE (name),
    CONSTRAINT categories_slug_key UNIQUE (slug)
);

CREATE TABLE IF NOT EXISTS subcategories (
    id          TEXT PRIMARY KEY,
    category_id TEXT        NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name        VARCHAR(150) NOT NULL,
    slug        VARCHAR(180) NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT subcategories_category_name_key UNIQUE (category_id, name),
    CONSTRAINT subcategories_category_slug_key UNIQUE (category_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_subcategories_category_id ON subcategories(category_id);

INSERT INTO categories (id, name, slug, created_at, updated_at)
SELECT DISTINCT
    md5('category:' || lower(trim(source.name))) AS id,
    trim(source.name)                            AS name,
    regexp_replace(lower(trim(source.name)), '[^a-z0-9]+', '-', 'g') AS slug,
    NOW(),
    NOW()
FROM (
    SELECT category AS name FROM products WHERE trim(category) <> ''
    UNION
    SELECT category AS name FROM global_attributes WHERE trim(category) <> ''
) AS source
WHERE trim(source.name) <> ''
ON CONFLICT (name) DO NOTHING;
