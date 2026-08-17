package product

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) (*Store, *pgxpool.Pool, context.Context) {
	t.Helper()
	pool := database.NewTestDB(t)
	return NewStore(pool), pool, context.Background()
}

// insertCategory test için doğrudan kategori ekler — category paketine
// bağımlılık yaratmamak için (import cycle olurdu).
func insertCategory(t *testing.T, pool *pgxpool.Pool, name, slug, axis string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO categories (name, slug, axis, is_active)
		 VALUES ($1, $2, $3, true) RETURNING id`,
		name, slug, axis,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func price(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	require.NoError(t, err)
	return d
}

func TestStore_Create(t *testing.T) {
	store, _, ctx := newTestStore(t)

	p, err := store.Create(ctx, CreateInput{
		Name:        "51 Gül Buket",
		Description: "Kırmızı güller",
		Price:       price(t, "1850.00"),
		IsActive:    true,
	}, "51-gul-buket")

	require.NoError(t, err)
	assert.Equal(t, "51 Gül Buket", p.Name)
	assert.Equal(t, "51-gul-buket", p.Slug)
	assert.True(t, p.Price.Equal(price(t, "1850.00")))
	assert.True(t, p.IsActive)
}

func TestStore_Create_PriceKeepsPrecision(t *testing.T) {
	store, _, ctx := newTestStore(t)

	p, err := store.Create(ctx, CreateInput{
		Name: "Test", Price: price(t, "1234.56"), IsActive: true,
	}, "test")

	require.NoError(t, err)
	assert.Equal(t, "1234.56", p.Price.StringFixed(2))
}

func TestStore_FindSlug_Current(t *testing.T) {
	store, _, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "100"), IsActive: true,
	}, "buket")
	require.NoError(t, err)

	id, isCurrent, err := store.FindSlug(ctx, "buket")

	require.NoError(t, err)
	assert.Equal(t, p.ID, id)
	assert.True(t, isCurrent)
}

func TestStore_FindSlug_NotFound(t *testing.T) {
	store, _, ctx := newTestStore(t)

	_, _, err := store.FindSlug(ctx, "yok-boyle-bir-slug")

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

// Spec §4.2: isim değişince eski slug is_current=false olur, yeni slug eklenir.
// Eski slug'a gelen istek 301 ile yönlendirilir.
func TestStore_AddSlug_OldSlugStillResolves(t *testing.T) {
	store, _, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "51 Gül Buket", Price: price(t, "1850"), IsActive: true,
	}, "51-gul-buket")
	require.NoError(t, err)

	require.NoError(t, store.AddSlug(ctx, p.ID, "51-kirmizi-gul-buketi"))

	oldID, oldIsCurrent, err := store.FindSlug(ctx, "51-gul-buket")
	require.NoError(t, err)
	assert.Equal(t, p.ID, oldID)
	assert.False(t, oldIsCurrent, "eski slug is_current=false olmalı")

	newID, newIsCurrent, err := store.FindSlug(ctx, "51-kirmizi-gul-buketi")
	require.NoError(t, err)
	assert.Equal(t, p.ID, newID)
	assert.True(t, newIsCurrent, "yeni slug güncel olmalı")
}

func TestStore_AddSlug_ProductSlugFieldReflectsCurrent(t *testing.T) {
	store, _, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "100"), IsActive: true,
	}, "buket")
	require.NoError(t, err)

	require.NoError(t, store.AddSlug(ctx, p.ID, "gul-buketi"))

	fetched, err := store.GetByID(ctx, p.ID)

	require.NoError(t, err)
	assert.Equal(t, "gul-buketi", fetched.Slug, "GetByID güncel slug'ı dönmeli")
}

func TestStore_SlugExists(t *testing.T) {
	store, _, ctx := newTestStore(t)
	_, err := store.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "100"), IsActive: true,
	}, "buket")
	require.NoError(t, err)

	exists, err := store.SlugExists(ctx, "buket")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = store.SlugExists(ctx, "yok")
	require.NoError(t, err)
	assert.False(t, exists)
}

// Eski slug da çakışma sayılır — aynı slug iki ürüne verilemez.
func TestStore_SlugExists_IncludesOldSlugs(t *testing.T) {
	store, _, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "100"), IsActive: true,
	}, "buket")
	require.NoError(t, err)
	require.NoError(t, store.AddSlug(ctx, p.ID, "gul-buketi"))

	exists, err := store.SlugExists(ctx, "buket")

	require.NoError(t, err)
	assert.True(t, exists, "eski slug hâlâ rezerve olmalı")
}

func TestStore_GetPublicByID_HidesInactive(t *testing.T) {
	store, _, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Pasif", Price: price(t, "100"), IsActive: false,
	}, "pasif")
	require.NoError(t, err)

	_, err = store.GetPublicByID(ctx, p.ID)

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestStore_SetCategories(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	dogumGunu := insertCategory(t, pool, "Doğum Günü", "dogum-gunu", "occasion")
	buket := insertCategory(t, pool, "Buket", "buket", "type")
	p, err := store.Create(ctx, CreateInput{
		Name: "Test", Price: price(t, "100"), IsActive: true,
	}, "test")
	require.NoError(t, err)

	require.NoError(t, store.SetCategories(ctx, p.ID, []int64{dogumGunu, buket}))

	fetched, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int64{dogumGunu, buket}, fetched.CategoryIDs)
}

func TestStore_SetCategories_ReplacesExisting(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	a := insertCategory(t, pool, "A", "a", "occasion")
	b := insertCategory(t, pool, "B", "b", "occasion")
	p, err := store.Create(ctx, CreateInput{
		Name: "Test", Price: price(t, "100"), IsActive: true,
	}, "test")
	require.NoError(t, err)
	require.NoError(t, store.SetCategories(ctx, p.ID, []int64{a}))

	require.NoError(t, store.SetCategories(ctx, p.ID, []int64{b}))

	fetched, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, []int64{b}, fetched.CategoryIDs)
}

func TestStore_SetCategories_Empty(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	a := insertCategory(t, pool, "A", "a", "occasion")
	p, err := store.Create(ctx, CreateInput{
		Name: "Test", Price: price(t, "100"), IsActive: true,
	}, "test")
	require.NoError(t, err)
	require.NoError(t, store.SetCategories(ctx, p.ID, []int64{a}))

	require.NoError(t, store.SetCategories(ctx, p.ID, []int64{}))

	fetched, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Empty(t, fetched.CategoryIDs)
}

func TestStore_Delete_CascadesSlugsAndCategories(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	cat := insertCategory(t, pool, "A", "a", "occasion")
	p, err := store.Create(ctx, CreateInput{
		Name: "Test", Price: price(t, "100"), IsActive: true,
	}, "test")
	require.NoError(t, err)
	require.NoError(t, store.SetCategories(ctx, p.ID, []int64{cat}))
	require.NoError(t, store.AddSlug(ctx, p.ID, "test-2"))

	require.NoError(t, store.Delete(ctx, p.ID))

	_, _, err = store.FindSlug(ctx, "test")
	assert.ErrorIs(t, err, errorsx.ErrNotFound, "slug geçmişi de silinmeli")

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM product_categories WHERE product_id = $1`, p.ID).Scan(&count))
	assert.Equal(t, 0, count, "junction kayıtları silinmeli")
}

// Kategori silinince ürün silinmez, sadece bağ kopar (spec §4.1).
func TestStore_CategoryDelete_KeepsProduct(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	cat := insertCategory(t, pool, "Silinecek", "silinecek", "occasion")
	p, err := store.Create(ctx, CreateInput{
		Name: "Ürün", Price: price(t, "100"), IsActive: true,
	}, "urun")
	require.NoError(t, err)
	require.NoError(t, store.SetCategories(ctx, p.ID, []int64{cat}))

	_, err = pool.Exec(ctx, `DELETE FROM categories WHERE id = $1`, cat)
	require.NoError(t, err)

	fetched, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err, "ürün silinmemeli")
	assert.Empty(t, fetched.CategoryIDs, "sadece bağ kopmalı")
}

// UpdateInput'ta nil olan alan değişmez (PATCH semantiği) — COALESCE
// parametrelerinin doğru sırayla eşleştiğini kanıtlar.
func TestStore_Update_PartialFieldsOnly(t *testing.T) {
	store, _, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name:        "Buket",
		Description: "Orijinal açıklama",
		Price:       price(t, "500.00"),
		IsActive:    true,
	}, "buket")
	require.NoError(t, err)

	newName := "Gül Buketi"
	updated, err := store.Update(ctx, p.ID, UpdateInput{Name: &newName}, "")

	require.NoError(t, err)
	assert.Equal(t, "Gül Buketi", updated.Name)
	assert.Equal(t, "Orijinal açıklama", updated.Description, "açıklama değişmemeli")
	assert.Equal(t, "500.00", updated.Price.StringFixed(2), "fiyat değişmemeli")
	assert.True(t, updated.IsActive, "is_active değişmemeli")
}

func TestStore_Update_AllFields(t *testing.T) {
	store, _, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name:        "Buket",
		Description: "Eski",
		Price:       price(t, "500.00"),
		IsActive:    true,
	}, "buket")
	require.NoError(t, err)

	newName := "Kırmızı Gül Buketi"
	newDesc := "Yeni açıklama"
	newPrice := price(t, "1850.50")
	newActive := false
	updated, err := store.Update(ctx, p.ID, UpdateInput{
		Name:        &newName,
		Description: &newDesc,
		Price:       &newPrice,
		IsActive:    &newActive,
	}, "")

	require.NoError(t, err)
	assert.Equal(t, "Kırmızı Gül Buketi", updated.Name)
	assert.Equal(t, "Yeni açıklama", updated.Description)
	assert.Equal(t, "1850.50", updated.Price.StringFixed(2))
	assert.False(t, updated.IsActive)
}

// Şemada UPDATE trigger'ı yok — updated_at'i sorgu elle set ediyor
// (`updated_at = now()`). O satır silinirse bu test yakalar.
func TestStore_Update_SetsUpdatedAt(t *testing.T) {
	store, _, ctx := newTestStore(t)
	before, err := store.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "500"), IsActive: true,
	}, "buket")
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	newName := "Gül Buketi"
	updated, err := store.Update(ctx, before.ID, UpdateInput{Name: &newName}, "")

	require.NoError(t, err)
	assert.True(t, updated.UpdatedAt.After(before.UpdatedAt),
		"updated_at artmalı: önce %v, sonra %v", before.UpdatedAt, updated.UpdatedAt)
}

// CategoryIDs nil ise kategorilere dokunulmaz.
func TestStore_Update_CategoryIDsNilDoesNotTouchCategories(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	dogumGunu := insertCategory(t, pool, "Doğum Günü", "dogum-gunu", "occasion")
	buket := insertCategory(t, pool, "Buket", "buket", "type")
	p, err := store.Create(ctx, CreateInput{
		Name: "Test", Price: price(t, "500"), IsActive: true,
		CategoryIDs: []int64{dogumGunu, buket},
	}, "test")
	require.NoError(t, err)

	newName := "Yeni Ad"
	updated, err := store.Update(ctx, p.ID, UpdateInput{Name: &newName}, "")

	require.NoError(t, err)
	assert.ElementsMatch(t, []int64{dogumGunu, buket}, updated.CategoryIDs,
		"CategoryIDs nil ise kategoriler aynen kalmalı")
}

// CategoryIDs boş slice ise tüm kategoriler kaldırılır. Bir önceki testle
// birlikte nil/boş ayrımını kilitler.
func TestStore_Update_CategoryIDsEmptySliceRemovesAll(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	dogumGunu := insertCategory(t, pool, "Doğum Günü", "dogum-gunu", "occasion")
	buket := insertCategory(t, pool, "Buket", "buket", "type")
	p, err := store.Create(ctx, CreateInput{
		Name: "Test", Price: price(t, "500"), IsActive: true,
		CategoryIDs: []int64{dogumGunu, buket},
	}, "test")
	require.NoError(t, err)

	updated, err := store.Update(ctx, p.ID, UpdateInput{CategoryIDs: []int64{}}, "")

	require.NoError(t, err)
	assert.Empty(t, updated.CategoryIDs, "boş slice tüm kategorileri kaldırmalı")
}

func TestStore_Update_NotFound(t *testing.T) {
	store, _, ctx := newTestStore(t)

	newName := "Yok"
	_, err := store.Update(ctx, 9999, UpdateInput{Name: &newName}, "")

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

// Slug ve isim aynı transaction'da yazılır — biri olup diğeri olmaz durumu
// oluşamaz. newSlug verilince eski slug is_current=false olur, yenisi güncel.
func TestStore_Update_WithNewSlugIsAtomic(t *testing.T) {
	store, _, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "500"), IsActive: true,
	}, "buket")
	require.NoError(t, err)

	newName := "Gül Buketi"
	updated, err := store.Update(ctx, p.ID, UpdateInput{Name: &newName}, "gul-buketi")

	require.NoError(t, err)
	assert.Equal(t, "Gül Buketi", updated.Name)
	assert.Equal(t, "gul-buketi", updated.Slug, "yeni slug güncel olmalı")

	// Eski slug hâlâ çözülmeli (301 için) ama güncel olmamalı.
	oldID, oldIsCurrent, err := store.FindSlug(ctx, "buket")
	require.NoError(t, err)
	assert.Equal(t, p.ID, oldID)
	assert.False(t, oldIsCurrent, "eski slug is_current=false olmalı")
}

// newSlug boşsa slug'a hiç dokunulmaz.
func TestStore_Update_EmptySlugLeavesSlugAlone(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "500"), IsActive: true,
	}, "buket")
	require.NoError(t, err)

	newPrice := price(t, "600")
	updated, err := store.Update(ctx, p.ID, UpdateInput{Price: &newPrice}, "")

	require.NoError(t, err)
	assert.Equal(t, "buket", updated.Slug)

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM product_slugs WHERE product_id = $1`, p.ID).Scan(&count))
	assert.Equal(t, 1, count, "yeni slug kaydı eklenmemeli")
}

// setStock test için doğrudan stok alanlarını yazar — service katmanına
// bağımlı olmadan store davranışını sınamak için (insertCategory deseni).
func setStock(t *testing.T, pool *pgxpool.Pool, id int64,
	track bool, qty, reserved int) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`UPDATE products SET track_stock=$2, stock_quantity=$3, stock_reserved=$4
		 WHERE id=$1`, id, track, qty, reserved)
	require.NoError(t, err)
}

func setDiscount(t *testing.T, pool *pgxpool.Pool, id int64,
	price string, quota, sold int) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`UPDATE products SET discount_price=$2, discount_quota=$3, discount_sold=$4
		 WHERE id=$1`, id, price, quota, sold)
	require.NoError(t, err)
}

func TestStore_GetByID_StokAlanlariOkunur(t *testing.T) {
	store, pool, ctx := newTestStore(t)

	p, err := store.Create(ctx, CreateInput{
		Name: "Gül Buketi", Price: price(t, "1850.00"), IsActive: true,
	}, "gul-buketi")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 10, 3)
	setDiscount(t, pool, p.ID, "1450.00", 10, 2)

	got, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err)

	assert.True(t, got.TrackStock)
	assert.Equal(t, 10, got.StockQuantity)
	assert.Equal(t, 3, got.StockReserved)
	require.NotNil(t, got.DiscountPrice)
	assert.Equal(t, "1450", got.DiscountPrice.String())
	require.NotNil(t, got.DiscountQuota)
	assert.Equal(t, 10, *got.DiscountQuota)
	assert.Equal(t, 2, got.DiscountSold)
	assert.True(t, got.DiscountActive())
}

func TestStore_GetByID_IndirimsizUrun_NilDoner(t *testing.T) {
	store, _, ctx := newTestStore(t)

	p, err := store.Create(ctx, CreateInput{
		Name: "Orkide", Price: price(t, "900.00"), IsActive: true,
	}, "orkide")
	require.NoError(t, err)

	got, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err)

	assert.False(t, got.TrackStock, "varsayılan takipsiz olmalı")
	assert.Nil(t, got.DiscountPrice)
	assert.Nil(t, got.DiscountQuota)
	assert.False(t, got.DiscountActive())
}

func TestStore_ListPublic_DiscountedOnly(t *testing.T) {
	store, pool, ctx := newTestStore(t)

	indirimli, err := store.Create(ctx, CreateInput{
		Name: "İndirimli Buket", Price: price(t, "1850.00"), IsActive: true,
	}, "indirimli-buket")
	require.NoError(t, err)
	setDiscount(t, pool, indirimli.ID, "1450.00", 10, 3)

	// Kotası dolmuş ürün indirimli listede GÖRÜNMEMELİ
	kotasiDolmus, err := store.Create(ctx, CreateInput{
		Name: "Kotası Dolmuş", Price: price(t, "1200.00"), IsActive: true,
	}, "kotasi-dolmus")
	require.NoError(t, err)
	setDiscount(t, pool, kotasiDolmus.ID, "1000.00", 5, 5)

	_, err = store.Create(ctx, CreateInput{
		Name: "Normal Ürün", Price: price(t, "700.00"), IsActive: true,
	}, "normal-urun")
	require.NoError(t, err)

	list, err := store.ListPublic(ctx, Filter{DiscountedOnly: true, Limit: 50})
	require.NoError(t, err)

	require.Len(t, list, 1, "yalnızca indirimi AKTİF ürün dönmeli")
	assert.Equal(t, "İndirimli Buket", list[0].Name)
}

func TestStore_Update_StokVeIndirimYazilir(t *testing.T) {
	store, _, ctx := newTestStore(t)

	p, err := store.Create(ctx, CreateInput{
		Name: "Lilyum", Price: price(t, "600.00"), IsActive: true,
	}, "lilyum")
	require.NoError(t, err)

	track := true
	qty := 25
	dp := price(t, "450.00")
	quota := 8
	updated, err := store.Update(ctx, p.ID, UpdateInput{
		TrackStock: &track, StockQuantity: &qty,
		DiscountPrice: &dp, DiscountQuota: &quota,
	}, "")
	require.NoError(t, err)

	assert.True(t, updated.TrackStock)
	assert.Equal(t, 25, updated.StockQuantity)
	require.NotNil(t, updated.DiscountPrice)
	assert.Equal(t, "450", updated.DiscountPrice.String())
	assert.Equal(t, 8, *updated.DiscountQuota)
}

func TestStore_Update_ClearDiscount_SayaciSifirlar(t *testing.T) {
	store, pool, ctx := newTestStore(t)

	p, err := store.Create(ctx, CreateInput{
		Name: "Papatya", Price: price(t, "300.00"), IsActive: true,
	}, "papatya")
	require.NoError(t, err)
	setDiscount(t, pool, p.ID, "250.00", 5, 4)

	updated, err := store.Update(ctx, p.ID, UpdateInput{ClearDiscount: true}, "")
	require.NoError(t, err)

	assert.Nil(t, updated.DiscountPrice)
	assert.Nil(t, updated.DiscountQuota)
	// Sıfırlanmazsa aynı ürüne ikinci indirim girildiğinde kota baştan
	// dolu görünür (spec §5.2).
	assert.Equal(t, 0, updated.DiscountSold, "sayaç sıfırlanmalı")
}
