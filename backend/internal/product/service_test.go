package product

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestProductService(t *testing.T) (*Service, *pgxpool.Pool, context.Context) {
	t.Helper()
	pool := database.NewTestDB(t)
	return NewService(NewStore(pool)), pool, context.Background()
}

func TestProductService_Create_GeneratesSlug(t *testing.T) {
	svc, _, ctx := newTestProductService(t)

	p, err := svc.Create(ctx, CreateInput{
		Name: "51 Gül Buket", Price: price(t, "1850"), IsActive: true,
	})

	require.NoError(t, err)
	assert.Equal(t, "51-gul-buket", p.Slug)
}

func TestProductService_Create_DuplicateNameGetsSuffix(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	_, err := svc.Create(ctx, CreateInput{
		Name: "Gül Buketi", Price: price(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	second, err := svc.Create(ctx, CreateInput{
		Name: "Gül Buketi", Price: price(t, "700"), IsActive: true,
	})

	require.NoError(t, err)
	assert.Equal(t, "gul-buketi-2", second.Slug)
}

func TestProductService_Create_EmptyName(t *testing.T) {
	svc, _, ctx := newTestProductService(t)

	_, err := svc.Create(ctx, CreateInput{Name: "   ", Price: price(t, "100")})

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestProductService_Create_NegativePrice(t *testing.T) {
	svc, _, ctx := newTestProductService(t)

	_, err := svc.Create(ctx, CreateInput{Name: "Test", Price: price(t, "-5")})

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

// Spec §4.2: isim değişince yeni slug üretilir, eskisi 301 için saklanır.
func TestProductService_Update_NameCreatesNewSlugAndKeepsOld(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	p, err := svc.Create(ctx, CreateInput{
		Name: "51 Gül Buket", Price: price(t, "1850"), IsActive: true,
	})
	require.NoError(t, err)

	newName := "51 Kırmızı Gül Buketi"
	updated, err := svc.Update(ctx, p.ID, UpdateInput{Name: &newName})

	require.NoError(t, err)
	assert.Equal(t, "51-kirmizi-gul-buketi", updated.Slug)

	// Eski slug 301 ile yönlendirmeli
	fetched, redirectTo, err := svc.GetPublicBySlug(ctx, "51-gul-buket")
	require.NoError(t, err)
	assert.Equal(t, "51-kirmizi-gul-buketi", redirectTo, "eski slug redirect vermeli")
	assert.Equal(t, p.ID, fetched.ID)
}

func TestProductService_Update_SameNameDoesNotCreateNewSlug(t *testing.T) {
	svc, pool, ctx := newTestProductService(t)
	p, err := svc.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	sameName := "Buket"
	_, err = svc.Update(ctx, p.ID, UpdateInput{Name: &sameName})
	require.NoError(t, err)

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM product_slugs WHERE product_id = $1`, p.ID).Scan(&count))
	assert.Equal(t, 1, count, "aynı isim yeni slug üretmemeli")
}

func TestProductService_Update_PriceOnlyDoesNotTouchSlug(t *testing.T) {
	svc, pool, ctx := newTestProductService(t)
	p, err := svc.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	newPrice := price(t, "600")
	updated, err := svc.Update(ctx, p.ID, UpdateInput{Price: &newPrice})

	require.NoError(t, err)
	assert.Equal(t, "buket", updated.Slug)
	assert.Equal(t, "600.00", updated.Price.StringFixed(2))

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM product_slugs WHERE product_id = $1`, p.ID).Scan(&count))
	assert.Equal(t, 1, count)
}

// Eski isme geri dönülürse: eski slug rezerve olduğu için -2 eki alır.
func TestProductService_Update_RevertToOldNameGetsSuffix(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	p, err := svc.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	changed := "Gül Buketi"
	_, err = svc.Update(ctx, p.ID, UpdateInput{Name: &changed})
	require.NoError(t, err)

	reverted := "Buket"
	updated, err := svc.Update(ctx, p.ID, UpdateInput{Name: &reverted})

	require.NoError(t, err)
	assert.Equal(t, "buket-2", updated.Slug, "eski slug rezerve, -2 almalı")
}

func TestProductService_GetPublicBySlug_CurrentNoRedirect(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	_, err := svc.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	p, redirectTo, err := svc.GetPublicBySlug(ctx, "buket")

	require.NoError(t, err)
	assert.Empty(t, redirectTo, "güncel slug redirect vermemeli")
	assert.Equal(t, "Buket", p.Name)
}

func TestProductService_GetPublicBySlug_NotFound(t *testing.T) {
	svc, _, ctx := newTestProductService(t)

	_, _, err := svc.GetPublicBySlug(ctx, "olmayan-urun")

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

// Pasif ürünün slug'ı çözülse bile public'te 404.
func TestProductService_GetPublicBySlug_InactiveIsNotFound(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	_, err := svc.Create(ctx, CreateInput{
		Name: "Pasif Ürün", Price: price(t, "500"), IsActive: false,
	})
	require.NoError(t, err)

	_, _, err = svc.GetPublicBySlug(ctx, "pasif-urun")

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestProductService_Delete(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	p, err := svc.Create(ctx, CreateInput{
		Name: "Silinecek", Price: price(t, "100"), IsActive: true,
	})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(ctx, p.ID))

	_, _, err = svc.GetPublicBySlug(ctx, "silinecek")
	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestProductService_ListPublic_DefaultLimit(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	_, err := svc.Create(ctx, CreateInput{
		Name: "A", Price: price(t, "100"), IsActive: true,
	})
	require.NoError(t, err)

	list, err := svc.ListPublic(ctx, Filter{})

	require.NoError(t, err)
	assert.Len(t, list, 1, "limit 0 verilse bile varsayılan uygulanmalı")
}

// --- öne çıkan ürünler (ana sayfa vitrini) ---

// Yeni ürün kendiliğinden ana sayfaya düşmemeli — esnaf bilinçli seçsin.
func TestProductService_Create_NotFeaturedByDefault(t *testing.T) {
	svc, _, ctx := newTestProductService(t)

	p, err := svc.Create(ctx, CreateInput{
		Name: "Yeni Buket", Price: price(t, "100"), IsActive: true,
	})

	require.NoError(t, err)
	assert.False(t, p.IsFeatured, "varsayılan false olmalı")
}

func TestProductService_ListPublic_FeaturedOnlyFilters(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	_, err := svc.Create(ctx, CreateInput{
		Name: "Vitrin Ürünü", Price: price(t, "100"), IsActive: true, IsFeatured: true,
	})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{
		Name: "Sıradan Ürün", Price: price(t, "200"), IsActive: true, IsFeatured: false,
	})
	require.NoError(t, err)

	vitrin, err := svc.ListPublic(ctx, Filter{FeaturedOnly: true})
	require.NoError(t, err)
	require.Len(t, vitrin, 1, "yalnızca öne çıkan gelmeli")
	assert.Equal(t, "Vitrin Ürünü", vitrin[0].Name)

	// Katalog sayfası öne çıkmaya bakmıyor — ikisi de gelmeli.
	hepsi, err := svc.ListPublic(ctx, Filter{})
	require.NoError(t, err)
	assert.Len(t, hepsi, 2)
}

// is_active=false her şeyi ezer: pasif ürün öne çıkarılmış olsa bile
// ana sayfada görünmemeli (kategorilerdeki kuralın aynısı).
func TestProductService_ListPublic_InactiveOverridesFeatured(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	_, err := svc.Create(ctx, CreateInput{
		Name: "Pasif Vitrin", Price: price(t, "100"), IsActive: false, IsFeatured: true,
	})
	require.NoError(t, err)

	list, err := svc.ListPublic(ctx, Filter{FeaturedOnly: true})

	require.NoError(t, err)
	assert.Empty(t, list, "pasif ürün öne çıkarılsa bile gelmemeli")
}

func TestProductService_Update_TogglesFeatured(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	p, err := svc.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "100"), IsActive: true,
	})
	require.NoError(t, err)

	yes := true
	updated, err := svc.Update(ctx, p.ID, UpdateInput{IsFeatured: &yes})

	require.NoError(t, err)
	assert.True(t, updated.IsFeatured)
	assert.Equal(t, "Buket", updated.Name, "dokunulmayan alan korunmalı")
}

// Öne çıkma ile eksen filtresi birlikte çalışmalı — ikisi de AND.
func TestProductService_ListPublic_FeaturedWithAxisFilter(t *testing.T) {
	svc, pool, ctx := newTestProductService(t)
	catID := insertCategory(t, pool, "Orkideler", "orkideler", "type")

	_, err := svc.Create(ctx, CreateInput{
		Name: "Öne Çıkan Orkide", Price: price(t, "100"), IsActive: true,
		IsFeatured: true, CategoryIDs: []int64{catID},
	})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{
		Name: "Öne Çıkan Kategorisiz", Price: price(t, "100"), IsActive: true,
		IsFeatured: true,
	})
	require.NoError(t, err)

	tip := "orkideler"
	list, err := svc.ListPublic(ctx, Filter{FeaturedOnly: true, TypeSlug: &tip})

	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Öne Çıkan Orkide", list[0].Name)
}
