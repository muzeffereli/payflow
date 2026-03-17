
CREATE TABLE IF NOT EXISTS product_images (
    id          TEXT        NOT NULL PRIMARY KEY,
    product_id  TEXT        NOT NULL,
    url         TEXT        NOT NULL,
    position    INT         NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_product_images_product_id ON product_images(product_id);

INSERT INTO product_images (id, product_id, url, position, created_at)
SELECT gen_random_uuid()::text, id, image_url, 0, NOW()
FROM products
WHERE image_url IS NOT NULL AND image_url != '';
