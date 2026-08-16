package idare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/internal/productoption"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestOptionAdminAPI seçenek admin uçlarını test edebilmek için app'i
// gerçek productoption.Service ile kurar — diğer handler_test.go
// dosyalarındaki newTestAdminAPI deseniyle aynı.
func newTestOptionAdminAPI(t *testing.T) (*fiber.App, string) {
	t.Helper()
	pool := database.NewTestDB(t)

	authSvc := auth.NewService(auth.NewStore(pool), testSecret)
	require.NoError(t, authSvc.CreateAdmin(context.Background(), "cicekci", "test-sifre-123"))

	optSvc := productoption.NewService(productoption.NewStore(pool))

	app := fiber.New()
	Register(app.Group("/api/admin"), Deps{
		AuthSvc:      authSvc,
		OptSvc:       optSvc,
		JWTSecret:    testSecret,
		SecureCookie: false,
	})

	token, err := auth.GenerateToken(1, "cicekci", testSecret)
	require.NoError(t, err)
	return app, token
}

// createTestGroup verilen ad/kind ile bir seçenek grubu oluşturur ve ID'sini
// döner — testlerde ön koşul olarak grup gerektiğinde kullanılır.
func createTestGroup(t *testing.T, app *fiber.App, token, name, kind string) int64 {
	t.Helper()
	resp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/option-groups",
		`{"name":"`+name+`","kind":"`+kind+`"}`, token))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var view OptionGroupView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	return view.ID
}

// Route sırası regresyonu: "/option-groups/reorder" isteği ":id"
// route'una düşerse "reorder" geçersiz id sayılır ve 400 döner.
// Bu test o sıralamayı kilitler.
func TestOptionGroups_ReorderRoute_IDRoutunaDusmez(t *testing.T) {
	app, token := newTestOptionAdminAPI(t)

	// Önce iki grup oluştur
	g1 := createTestGroup(t, app, token, "Ambalaj", "color")
	g2 := createTestGroup(t, app, token, "Kurdele", "color")

	resp, err := app.Test(authedRequest(http.MethodPut, "/api/admin/option-groups/reorder",
		`{"ids":[`+itoa(g2)+`,`+itoa(g1)+`]}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"reorder ':id' route'una düşmemeli")
}

func TestOptionGroups_YetkisizErisimReddedilir(t *testing.T) {
	app, _ := newTestOptionAdminAPI(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/admin/option-groups", nil))

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestOptionValues_GecersizHex400Doner(t *testing.T) {
	app, token := newTestOptionAdminAPI(t)

	gid := createTestGroup(t, app, token, "Ambalaj", "color")

	resp, err := app.Test(authedRequest(http.MethodPost,
		"/api/admin/option-groups/"+itoa(gid)+"/values",
		`{"name":"Pembe","swatch_hex":"F0A6CA"}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
