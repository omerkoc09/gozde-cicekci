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
	       recipient_name, recipient_phone, delivery_address,
	       delivery_date, delivery_slot, COALESCE(card_message, ''),
	       items_total, delivery_fee, total,
	       COALESCE(note, ''), created_at, updated_at
	FROM orders`

func scanOrder(row pgx.Row) (*Order, error) {
	var o Order

	err := row.Scan(&o.ID, &o.OrderNo, &o.Status,
		&o.BuyerName, &o.BuyerPhone, &o.BuyerEmail,
		&o.RecipientName, &o.RecipientPhone, &o.DeliveryAddress,
		&o.DeliveryDate, &o.DeliverySlot, &o.CardMessage,
		&o.ItemsTotal, &o.DeliveryFee, &o.Total,
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
			recipient_name, recipient_phone, delivery_address,
			delivery_date, delivery_slot, card_message,
			items_total, delivery_fee, total
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id`,
		orderNo, in.BuyerName, in.BuyerPhone, nullIfEmpty(in.BuyerEmail),
		in.RecipientName, in.RecipientPhone, in.DeliveryAddress,
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
	q := orderSelect
	args := []any{}

	if status != "" {
		q += ` WHERE status = $1`
		args = append(args, status)
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

	// Liste ekranı kalemleri de gösteriyor (ürün adları)
	for i := range orders {
		items, err := s.itemsOf(ctx, orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].Items = items
	}

	return orders, nil
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
