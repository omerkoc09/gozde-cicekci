CREATE TABLE slides (
    id         BIGSERIAL PRIMARY KEY,
    title      TEXT NOT NULL,
    subtitle   TEXT NOT NULL DEFAULT '',
    image_key  TEXT NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Ana sayfa her istekte aktif slaytları sıraya dizip çekiyor.
CREATE INDEX idx_slides_active_order ON slides(sort_order, id) WHERE is_active;
