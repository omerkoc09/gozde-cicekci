package order

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDeliveryConfig() DeliveryConfig {
	return DeliveryConfig{
		Fee:           "50",
		Slots:         []string{"09:00-12:00", "12:00-15:00", "15:00-18:00"},
		SameDayCutoff: "16:00",
		MaxDays:       30,
		Districts:     []string{"Ödemiş", "Tire", "Bayındır"},
		DistrictFees:  map[string]string{"Tire": "80"},
	}
}

func testCreateInput(productID int64) CreateInput {
	return CreateInput{
		Items:            []CreateItem{{ProductID: productID, Quantity: 2}},
		BuyerName:        "Ahmet Yılmaz",
		BuyerPhone:       "05551112233",
		RecipientName:    "Ayşe Yılmaz",
		RecipientPhone:   "05554445566",
		DeliveryAddress:  "Teşvikiye Cad. No:1",
		DeliveryDistrict: "Ödemiş",
		DeliveryDate:     time.Now().AddDate(0, 0, 2),
		DeliverySlot:     "12:00-15:00",
	}
}

// setupService gerçek DB ile service kurar ve bir aktif ürün oluşturur.
//
// pool'u da döndürüyor: NewTestDB TRUNCATE çalıştırdığı için testlerin
// ikinci kez çağırması veriyi siler — aynı pool paylaşılmalı.
func setupService(t *testing.T) (svc *Service, pool *pgxpool.Pool, productID int64) {
	t.Helper()
	pool = database.NewTestDB(t)

	err := pool.QueryRow(context.Background(),
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ('51 Gül Buket', 'test', 1850.00, true) RETURNING id`).Scan(&productID)
	require.NoError(t, err)

	svc = NewService(NewStore(pool), product.NewStore(pool), testDeliveryConfig())

	return svc, pool, productID
}

// EN KRİTİK TEST: fiyat sepetten değil DB'den okunur.
func TestService_Create_FiyatDBdenOkunur(t *testing.T) {
	svc, _, productID := setupService(t)

	o, err := svc.Create(context.Background(), testCreateInput(productID))
	require.NoError(t, err)

	// Ürün 1850, adet 2 → 3700 + 50 teslimat (Ödemiş = genel ücret) = 3750
	assert.Equal(t, "3700", o.ItemsTotal.String())
	assert.Equal(t, "50", o.DeliveryFee.String())
	assert.Equal(t, "3750", o.Total.String())
	assert.Equal(t, "1850", o.Items[0].PriceAtOrder.String())
}

// İlçeye özel ücret varsa genel ücret yerine o kullanılır.
func TestService_Create_IlceyeOzelUcretKullanilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.DeliveryDistrict = "Tire" // testDeliveryConfig: Tire=80, genel=50

	o, err := svc.Create(context.Background(), in)
	require.NoError(t, err)

	assert.Equal(t, "80", o.DeliveryFee.String())
	assert.Equal(t, "3780", o.Total.String())
}

// DistrictFees'te olmayan ilçe genel ücrete düşer.
func TestService_Create_IlceOzelUcretiYoksaGenelUcreteDuser(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.DeliveryDistrict = "Bayındır" // DistrictFees'te yok, genel=50

	o, err := svc.Create(context.Background(), in)
	require.NoError(t, err)

	assert.Equal(t, "50", o.DeliveryFee.String())
}

func TestService_Create_PasifUrunReddedilir(t *testing.T) {
	svc, pool, productID := setupService(t)

	// Ürünü pasif yap — sepette dururken esnaf pasif yapmış olabilir
	_, err := pool.Exec(context.Background(),
		`UPDATE products SET is_active = false WHERE id = $1`, productID)
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), testCreateInput(productID))
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_GecmisTarihReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.DeliveryDate = time.Now().AddDate(0, 0, -1)

	_, err := svc.Create(context.Background(), in)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_CokIleriTarihReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.DeliveryDate = time.Now().AddDate(0, 0, 60) // MaxDays=30

	_, err := svc.Create(context.Background(), in)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_GecersizSlotReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.DeliverySlot = "03:00-04:00" // config'de yok

	_, err := svc.Create(context.Background(), in)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_GecersizIlceReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.DeliveryDistrict = "İstanbul" // config'de yok (Ödemiş, Tire, Bayındır)

	_, err := svc.Create(context.Background(), in)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_BosSepetReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.Items = nil

	_, err := svc.Create(context.Background(), in)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_SifirAdetReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.Items = []CreateItem{{ProductID: productID, Quantity: 0}}

	_, err := svc.Create(context.Background(), in)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_ZorunluAlanBosReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.RecipientPhone = "  " // kurye alıcıyı arayamaz

	_, err := svc.Create(context.Background(), in)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Update_GecersizStatusReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	o, err := svc.Create(context.Background(), testCreateInput(productID))
	require.NoError(t, err)

	bad := "uydurma"
	_, err = svc.Update(context.Background(), o.ID, &bad, nil)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}
