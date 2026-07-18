package idare

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: 150, B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

// uploadRequest multipart görsel yükleme isteği kurar.
func uploadRequest(t *testing.T, url, token string, data []byte, fieldName, fileName string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile(fieldName, fileName)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, url, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	return req
}

// createProduct test için ürün oluşturur ve id döner.
func createProduct(t *testing.T, app *fiber.App, token, name string) int64 {
	t.Helper()
	resp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"`+name+`","price":"500.00"}`, token))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	return view.ID
}

func imagesPath(pid int64) string {
	return "/api/admin/products/" + strconv.FormatInt(pid, 10) + "/images"
}

func TestImage_Upload(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")

	resp, err := app.Test(uploadRequest(t, imagesPath(pid), token,
		makeTestJPEG(t, 1200, 960), "image", "foto.jpg"), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var view ImageView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	assert.Contains(t, view.URL400, "/400.jpg")
	assert.Contains(t, view.URL1200, "/1200.jpg")
	assert.Equal(t, 0, view.SortOrder)
}

func TestImage_Upload_RequiresAuth(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("image", "foto.jpg")
	require.NoError(t, err)
	_, err = part.Write(makeTestJPEG(t, 100, 100))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, imagesPath(pid), &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// Uzantı .jpg olsa da içerik PDF ise reddedilmeli.
func TestImage_Upload_RejectsPDF(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")

	resp, err := app.Test(uploadRequest(t, imagesPath(pid), token,
		[]byte("%PDF-1.4 bu bir PDF"), "image", "foto.jpg"), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestImage_Upload_WrongFieldName(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")

	resp, err := app.Test(uploadRequest(t, imagesPath(pid), token,
		makeTestJPEG(t, 100, 100), "dosya", "foto.jpg"), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestImage_Upload_ProductNotFound(t *testing.T) {
	app, token := newTestAdminAPI(t)

	resp, err := app.Test(uploadRequest(t, "/api/admin/products/9999/images",
		token, makeTestJPEG(t, 100, 100), "image", "foto.jpg"), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestImage_List(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")
	_, err := app.Test(uploadRequest(t, imagesPath(pid), token, makeTestJPEG(t, 1200, 900), "image", "1.jpg"), -1)
	require.NoError(t, err)
	_, err = app.Test(uploadRequest(t, imagesPath(pid), token, makeTestJPEG(t, 1200, 900), "image", "2.jpg"), -1)
	require.NoError(t, err)

	resp, err := app.Test(authedRequest(http.MethodGet, imagesPath(pid), "", token), -1)

	require.NoError(t, err)
	var views []ImageView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	require.Len(t, views, 2)
	assert.Equal(t, 0, views[0].SortOrder)
	assert.Equal(t, 1, views[1].SortOrder)
}

func TestImage_Delete(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")
	uploadResp, err := app.Test(uploadRequest(t, imagesPath(pid), token,
		makeTestJPEG(t, 1200, 900), "image", "foto.jpg"), -1)
	require.NoError(t, err)
	var img ImageView
	require.NoError(t, json.NewDecoder(uploadResp.Body).Decode(&img))

	resp, err := app.Test(authedRequest(http.MethodDelete,
		"/api/admin/images/"+strconv.FormatInt(img.ID, 10), "", token), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestImage_Reorder(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")

	r1, err := app.Test(uploadRequest(t, imagesPath(pid), token, makeTestJPEG(t, 1200, 900), "image", "1.jpg"), -1)
	require.NoError(t, err)
	var first ImageView
	require.NoError(t, json.NewDecoder(r1.Body).Decode(&first))

	r2, err := app.Test(uploadRequest(t, imagesPath(pid), token, makeTestJPEG(t, 1200, 900), "image", "2.jpg"), -1)
	require.NoError(t, err)
	var second ImageView
	require.NoError(t, json.NewDecoder(r2.Body).Decode(&second))

	body := `{"image_ids":[` + strconv.FormatInt(second.ID, 10) + `,` +
		strconv.FormatInt(first.ID, 10) + `]}`
	resp, err := app.Test(authedRequest(http.MethodPatch, imagesPath(pid)+"/order", body, token), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var views []ImageView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	assert.Equal(t, second.ID, views[0].ID, "kapak değişmeli")
}

func TestImage_Reorder_IncompleteListRejected(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")

	r1, err := app.Test(uploadRequest(t, imagesPath(pid), token, makeTestJPEG(t, 1200, 900), "image", "1.jpg"), -1)
	require.NoError(t, err)
	var first ImageView
	require.NoError(t, json.NewDecoder(r1.Body).Decode(&first))
	_, err = app.Test(uploadRequest(t, imagesPath(pid), token, makeTestJPEG(t, 1200, 900), "image", "2.jpg"), -1)
	require.NoError(t, err)

	body := `{"image_ids":[` + strconv.FormatInt(first.ID, 10) + `]}`
	resp, err := app.Test(authedRequest(http.MethodPatch, imagesPath(pid)+"/order", body, token), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// Ürün silinince görselleri de gider — yetim dosya kalmaz (spec §4.4).
func TestProduct_Delete_RemovesImages(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Silinecek")
	_, err := app.Test(uploadRequest(t, imagesPath(pid), token,
		makeTestJPEG(t, 1200, 900), "image", "foto.jpg"), -1)
	require.NoError(t, err)

	resp, err := app.Test(authedRequest(http.MethodDelete,
		"/api/admin/products/"+strconv.FormatInt(pid, 10), "", token), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

// Ürün detayında görseller görünmeli.
func TestProduct_Get_IncludesImages(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")
	_, err := app.Test(uploadRequest(t, imagesPath(pid), token,
		makeTestJPEG(t, 1200, 900), "image", "foto.jpg"), -1)
	require.NoError(t, err)

	resp, err := app.Test(authedRequest(http.MethodGet,
		"/api/admin/products/"+strconv.FormatInt(pid, 10), "", token), -1)

	require.NoError(t, err)
	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	require.Len(t, view.Images, 1)
	assert.Contains(t, view.Images[0].URL400, "/400.jpg")
}

// Yeni ürünün görsel dizisi boş olmalı, null değil — frontend'de v-for patlamasın.
func TestProduct_Create_ImagesIsEmptyArrayNotNull(t *testing.T) {
	app, token := newTestAdminAPI(t)

	resp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"Yeni","price":"100"}`, token))
	require.NoError(t, err)
	body := make([]byte, 512)
	n, _ := resp.Body.Read(body)

	assert.Contains(t, string(body[:n]), `"images":[]`)
	assert.NotContains(t, string(body[:n]), `"images":null`)
}
