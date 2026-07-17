ALTER TABLE orders ADD COLUMN delivery_district TEXT NOT NULL DEFAULT '';
ALTER TABLE orders ALTER COLUMN delivery_district DROP DEFAULT;
