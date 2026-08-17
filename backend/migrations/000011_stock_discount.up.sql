-- Stok takibi ürün başına İSTEĞE BAĞLI. Varsayılan false: mevcut ürünler
-- aynen çalışmaya devam eder, hiçbiri "tükendi" görünmez.
ALTER TABLE products
    ADD COLUMN track_stock     BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN stock_quantity  INT     NOT NULL DEFAULT 0,
    -- Ödeme bekleyen adet. Satılabilir = stock_quantity - stock_reserved.
    ADD COLUMN stock_reserved  INT     NOT NULL DEFAULT 0,
    ADD COLUMN discount_price  NUMERIC(10,2),
    ADD COLUMN discount_quota  INT,
    ADD COLUMN discount_sold   INT     NOT NULL DEFAULT 0;

ALTER TABLE products
    ADD CONSTRAINT products_stock_nonneg
        CHECK (stock_quantity >= 0 AND stock_reserved >= 0),
    ADD CONSTRAINT products_discount_sold_nonneg
        CHECK (discount_sold >= 0),
    -- Kotasız indirim süresiz indirimdir; bu özelliğin amacı değil.
    ADD CONSTRAINT products_discount_pair
        CHECK ((discount_price IS NULL AND discount_quota IS NULL)
            OR (discount_price IS NOT NULL AND discount_quota > 0));

-- /indirimli sayfası bu koşulla okuyor
CREATE INDEX idx_products_discount_active ON products (id)
    WHERE discount_price IS NOT NULL;

-- Sipariş anında indirimli satıldı mı. Sonradan ürünün fiyatı değişebileceği
-- için price_at_order karşılaştırmasıyla anlaşılamaz; o anki durum kopyalanır
-- (product_name / price_at_order deseninin aynısı).
ALTER TABLE order_items
    ADD COLUMN was_discounted BOOLEAN NOT NULL DEFAULT false;

-- Süpürülmüş sipariş işareti: aynı sipariş iki kez süpürülürse rezerve
-- negatife düşer (spec §4.3).
ALTER TABLE orders
    ADD COLUMN stock_swept BOOLEAN NOT NULL DEFAULT false;

-- Süpürücü bu koşulla tarıyor
CREATE INDEX idx_orders_sweep ON orders (created_at)
    WHERE status = 'awaiting_payment' AND NOT stock_swept;

-- Her stok değişiminin izi. İki soruyu cevaplar: "bu ay WhatsApp'tan kaç
-- sattım" ve "stok neden bu sayıda".
CREATE TABLE stock_movements (
    id             BIGSERIAL PRIMARY KEY,
    product_id     BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    delta          INT NOT NULL,          -- negatif: düşüş, pozitif: giriş
    reason         TEXT NOT NULL CHECK (reason IN
                     ('siparis','whatsapp_satisi','sayim_duzeltme',
                      'yeni_parti','iptal_iade','rezervasyon_iptal')),
    -- Sipariş silinirse hareket kaydı ölmemeli (order_items.product_id deseni)
    order_id       BIGINT REFERENCES orders(id) ON DELETE SET NULL,
    -- İndirim kotasının WhatsApp satışlarını da sayabilmesi için
    was_discounted BOOLEAN NOT NULL DEFAULT false,
    note           TEXT NOT NULL DEFAULT '',
    -- Hareketi yapan panel kullanıcısı; kullanıcı silinse de kayıt kalır
    admin_user_id  BIGINT REFERENCES admin_users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_stock_movements_product
    ON stock_movements (product_id, created_at DESC);
