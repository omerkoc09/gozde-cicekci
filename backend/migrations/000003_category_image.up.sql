-- Kategori kartı görseli. NULL olabilir: mevcut kategorilerin görseli yok ve
-- görsel zorunlu değil — yüklenmemişse frontend yedek görsele düşüyor.
ALTER TABLE categories ADD COLUMN image_key TEXT;
