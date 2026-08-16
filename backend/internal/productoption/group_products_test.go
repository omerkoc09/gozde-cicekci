package productoption

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestStoreWithPool ürün kaydı da gerektiğinde pool'a doğrudan erişim
// verir — bu paketin kendi Store'unda ürün oluşturma metodu yok.
func newTestStoreWithPool(t *testing.T) (*Store, *pgxpool.Pool, context.Context) {
	t.Helper()
	pool := database.NewTestDB(t)
	return NewStore(pool), pool, context.Background()
}

func urunEkle(t *testing.T, pool *pgxpool.Pool, ad string, aktif bool) int64 {
	t.Helper()
	var id int64
	require.NoError(t, pool.QueryRow(context.Background(),
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ($1, 'test', 100.00, $2) RETURNING id`, ad, aktif).Scan(&id))
	return id
}

func TestStore_ProductsUsingGroup_KullananUrunleriDoner(t *testing.T) {
	store, pool, ctx := newTestStoreWithPool(t)

	grup, err := store.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindColor}, "ambalaj")
	require.NoError(t, err)
	baskaGrup, err := store.CreateGroup(ctx, CreateGroupInput{Name: "Kurdele", Kind: KindColor}, "kurdele")
	require.NoError(t, err)

	kullanan1 := urunEkle(t, pool, "Buket A", true)
	kullanan2 := urunEkle(t, pool, "Buket B", true)
	kullanmayan := urunEkle(t, pool, "Buket C", true)

	require.NoError(t, store.SetProductGroups(ctx, kullanan1, []ProductGroupLink{{GroupID: grup.ID}}))
	require.NoError(t, store.SetProductGroups(ctx, kullanan2, []ProductGroupLink{{GroupID: grup.ID}}))
	// Bu ürün BAŞKA grubu kullanıyor — listede olmamalı.
	require.NoError(t, store.SetProductGroups(ctx, kullanmayan, []ProductGroupLink{{GroupID: baskaGrup.ID}}))

	list, err := store.ProductsUsingGroup(ctx, grup.ID)

	require.NoError(t, err)
	require.Len(t, list, 2, "yalnızca bu grubu kullanan ürünler")
	adlar := []string{list[0].Name, list[1].Name}
	assert.Contains(t, adlar, "Buket A")
	assert.Contains(t, adlar, "Buket B")
	assert.NotContains(t, adlar, "Buket C")
}

// Pasif ürünler de listelenir — esnaf "bu grubu silersem nereler etkilenir"
// sorusunun tam cevabını görmeli. Aktiflik bilgisi ayrı alanla geliyor.
func TestStore_ProductsUsingGroup_PasifUrunuDeIcerirAmaIsaretler(t *testing.T) {
	store, pool, ctx := newTestStoreWithPool(t)

	grup, err := store.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindColor}, "ambalaj")
	require.NoError(t, err)

	aktif := urunEkle(t, pool, "Aktif Buket", true)
	pasif := urunEkle(t, pool, "Pasif Buket", false)

	require.NoError(t, store.SetProductGroups(ctx, aktif, []ProductGroupLink{{GroupID: grup.ID}}))
	require.NoError(t, store.SetProductGroups(ctx, pasif, []ProductGroupLink{{GroupID: grup.ID}}))

	list, err := store.ProductsUsingGroup(ctx, grup.ID)

	require.NoError(t, err)
	require.Len(t, list, 2)

	durum := make(map[string]bool, 2)
	for _, p := range list {
		durum[p.Name] = p.IsActive
	}
	assert.True(t, durum["Aktif Buket"])
	assert.False(t, durum["Pasif Buket"])
}

func TestStore_ProductsUsingGroup_AdaGoreSirali(t *testing.T) {
	store, pool, ctx := newTestStoreWithPool(t)

	grup, err := store.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindColor}, "ambalaj")
	require.NoError(t, err)

	// Ekleme sırası alfabetik DEĞİL — sıralama gerçekten uygulanmalı.
	for _, ad := range []string{"Zambak", "Ay Çiçeği", "Menekşe"} {
		id := urunEkle(t, pool, ad, true)
		require.NoError(t, store.SetProductGroups(ctx, id, []ProductGroupLink{{GroupID: grup.ID}}))
	}

	list, err := store.ProductsUsingGroup(ctx, grup.ID)

	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, "Ay Çiçeği", list[0].Name)
	assert.Equal(t, "Menekşe", list[1].Name)
	assert.Equal(t, "Zambak", list[2].Name)
}

func TestStore_ProductsUsingGroup_KullanilmayanGrupBosDoner(t *testing.T) {
	store, _, ctx := newTestStoreWithPool(t)

	grup, err := store.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindColor}, "ambalaj")
	require.NoError(t, err)

	list, err := store.ProductsUsingGroup(ctx, grup.ID)

	require.NoError(t, err)
	assert.Empty(t, list, "nil değil boş slice — JSON'da null yerine [] çıksın")
	assert.NotNil(t, list)
}
