package product

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// Reason stok hareketinin sebebi. DB'deki CHECK ile birebir aynı olmalı.
type Reason string

const (
	ReasonSiparis          Reason = "siparis"
	ReasonWhatsApp         Reason = "whatsapp_satisi"
	ReasonSayim            Reason = "sayim_duzeltme"
	ReasonYeniParti        Reason = "yeni_parti"
	ReasonIptalIade        Reason = "iptal_iade"
	ReasonRezervasyonIptal Reason = "rezervasyon_iptal"
)

func (r Reason) Valid() bool {
	switch r {
	case ReasonSiparis, ReasonWhatsApp, ReasonSayim,
		ReasonYeniParti, ReasonIptalIade, ReasonRezervasyonIptal:
		return true
	}
	return false
}

// Movement tek bir stok hareketi.
type Movement struct {
	ID            int64     `json:"id"`
	ProductID     int64     `json:"product_id"`
	Delta         int       `json:"delta"`
	Reason        Reason    `json:"reason"`
	OrderID       *int64    `json:"order_id,omitempty"`
	WasDiscounted bool      `json:"was_discounted"`
	Note          string    `json:"note"`
	AdminUserID   *int64    `json:"admin_user_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// StockError stok yetmediğinde döner. Kaç adet kaldığını taşır — müşteri
// sepetini düzeltebilsin diye mesajda bu bilgi geçiyor (spec §8).
type StockError struct {
	ProductID   int64
	ProductName string
	Available   int
}

func (e *StockError) Error() string {
	if e.Available == 0 {
		return fmt.Sprintf("%q tükendi", e.ProductName)
	}
	return fmt.Sprintf("%q için yalnızca %d adet kaldı", e.ProductName, e.Available)
}

// Unwrap sayesinde errors.Is(err, errorsx.ErrInvalidInput) çalışır ve
// handler katmanı bu hatayı 400 olarak döner (mevcut httperr deseni).
func (e *StockError) Unwrap() error { return errorsx.ErrInvalidInput }

// Reserve stoğu rezerve eder. Sipariş transaction'ına KATILIR (tx parametresi)
// — sipariş yazılamazsa rezervasyon da geri alınır.
//
// Kontrol TEK ifadede: koşul WHERE içinde olduğu için Postgres satırı kilitler
// ve eşzamanlı iki istekten yalnızca biri kazanır. "Önce oku, sonra karar ver,
// sonra yaz" yapılırsa okuma ile yazma arasında başkası araya girer ve son
// ürün iki kişiye satılır.
func (s *Store) Reserve(ctx context.Context, tx pgx.Tx, productID int64, qty int) error {
	// Takipsiz üründe sayaç ARTMAZ (CASE) ama satır yine güncellenir ve
	// rezervasyon başarılı sayılır. Artsaydı sayaç sonsuza kadar büyür,
	// esnaf sonradan stok takibini açtığında ürün anında tükendi görünürdü.
	ct, err := tx.Exec(ctx, `
		UPDATE products
		   SET stock_reserved = CASE WHEN track_stock
		                             THEN stock_reserved + $2
		                             ELSE stock_reserved END
		 WHERE id = $1
		   AND (NOT track_stock OR stock_quantity - stock_reserved >= $2)`,
		productID, qty)
	if err != nil {
		return fmt.Errorf("stok rezerve: %w", err)
	}
	if ct.RowsAffected() == 1 {
		return nil
	}

	// Satır güncellenmedi: ya ürün yok ya stok yetmiyor. Hangisi olduğunu
	// ve kaç adet kaldığını öğren — müşteriye anlamlı mesaj vermek için.
	var name string
	var qtyDB, reserved int
	var track bool
	err = tx.QueryRow(ctx,
		`SELECT name, track_stock, stock_quantity, stock_reserved
		   FROM products WHERE id = $1`, productID).
		Scan(&name, &track, &qtyDB, &reserved)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsx.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("stok kontrol: %w", err)
	}

	kalan := qtyDB - reserved
	if kalan < 0 {
		kalan = 0
	}
	return &StockError{ProductID: productID, ProductName: name, Available: kalan}
}

// CommitReservation rezervasyonu kesin düşüşe çevirir. Ödemeyi paid yapan
// transaction'a KATILIR — para hareketi ile stok asla ayrışmamalı (spec §8).
//
// discounted true ise indirim kotası da tüketilir.
func (s *Store) CommitReservation(ctx context.Context, tx pgx.Tx,
	productID int64, qty int, orderID int64, discounted bool) error {
	// TEK ifade: stok ve kota bağımsız kavramlar (spec §2) ama ikisi de aynı
	// satırda. Takipsiz üründe stok alanlarına dokunulmaz, kota yine tüketilir
	// — bu yüzden koşullar SET içinde, WHERE'de değil.
	//
	// GREATEST(...,0): elle düzeltme sonrası rezerve beklenenden az olabilir;
	// negatife düşüp CHECK constraint'i patlatmasın.
	_, err := tx.Exec(ctx, `
		UPDATE products
		   SET stock_reserved = CASE WHEN track_stock
		                             THEN GREATEST(stock_reserved - $2, 0)
		                             ELSE stock_reserved END,
		       stock_quantity = CASE WHEN track_stock
		                             THEN GREATEST(stock_quantity - $2, 0)
		                             ELSE stock_quantity END,
		       discount_sold  = discount_sold + CASE WHEN $3 THEN $2 ELSE 0 END,
		       updated_at     = now()
		 WHERE id = $1`,
		productID, qty, discounted)
	if err != nil {
		return fmt.Errorf("stok kesinleştir: %w", err)
	}

	var oid *int64
	if orderID > 0 {
		oid = &orderID
	}
	return s.addMovementTx(ctx, tx, Movement{
		ProductID: productID, Delta: -qty, Reason: ReasonSiparis,
		OrderID: oid, WasDiscounted: discounted,
	})
}

// Release rezervasyonu serbest bırakır (ödeme gelmedi). Fiziksel stok
// değişmez — ürün hiç satılmadı.
func (s *Store) Release(ctx context.Context, productID int64, qty int, orderID *int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE products
		   SET stock_reserved = GREATEST(stock_reserved - $2, 0), updated_at = now()
		 WHERE id = $1 AND track_stock`, productID, qty); err != nil {
		return fmt.Errorf("rezervasyon serbest: %w", err)
	}

	// delta 0: fiziksel stok değişmedi, ama izi kalmalı — sessizce kaybolan
	// bir şey olmasın (spec §4.3).
	if err := s.addMovementTx(ctx, tx, Movement{
		ProductID: productID, Delta: 0, Reason: ReasonRezervasyonIptal,
		OrderID: orderID, Note: fmt.Sprintf("%d adet rezervasyon serbest bırakıldı", qty),
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RestoreStock iade sonrası ürünü rafa geri koyar.
func (s *Store) RestoreStock(ctx context.Context, productID int64, qty int, orderID *int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE products
		   SET stock_quantity = stock_quantity + $2, updated_at = now()
		 WHERE id = $1 AND track_stock`, productID, qty); err != nil {
		return fmt.Errorf("stok iade: %w", err)
	}
	if err := s.addMovementTx(ctx, tx, Movement{
		ProductID: productID, Delta: qty, Reason: ReasonIptalIade, OrderID: orderID,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ManualAdjustInput panelden yapılan elle stok düzeltmesi.
type ManualAdjustInput struct {
	ProductID     int64
	Delta         int
	Reason        Reason
	WasDiscounted bool
	Note          string
	AdminUserID   *int64
}

// ManualAdjust elle stok düzeltmesi yapar (WhatsApp satışı, sayım, yeni parti).
// Stoğun altına düşen düzeltme reddedilir.
func (s *Store) ManualAdjust(ctx context.Context, in ManualAdjustInput) (*Product, error) {
	if in.Delta == 0 {
		return nil, fmt.Errorf("%w: değişim miktarı sıfır olamaz", errorsx.ErrInvalidInput)
	}
	if !in.Reason.Valid() {
		return nil, fmt.Errorf("%w: geçersiz sebep", errorsx.ErrInvalidInput)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	// Stok yetmiyorsa satır güncellenmez — Reserve ile aynı atomik desen.
	ct, err := tx.Exec(ctx, `
		UPDATE products
		   SET stock_quantity = stock_quantity + $2,
		       discount_sold  = discount_sold +
		           CASE WHEN $3 AND $2 < 0 THEN -$2 ELSE 0 END,
		       updated_at     = now()
		 WHERE id = $1 AND stock_quantity + $2 >= 0`,
		in.ProductID, in.Delta, in.WasDiscounted)
	if err != nil {
		return nil, fmt.Errorf("stok düzelt: %w", err)
	}
	if ct.RowsAffected() == 0 {
		var name string
		var qty int
		err = tx.QueryRow(ctx, `SELECT name, stock_quantity FROM products WHERE id = $1`,
			in.ProductID).Scan(&name, &qty)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsx.ErrNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("stok kontrol: %w", err)
		}
		return nil, fmt.Errorf("%w: %q stoğu %d, %d adet düşülemez",
			errorsx.ErrInvalidInput, name, qty, -in.Delta)
	}

	if err := s.addMovementTx(ctx, tx, Movement{
		ProductID: in.ProductID, Delta: in.Delta, Reason: in.Reason,
		WasDiscounted: in.WasDiscounted, Note: in.Note, AdminUserID: in.AdminUserID,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx commit: %w", err)
	}
	return s.GetByID(ctx, in.ProductID)
}

func (s *Store) addMovementTx(ctx context.Context, tx pgx.Tx, m Movement) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO stock_movements
		  (product_id, delta, reason, order_id, was_discounted, note, admin_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		m.ProductID, m.Delta, m.Reason, m.OrderID, m.WasDiscounted, m.Note, m.AdminUserID)
	if err != nil {
		return fmt.Errorf("hareket kaydı: %w", err)
	}
	return nil
}

// ListMovements ürünün stok hareketlerini yeniden eskiye döner.
func (s *Store) ListMovements(ctx context.Context, productID int64, limit int) ([]Movement, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, product_id, delta, reason, order_id, was_discounted,
		       note, admin_user_id, created_at
		  FROM stock_movements
		 WHERE product_id = $1
		 ORDER BY created_at DESC, id DESC
		 LIMIT $2`, productID, limit)
	if err != nil {
		return nil, fmt.Errorf("hareket listele: %w", err)
	}
	defer rows.Close()

	out := make([]Movement, 0)
	for rows.Next() {
		var m Movement
		if err := rows.Scan(&m.ID, &m.ProductID, &m.Delta, &m.Reason, &m.OrderID,
			&m.WasDiscounted, &m.Note, &m.AdminUserID, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("hareket scan: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
