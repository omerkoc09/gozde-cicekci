package product

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insertAwaitingOrder test için ödeme bekleyen sipariş oluşturur.
// order paketine bağımlılık yaratmamak için doğrudan SQL (insertCategory
// deseninin aynısı).
func insertAwaitingOrder(t *testing.T, pool *pgxpool.Pool, productID int64,
	qty int, yas time.Duration) int64 {
	t.Helper()
	ctx := context.Background()
	var orderID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO orders (order_no, status, buyer_name, buyer_phone,
		   recipient_name, recipient_phone, delivery_address, delivery_district,
		   delivery_date, delivery_slot, items_total, delivery_fee, total, created_at)
		VALUES ($1, 'awaiting_payment', 'Test Alıcı', '5551112233',
		   'Test Gönderilen', '5554445566', 'Adres', 'Konak',
		   CURRENT_DATE, '09:00-12:00', 100, 0, 100, now() - $2::interval)
		RETURNING id`,
		"TEST-"+time.Now().Format("150405.000000000"),
		yas.String()).Scan(&orderID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO order_items (order_id, product_id, product_name,
		                         price_at_order, quantity)
		VALUES ($1, $2, 'Test Ürün', 100, $3)`, orderID, productID, qty)
	require.NoError(t, err)
	return orderID
}

func TestSweeper_SuresiGecenRezervasyonSerbestKalir(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Gül", Price: price(t, "100.00"), IsActive: true,
	}, "gul-sweep")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 5, 2)
	insertAwaitingOrder(t, pool, p.ID, 2, 30*time.Minute)

	sweeper := NewSweeper(store, 20*time.Minute, time.Minute)
	sayi, err := sweeper.SweepOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, sayi)
	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 0, got.StockReserved, "rezervasyon serbest kalmalı")
	assert.Equal(t, 5, got.StockQuantity, "fiziksel stok değişmemeli")
}

func TestSweeper_SuresiGecmeyeniDokunmaz(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Orkide", Price: price(t, "100.00"), IsActive: true,
	}, "orkide-sweep")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 5, 2)
	// 19 dakikalık sipariş — henüz süresi geçmedi
	insertAwaitingOrder(t, pool, p.ID, 2, 19*time.Minute)

	sweeper := NewSweeper(store, 20*time.Minute, time.Minute)
	sayi, err := sweeper.SweepOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, 0, sayi)
	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 2, got.StockReserved, "süresi geçmemiş rezervasyon durmalı")
}

func TestSweeper_AyniSiparisIkiKezSupurulmez(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Lale", Price: price(t, "100.00"), IsActive: true,
	}, "lale-sweep")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 5, 2)
	insertAwaitingOrder(t, pool, p.ID, 2, 30*time.Minute)

	sweeper := NewSweeper(store, 20*time.Minute, time.Minute)
	_, err = sweeper.SweepOnce(ctx)
	require.NoError(t, err)

	sayi, err := sweeper.SweepOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, 0, sayi, "ikinci süpürme aynı siparişi tekrar işlememeli")
	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 0, got.StockReserved, "rezerve negatife düşmemeli")
}

func TestSweeper_OdenmisSiparisiSupurmez(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Kaktüs", Price: price(t, "100.00"), IsActive: true,
	}, "kaktus-sweep")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 5, 2)
	orderID := insertAwaitingOrder(t, pool, p.ID, 2, 30*time.Minute)
	_, err = pool.Exec(ctx, `UPDATE orders SET status='paid' WHERE id=$1`, orderID)
	require.NoError(t, err)

	sweeper := NewSweeper(store, 20*time.Minute, time.Minute)
	sayi, err := sweeper.SweepOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, 0, sayi, "ödenmiş siparişin rezervasyonu kesinleşmiştir")
	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 2, got.StockReserved)
}
