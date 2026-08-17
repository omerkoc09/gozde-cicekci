package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/customer"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/internal/payment"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/internal/productoption"
	"github.com/omerkoc/cicekci/internal/slider"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testJWTSecret app paketi testlerinde kullanılan sabit imzalama anahtarı.
const testJWTSecret = "test-secret-app-paketi"

func newTestAPI(t *testing.T) (*fiber.App, *product.Service, *category.Service) {
	t.Helper()
	app, _, prodSvc, catSvc, _, _ := newTestAPIFull(t)
	return app, prodSvc, catSvc
}

// newTestAPIFull tam kurulum — müşteri uçlarını da test edebilmek için
// order.Service, customer.Service ve pool'u da döner (test verisi
// hazırlamak için doğrudan SQL gerekebiliyor — örn. products insert).
func newTestAPIFull(t *testing.T) (fiberApp *fiber.App, orderSvc *order.Service, prodSvc *product.Service, catSvc *category.Service, custSvc *customer.Service, pool *pgxpool.Pool) {
	t.Helper()
	pool = database.NewTestDB(t)
	prodSvc = product.NewService(product.NewStore(pool))

	imgStore, err := image.NewLocalStore(t.TempDir(), "http://localhost:8080/uploads")
	require.NoError(t, err)
	imgSvc := image.NewService(imgStore, image.NewDB(pool))

	catSvc = category.NewService(category.NewStore(pool), imgSvc)
	sliderSvc := slider.NewService(slider.NewStore(pool), imgSvc)

	deliveryCfg := order.DeliveryConfig{
		Fee: "50", Slots: []string{"09:00-12:00", "12:00-15:00", "15:00-18:00"},
		SameDayCutoff: "16:00", MaxDays: 30,
		Districts: []string{"Ödemiş", "Tire"},
	}
	optSvc := productoption.NewService(productoption.NewStore(pool))
	orderSvc = order.NewService(order.NewStore(pool), product.NewStore(pool), optSvc,
		product.NewStore(pool), deliveryCfg,
		payment.NewMockProvider(), "https://example.com/ok", "https://example.com/fail")

	custSvc = customer.NewService(customer.NewStore(pool), testJWTSecret)

	fiberApp = fiber.New()
	Register(fiberApp.Group("/api"), catSvc, prodSvc, imgSvc, sliderSvc, orderSvc, deliveryCfg,
		custSvc, optSvc, testJWTSecret, false)
	return fiberApp, orderSvc, prodSvc, catSvc, custSvc, pool
}

func mustPrice(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	require.NoError(t, err)
	return d
}

func TestProductHandler_List(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	_, err := svc.Create(context.Background(), product.CreateInput{
		Name: "51 Gül Buket", Price: mustPrice(t, "1850"), IsActive: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products", nil))
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var views []ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	require.Len(t, views, 1)
	assert.Equal(t, "51 Gül Buket", views[0].Name)
	assert.Equal(t, "1850.00", views[0].Price)
}

// Spec §4.6: public uçlar pasif ürünü hiç görmez.
func TestProductHandler_List_HidesInactive(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	_, err := svc.Create(context.Background(), product.CreateInput{
		Name: "Pasif", Price: mustPrice(t, "100"), IsActive: false,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products", nil))
	require.NoError(t, err)

	var views []ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	assert.Empty(t, views)
}

// Public viewmodel'de is_active alanı hiç olmamalı.
func TestProductHandler_List_ViewHasNoIsActiveField(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	_, err := svc.Create(context.Background(), product.CreateInput{
		Name: "Aktif", Price: mustPrice(t, "100"), IsActive: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products", nil))
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.NotContains(t, string(body), "is_active")
}

func TestProductHandler_GetBySlug(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	_, err := svc.Create(context.Background(), product.CreateInput{
		Name: "Buket", Price: mustPrice(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products/buket", nil))
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	assert.Equal(t, "Buket", view.Name)
}

// Spec §4.2: eski slug 301 ile güncel URL'e yönlendirir.
func TestProductHandler_GetBySlug_OldSlugRedirects(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	ctx := context.Background()
	p, err := svc.Create(ctx, product.CreateInput{
		Name: "51 Gül Buket", Price: mustPrice(t, "1850"), IsActive: true,
	})
	require.NoError(t, err)
	newName := "51 Kırmızı Gül Buketi"
	_, err = svc.Update(ctx, p.ID, product.UpdateInput{Name: &newName})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products/51-gul-buket", nil))
	require.NoError(t, err)

	assert.Equal(t, http.StatusMovedPermanently, resp.StatusCode)
	assert.Equal(t, "/api/products/51-kirmizi-gul-buketi", resp.Header.Get("Location"))
}

func TestProductHandler_GetBySlug_NotFound(t *testing.T) {
	app, _, _ := newTestAPI(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products/yok", nil))
	require.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCategoryHandler_Featured_RouteNotShadowed(t *testing.T) {
	app, _, catSvc := newTestAPI(t)
	_, err := catSvc.Create(context.Background(), category.CreateInput{
		Name: "Doğum Günü", Axis: category.AxisOccasion, IsActive: true, IsFeatured: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/categories/featured", nil))
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var views []CategoryView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	require.Len(t, views, 1)
	assert.Equal(t, "Doğum Günü", views[0].Name)
}

// Public görsel gösteriminde iç detay sızmamalı — image_key, id, sort_order yok.
func TestProductHandler_ImageViewHasNoInternalFields(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	_, err := svc.Create(context.Background(), product.CreateInput{
		Name: "Buket", Price: mustPrice(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products/buket", nil))
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.NotContains(t, string(body), "image_key")
	assert.NotContains(t, string(body), "sort_order")
}

// Görseli olmayan ürün boş dizi döner, null değil — frontend'de v-for patlamasın.
func TestProductHandler_NoImagesReturnsEmptyArray(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	_, err := svc.Create(context.Background(), product.CreateInput{
		Name: "Görselsiz", Price: mustPrice(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products/gorselsiz", nil))
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), `"images":[]`)
	assert.NotContains(t, string(body), `"images":null`)
}

// Liste ucunda da görseller boş dizi olmalı.
func TestProductHandler_ListImagesIsEmptyArrayNotNull(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	_, err := svc.Create(context.Background(), product.CreateInput{
		Name: "Görselsiz", Price: mustPrice(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products", nil))
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), `"images":[]`)
	assert.NotContains(t, string(body), `"images":null`)
}

// Spec: public uç pasif GRUBU hiç göstermez, ama admin (GroupsForProduct
// onlyActive=false — idare handler'ının kullandığı çağrının aynısı) hâlâ
// görür. Pasif grubun kendisi aktif olsaydı bile görünecek aktif bir
// değeri var; filtrelenen tek şey grubun is_active=false olması.
func TestProductHandler_GetBySlug_PasifGrupGizlenir(t *testing.T) {
	app, _, prodSvc, _, _, pool := newTestAPIFull(t)
	optSvc := productoption.NewService(productoption.NewStore(pool))
	ctx := context.Background()

	p, err := prodSvc.Create(ctx, product.CreateInput{
		Name: "Pasif Grup Testi", Price: mustPrice(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	g, err := optSvc.CreateGroup(ctx, productoption.CreateGroupInput{Name: "Ambalaj", Kind: productoption.KindColor})
	require.NoError(t, err)
	_, err = optSvc.CreateValue(ctx, productoption.CreateValueInput{GroupID: g.ID, Name: "Pembe", SwatchHex: "#F0A6CA"})
	require.NoError(t, err)
	require.NoError(t, optSvc.SetProductGroups(ctx, p.ID, []productoption.ProductGroupLink{{GroupID: g.ID}}))

	// Grubun kendisini pasifle.
	inactive := false
	_, err = optSvc.UpdateGroup(ctx, g.ID, productoption.UpdateGroupInput{IsActive: &inactive})
	require.NoError(t, err)

	// Admin görüşü (idare handler'ının kullandığı aynı çağrı, onlyActive=false):
	// pasif grup hâlâ dönmeli.
	adminGroups, err := optSvc.GroupsForProduct(ctx, p.ID, false)
	require.NoError(t, err)
	require.Len(t, adminGroups, 1, "admin pasif grubu da görmeli")
	assert.False(t, adminGroups[0].IsActive)

	// Public uç: pasif grup hiç görünmemeli.
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products/pasif-grup-testi", nil))
	require.NoError(t, err)
	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	assert.Empty(t, view.OptionGroups, "pasif grup public yanıtta görünmemeli")
}

// Spec: aynı grubun aktif değerleri public'te görünürken pasif değerleri
// görünmez.
func TestProductHandler_GetBySlug_PasifDegerGizlenirAktifKalir(t *testing.T) {
	app, _, prodSvc, _, _, pool := newTestAPIFull(t)
	optSvc := productoption.NewService(productoption.NewStore(pool))
	ctx := context.Background()

	p, err := prodSvc.Create(ctx, product.CreateInput{
		Name: "Karma Deger Testi", Price: mustPrice(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	g, err := optSvc.CreateGroup(ctx, productoption.CreateGroupInput{Name: "Kurdele", Kind: productoption.KindColor})
	require.NoError(t, err)
	vAktif, err := optSvc.CreateValue(ctx, productoption.CreateValueInput{GroupID: g.ID, Name: "Kırmızı", SwatchHex: "#FF0000"})
	require.NoError(t, err)
	vPasif, err := optSvc.CreateValue(ctx, productoption.CreateValueInput{GroupID: g.ID, Name: "Mavi", SwatchHex: "#0000FF"})
	require.NoError(t, err)
	require.NoError(t, optSvc.SetProductGroups(ctx, p.ID, []productoption.ProductGroupLink{{GroupID: g.ID}}))

	inactive := false
	_, err = optSvc.UpdateValue(ctx, vPasif.ID, productoption.UpdateValueInput{IsActive: &inactive})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products/karma-deger-testi", nil))
	require.NoError(t, err)
	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))

	require.Len(t, view.OptionGroups, 1, "grup hâlâ görünmeli (en az bir aktif değeri var)")
	require.Len(t, view.OptionGroups[0].Values, 1, "yalnızca aktif değer görünmeli")
	assert.Equal(t, vAktif.ID, view.OptionGroups[0].Values[0].ID)
	assert.Equal(t, "Kırmızı", view.OptionGroups[0].Values[0].Name)
}

// Spec: bir grubun TÜM değerleri pasifse (len(g.Values)==0 filtresi
// devreye girer), grup public yanıtta hiç görünmemeli — seçenek sunmayan
// bir başlık müşteriyi kafa karıştırır.
func TestProductHandler_GetBySlug_TumDegerleriPasifGrupHicGorunmez(t *testing.T) {
	app, _, prodSvc, _, _, pool := newTestAPIFull(t)
	optSvc := productoption.NewService(productoption.NewStore(pool))
	ctx := context.Background()

	p, err := prodSvc.Create(ctx, product.CreateInput{
		Name: "Tum Degerler Pasif", Price: mustPrice(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	g, err := optSvc.CreateGroup(ctx, productoption.CreateGroupInput{Name: "Boyut", Kind: productoption.KindText})
	require.NoError(t, err)
	v, err := optSvc.CreateValue(ctx, productoption.CreateValueInput{GroupID: g.ID, Name: "Büyük"})
	require.NoError(t, err)
	require.NoError(t, optSvc.SetProductGroups(ctx, p.ID, []productoption.ProductGroupLink{{GroupID: g.ID}}))

	inactive := false
	_, err = optSvc.UpdateValue(ctx, v.ID, productoption.UpdateValueInput{IsActive: &inactive})
	require.NoError(t, err)

	// Grubun kendisi hâlâ aktif — yalnızca tek değeri pasif.
	adminGroups, err := optSvc.GroupsForProduct(ctx, p.ID, false)
	require.NoError(t, err)
	require.Len(t, adminGroups, 1, "admin grubu hâlâ görmeli (değerleri pasif de olsa)")
	assert.True(t, adminGroups[0].IsActive, "grubun kendisi aktif kalmalı, yalnız değeri pasif")

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products/tum-degerler-pasif", nil))
	require.NoError(t, err)
	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	assert.Empty(t, view.OptionGroups, "değeri kalmamış grup public yanıtta hiç görünmemeli")
}

// --- Stok ve indirim gösterimi (spec §6.1, §7) ---

// setStockAndDiscount test ürününe stok/indirim alanlarını doğrudan yazar.
func setStockAndDiscount(t *testing.T, pool *pgxpool.Pool, id int64, sql string, args ...any) {
	t.Helper()
	all := append([]any{id}, args...)
	_, err := pool.Exec(context.Background(), sql, all...)
	require.NoError(t, err)
}

func TestProductHandler_List_IndirimliUrun_EskiVeYeniFiyat(t *testing.T) {
	app, _, prodSvc, _, _, pool := newTestAPIFull(t)
	p, err := prodSvc.Create(context.Background(), product.CreateInput{
		Name: "Gül Buketi", Price: mustPrice(t, "1850"), IsActive: true,
	})
	require.NoError(t, err)
	setStockAndDiscount(t, pool, p.ID,
		`UPDATE products SET discount_price=1450.00, discount_quota=10, discount_sold=3
		 WHERE id=$1`)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products", nil))
	require.NoError(t, err)

	var views []ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	require.Len(t, views, 1)
	assert.Equal(t, "1450.00", views[0].Price, "price indirimli fiyat olmalı")
	require.NotNil(t, views[0].OldPrice)
	assert.Equal(t, "1850.00", *views[0].OldPrice, "eski fiyat da gösterilmeli")
	require.NotNil(t, views[0].DiscountRemaining)
	assert.Equal(t, 7, *views[0].DiscountRemaining)
	assert.True(t, views[0].InStock)
}

func TestProductHandler_List_KotasiDolmus_EskiFiyatGosterilmez(t *testing.T) {
	app, _, prodSvc, _, _, pool := newTestAPIFull(t)
	p, err := prodSvc.Create(context.Background(), product.CreateInput{
		Name: "Kotası Dolmuş", Price: mustPrice(t, "1850"), IsActive: true,
	})
	require.NoError(t, err)
	setStockAndDiscount(t, pool, p.ID,
		`UPDATE products SET discount_price=1450.00, discount_quota=5, discount_sold=5
		 WHERE id=$1`)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products", nil))
	require.NoError(t, err)

	var views []ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	require.Len(t, views, 1)
	assert.Equal(t, "1850.00", views[0].Price, "kota dolunca normal fiyata dönmeli")
	assert.Nil(t, views[0].OldPrice)
	assert.Nil(t, views[0].DiscountRemaining)
}

func TestProductHandler_List_TukenenUrun_ListedeKalir(t *testing.T) {
	app, _, prodSvc, _, _, pool := newTestAPIFull(t)
	p, err := prodSvc.Create(context.Background(), product.CreateInput{
		Name: "Tükenen", Price: mustPrice(t, "500"), IsActive: true,
	})
	require.NoError(t, err)
	setStockAndDiscount(t, pool, p.ID,
		`UPDATE products SET track_stock=true, stock_quantity=2, stock_reserved=2
		 WHERE id=$1`)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products", nil))
	require.NoError(t, err)

	var views []ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	require.Len(t, views, 1, "tükenen ürün listeden GİZLENMEZ (spec §6.1)")
	assert.False(t, views[0].InStock)
	require.NotNil(t, views[0].StockQuantity)
	assert.Equal(t, 0, *views[0].StockQuantity)
}

func TestProductHandler_List_TakipsizUrun_StokNull(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	_, err := svc.Create(context.Background(), product.CreateInput{
		Name: "Takipsiz", Price: mustPrice(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products", nil))
	require.NoError(t, err)

	var views []ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	require.Len(t, views, 1)
	assert.True(t, views[0].InStock, "takipsiz ürün her zaman satılabilir")
	assert.Nil(t, views[0].StockQuantity, "takipsiz üründe adet null gitmeli")
}

func TestProductHandler_List_IndirimliFiltresi(t *testing.T) {
	app, _, prodSvc, _, _, pool := newTestAPIFull(t)
	indirimli, err := prodSvc.Create(context.Background(), product.CreateInput{
		Name: "İndirimli", Price: mustPrice(t, "1850"), IsActive: true,
	})
	require.NoError(t, err)
	setStockAndDiscount(t, pool, indirimli.ID,
		`UPDATE products SET discount_price=1450.00, discount_quota=10, discount_sold=0
		 WHERE id=$1`)

	_, err = prodSvc.Create(context.Background(), product.CreateInput{
		Name: "Normal", Price: mustPrice(t, "700"), IsActive: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products?indirimli=true", nil))
	require.NoError(t, err)

	var views []ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	require.Len(t, views, 1, "yalnızca indirimi aktif ürün dönmeli")
	assert.Equal(t, "İndirimli", views[0].Name)
}
