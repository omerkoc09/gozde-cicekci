package idare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/internal/payment"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Admin uçları JWT ister — cookie'siz istek 401 dönmeli.
func TestOrders_AuthGerekli(t *testing.T) {
	app, _ := newTestAdminAPI(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/admin/orders", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// newTestOrderAdminAPI order akışını uçtan uca test edebilmek için app'i
// gerçek order.Service ve mock ödeme sağlayıcısıyla kurar; svc'yi de
// döndürür ki test paid sipariş üretmek için ApplyCallback çağırabilsin.
func newTestOrderAdminAPI(t *testing.T) (app *fiber.App, token string, svc *order.Service, productID int64) {
	t.Helper()
	pool := database.NewTestDB(t)

	authSvc := auth.NewService(auth.NewStore(pool), testSecret)
	require.NoError(t, authSvc.CreateAdmin(context.Background(), "cicekci", "test-sifre-123"))

	prodStore := product.NewStore(pool)
	deliveryCfg := order.DeliveryConfig{
		Fee: "50", Slots: []string{"09:00-12:00", "12:00-15:00", "15:00-18:00"},
		SameDayCutoff: "16:00", MaxDays: 30,
		Districts: []string{"Ödemiş", "Tire"},
	}
	svc = order.NewService(order.NewStore(pool), prodStore, deliveryCfg,
		payment.NewMockProvider(), "https://example.com/ok", "https://example.com/fail")

	app = fiber.New()
	Register(app.Group("/api/admin"), Deps{
		AuthSvc:      authSvc,
		OrderSvc:     svc,
		JWTSecret:    testSecret,
		SecureCookie: false,
	})

	token, err := auth.GenerateToken(1, "cicekci", testSecret)
	require.NoError(t, err)

	err = pool.QueryRow(context.Background(),
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ('51 Gül Buket', 'test', 1850.00, true) RETURNING id`).Scan(&productID)
	require.NoError(t, err)

	return app, token, svc, productID
}

func newTestPaidOrder(t *testing.T, svc *order.Service, productID int64) *order.Order {
	t.Helper()
	in := order.CreateInput{
		Items:            []order.CreateItem{{ProductID: productID, Quantity: 2}},
		BuyerName:        "Ahmet Yılmaz",
		BuyerPhone:       "05551112233",
		RecipientName:    "Ayşe Yılmaz",
		RecipientPhone:   "05554445566",
		DeliveryAddress:  "Teşvikiye Cad. No:1",
		DeliveryDistrict: "Ödemiş",
		DeliveryDate:     time.Now().AddDate(0, 0, 2),
		DeliverySlot:     "12:00-15:00",
	}
	o, _, err := svc.Create(context.Background(), in, "127.0.0.1")
	require.NoError(t, err)

	_, err = svc.ApplyCallback(context.Background(), payment.CallbackInput{
		MerchantOID: o.PaymentRef,
		Status:      "success",
		TotalAmount: "375000",
		Hash:        "gecerli-hash",
	}, []byte(`{"status":"success"}`))
	require.NoError(t, err)

	got, err := svc.Get(context.Background(), o.ID)
	require.NoError(t, err)
	return got
}

// İade ucu da JWT ister — cookie'siz istek 401 dönmeli.
func TestRefund_AuthGerekli(t *testing.T) {
	app, _, svc, productID := newTestOrderAdminAPI(t)
	o := newTestPaidOrder(t, svc, productID)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/orders/"+
		strconv.FormatInt(o.ID, 10)+"/refund", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// paid sipariş iade edilince statü refunded olur ve ödeme alanları görünüme yansır.
func TestRefund_PaidSiparisRefunded(t *testing.T) {
	app, token, svc, productID := newTestOrderAdminAPI(t)
	o := newTestPaidOrder(t, svc, productID)
	require.NotNil(t, o.PaidAt)

	resp, err := app.Test(authedRequest(http.MethodPost,
		"/api/admin/orders/"+strconv.FormatInt(o.ID, 10)+"/refund", "", token))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var got orderView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	assert.Equal(t, "refunded", got.Status)
	assert.NotNil(t, got.RefundedAt)
	assert.NotNil(t, got.PaidAt)
	assert.NotEmpty(t, got.PaymentRef)
}
