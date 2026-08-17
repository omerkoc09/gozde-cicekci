DROP TABLE IF EXISTS stock_movements;

DROP INDEX IF EXISTS idx_orders_sweep;
ALTER TABLE orders DROP COLUMN IF EXISTS stock_swept;

ALTER TABLE order_items DROP COLUMN IF EXISTS was_discounted;

DROP INDEX IF EXISTS idx_products_discount_active;

ALTER TABLE products
    DROP CONSTRAINT IF EXISTS products_stock_nonneg,
    DROP CONSTRAINT IF EXISTS products_discount_sold_nonneg,
    DROP CONSTRAINT IF EXISTS products_discount_pair;

ALTER TABLE products
    DROP COLUMN IF EXISTS track_stock,
    DROP COLUMN IF EXISTS stock_quantity,
    DROP COLUMN IF EXISTS stock_reserved,
    DROP COLUMN IF EXISTS discount_price,
    DROP COLUMN IF EXISTS discount_quota,
    DROP COLUMN IF EXISTS discount_sold;
