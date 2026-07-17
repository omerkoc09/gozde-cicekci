package order

import (
	"context"
	"testing"
	"time"

	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewOrder() NewOrder {
	return NewOrder{
		BuyerName:        "Ahmet Yılmaz",
		BuyerPhone:       "05551112233",
		BuyerEmail:       "ahmet@example.com",
		RecipientName:    "Ayşe Yılmaz",
		RecipientPhone:   "05554445566",
		DeliveryAddress:  "Teşvikiye Cad. No:1, Şişli/İstanbul",
		DeliveryDistrict: "Ödemiş",
		DeliveryDate:     time.Now().AddDate(0, 0, 1),
		DeliverySlot:     "12:00-15:00",
		CardMessage:      "Doğum günün kutlu olsun",
		ItemsTotal:       decimal.RequireFromString("1850.00"),
		DeliveryFee:      decimal.RequireFromString("50.00"),
		Total:            decimal.RequireFromString("1900.00"),
	}
}

func TestStore_Create(t *testing.T) {
	pool := database.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	// Sipariş için gerçek bir ürün gerekiyor (FK)
	var productID int64
	err := pool.QueryRow(ctx,
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ('51 Gül Buket', 'test', 1850.00, true) RETURNING id`).Scan(&productID)
	require.NoError(t, err)

	in := testNewOrder()
	in.Items = []NewOrderItem{{
		ProductID:    productID,
		ProductName:  "51 Gül Buket",
		PriceAtOrder: decimal.RequireFromString("1850.00"),
		Quantity:     1,
	}}

	store := NewStore(pool)
	o, err := store.Create(ctx, in)
	require.NoError(t, err)

	assert.NotZero(t, o.ID)
	assert.NotEmpty(t, o.OrderNo)
	assert.Equal(t, StatusPending, o.Status)
	assert.Equal(t, "Ahmet Yılmaz", o.BuyerName)
	assert.Equal(t, "Ödemiş", o.DeliveryDistrict)
	assert.True(t, o.Total.Equal(decimal.RequireFromString("1900.00")))
	require.Len(t, o.Items, 1)
	assert.Equal(t, "51 Gül Buket", o.Items[0].ProductName)
}

func TestStore_Create_OrderNoGunlukArtar(t *testing.T) {
	pool := database.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	var productID int64
	err := pool.QueryRow(ctx,
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ('Test', 'test', 100.00, true) RETURNING id`).Scan(&productID)
	require.NoError(t, err)

	store := NewStore(pool)

	mk := func() *Order {
		in := testNewOrder()
		in.Items = []NewOrderItem{{
			ProductID: productID, ProductName: "Test",
			PriceAtOrder: decimal.RequireFromString("100.00"), Quantity: 1,
		}}
		o, err := store.Create(ctx, in)
		require.NoError(t, err)
		return o
	}

	o1 := mk()
	o2 := mk()

	// Aynı gün: sıra artmalı, ikisi farklı olmalı
	assert.NotEqual(t, o1.OrderNo, o2.OrderNo)
	assert.Equal(t, FormatOrderNo(time.Now(), 1), o1.OrderNo)
	assert.Equal(t, FormatOrderNo(time.Now(), 2), o2.OrderNo)
}

func TestStore_Create_Atomik(t *testing.T) {
	pool := database.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	store := NewStore(pool)

	in := testNewOrder()
	// Var olmayan ürün → order_items INSERT'i FK'dan patlar.
	// orders satırı da yazılmamalı (tek transaction).
	in.Items = []NewOrderItem{{
		ProductID:    999999,
		ProductName:  "Yok",
		PriceAtOrder: decimal.RequireFromString("100.00"),
		Quantity:     1,
	}}

	_, err := store.Create(ctx, in)
	require.Error(t, err)

	var count int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM orders`).Scan(&count)
	require.NoError(t, err)
	assert.Zero(t, count, "orders satırı rollback edilmeliydi")
}

func TestStore_List_StatusFiltresi(t *testing.T) {
	pool := database.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	var productID int64
	err := pool.QueryRow(ctx,
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ('Test', 'test', 100.00, true) RETURNING id`).Scan(&productID)
	require.NoError(t, err)

	store := NewStore(pool)
	in := testNewOrder()
	in.Items = []NewOrderItem{{
		ProductID: productID, ProductName: "Test",
		PriceAtOrder: decimal.RequireFromString("100.00"), Quantity: 1,
	}}

	o, err := store.Create(ctx, in)
	require.NoError(t, err)

	confirmed := string(StatusConfirmed)
	_, err = store.Update(ctx, o.ID, &confirmed, nil)
	require.NoError(t, err)

	// pending filtresi: boş dönmeli
	list, err := store.List(ctx, string(StatusPending), 50, 0)
	require.NoError(t, err)
	assert.Empty(t, list)

	// confirmed filtresi: bir kayıt
	list, err = store.List(ctx, string(StatusConfirmed), 50, 0)
	require.NoError(t, err)
	require.Len(t, list, 1)

	// filtresiz: bir kayıt
	list, err = store.List(ctx, "", 50, 0)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestStore_GetByID_Yok(t *testing.T) {
	pool := database.NewTestDB(t)
	defer pool.Close()

	store := NewStore(pool)
	_, err := store.GetByID(context.Background(), 999999)
	assert.ErrorIs(t, err, errorsx.ErrNotFound)
}
