package idare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/internal/payment"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/internal/productoption"
	"github.com/omerkoc/cicekci/internal/slider"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-that-is-long-enough-32"

func newTestAdminAPI(t *testing.T) (*fiber.App, string) {
	t.Helper()
	app, token, _ := newTestAdminAPIWithPool(t)
	return app, token
}

// newTestAdminAPIWithPool newTestAdminAPI ile aynı, ama pool'u da döner —
// stok/indirim alanlarını doğrudan SQL ile hazırlamak gerekiyor.
func newTestAdminAPIWithPool(t *testing.T) (*fiber.App, string, *pgxpool.Pool) {
	t.Helper()
	pool := database.NewTestDB(t)

	authSvc := auth.NewService(auth.NewStore(pool), testSecret)
	require.NoError(t, authSvc.CreateAdmin(context.Background(), "cicekci", "test-sifre-123"))

	imgStore, err := image.NewLocalStore(t.TempDir(), "http://localhost:8080/uploads")
	require.NoError(t, err)
	imgSvc := image.NewService(imgStore, image.NewDB(pool))

	prodSvc := product.NewService(product.NewStore(pool))
	deliveryCfg := order.DeliveryConfig{
		Fee: "50", Slots: []string{"09:00-12:00", "12:00-15:00", "15:00-18:00"},
		SameDayCutoff: "16:00", MaxDays: 30,
		Districts: []string{"Ödemiş", "Tire"},
	}
	optSvc := productoption.NewService(productoption.NewStore(pool))
	orderSvc := order.NewService(order.NewStore(pool), product.NewStore(pool), optSvc,
		product.NewStore(pool), deliveryCfg,
		payment.NewMockProvider(), "https://example.com/ok", "https://example.com/fail")

	app := fiber.New()
	Register(app.Group("/api/admin"), Deps{
		AuthSvc:      authSvc,
		CatSvc:       category.NewService(category.NewStore(pool), imgSvc),
		ProdSvc:      prodSvc,
		ImgSvc:       imgSvc,
		SliderSvc:    slider.NewService(slider.NewStore(pool), imgSvc),
		OrderSvc:     orderSvc,
		OptSvc:       productoption.NewService(productoption.NewStore(pool)),
		JWTSecret:    testSecret,
		SecureCookie: false,
	})

	token, err := auth.GenerateToken(1, "cicekci", testSecret)
	require.NoError(t, err)
	return app, token, pool
}

func authedRequest(method, path, body, token string) *http.Request {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	return req
}

func TestAdmin_ProductsRequireAuth(t *testing.T) {
	app, _ := newTestAdminAPI(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/admin/products", nil))

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAdmin_Login_SetsHttpOnlyCookie(t *testing.T) {
	app, _ := newTestAdminAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/login",
		strings.NewReader(`{"username":"cicekci","password":"test-sifre-123"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	cookies := resp.Cookies()
	require.NotEmpty(t, cookies)
	var tokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.CookieName {
			tokenCookie = c
		}
	}
	require.NotNil(t, tokenCookie, "token cookie'si set edilmeli")
	assert.True(t, tokenCookie.HttpOnly, "cookie HttpOnly olmalı (spec §4.5)")
	assert.NotEmpty(t, tokenCookie.Value)
}

func TestAdmin_Login_WrongPassword(t *testing.T) {
	app, _ := newTestAdminAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/login",
		strings.NewReader(`{"username":"cicekci","password":"yanlis"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAdmin_CreateProduct(t *testing.T) {
	app, token := newTestAdminAPI(t)

	resp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"51 Gül Buket","description":"Kırmızı","price":"1850.00"}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	assert.Equal(t, "51-gul-buket", view.Slug)
	assert.Equal(t, "1850.00", view.Price)
	assert.True(t, view.IsActive, "varsayılan aktif olmalı")
}

func TestAdmin_CreateProduct_InvalidPrice(t *testing.T) {
	app, token := newTestAdminAPI(t)

	resp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"Test","price":"bes-yuz-lira"}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// Admin listesi pasif ürünleri de görür (spec §4.6).
func TestAdmin_ListProducts_ShowsInactive(t *testing.T) {
	app, token := newTestAdminAPI(t)
	_, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"Pasif","price":"100","is_active":false}`, token))
	require.NoError(t, err)

	resp, err := app.Test(authedRequest(http.MethodGet, "/api/admin/products", "", token))

	require.NoError(t, err)
	var views []ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	require.Len(t, views, 1)
	assert.False(t, views[0].IsActive)
}

func TestAdmin_CategoryProductCount(t *testing.T) {
	app, token := newTestAdminAPI(t)

	catResp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/categories",
		`{"name":"Buket","axis":"type"}`, token))
	require.NoError(t, err)
	var cat CategoryView
	require.NoError(t, json.NewDecoder(catResp.Body).Decode(&cat))

	_, err = app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"Test","price":"100","category_ids":[`+itoa(cat.ID)+`]}`, token))
	require.NoError(t, err)

	resp, err := app.Test(authedRequest(http.MethodGet,
		"/api/admin/categories/"+itoa(cat.ID)+"/product-count", "", token))

	require.NoError(t, err)
	var body struct {
		ProductCount int `json:"product_count"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, 1, body.ProductCount)
}

func itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}

// Olmayan kategori için product-count 404 dönmeli — count(*) aggregate
// olduğu için store tek başına ayırt edemez, service katmanı doğruluyor.
func TestAdmin_ProductCount_NotFound(t *testing.T) {
	app, token := newTestAdminAPI(t)

	resp, err := app.Test(authedRequest(http.MethodGet,
		"/api/admin/categories/9999/product-count", "", token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// PATCH ile sadece name gönderilince description ve price aynen kalmalı
// (pointer/nil PATCH semantiği — spec, store seviyesinde zaten test edilmişti,
// burada HTTP seviyesinde de kanıtlanıyor).
func TestAdmin_UpdateProduct_PartialOnlyChangesGivenFields(t *testing.T) {
	app, token := newTestAdminAPI(t)

	createResp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"Eski Ad","description":"Eski açıklama","price":"250.00"}`, token))
	require.NoError(t, err)
	var created ProductView
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	resp, err := app.Test(authedRequest(http.MethodPatch,
		"/api/admin/products/"+itoa(created.ID), `{"name":"Yeni Ad"}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var updated ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&updated))
	assert.Equal(t, "Yeni Ad", updated.Name)
	assert.Equal(t, "Eski açıklama", updated.Description, "description değişmemeli")
	assert.Equal(t, "250.00", updated.Price, "price değişmemeli")
}

// PATCH ile category_ids:[] gönderilince tüm kategoriler kaldırılmalı —
// JSON'da [] boş slice'a çözümlenir (nil değil), bu da "hepsini kaldır" demektir.
func TestAdmin_UpdateProduct_EmptyCategoryIDsRemovesAll(t *testing.T) {
	app, token := newTestAdminAPI(t)

	catResp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/categories",
		`{"name":"Buket","axis":"type"}`, token))
	require.NoError(t, err)
	var cat CategoryView
	require.NoError(t, json.NewDecoder(catResp.Body).Decode(&cat))

	createResp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"Test","price":"100","category_ids":[`+itoa(cat.ID)+`]}`, token))
	require.NoError(t, err)
	var created ProductView
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	require.Len(t, created.CategoryIDs, 1, "ürün kategoriyle oluşmalı")

	resp, err := app.Test(authedRequest(http.MethodPatch,
		"/api/admin/products/"+itoa(created.ID), `{"category_ids":[]}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var updated ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&updated))
	assert.Empty(t, updated.CategoryIDs, "category_ids boş olmalı")
}

// createTestProduct verilen body ile POST /api/admin/products çağırır ve
// oluşan ürünün ID'sini döner.
func createTestProduct(t *testing.T, app *fiber.App, token string, body map[string]any) int64 {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	resp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products", string(raw), token))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	return view.ID
}

// getProduct GET /api/admin/products/:id çağırır ve view'ı döner.
func getProduct(t *testing.T, app *fiber.App, token string, id int64) ProductView {
	t.Helper()
	resp, err := app.Test(authedRequest(http.MethodGet, "/api/admin/products/"+itoa(id), "", token))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	return view
}

// option_groups nil gönderilirse mevcut bağlar KORUNUR (PATCH semantiği).
// Boş dizi gönderilirse hepsi kaldırılır.
func TestProduct_OptionGroups_PatchSemantigi(t *testing.T) {
	app, token := newTestAdminAPI(t)

	gid := createTestGroup(t, app, token, "Ambalaj", "color")
	pid := createTestProduct(t, app, token, map[string]any{
		"name":  "Buket",
		"price": "100.00",
		"option_groups": []map[string]any{
			{"group_id": gid},
		},
	})

	// option_groups GÖNDERİLMEDEN isim güncelle → bağ korunmalı
	resp, err := app.Test(authedRequest(http.MethodPatch,
		"/api/admin/products/"+itoa(pid), `{"name":"Yeni Buket"}`, token))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	urun := getProduct(t, app, token, pid)
	require.Len(t, urun.OptionGroups, 1, "option_groups gönderilmediyse bağ korunmalı")
	assert.Equal(t, gid, urun.OptionGroups[0].ID, "korunan bağ aynı grup olmalı")

	// Boş dizi → hepsi kalkmalı
	resp2, err := app.Test(authedRequest(http.MethodPatch,
		"/api/admin/products/"+itoa(pid), `{"option_groups":[]}`, token))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	urun2 := getProduct(t, app, token, pid)
	assert.Empty(t, urun2.OptionGroups, "boş dizi tüm bağları kaldırmalı")
}

// --- Stok yönetimi uçları (spec §5, §7) ---

func createProductWithStock(t *testing.T, app *fiber.App, token string,
	pool *pgxpool.Pool, name string, qty int) int64 {
	t.Helper()
	resp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"`+name+`","price":"1850.00"}`, token))
	require.NoError(t, err)
	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))

	_, err = pool.Exec(context.Background(),
		`UPDATE products SET track_stock=true, stock_quantity=$2 WHERE id=$1`,
		view.ID, qty)
	require.NoError(t, err)
	return view.ID
}

func TestAdmin_AdjustStock_WhatsAppSatisi(t *testing.T) {
	app, token, pool := newTestAdminAPIWithPool(t)
	id := createProductWithStock(t, app, token, pool, "Gül Buketi", 12)
	_, err := pool.Exec(context.Background(),
		`UPDATE products SET discount_price=1450.00, discount_quota=10 WHERE id=$1`, id)
	require.NoError(t, err)

	resp, err := app.Test(authedRequest(http.MethodPost,
		"/api/admin/products/"+itoa(id)+"/stock",
		`{"delta":-1,"reason":"whatsapp_satisi","was_discounted":true}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	assert.Equal(t, 11, view.StockQuantity)
	assert.Equal(t, 1, view.DiscountSold, "indirimli WhatsApp satışı kotayı tüketir")
}

func TestAdmin_AdjustStock_GecersizSebep(t *testing.T) {
	app, token, pool := newTestAdminAPIWithPool(t)
	id := createProductWithStock(t, app, token, pool, "Orkide", 5)

	resp, err := app.Test(authedRequest(http.MethodPost,
		"/api/admin/products/"+itoa(id)+"/stock",
		`{"delta":-1,"reason":"hatali_sebep"}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAdmin_AdjustStock_StokAltinda(t *testing.T) {
	app, token, pool := newTestAdminAPIWithPool(t)
	id := createProductWithStock(t, app, token, pool, "Lilyum", 2)

	resp, err := app.Test(authedRequest(http.MethodPost,
		"/api/admin/products/"+itoa(id)+"/stock",
		`{"delta":-5,"reason":"whatsapp_satisi"}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAdmin_AdjustStock_AuthGerekli(t *testing.T) {
	app, token, pool := newTestAdminAPIWithPool(t)
	id := createProductWithStock(t, app, token, pool, "Papatya", 5)

	resp, err := app.Test(httptest.NewRequest(http.MethodPost,
		"/api/admin/products/"+itoa(id)+"/stock", strings.NewReader(`{"delta":-1}`)))

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAdmin_ListMovements(t *testing.T) {
	app, token, pool := newTestAdminAPIWithPool(t)
	id := createProductWithStock(t, app, token, pool, "Karanfil", 20)

	_, err := app.Test(authedRequest(http.MethodPost,
		"/api/admin/products/"+itoa(id)+"/stock",
		`{"delta":-2,"reason":"whatsapp_satisi"}`, token))
	require.NoError(t, err)
	_, err = app.Test(authedRequest(http.MethodPost,
		"/api/admin/products/"+itoa(id)+"/stock",
		`{"delta":10,"reason":"yeni_parti"}`, token))
	require.NoError(t, err)

	resp, err := app.Test(authedRequest(http.MethodGet,
		"/api/admin/products/"+itoa(id)+"/movements", "", token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var views []MovementView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	require.Len(t, views, 2)
	// Yeniden eskiye sıralı
	assert.Equal(t, "yeni_parti", views[0].Reason)
	assert.Equal(t, 10, views[0].Delta)
	assert.Equal(t, "whatsapp_satisi", views[1].Reason)
	assert.Equal(t, -2, views[1].Delta)
}

func TestAdmin_UpdateProduct_IndirimGirilir(t *testing.T) {
	app, token := newTestAdminAPI(t)
	createResp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"İndirimli Buket","price":"1850.00"}`, token))
	require.NoError(t, err)
	var created ProductView
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	resp, err := app.Test(authedRequest(http.MethodPatch,
		"/api/admin/products/"+itoa(created.ID),
		`{"discount_price":"1450.00","discount_quota":10}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	require.NotNil(t, view.DiscountPrice)
	assert.Equal(t, "1450.00", *view.DiscountPrice)
	require.NotNil(t, view.DiscountQuota)
	assert.Equal(t, 10, *view.DiscountQuota)
	assert.Equal(t, "1850.00", view.Price, "panelde normal fiyat gösterilir")
}

func TestAdmin_UpdateProduct_KotasizIndirimReddedilir(t *testing.T) {
	app, token := newTestAdminAPI(t)
	createResp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"Test","price":"1850.00"}`, token))
	require.NoError(t, err)
	var created ProductView
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	resp, err := app.Test(authedRequest(http.MethodPatch,
		"/api/admin/products/"+itoa(created.ID),
		`{"discount_price":"1450.00"}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
		"kotasız indirim süresiz indirimdir — reddedilmeli")
}

func TestAdmin_UpdateProduct_IndirimKaldirilir(t *testing.T) {
	app, token, pool := newTestAdminAPIWithPool(t)
	createResp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"Test","price":"1850.00"}`, token))
	require.NoError(t, err)
	var created ProductView
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	_, err = pool.Exec(context.Background(),
		`UPDATE products SET discount_price=1450.00, discount_quota=5, discount_sold=4
		 WHERE id=$1`, created.ID)
	require.NoError(t, err)

	resp, err := app.Test(authedRequest(http.MethodPatch,
		"/api/admin/products/"+itoa(created.ID), `{"clear_discount":true}`, token))

	require.NoError(t, err)
	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	assert.Nil(t, view.DiscountPrice)
	assert.Equal(t, 0, view.DiscountSold, "sayaç sıfırlanmalı (spec §5.2)")
}

func TestAdmin_UpdateProduct_StokTakibiAcilir(t *testing.T) {
	app, token := newTestAdminAPI(t)
	createResp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"Stoklu","price":"500.00"}`, token))
	require.NoError(t, err)
	var created ProductView
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	assert.False(t, created.TrackStock, "yeni ürün varsayılan takipsiz")

	resp, err := app.Test(authedRequest(http.MethodPatch,
		"/api/admin/products/"+itoa(created.ID),
		`{"track_stock":true,"stock_quantity":25}`, token))

	require.NoError(t, err)
	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	assert.True(t, view.TrackStock)
	assert.Equal(t, 25, view.StockQuantity)
}
