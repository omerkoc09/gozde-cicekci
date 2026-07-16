-- Ana sayfa "En Çok Tercih Edilenler" bölümü. Varsayılan false: yeni ürün
-- kendiliğinden ana sayfaya düşmesin, esnaf bilinçli olarak seçsin.
ALTER TABLE products ADD COLUMN is_featured BOOLEAN NOT NULL DEFAULT false;

-- Ana sayfa her istekte öne çıkan aktif ürünleri çekiyor.
CREATE INDEX idx_products_featured ON products(id) WHERE is_active AND is_featured;
