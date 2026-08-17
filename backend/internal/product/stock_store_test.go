package product

import (
	"context"
	"sync"
	"testing"

	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reserveTx Reserve'i kendi transaction'ında çalıştırır — üretimde sipariş
// transaction'ına katılıyor, testte tek başına sınanıyor.
func reserveTx(t *testing.T, store *Store, productID int64, qty int) error {
	t.Helper()
	ctx := context.Background()
	tx, err := store.pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	if err := store.Reserve(ctx, tx, productID, qty); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func TestReserve_TakipsizUrun_HerZamanBasarili(t *testing.T) {
	store, _, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Takipsiz", Price: price(t, "100.00"), IsActive: true,
	}, "takipsiz")
	require.NoError(t, err)

	require.NoError(t, reserveTx(t, store, p.ID, 1000))

	got, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, got.StockReserved, "takipsiz üründe rezerve artmamalı")
	assert.Equal(t, 0, got.StockQuantity)
}

func TestReserve_YeterliStok_RezerveArtar(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Gül", Price: price(t, "100.00"), IsActive: true,
	}, "gul")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 5, 0)

	require.NoError(t, reserveTx(t, store, p.ID, 3))

	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 3, got.StockReserved)
	assert.Equal(t, 5, got.StockQuantity, "fiziksel stok rezervasyonda değişmez")
}

func TestReserve_YetersizStok_HataDoner(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Orkide", Price: price(t, "100.00"), IsActive: true,
	}, "orkide")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 2, 0)

	err = reserveTx(t, store, p.ID, 3)

	require.Error(t, err)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)

	var se *StockError
	require.ErrorAs(t, err, &se, "müşteriye kaç adet kaldığı söylenebilmeli")
	assert.Equal(t, 2, se.Available)
	assert.Equal(t, "Orkide", se.ProductName)

	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 0, got.StockReserved, "başarısız rezervasyon iz bırakmamalı")
}

// TestReserve_YarisDurumu tasarımın EN KRİTİK iddiasını kanıtlar:
// son ürün asla iki kişiye satılamaz.
func TestReserve_YarisDurumu_TekKazanan(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Son Ürün", Price: price(t, "100.00"), IsActive: true,
	}, "son-urun")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 1, 0)

	const esZamanli = 20
	var wg sync.WaitGroup
	sonuc := make([]error, esZamanli)

	for i := 0; i < esZamanli; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sonuc[idx] = reserveTx(t, store, p.ID, 1)
		}(i)
	}
	wg.Wait()

	basarili := 0
	for _, err := range sonuc {
		if err == nil {
			basarili++
		}
	}

	assert.Equal(t, 1, basarili, "1 adetlik stoktan tam olarak 1 rezervasyon geçmeli")

	got, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.StockReserved, "rezerve stoğu aşamaz")
	adet, _ := got.Available()
	assert.Equal(t, 0, adet)
}

func TestCommitReservation_StokVeKotaDuser(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "100.00"), IsActive: true,
	}, "buket")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 10, 0)
	setDiscount(t, pool, p.ID, "80.00", 5, 0)
	require.NoError(t, reserveTx(t, store, p.ID, 2))

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, store.CommitReservation(ctx, tx, p.ID, 2, 0, true))
	require.NoError(t, tx.Commit(ctx))

	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 8, got.StockQuantity, "fiziksel stok düşmeli")
	assert.Equal(t, 0, got.StockReserved, "rezerve serbest kalmalı")
	assert.Equal(t, 2, got.DiscountSold, "indirimli satış kotayı tüketmeli")
}

func TestCommitReservation_TakipsizUrun_KotaYineTuketilir(t *testing.T) {
	// Stok ve indirim bağımsız kavramlar (spec §2): stok takibi kapalı olsa
	// da indirim kotası tüketilmeli, yoksa esnaf sınırsız indirimli satar.
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Takipsiz İndirimli", Price: price(t, "100.00"), IsActive: true,
	}, "takipsiz-indirimli")
	require.NoError(t, err)
	setDiscount(t, pool, p.ID, "80.00", 5, 0)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, store.CommitReservation(ctx, tx, p.ID, 2, 0, true))
	require.NoError(t, tx.Commit(ctx))

	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 2, got.DiscountSold, "takipsiz üründe de kota tüketilmeli")
	assert.Equal(t, 0, got.StockQuantity, "takipsiz üründe stok alanları değişmemeli")
	assert.Equal(t, 0, got.StockReserved)
}

func TestRelease_RezerveSerbestKalir(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Lale", Price: price(t, "100.00"), IsActive: true,
	}, "lale")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 5, 0)
	require.NoError(t, reserveTx(t, store, p.ID, 2))

	require.NoError(t, store.Release(ctx, p.ID, 2, nil))

	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 0, got.StockReserved)
	assert.Equal(t, 5, got.StockQuantity, "serbest bırakma fiziksel stoğu değiştirmez")

	hareketler, err := store.ListMovements(ctx, p.ID, 10)
	require.NoError(t, err)
	require.Len(t, hareketler, 1, "serbest bırakma iz bırakmalı")
	assert.Equal(t, ReasonRezervasyonIptal, hareketler[0].Reason)
}

func TestRestoreStock_IadeStoguGeriEkler(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Nergis", Price: price(t, "100.00"), IsActive: true,
	}, "nergis")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 5, 0)

	require.NoError(t, store.RestoreStock(ctx, p.ID, 2, nil))

	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 7, got.StockQuantity)

	hareketler, err := store.ListMovements(ctx, p.ID, 10)
	require.NoError(t, err)
	require.Len(t, hareketler, 1)
	assert.Equal(t, ReasonIptalIade, hareketler[0].Reason)
	assert.Equal(t, 2, hareketler[0].Delta)
}

func TestManualAdjust_WhatsAppSatisi(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Karanfil", Price: price(t, "100.00"), IsActive: true,
	}, "karanfil")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 12, 0)
	setDiscount(t, pool, p.ID, "80.00", 10, 0)

	got, err := store.ManualAdjust(ctx, ManualAdjustInput{
		ProductID: p.ID, Delta: -1,
		Reason: ReasonWhatsApp, WasDiscounted: true,
	})
	require.NoError(t, err)

	assert.Equal(t, 11, got.StockQuantity)
	assert.Equal(t, 1, got.DiscountSold, "WhatsApp satışı da kotayı tüketir (spec §4.4)")

	hareketler, err := store.ListMovements(ctx, p.ID, 10)
	require.NoError(t, err)
	require.Len(t, hareketler, 1)
	assert.Equal(t, -1, hareketler[0].Delta)
	assert.Equal(t, ReasonWhatsApp, hareketler[0].Reason)
	assert.True(t, hareketler[0].WasDiscounted)
}

func TestManualAdjust_StokAltinaDusemez(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Zambak", Price: price(t, "100.00"), IsActive: true,
	}, "zambak")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 2, 0)

	_, err = store.ManualAdjust(ctx, ManualAdjustInput{
		ProductID: p.ID, Delta: -5, Reason: ReasonWhatsApp,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)

	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 2, got.StockQuantity, "başarısız düzeltme stoğu bozmamalı")
}

func TestManualAdjust_YeniParti_StokArtar(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Şebboy", Price: price(t, "100.00"), IsActive: true,
	}, "sebboy")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 3, 0)

	got, err := store.ManualAdjust(ctx, ManualAdjustInput{
		ProductID: p.ID, Delta: 20, Reason: ReasonYeniParti,
	})
	require.NoError(t, err)

	assert.Equal(t, 23, got.StockQuantity)
}

func TestManualAdjust_GecersizSebep_Reddedilir(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Sümbül", Price: price(t, "100.00"), IsActive: true,
	}, "sumbul")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 5, 0)

	_, err = store.ManualAdjust(ctx, ManualAdjustInput{
		ProductID: p.ID, Delta: -1, Reason: Reason("hatali_sebep"),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}
