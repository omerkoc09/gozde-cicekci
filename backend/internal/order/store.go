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
	       paid_at, refunded_at, COALESCE(payment_ref, ''),
	       COALESCE(note, ''), created_at, updated_at
	FROM orders`

func scanOrder(row pgx.Row) (*Order, error) {
	var o Order

	err := row.Scan(&o.ID, &o.OrderNo, &o.Status,
		&o.BuyerName, &o.BuyerPhone, &o.BuyerEmail,
		&o.RecipientName, &o.RecipientPhone, &o.DeliveryAddress, &o.DeliveryDistrict,
		&o.DeliveryDate, &o.DeliverySlot, &o.CardMessage,
		&o.ItemsTotal, &o.DeliveryFee, &o.Total,
		&o.PaidAt, &o.RefundedAt, &o.PaymentRef,
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
			items_total, delivery_fee, total
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id`,
		orderNo, in.BuyerName, in.BuyerPhone, nullIfEmpty(in.BuyerEmail),
		in.RecipientName, in.RecipientPhone, in.DeliveryAddress, in.DeliveryDistrict,
		in.DeliveryDate, in.DeliverySlot, nullIfEmpty(in.CardMessage),
		in.ItemsTotal, in.DeliveryFee, in.Total,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	for _, it := range in.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (order_id, product_id, product_name, price_at_order, quantity)
			VALUES ($1,$2,$3,$4,$5)`,
			id, it.ProductID, it.ProductName, it.PriceAtOrder, it.Quantity)
		if err != nil {
			return nil, err
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

	return items, rows.Err()
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
	for rows.Next() {
		var it OrderItem
		var orderID int64
		if err := rows.Scan(&it.ID, &orderID, &it.ProductID, &it.ProductName, &it.PriceAtOrder, &it.Quantity); err != nil {
			return nil, err
		}
		itemsByOrder[orderID] = append(itemsByOrder[orderID], it)
	}

	return itemsByOrder, rows.Err()
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
