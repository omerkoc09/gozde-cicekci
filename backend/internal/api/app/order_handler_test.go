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
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateOrder_FiyatGovdedenGelmez(t *testing.T) {
	pool := database.NewTestDB(t)

	var productID int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ('51 Gül Buket', 'test', 1850.00, true) RETURNING id`).Scan(&productID)
	require.NoError(t, err)

	cfg := order.DeliveryConfig{
		Fee: "50", Slots: []string{"12:00-15:00"},
		SameDayCutoff: "16:00", MaxDays: 30,
		Districts: []string{"Ödemiş"},
	}
	svc := order.NewService(order.NewStore(pool), product.NewStore(pool), cfg)

	f := fiber.New()
	oh := &orderHandler{svc: svc, cfg: cfg}
	f.Post("/orders", oh.create)

	// Gövdede fiyat göndermeye çalış — yok sayılmalı, DB fiyatı kullanılmalı
	body := fmt.Sprintf(`{
		"items": [{"product_id": %d, "quantity": 2, "price": "1.00"}],
		"buyer": {"name": "Ahmet", "phone": "05551112233"},
		"recipient": {"name": "Ayşe", "phone": "05554445566"},
		"delivery": {"address": "Test Cad. 1", "district": "Ödemiş", "date": "%s", "slot": "12:00-15:00"}
	}`, productID, time.Now().AddDate(0, 0, 2).Format("2006-01-02"))

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var out createOrderResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))

	// 1850 × 2 + 50 = 3750 — gövdedeki "1.00" yok sayıldı
	assert.Equal(t, "3750.00", out.Total)
	assert.NotEmpty(t, out.OrderNo)
}

func TestDeliveryConfig(t *testing.T) {
	cfg := order.DeliveryConfig{
		Fee: "50", Slots: []string{"09:00-12:00", "12:00-15:00"},
		SameDayCutoff: "16:00", MaxDays: 30,
		Districts:    []string{"Ödemiş", "Tire", "Bayındır"},
		DistrictFees: map[string]string{"Tire": "80"},
	}

	f := fiber.New()
	oh := &orderHandler{svc: nil, cfg: cfg}
	f.Get("/delivery-config", oh.deliveryConfig)

	resp, err := f.Test(httptest.NewRequest(http.MethodGet, "/delivery-config", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var out deliveryConfigResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))

	assert.Equal(t, "50", out.Fee)
	assert.Len(t, out.Slots, 2)
	assert.Equal(t, 30, out.MaxDays)
	assert.Len(t, out.Districts, 3)
	assert.Equal(t, "80", out.DistrictFees["Tire"])
}
