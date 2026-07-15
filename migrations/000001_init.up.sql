CREATE TABLE products (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price       NUMERIC(10,2) NOT NULL CHECK (price >= 0),
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE product_slugs (
    slug       TEXT PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    is_current BOOLEAN NOT NULL DEFAULT true
);

CREATE INDEX idx_product_slugs_product_id ON product_slugs(product_id);
CREATE UNIQUE INDEX idx_product_slugs_one_current
    ON product_slugs(product_id) WHERE is_current;

CREATE TABLE product_images (
    id         BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    image_key  TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_product_images_product_id ON product_images(product_id, sort_order);

CREATE TABLE categories (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    axis        TEXT NOT NULL CHECK (axis IN ('occasion','type')),
    is_active   BOOLEAN NOT NULL DEFAULT true,
    is_featured BOOLEAN NOT NULL DEFAULT false,
    sort_order  INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_categories_axis ON categories(axis);

CREATE TABLE product_categories (
    product_id  BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, category_id)
);

CREATE INDEX idx_product_categories_category_id ON product_categories(category_id);

CREATE TABLE admin_users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL
);
