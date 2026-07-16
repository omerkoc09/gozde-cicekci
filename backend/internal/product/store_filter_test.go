package product

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Spec §5.6: iki eksen de seçiliyse AND — her iki koşula da uyan ürünler.
// AND/OR karışması bu tür sorgularda klasik bir hatadır (spec §6).
func TestStore_ListPublic_TwoAxisFilterIsAND(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	dogumGunu := insertCategory(t, pool, "Doğum Günü", "dogum-gunu", "occasion")
	taziye := insertCategory(t, pool, "Taziye", "taziye", "occasion")
	buket := insertCategory(t, pool, "Buket", "buket", "type")
	orkide := insertCategory(t, pool, "Orkide", "orkide", "type")

	// Doğum Günü + Buket — eşleşmeli
	match, err := store.Create(ctx, CreateInput{
		Name: "Doğum Günü Buketi", Price: price(t, "500"), IsActive: true,
		CategoryIDs: []int64{dogumGunu, buket},
	}, "dogum-gunu-buketi")
	require.NoError(t, err)

	// Doğum Günü + Orkide — tip uymuyor
	_, err = store.Create(ctx, CreateInput{
		Name: "Doğum Günü Orkidesi", Price: price(t, "800"), IsActive: true,
		CategoryIDs: []int64{dogumGunu, orkide},
	}, "dogum-gunu-orkidesi")
	require.NoError(t, err)

	// Taziye + Buket — amaç uymuyor
	_, err = store.Create(ctx, CreateInput{
		Name: "Taziye Buketi", Price: price(t, "600"), IsActive: true,
		CategoryIDs: []int64{taziye, buket},
	}, "taziye-buketi")
	require.NoError(t, err)

	occasion, typ := "dogum-gunu", "buket"
	list, err := store.ListPublic(ctx, Filter{
		OccasionSlug: &occasion, TypeSlug: &typ, Limit: 20,
	})

	require.NoError(t, err)
	require.Len(t, list, 1, "sadece iki koşula da uyan ürün gelmeli (AND, OR değil)")
	assert.Equal(t, match.ID, list[0].ID)
}

func TestStore_ListPublic_SingleAxisFilter(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	dogumGunu := insertCategory(t, pool, "Doğum Günü", "dogum-gunu", "occasion")
	buket := insertCategory(t, pool, "Buket", "buket", "type")
	orkide := insertCategory(t, pool, "Orkide", "orkide", "type")

	_, err := store.Create(ctx, CreateInput{
		Name: "A", Price: price(t, "100"), IsActive: true,
		CategoryIDs: []int64{dogumGunu, buket},
	}, "a")
	require.NoError(t, err)
	_, err = store.Create(ctx, CreateInput{
		Name: "B", Price: price(t, "100"), IsActive: true,
		CategoryIDs: []int64{dogumGunu, orkide},
	}, "b")
	require.NoError(t, err)

	occasion := "dogum-gunu"
	list, err := store.ListPublic(ctx, Filter{OccasionSlug: &occasion, Limit: 20})

	require.NoError(t, err)
	assert.Len(t, list, 2, "tek eksen filtresi ikisini de getirmeli")
}

func TestStore_ListPublic_NoFilter(t *testing.T) {
	store, _, ctx := newTestStore(t)
	_, err := store.Create(ctx, CreateInput{
		Name: "A", Price: price(t, "100"), IsActive: true,
	}, "a")
	require.NoError(t, err)
	_, err = store.Create(ctx, CreateInput{
		Name: "B", Price: price(t, "100"), IsActive: true,
	}, "b")
	require.NoError(t, err)

	list, err := store.ListPublic(ctx, Filter{Limit: 20})

	require.NoError(t, err)
	assert.Len(t, list, 2)
}

// Spec §6: is_active sızıntısı regresyon testi.
func TestStore_ListPublic_HidesInactive(t *testing.T) {
	store, _, ctx := newTestStore(t)
	_, err := store.Create(ctx, CreateInput{
		Name: "Aktif", Price: price(t, "100"), IsActive: true,
	}, "aktif")
	require.NoError(t, err)
	_, err = store.Create(ctx, CreateInput{
		Name: "Pasif", Price: price(t, "100"), IsActive: false,
	}, "pasif")
	require.NoError(t, err)

	list, err := store.ListPublic(ctx, Filter{Limit: 20})

	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Aktif", list[0].Name)
}

// Pasif kategoriye bağlı ürün, o kategori filtresiyle gelmemeli.
func TestStore_ListPublic_InactiveCategoryNotFilterable(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	var catID int64
	err := pool.QueryRow(ctx,
		`INSERT INTO categories (name, slug, axis, is_active)
		 VALUES ('Anneler Günü', 'anneler-gunu', 'occasion', false) RETURNING id`,
	).Scan(&catID)
	require.NoError(t, err)

	_, err = store.Create(ctx, CreateInput{
		Name: "Anneler Günü Buketi", Price: price(t, "500"), IsActive: true,
		CategoryIDs: []int64{catID},
	}, "anneler-gunu-buketi")
	require.NoError(t, err)

	slug := "anneler-gunu"
	list, err := store.ListPublic(ctx, Filter{OccasionSlug: &slug, Limit: 20})

	require.NoError(t, err)
	assert.Empty(t, list, "pasif kategori filtresi sonuç döndürmemeli")
}

func TestStore_ListPublic_Pagination(t *testing.T) {
	store, _, ctx := newTestStore(t)
	for _, name := range []string{"A", "B", "C"} {
		_, err := store.Create(ctx, CreateInput{
			Name: name, Price: price(t, "100"), IsActive: true,
		}, Slugify(name))
		require.NoError(t, err)
	}

	first, err := store.ListPublic(ctx, Filter{Limit: 2, Offset: 0})
	require.NoError(t, err)
	second, err := store.ListPublic(ctx, Filter{Limit: 2, Offset: 2})
	require.NoError(t, err)

	assert.Len(t, first, 2)
	assert.Len(t, second, 1)
}

func TestStore_ListAdmin_ShowsInactive(t *testing.T) {
	store, _, ctx := newTestStore(t)
	_, err := store.Create(ctx, CreateInput{
		Name: "Aktif", Price: price(t, "100"), IsActive: true,
	}, "aktif")
	require.NoError(t, err)
	_, err = store.Create(ctx, CreateInput{
		Name: "Pasif", Price: price(t, "100"), IsActive: false,
	}, "pasif")
	require.NoError(t, err)

	list, err := store.ListAdmin(ctx, 20, 0)

	require.NoError(t, err)
	assert.Len(t, list, 2)
}
