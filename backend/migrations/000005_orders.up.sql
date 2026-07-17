CREATE TABLE orders (
    id               BIGSERIAL PRIMARY KEY,
    order_no         TEXT NOT NULL UNIQUE,
    status           TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending','confirmed','delivered','cancelled')),

    buyer_name       TEXT NOT NULL,
    buyer_phone      TEXT NOT NULL,
    buyer_email      TEXT,

    recipient_name   TEXT NOT NULL,
    recipient_phone  TEXT NOT NULL,
    delivery_address TEXT NOT NULL,
    delivery_date    DATE NOT NULL,
    delivery_slot    TEXT NOT NULL,
    card_message     TEXT,

    items_total      NUMERIC(10,2) NOT NULL,
    delivery_fee     NUMERIC(10,2) NOT NULL,
    total            NUMERIC(10,2) NOT NULL,

    note             TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Panel listesi status filtresi + tarih sırasıyla okuyor
CREATE INDEX idx_orders_status_created ON orders (status, created_at DESC);

CREATE TABLE order_items (
    id             BIGSERIAL PRIMARY KEY,
    order_id       BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    -- Ürün silinirse sipariş ölmemeli; ad/fiyat kopya olduğu için okunabilir kalır
    product_id     BIGINT REFERENCES products(id) ON DELETE SET NULL,
    product_name   TEXT NOT NULL,
    price_at_order NUMERIC(10,2) NOT NULL,
    quantity       INT NOT NULL CHECK (quantity > 0)
);

CREATE INDEX idx_order_items_order ON order_items (order_id);
