-- IF EXISTS: kısmen uygulanmış bir up'tan sonra down'ın yarıda patlayıp
-- şemayı arada bırakmasını engeller (000005_orders.down.sql ile aynı stil).
-- idx_orders_customer ayrıca DROP edilmiyor: kolon düşünce bağımlı indeks
-- de otomatik düşer.
ALTER TABLE orders DROP COLUMN IF EXISTS customer_id;
DROP TABLE IF EXISTS customers;
