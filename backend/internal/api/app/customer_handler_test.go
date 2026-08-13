package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/internal/customer"
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registerTestCustomer /customer/register çağırır ve set edilen customer_token
// cookie'sinin değerini döner.
func registerTestCustomer(t *testing.T, app *fiber.App, email, password, name, phone string) (cookieValue string, resp *http.Response) {
	t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":%q,"name":%q,"phone":%q}`, email, password, name, phone)
	req := httptest.NewRequest(http.MethodPost, "/api/customer/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	for _, ck := range resp.Cookies() {
		if ck.Name == customer.CookieName {
			return ck.Value, resp
		}
	}
	return "", resp
}

func TestCustomer_RegisterLogin_CookieSet(t *testing.T) {
	app, _, _, _, _, _ := newTestAPIFull(t)

	// Register → 201 + customer_token cookie set.
	tok, resp := registerTestCustomer(t, app, "kayit@example.com", "sifre1234", "Ali Veli", "05551112233")
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	require.NotEmpty(t, tok, "register sonrası customer_token cookie'si set edilmeli")

	var cv customerView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&cv))
	assert.Equal(t, "kayit@example.com", cv.Email)
	assert.Equal(t, "Ali Veli", cv.Name)
	assert.Equal(t, "05551112233", cv.Phone)
	assert.NotZero(t, cv.ID)

	// Şifre hash'i asla JSON'a çıkmamalı.
	raw, err := json.Marshal(cv)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "password")
	assert.NotContains(t, string(raw), "$2a$") // bcrypt hash öneki

	// Login → ok + cookie set.
	loginBody := `{"email":"kayit@example.com","password":"sifre1234"}`
	req := httptest.NewRequest(http.MethodPost, "/api/customer/login", bytes.NewBufferString(loginBody))
	req.Header.Set("Content-Type", "application/json")
	loginResp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, loginResp.StatusCode)

	var loginCookie string
	for _, ck := range loginResp.Cookies() {
		if ck.Name == customer.CookieName {
			loginCookie = ck.Value
		}
	}
	assert.NotEmpty(t, loginCookie, "login sonrası da customer_token cookie'si set edilmeli")
}

func TestCustomer_Me_AuthGerekli(t *testing.T) {
	app, _, _, _, _, _ := newTestAPIFull(t)

	// Cookie'siz GET /customer/me → 401.
	req := httptest.NewRequest(http.MethodGet, "/api/customer/me", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

// TestCustomer_AdminTokenCustomerUcunaErisemez GÜVENLİK testi: admin
// token'ının (auth.GenerateToken) customer_token cookie'sine konsa bile
// müşteri uçlarına erişemediğini kanıtlar (ParseToken Type=="customer" şartı).
func TestCustomer_AdminTokenCustomerUcunaErisemez(t *testing.T) {
	app, _, _, _, _, _ := newTestAPIFull(t)

	adminToken, err := auth.GenerateToken(1, "cicekci", testJWTSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/customer/me", nil)
	req.AddCookie(&http.Cookie{Name: customer.CookieName, Value: adminToken})

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode,
		"admin token customer_token cookie'sine konsa bile reddedilmeli")
}

// TestCustomer_Orders_YalnizKendisi iki müşteri sipariş verir; A'nın
// /orders'ı yalnız A'nınkini döner — sızıntı yok.
func TestCustomer_Orders_YalnizKendisi(t *testing.T) {
	app, orderSvc, _, _, custSvc, pool := newTestAPIFull(t)

	// İki müşteri kaydet.
	tokenA, custA, err := custSvc.Register(context.Background(), "musteri-a@example.com", "sifre1234", "Müşteri A", "05550000001")
	require.NoError(t, err)
	_, custB, err := custSvc.Register(context.Background(), "musteri-b@example.com", "sifre1234", "Müşteri B", "05550000002")
	require.NoError(t, err)

	// Ürün oluştur (order.Service ürünü DB'den okuyor).
	var productID int64
	err = pool.QueryRow(context.Background(),
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ('Test Buket', 'test', 500.00, true) RETURNING id`).Scan(&productID)
	require.NoError(t, err)

	custAID := custA.ID
	custBID := custB.ID

	inA := order.CreateInput{
		Items:            []order.CreateItem{{ProductID: productID, Quantity: 1}},
		BuyerName:        "Müşteri A", BuyerPhone: "05550000001",
		RecipientName: "Alıcı A", RecipientPhone: "05559999991",
		DeliveryAddress: "Adres A", DeliveryDistrict: "Ödemiş",
		DeliveryDate: time.Now().AddDate(0, 0, 2), DeliverySlot: "12:00-15:00",
	}
	oA, _, err := orderSvc.Create(context.Background(), inA, "127.0.0.1", &custAID)
	require.NoError(t, err)

	inB := order.CreateInput{
		Items:            []order.CreateItem{{ProductID: productID, Quantity: 1}},
		BuyerName:        "Müşteri B", BuyerPhone: "05550000002",
		RecipientName: "Alıcı B", RecipientPhone: "05559999992",
		DeliveryAddress: "Adres B", DeliveryDistrict: "Ödemiş",
		DeliveryDate: time.Now().AddDate(0, 0, 2), DeliverySlot: "12:00-15:00",
	}
	_, _, err = orderSvc.Create(context.Background(), inB, "127.0.0.1", &custBID)
	require.NoError(t, err)

	// A olarak /customer/orders çağır.
	req := httptest.NewRequest(http.MethodGet, "/api/customer/orders", nil)
	req.AddCookie(&http.Cookie{Name: customer.CookieName, Value: tokenA})
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var list []createOrderCustomerView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
	require.Len(t, list, 1, "A yalnız kendi siparişini görmeli")
	assert.Equal(t, oA.OrderNo, list[0].OrderNo)
	// items_total(500) + delivery_fee(50) = 550
	assert.Equal(t, "550.00", list[0].Total)
	require.Len(t, list[0].Items, 1)
	assert.Equal(t, "Test Buket", list[0].Items[0].ProductName)
	assert.Equal(t, 1, list[0].Items[0].Quantity)
}
