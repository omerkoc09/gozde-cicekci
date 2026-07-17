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
	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/internal/slider"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-that-is-long-enough-32"

func newTestAdminAPI(t *testing.T) (*fiber.App, string) {
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
	}
	orderSvc := order.NewService(order.NewStore(pool), product.NewStore(pool), deliveryCfg)

	app := fiber.New()
	Register(app.Group("/api/admin"), Deps{
		AuthSvc:      authSvc,
		CatSvc:       category.NewService(category.NewStore(pool), imgSvc),
		ProdSvc:      prodSvc,
		ImgSvc:       imgSvc,
		SliderSvc:    slider.NewService(slider.NewStore(pool), imgSvc),
		OrderSvc:     orderSvc,
		JWTSecret:    testSecret,
		SecureCookie: false,
	})

	token, err := auth.GenerateToken(1, "cicekci", testSecret)
	require.NoError(t, err)
	return app, token
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
