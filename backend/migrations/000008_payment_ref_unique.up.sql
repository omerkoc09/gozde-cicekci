-- payment_ref sipariş başına PayTR merchant_oid'i tutar. Aynı referansın iki
-- siparişte kullanılması (ör. race condition ya da manuel hata) callback'in
-- yanlış siparişi paid yapmasına yol açabilir — partial unique index bunu
-- veritabanı seviyesinde engeller. NULL'lar hariç tutuluyor çünkü teoride
-- payment_ref set edilmeden önce geçici olarak NULL olabilecek satırlar var
-- olabilir (savunma amaçlı; SetPaymentRef normalde hemen set ediyor).
CREATE UNIQUE INDEX idx_orders_payment_ref ON orders (payment_ref) WHERE payment_ref IS NOT NULL;
