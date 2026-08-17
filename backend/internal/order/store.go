package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// orderNoMaxRetry aynı anda gelen siparişler aynı sırayı görebilir; UNIQUE
// constraint çakışmayı yakalar, bir sonraki sırayla tekrar denenir.
// Plan 1'de uniqueSlug'da aynı yarış vardı — orada hata kaba düşüyordu.
const orderNoMaxRetry = 5

const orderSelect = `
	SELECT id, order_no, status,
	       buyer_name, buyer_phone, COALESCE(buyer_email, ''),
	       recipient_name, recipient_phone, delivery_address, delivery_district,
	       delivery_date, delivery_slot, COALESCE(card_message, ''),
	       items_total, delivery_fee, total,
	       paid_at, refunded_at, COALESCE(payment_ref, ''), customer_id,
	       COALESCE(note, ''), created_at, updated_at
	FROM orders`

func scanOrder(row pgx.Row) (*Order, error) {
	var o Order

	err := row.Scan(&o.ID, &o.OrderNo, &o.Status,
		&o.BuyerName, &o.BuyerPhone, &o.BuyerEmail,
		&o.RecipientName, &o.RecipientPhone, &o.DeliveryAddress, &o.DeliveryDistrict,
		&o.DeliveryDate, &o.DeliverySlot, &o.CardMessage,
		&o.ItemsTotal, &o.DeliveryFee, &o.Total,
		&o.PaidAt, &o.RefundedAt, &o.PaymentRef, &o.CustomerID,
		&o.Note, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &o, nil
}

// Create siparişi ve kalemlerini TEK transaction'da yazar.
// Yarısı yazılıp kalanı patlarsa tutarsız sipariş kalır — Plan 1'de slug
// atomikliğinde aynı ders alınmıştı.
func (s *Store) Create(ctx context.Context, in NewOrder) (*Order, error) {
	var lastErr error

	for attempt := 0; attempt < orderNoMaxRetry; attempt++ {
		o, err := s.createOnce(ctx, in)
		if err == nil {
			return o, nil
		}
		// order_no çakışması → bir sonraki sırayla tekrar dene
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" &&
			strings.Contains(pgErr.ConstraintName, "order_no") {
			lastErr = err
			continue
		}

		// customer_id FK ihlali (23503): token geçerli görünse de müşteri
		// artık DB'de yok (silinmiş) — JWT 7 gün canlı kalabildiği için
		// mümkün. Sipariş verilmesini engellemek yerine misafir siparişine
		// düş: checkout'un kırılmaması guest akışında en önemli kural.
		if errors.As(err, &pgErr) && pgErr.Code == "23503" &&
			strings.Contains(pgErr.ConstraintName, "customer_id") && in.CustomerID != nil {
			guestIn := in
			guestIn.CustomerID = nil
			return s.createOnce(ctx, guestIn)
		}
		return nil, err
	}

	return nil, fmt.Errorf("sipariş numarası üretilemedi: %w", lastErr)
}

func (s *Store) createOnce(ctx context.Context, in NewOrder) (*Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Stok rezervasyonu sipariş ile AYNI transaction'da: sipariş yazılamazsa
	// rezervasyon da geri alınır, rezervasyon başarısızsa sipariş hiç
	// oluşmaz (spec §4.1). Nil ise stok yönetimi devrede değil.
	if in.Reserve != nil {
		if err := in.Reserve(ctx, tx); err != nil {
			return nil, err
		}
	}

	// Bugünün kaçıncı siparişi
	now := time.Now()
	var todayCount int
	err = tx.QueryRow(ctx,
		`SELECT count(*) FROM orders WHERE created_at::date = CURRENT_DATE`).Scan(&todayCount)
	if err != nil {
		return nil, err
	}
	orderNo := FormatOrderNo(now, todayCount+1)

	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (
			order_no, buyer_name, buyer_phone, buyer_email,
			recipient_name, recipient_phone, delivery_address, delivery_district,
			delivery_date, delivery_slot, card_message,
			items_total, delivery_fee, total, customer_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id`,
		orderNo, in.BuyerName, in.BuyerPhone, nullIfEmpty(in.BuyerEmail),
		in.RecipientName, in.RecipientPhone, in.DeliveryAddress, in.DeliveryDistrict,
		in.DeliveryDate, in.DeliverySlot, nullIfEmpty(in.CardMessage),
		in.ItemsTotal, in.DeliveryFee, in.Total, in.CustomerID,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	for _, it := range in.Items {
		var itemID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO order_items (order_id, product_id, product_name,
			                         price_at_order, quantity, was_discounted)
			VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
			id, it.ProductID, it.ProductName, it.PriceAtOrder, it.Quantity,
			it.WasDiscounted).Scan(&itemID)
		if err != nil {
			return nil, err
		}

		for _, o := range it.Options {
			if _, err := tx.Exec(ctx, `
				INSERT INTO order_item_options
					(order_item_id, group_name, value_name, swatch_hex, sort_order)
				VALUES ($1,$2,$3,$4,$5)`,
				itemID, o.GroupName, o.ValueName, o.SwatchHex, o.SortOrder); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.GetByID(ctx, id)
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Order, error) {
	o, err := scanOrder(s.pool.QueryRow(ctx, orderSelect+` WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsx.ErrNotFound
		}
		return nil, err
	}

	items, err := s.itemsOf(ctx, id)
	if err != nil {
		return nil, err
	}
	o.Items = items

	return o, nil
}

func (s *Store) itemsOf(ctx context.Context, orderID int64) ([]OrderItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, product_id, product_name, price_at_order, quantity
		FROM order_items WHERE order_id = $1 ORDER BY id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []OrderItem{}
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.ID, &it.ProductID, &it.ProductName, &it.PriceAtOrder, &it.Quantity); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := s.attachOptions(ctx, items); err != nil {
		return nil, err
	}

	return items, nil
}

// optionsOfMany kalem seçimlerini tek sorguda çeker. Kalem başına ayrı
// sorgu N+1 olurdu — Store.List'te aynı ders alınmıştı.
func (s *Store) optionsOfMany(ctx context.Context, itemIDs []int64) (map[int64][]OrderItemOption, error) {
	if len(itemIDs) == 0 {
		return map[int64][]OrderItemOption{}, nil
	}

	rows, err := s.pool.Query(ctx, `
		SELECT order_item_id, group_name, value_name, swatch_hex, sort_order
		FROM order_item_options
		WHERE order_item_id = ANY($1)
		ORDER BY order_item_id, sort_order, id`, itemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64][]OrderItemOption, len(itemIDs))
	for rows.Next() {
		var itemID int64
		var o OrderItemOption
		if err := rows.Scan(&itemID, &o.GroupName, &o.ValueName, &o.SwatchHex, &o.SortOrder); err != nil {
			return nil, err
		}
		out[itemID] = append(out[itemID], o)
	}
	return out, rows.Err()
}

// attachOptions kalemlere seçimlerini bağlar. Seçimi olmayan kalem boş
// slice alır (nil değil) — JSON'da null yerine [] çıksın.
func (s *Store) attachOptions(ctx context.Context, items []OrderItem) error {
	ids := make([]int64, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}

	byItem, err := s.optionsOfMany(ctx, ids)
	if err != nil {
		return err
	}

	for i := range items {
		if o, ok := byItem[items[i].ID]; ok {
			items[i].Options = o
		} else {
			items[i].Options = []OrderItemOption{}
		}
	}
	return nil
}

// List siparişleri en yeniden eskiye listeler. status boşsa hepsi.
func (s *Store) List(ctx context.Context, status string, limit, offset int) ([]Order, error) {
	if status != "" {
		return s.listWhere(ctx, `status = $1`, []any{status}, limit, offset)
	}
	return s.listWhere(ctx, "", nil, limit, offset)
}

// ListVisible awaiting_payment HARİÇ tüm siparişleri listeler (esnaf görünümü).
// Ödeme başlatılıp tamamlanmamış siparişler esnafın önüne çöp olarak düşmesin.
func (s *Store) ListVisible(ctx context.Context, limit, offset int) ([]Order, error) {
	return s.listWhere(ctx, `status <> 'awaiting_payment'`, nil, limit, offset)
}

// listWhere List ve ListVisible'ın ortak gövdesi — sorgu + kalem doldurma
// (itemsOfMany) tek yerde. whereClause boşsa WHERE eklenmez.
func (s *Store) listWhere(ctx context.Context, whereClause string, args []any, limit, offset int) ([]Order, error) {
	q := orderSelect
	if whereClause != "" {
		q += ` WHERE ` + whereClause
	}
	q += fmt.Sprintf(` ORDER BY created_at DESC LIMIT %d OFFSET %d`, limit, offset)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Liste ekranı kalemleri de gösteriyor (ürün adları) — tek sorguda topluca
	if len(orders) > 0 {
		ids := make([]int64, len(orders))
		for i, o := range orders {
			ids[i] = o.ID
		}

		itemsByOrder, err := s.itemsOfMany(ctx, ids)
		if err != nil {
			return nil, err
		}
		for i := range orders {
			orders[i].Items = itemsByOrder[orders[i].ID]
		}
	}

	return orders, nil
}

func (s *Store) itemsOfMany(ctx context.Context, orderIDs []int64) (map[int64][]OrderItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, order_id, product_id, product_name, price_at_order, quantity
		FROM order_items WHERE order_id = ANY($1) ORDER BY id`, orderIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	itemsByOrder := make(map[int64][]OrderItem, len(orderIDs))
	all := make([]OrderItem, 0)
	itemOrder := make([]int64, 0)
	for rows.Next() {
		var it OrderItem
		var orderID int64
		if err := rows.Scan(&it.ID, &orderID, &it.ProductID, &it.ProductName, &it.PriceAtOrder, &it.Quantity); err != nil {
			return nil, err
		}
		itemsByOrder[orderID] = append(itemsByOrder[orderID], it)
		all = append(all, it)
		itemOrder = append(itemOrder, orderID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Seçimler TÜM siparişlerin kalemleri için tek batch sorguyla çekilir —
	// sipariş başına ayrı sorgu N+1 olurdu (aynı ders itemsOfMany'nin
	// kendisinde List için önceden alınmıştı).
	if err := s.attachOptions(ctx, all); err != nil {
		return nil, err
	}
	itemsByOrder = make(map[int64][]OrderItem, len(orderIDs))
	for i, it := range all {
		itemsByOrder[itemOrder[i]] = append(itemsByOrder[itemOrder[i]], it)
	}

	return itemsByOrder, nil
}

// Update status ve/veya note günceller. nil olan alan değişmez (PATCH semantiği).
func (s *Store) Update(ctx context.Context, id int64, status *string, note *string) (*Order, error) {
	ct, err := s.pool.Exec(ctx, `
		UPDATE orders SET
			status = COALESCE($2, status),
			note = COALESCE($3, note),
			updated_at = now()
		WHERE id = $1`, id, status, note)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, errorsx.ErrNotFound
	}

	return s.GetByID(ctx, id)
}

// SetPaymentRef siparişe PayTR merchant_oid yazar (token isteğinden sonra).
func (s *Store) SetPaymentRef(ctx context.Context, id int64, ref string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE orders SET payment_ref = $2, updated_at = now() WHERE id = $1`, id, ref)
	return err
}

// GetByPaymentRef callback'te merchant_oid ile siparişi bulur.
func (s *Store) GetByPaymentRef(ctx context.Context, ref string) (*Order, error) {
	o, err := scanOrder(s.pool.QueryRow(ctx, orderSelect+` WHERE payment_ref = $1`, ref))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsx.ErrNotFound
		}
		return nil, err
	}
	items, err := s.itemsOf(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return o, nil
}

// SetPaid siparişi paid yapar (yalnızca awaiting_payment'tan). Zaten paid ise
// dokunmaz — idempotency callback'te AddPaymentEvent kontrolüyle sağlanır ama
// bu koşul çift güvenlik.
func (s *Store) SetPaid(ctx context.Context, id int64) (*Order, error) {
	ct, err := s.pool.Exec(ctx, `
		UPDATE orders SET status = 'paid', paid_at = now(), updated_at = now()
		WHERE id = $1 AND status = 'awaiting_payment'`, id)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, errorsx.ErrNotFound
	}
	return s.GetByID(ctx, id)
}

// SetRefunded siparişi refunded yapar (paid veya delivered'dan).
func (s *Store) SetRefunded(ctx context.Context, id int64) (*Order, error) {
	ct, err := s.pool.Exec(ctx, `
		UPDATE orders SET status = 'refunded', refunded_at = now(), updated_at = now()
		WHERE id = $1 AND status IN ('paid','delivered')`, id)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, errorsx.ErrNotFound
	}
	return s.GetByID(ctx, id)
}

// AddPaymentEvent denetim izi kaydı ekler.
func (s *Store) AddPaymentEvent(ctx context.Context, orderID int64, eventType string, payload []byte) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO payment_events (order_id, event_type, raw_payload) VALUES ($1,$2,$3)`,
		orderID, eventType, payload)
	return err
}

// HasPaymentEvent bu tip olay bu sipariş için işlenmiş mi (idempotency).
func (s *Store) HasPaymentEvent(ctx context.Context, orderID int64, eventType string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM payment_events WHERE order_id=$1 AND event_type=$2)`,
		orderID, eventType).Scan(&exists)
	return exists, err
}

// customerOrderLimit bir müşterinin hesabım ekranında dönülen en fazla
// sipariş sayısı.
//
// Sayfalama BİLİNÇLİ olarak yok: tek şubeli bir çiçekçide tek müşterinin
// 200'ü aşan siparişi gerçekçi değil ve "hesabım" ekranı bir arşiv değil,
// son siparişlere bakma yeri. Sınır aşılırsa en eskiler sessizce düşer —
// müşteri sayısı bu ölçeğe gelirse burası sayfalamaya çevrilmeli
// (listWhere zaten limit/offset alıyor, değişiklik küçük olur).
const customerOrderLimit = 200

// ListByCustomer bir müşterinin kendi siparişlerini en yeniden eskiye döner.
func (s *Store) ListByCustomer(ctx context.Context, customerID int64) ([]Order, error) {
	return s.listWhere(ctx, "customer_id = $1", []any{customerID}, customerOrderLimit, 0)
}

// RecentAddress müşterinin daha önce gönderdiği bir teslimat adresi.
// Adres defteri tablosu YOK — bu veri geçmiş siparişlerden türetiliyor.
type RecentAddress struct {
	RecipientName    string `json:"recipient_name"`
	RecipientPhone   string `json:"recipient_phone"`
	DeliveryAddress  string `json:"delivery_address"`
	DeliveryDistrict string `json:"delivery_district"`
}

// recentAddressLimit sipariş formunda önerilecek en fazla adres sayısı.
// Uzun liste seçimi kolaylaştırmaz, zorlaştırır.
const recentAddressLimit = 5

// RecentAddresses müşterinin geçmiş siparişlerinden benzersiz teslimat
// adreslerini en yeniden eskiye döner.
//
// Aynı alıcıya birden çok sipariş verilmişse tek satır döner (DISTINCT ON);
// hangi sipariş olduğu önemli değil, adresin kendisi öneriliyor. Sıralama
// için son kullanım tarihi baz alınır — en son gönderilen en üstte.
func (s *Store) RecentAddresses(ctx context.Context, customerID int64) ([]RecentAddress, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT recipient_name, recipient_phone, delivery_address, delivery_district
		FROM (
			SELECT DISTINCT ON (recipient_name, recipient_phone, delivery_address, delivery_district)
			       recipient_name, recipient_phone, delivery_address, delivery_district, created_at
			FROM orders
			WHERE customer_id = $1
			ORDER BY recipient_name, recipient_phone, delivery_address, delivery_district, created_at DESC
		) AS benzersiz
		ORDER BY created_at DESC
		LIMIT $2`, customerID, recentAddressLimit)
	if err != nil {
		return nil, fmt.Errorf("geçmiş adresler: %w", err)
	}
	defer rows.Close()

	adresler := []RecentAddress{}
	for rows.Next() {
		var a RecentAddress
		if err := rows.Scan(&a.RecipientName, &a.RecipientPhone, &a.DeliveryAddress, &a.DeliveryDistrict); err != nil {
			return nil, err
		}
		adresler = append(adresler, a)
	}
	return adresler, rows.Err()
}

// BeginTx sipariş ve stok işlemlerinin aynı transaction'da yürümesi için.
func (s *Store) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.pool.Begin(ctx)
}

// SetPaidTx SetPaid'in transaction'a katılan varyantı. Stok kesinleşmesi
// ile ödeme aynı transaction'da olmalı — para hareketi ile stok ayrışamaz
// (spec §8).
func (s *Store) SetPaidTx(ctx context.Context, tx pgx.Tx, id int64) error {
	ct, err := tx.Exec(ctx, `
		UPDATE orders SET status = 'paid', paid_at = now(), updated_at = now()
		WHERE id = $1 AND status = 'awaiting_payment'`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

// StockLine stok işlemleri için sipariş kalemi özeti.
type StockLine struct {
	ProductID     int64
	Quantity      int
	WasDiscounted bool
}

// ItemsForStock siparişin stok etkileyen kalemlerini döner. Ürünü silinmiş
// kalem (product_id NULL) atlanır — düşülecek stok yok.
func (s *Store) ItemsForStock(ctx context.Context, orderID int64) ([]StockLine, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT product_id, quantity, was_discounted
		  FROM order_items
		 WHERE order_id = $1 AND product_id IS NOT NULL`, orderID)
	if err != nil {
		return nil, fmt.Errorf("stok kalemleri: %w", err)
	}
	defer rows.Close()

	out := make([]StockLine, 0)
	for rows.Next() {
		var l StockLine
		if err := rows.Scan(&l.ProductID, &l.Quantity, &l.WasDiscounted); err != nil {
			return nil, fmt.Errorf("stok kalemi scan: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
