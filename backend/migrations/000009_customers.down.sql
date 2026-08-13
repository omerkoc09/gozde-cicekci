DROP INDEX idx_orders_customer;
ALTER TABLE orders DROP COLUMN customer_id;
DROP TABLE customers;
