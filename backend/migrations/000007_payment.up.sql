-- Statü seti Faz 3 ile değişiyor: awaiting_payment / paid / delivered / refunded.
-- Canlıda henüz gerçek sipariş yok (Faz 2+3 birlikte yayınlanacaktı), veri kaybı yok.
ALTER TABLE orders ALTER COLUMN status SET DEFAULT 'awaiting_payment';
ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('awaiting_payment','paid','delivered','refunded'));

ALTER TABLE orders
    ADD COLUMN paid_at     TIMESTAMPTZ,
    ADD COLUMN refunded_at TIMESTAMPTZ,
    ADD COLUMN payment_ref TEXT;

CREATE TABLE payment_events (
    id          BIGSERIAL PRIMARY KEY,
    order_id    BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    event_type  TEXT NOT NULL,   -- 'token_requested','callback_ok','callback_fail','refund'
    raw_payload JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_payment_events_order ON payment_events (order_id);
