-- Seçenek grubu: "Ambalaj Rengi", "Kurdele Rengi", "Çiçek Rengi"...
-- Panelden eklenir; kodda gömülü liste YOK.
CREATE TABLE option_groups (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL UNIQUE,
    -- kind oluşturulduktan sonra DEĞİŞMEZ: color'dan text'e geçiş mevcut
    -- hex değerlerini anlamsız kılar (categories.axis ile aynı kural).
    kind       TEXT NOT NULL CHECK (kind IN ('color','text')),
    sort_order INT NOT NULL DEFAULT 0,
    is_active  BOOLEAN NOT NULL DEFAULT true
);

-- Gruba bağlı değerler: "Pembe" #F5A9C8
CREATE TABLE option_values (
    id         BIGSERIAL PRIMARY KEY,
    group_id   BIGINT NOT NULL REFERENCES option_groups(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    -- kind='text' olan grupta boş kalır.
    swatch_hex TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    is_active  BOOLEAN NOT NULL DEFAULT true
);

CREATE INDEX idx_option_values_group ON option_values(group_id, sort_order);

-- Hangi üründe hangi grup soruluyor.
CREATE TABLE product_option_groups (
    product_id  BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    group_id    BIGINT NOT NULL REFERENCES option_groups(id) ON DELETE CASCADE,
    is_required BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (product_id, group_id)
);

-- Sipariş anındaki seçim. Gruba/değere REFERANS YOK — isim ve hex
-- kopyalanır. Esnaf sonradan "Pembe"yi silerse veya rengini değiştirirse
-- eski siparişin ne olduğu bilgisi bozulmamalı
-- (order_items.product_name / price_at_order deseninin aynısı).
CREATE TABLE order_item_options (
    id            BIGSERIAL PRIMARY KEY,
    order_item_id BIGINT NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    group_name    TEXT NOT NULL,
    value_name    TEXT NOT NULL,
    swatch_hex    TEXT NOT NULL DEFAULT '',
    sort_order    INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_order_item_options_item
    ON order_item_options(order_item_id, sort_order);

-- Başlangıç verisi. Panelden düzenlenebilir/silinebilir — kodda gömülü
-- değil, esnaf "Çiçek Rengi" grubunu sonradan kendisi ekleyebilir.
INSERT INTO option_groups (name, slug, kind, sort_order) VALUES
    ('Ambalaj Rengi',  'ambalaj-rengi',  'color', 0),
    ('Kurdele Rengi',  'kurdele-rengi',  'color', 1),
    ('Kutu Rengi',     'kutu-rengi',     'color', 2);

INSERT INTO option_values (group_id, name, swatch_hex, sort_order)
SELECT g.id, v.name, v.hex, v.ord
FROM option_groups g
CROSS JOIN (VALUES
    ('Pembe',    '#F0A6CA', 0),
    ('Beyaz',    '#FFFFFF', 1),
    ('Gri',      '#C4C4C4', 2),
    ('Kırmızı',  '#D93A34', 3),
    ('Fuşya',    '#E0219A', 4),
    ('Mor',      '#7B2FF7', 5),
    ('Lacivert', '#41618A', 6),
    ('Siyah',    '#000000', 7),
    ('Hardal',   '#D9A441', 8),
    ('Yeşil',    '#8CD147', 9)
) AS v(name, hex, ord)
WHERE g.slug IN ('ambalaj-rengi', 'kurdele-rengi', 'kutu-rengi');
