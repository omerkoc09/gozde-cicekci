DROP TABLE payment_events;

ALTER TABLE orders
    DROP COLUMN payment_ref,
    DROP COLUMN refunded_at,
    DROP COLUMN paid_at;

ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending','confirmed','delivered','cancelled'));
ALTER TABLE orders ALTER COLUMN status SET DEFAULT 'pending';
