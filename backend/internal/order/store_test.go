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
	assert.Equal(t, StatusAwaitingPayment, o.Status)
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

	paid := string(StatusPaid)
	_, err = store.Update(ctx, o.ID, &paid, nil)
	require.NoError(t, err)

	// awaiting_payment filtresi: boş dönmeli
	list, err := store.List(ctx, string(StatusAwaitingPayment), 50, 0)
	require.NoError(t, err)
	assert.Empty(t, list)

	// paid filtresi: bir kayıt
	list, err = store.List(ctx, string(StatusPaid), 50, 0)
	require.NoError(t, err)
	require.Len(t, list, 1)

	// filtresiz: bir kayıt
	list, err = store.List(ctx, "", 50, 0)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Len(t, list[0].Items, 1)
	assert.Equal(t, "Test", list[0].Items[0].ProductName)
}

// List her siparişin kendi kalemlerini döndürmeli — batch sorgu (itemsOfMany)
// siparişleri birbirine karıştırmamalı.
func TestStore_List_HerSiparisKendiKalemleriniAlir(t *testing.T) {
	pool := database.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	var productA, productB int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ('Ürün A', 'test', 100.00, true) RETURNING id`).Scan(&productA))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ('Ürün B', 'test', 50.00, true) RETURNING id`).Scan(&productB))

	store := NewStore(pool)

	in1 := testNewOrder()
	in1.Items = []NewOrderItem{{
		ProductID: productA, ProductName: "Ürün A",
		PriceAtOrder: decimal.RequireFromString("100.00"), Quantity: 1,
	}}
	o1, err := store.Create(ctx, in1)
	require.NoError(t, err)

	in2 := testNewOrder()
	in2.Items = []NewOrderItem{{
		ProductID: productB, ProductName: "Ürün B",
		PriceAtOrder: decimal.RequireFromString("50.00"), Quantity: 2,
	}}
	o2, err := store.Create(ctx, in2)
	require.NoError(t, err)

	list, err := store.List(ctx, "", 50, 0)
	require.NoError(t, err)
	require.Len(t, list, 2)

	byID := map[int64]Order{}
	for _, o := range list {
		byID[o.ID] = o
	}

	require.Len(t, byID[o1.ID].Items, 1)
	assert.Equal(t, "Ürün A", byID[o1.ID].Items[0].ProductName)

	require.Len(t, byID[o2.ID].Items, 1)
	assert.Equal(t, "Ürün B", byID[o2.ID].Items[0].ProductName)
}

func TestStore_GetByID_Yok(t *testing.T) {
	pool := database.NewTestDB(t)
	defer pool.Close()

	store := NewStore(pool)
	_, err := store.GetByID(context.Background(), 999999)
	assert.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestStore_SetPaid_AwaitingdanPaid(t *testing.T) {
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

	paid, err := store.SetPaid(ctx, o.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPaid, paid.Status)
	require.NotNil(t, paid.PaidAt)
}

func TestStore_SetPaid_AwaitingDegilseNotFound(t *testing.T) {
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

	_, err = store.SetPaid(ctx, o.ID)
	require.NoError(t, err)

	// zaten paid — ikinci SetPaid awaiting_payment koşuluna uymaz
	_, err = store.SetPaid(ctx, o.ID)
	assert.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestStore_SetRefunded_PaidtenRefunded(t *testing.T) {
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

	_, err = store.SetPaid(ctx, o.ID)
	require.NoError(t, err)

	refunded, err := store.SetRefunded(ctx, o.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusRefunded, refunded.Status)
	require.NotNil(t, refunded.RefundedAt)
}

func TestStore_SetPaymentRef_GetByPaymentRef(t *testing.T) {
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

	err = store.SetPaymentRef(ctx, o.ID, "merchant-oid-123")
	require.NoError(t, err)

	found, err := store.GetByPaymentRef(ctx, "merchant-oid-123")
	require.NoError(t, err)
	assert.Equal(t, o.ID, found.ID)
	assert.Equal(t, "merchant-oid-123", found.PaymentRef)
}

func TestStore_GetByPaymentRef_Yok(t *testing.T) {
	pool := database.NewTestDB(t)
	defer pool.Close()

	store := NewStore(pool)
	_, err := store.GetByPaymentRef(context.Background(), "olmayan-ref")
	assert.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestStore_HasPaymentEvent_Idempotency(t *testing.T) {
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

	has, err := store.HasPaymentEvent(ctx, o.ID, "callback_ok")
	require.NoError(t, err)
	assert.False(t, has, "başlangıçta olay olmamalı")

	err = store.AddPaymentEvent(ctx, o.ID, "callback_ok", []byte(`{}`))
	require.NoError(t, err)

	has, err = store.HasPaymentEvent(ctx, o.ID, "callback_ok")
	require.NoError(t, err)
	assert.True(t, has, "olay eklendikten sonra true olmalı")
}
