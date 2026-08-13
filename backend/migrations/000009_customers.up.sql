CREATE TABLE customers (
  id            BIGSERIAL PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  name          TEXT NOT NULL,
  phone         TEXT NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE orders
  ADD COLUMN customer_id BIGINT REFERENCES customers(id) ON DELETE SET NULL;

CREATE INDEX idx_orders_customer ON orders (customer_id, created_at DESC);
