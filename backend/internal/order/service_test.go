package order

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/internal/payment"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePay PaymentStarter test double'ı. Gerçek PayTR imza/hash doğrulaması
// payment paketinde test edilir — burada sadece order servisinin fakePay ile
// nasıl davrandığı (paid yapar mı, yapmaz mı) test edilir.
type fakePay struct {
	callbackOK bool
	refundErr  error
	started    bool
}

func (f *fakePay) Start(_ context.Context, _ payment.StartInput) (payment.StartResult, error) {
	f.started = true
	return payment.StartResult{Token: "tok"}, nil
}
func (f *fakePay) VerifyCallback(in payment.CallbackInput) payment.CallbackResult {
	return payment.CallbackResult{OK: f.callbackOK, MerchantOID: in.MerchantOID}
}
func (f *fakePay) Refund(_ context.Context, _ payment.RefundInput) error { return f.refundErr }

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
	return setupServiceWithPay(t, &fakePay{callbackOK: true})
}

// setupServiceWithPay setupService ile aynı, ama çağıran kendi fakePay'ini
// (ör. callbackOK:false, refundErr set edilmiş) enjekte edebilir.
func setupServiceWithPay(t *testing.T, pay PaymentStarter) (svc *Service, pool *pgxpool.Pool, productID int64) {
	t.Helper()
	pool = database.NewTestDB(t)

	err := pool.QueryRow(context.Background(),
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ('51 Gül Buket', 'test', 1850.00, true) RETURNING id`).Scan(&productID)
	require.NoError(t, err)

	svc = NewService(NewStore(pool), product.NewStore(pool), testDeliveryConfig(),
		pay, "https://example.com/ok", "https://example.com/fail")

	return svc, pool, productID
}

// EN KRİTİK TEST: fiyat sepetten değil DB'den okunur.
func TestService_Create_FiyatDBdenOkunur(t *testing.T) {
	svc, _, productID := setupService(t)

	o, _, err := svc.Create(context.Background(), testCreateInput(productID), "127.0.0.1", nil)
	require.NoError(t, err)

	// Ürün 1850, adet 2 → 3700 + 50 teslimat (Ödemiş = genel ücret) = 3750
	assert.Equal(t, "3700", o.ItemsTotal.String())
	assert.Equal(t, "50", o.DeliveryFee.String())
	assert.Equal(t, "3750", o.Total.String())
	assert.Equal(t, "1850", o.Items[0].PriceAtOrder.String())
}

// Create'e verilen customerID sipariş kaydına bağlanmalı; nil (misafir)
// verilirse sipariş customer_id'siz kalmalı.
func TestService_Create_CustomerIDBaglar(t *testing.T) {
	svc, pool, productID := setupService(t)
	ctx := context.Background()

	var customerID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO customers (email, password_hash, name, phone)
		 VALUES ('musteri@example.com', 'hash', 'Test Müşteri', '05551112233') RETURNING id`).
		Scan(&customerID))

	o, _, err := svc.Create(ctx, testCreateInput(productID), "127.0.0.1", &customerID)
	require.NoError(t, err)
	require.NotNil(t, o.CustomerID)
	assert.Equal(t, customerID, *o.CustomerID)

	guest, _, err := svc.Create(ctx, testCreateInput(productID), "127.0.0.1", nil)
	require.NoError(t, err)
	assert.Nil(t, guest.CustomerID)
}

// İlçeye özel ücret varsa genel ücret yerine o kullanılır.
func TestService_Create_IlceyeOzelUcretKullanilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.DeliveryDistrict = "Tire" // testDeliveryConfig: Tire=80, genel=50

	o, _, err := svc.Create(context.Background(), in, "127.0.0.1", nil)
	require.NoError(t, err)

	assert.Equal(t, "80", o.DeliveryFee.String())
	assert.Equal(t, "3780", o.Total.String())
}

// DistrictFees'te olmayan ilçe genel ücrete düşer.
func TestService_Create_IlceOzelUcretiYoksaGenelUcreteDuser(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.DeliveryDistrict = "Bayındır" // DistrictFees'te yok, genel=50

	o, _, err := svc.Create(context.Background(), in, "127.0.0.1", nil)
	require.NoError(t, err)

	assert.Equal(t, "50", o.DeliveryFee.String())
}

func TestService_Create_PasifUrunReddedilir(t *testing.T) {
	svc, pool, productID := setupService(t)

	// Ürünü pasif yap — sepette dururken esnaf pasif yapmış olabilir
	_, err := pool.Exec(context.Background(),
		`UPDATE products SET is_active = false WHERE id = $1`, productID)
	require.NoError(t, err)

	_, _, err = svc.Create(context.Background(), testCreateInput(productID), "127.0.0.1", nil)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_GecmisTarihReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.DeliveryDate = time.Now().AddDate(0, 0, -1)

	_, _, err := svc.Create(context.Background(), in, "127.0.0.1", nil)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_CokIleriTarihReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.DeliveryDate = time.Now().AddDate(0, 0, 60) // MaxDays=30

	_, _, err := svc.Create(context.Background(), in, "127.0.0.1", nil)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_GecersizSlotReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.DeliverySlot = "03:00-04:00" // config'de yok

	_, _, err := svc.Create(context.Background(), in, "127.0.0.1", nil)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_GecersizIlceReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.DeliveryDistrict = "İstanbul" // config'de yok (Ödemiş, Tire, Bayındır)

	_, _, err := svc.Create(context.Background(), in, "127.0.0.1", nil)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_BosSepetReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.Items = nil

	_, _, err := svc.Create(context.Background(), in, "127.0.0.1", nil)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_SifirAdetReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.Items = []CreateItem{{ProductID: productID, Quantity: 0}}

	_, _, err := svc.Create(context.Background(), in, "127.0.0.1", nil)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_ZorunluAlanBosReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	in := testCreateInput(productID)
	in.RecipientPhone = "  " // kurye alıcıyı arayamaz

	_, _, err := svc.Create(context.Background(), in, "127.0.0.1", nil)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Update_GecersizStatusReddedilir(t *testing.T) {
	svc, _, productID := setupService(t)

	o, _, err := svc.Create(context.Background(), testCreateInput(productID), "127.0.0.1", nil)
	require.NoError(t, err)

	bad := "uydurma"
	_, err = svc.Update(context.Background(), o.ID, &bad, nil)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

// EN KRİTİK TEST: hash geçerli + status=success → sipariş paid olur.
func TestService_ApplyCallback_SuccessPaidYapar(t *testing.T) {
	pay := &fakePay{callbackOK: true}
	svc, _, productID := setupServiceWithPay(t, pay)

	o, token, err := svc.Create(context.Background(), testCreateInput(productID), "127.0.0.1", nil)
	require.NoError(t, err)
	require.Equal(t, "tok", token)
	require.True(t, pay.started)
	require.NotEmpty(t, o.PaymentRef)

	accepted, err := svc.ApplyCallback(context.Background(), payment.CallbackInput{
		MerchantOID: o.PaymentRef,
		Status:      "success",
		TotalAmount: "375000",
		Hash:        "gecerli-hash",
	}, []byte(`{"status":"success"}`))
	require.NoError(t, err)
	assert.True(t, accepted)

	got, err := svc.Get(context.Background(), o.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPaid, got.Status)
	assert.NotNil(t, got.PaidAt)
}

// Aynı callback iki kez işlenirse sipariş bir kez paid olur, ikinci çağrı
// no-op ama yine accepted=true döner (PayTR'nin tekrar denemesini durdurmak için).
func TestService_ApplyCallback_Idempotent(t *testing.T) {
	pay := &fakePay{callbackOK: true}
	svc, _, productID := setupServiceWithPay(t, pay)

	o, _, err := svc.Create(context.Background(), testCreateInput(productID), "127.0.0.1", nil)
	require.NoError(t, err)

	cbIn := payment.CallbackInput{
		MerchantOID: o.PaymentRef,
		Status:      "success",
		TotalAmount: "375000",
		Hash:        "gecerli-hash",
	}

	accepted1, err := svc.ApplyCallback(context.Background(), cbIn, []byte(`{}`))
	require.NoError(t, err)
	assert.True(t, accepted1)

	accepted2, err := svc.ApplyCallback(context.Background(), cbIn, []byte(`{}`))
	require.NoError(t, err)
	assert.True(t, accepted2)

	got, err := svc.Get(context.Background(), o.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPaid, got.Status)
}

// Hash geçersiz (veya status=failed) → VerifyCallback OK=false döner,
// sipariş asla paid yapılmaz. Bu, sahte ödeme bildirimlerine karşı ana koruma.
func TestService_ApplyCallback_HashGecersizPaidYapmaz(t *testing.T) {
	pay := &fakePay{callbackOK: false}
	svc, _, productID := setupServiceWithPay(t, pay)

	o, _, err := svc.Create(context.Background(), testCreateInput(productID), "127.0.0.1", nil)
	require.NoError(t, err)

	accepted, err := svc.ApplyCallback(context.Background(), payment.CallbackInput{
		MerchantOID: o.PaymentRef,
		Status:      "failed",
		TotalAmount: "375000",
		Hash:        "sahte-hash",
	}, []byte(`{"status":"failed"}`))
	require.NoError(t, err)
	// oid meşru (sipariş bulundu) → PayTR'ye OK dönülür, ama sipariş paid
	// yapılmaz — para hareketi yok, sadece iz bırakılır.
	assert.True(t, accepted)

	got, err := svc.Get(context.Background(), o.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusAwaitingPayment, got.Status)
	assert.Nil(t, got.PaidAt)
}

// Callback'teki tutar sipariş tutarıyla uyuşmuyorsa (ör. kısmi ödeme, PayTR
// hatası) hash geçerli olsa bile sipariş paid yapılmaz — para bir finansal
// kayıttır, sessizce farklı tutarla kapatılamaz.
func TestService_ApplyCallback_TutarUyusmazsaPaidYapmaz(t *testing.T) {
	pay := &fakePay{callbackOK: true}
	svc, _, productID := setupServiceWithPay(t, pay)

	o, _, err := svc.Create(context.Background(), testCreateInput(productID), "127.0.0.1", nil)
	require.NoError(t, err)
	require.Equal(t, "3750", o.Total.String())

	accepted, err := svc.ApplyCallback(context.Background(), payment.CallbackInput{
		MerchantOID: o.PaymentRef,
		Status:      "success",
		TotalAmount: "1", // sipariş 375000 kuruş iken yanlış tutar
		Hash:        "gecerli-hash",
	}, []byte(`{"status":"success"}`))
	require.NoError(t, err)
	// hash geçerli/meşru callback → PayTR'ye OK dönülür (retry döngüsüne girmesin)
	assert.True(t, accepted)

	got, err := svc.Get(context.Background(), o.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusAwaitingPayment, got.Status)
	assert.Nil(t, got.PaidAt)

	hasFail, err := svc.store.HasPaymentEvent(context.Background(), o.ID, "callback_fail")
	require.NoError(t, err)
	assert.True(t, hasFail, "tutar uyuşmazlığı callback_fail izi bırakmalı")

	hasOK, err := svc.store.HasPaymentEvent(context.Background(), o.ID, "callback_ok")
	require.NoError(t, err)
	assert.False(t, hasOK, "tutar uyuşmazsa callback_ok yazılmamalı")
}

// Kalıcı kilitlenme olmadığını kanıtlar: yanlış tutarlı callback siparişi
// awaiting_payment'ta bırakır (yukarıdaki test), ama aynı merchant_oid ile
// SONRA gelen doğru tutarlı bir callback siparişi yine de paid yapabilmeli.
func TestService_ApplyCallback_YanlisTutarSonraDogruTutarPaidYapar(t *testing.T) {
	pay := &fakePay{callbackOK: true}
	svc, _, productID := setupServiceWithPay(t, pay)

	o, _, err := svc.Create(context.Background(), testCreateInput(productID), "127.0.0.1", nil)
	require.NoError(t, err)
	require.Equal(t, "3750", o.Total.String())

	// 1. Yanlış tutarlı callback — sipariş awaiting_payment'ta kalmalı.
	accepted, err := svc.ApplyCallback(context.Background(), payment.CallbackInput{
		MerchantOID: o.PaymentRef,
		Status:      "success",
		TotalAmount: "1", // sipariş 375000 kuruş iken yanlış tutar
		Hash:        "gecerli-hash",
	}, []byte(`{"status":"success","attempt":"1"}`))
	require.NoError(t, err)
	assert.True(t, accepted)

	got, err := svc.Get(context.Background(), o.ID)
	require.NoError(t, err)
	require.Equal(t, StatusAwaitingPayment, got.Status)
	require.Nil(t, got.PaidAt)

	// 2. Aynı merchant_oid ile SONRA gelen doğru tutarlı callback — sipariş
	// paid olmalı. Önceki callback_fail izi bunu kilitlememeli.
	accepted, err = svc.ApplyCallback(context.Background(), payment.CallbackInput{
		MerchantOID: o.PaymentRef,
		Status:      "success",
		TotalAmount: "375000", // doğru tutar
		Hash:        "gecerli-hash",
	}, []byte(`{"status":"success","attempt":"2"}`))
	require.NoError(t, err)
	assert.True(t, accepted)

	got, err = svc.Get(context.Background(), o.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPaid, got.Status, "doğru tutarlı ikinci callback siparişi paid yapmalı")
	assert.NotNil(t, got.PaidAt)

	hasOK, err := svc.store.HasPaymentEvent(context.Background(), o.ID, "callback_ok")
	require.NoError(t, err)
	assert.True(t, hasOK, "doğru tutarlı callback callback_ok izi bırakmalı")
}

// Var olmayan merchant_oid → GetByPaymentRef ErrNotFound, accepted=false.
func TestService_ApplyCallback_BilinmeyenOidReddedilir(t *testing.T) {
	pay := &fakePay{callbackOK: true}
	svc, _, _ := setupServiceWithPay(t, pay)

	accepted, err := svc.ApplyCallback(context.Background(), payment.CallbackInput{
		MerchantOID: "yok-boyle-bir-oid",
		Status:      "success",
		TotalAmount: "100",
		Hash:        "h",
	}, []byte(`{}`))
	assert.Error(t, err)
	assert.False(t, accepted)
}

// Refund yalnızca paid veya delivered siparişte çalışır.
func TestService_Refund_YalnizPaidVeyaDelivered(t *testing.T) {
	pay := &fakePay{callbackOK: true}
	svc, _, productID := setupServiceWithPay(t, pay)

	o, _, err := svc.Create(context.Background(), testCreateInput(productID), "127.0.0.1", nil)
	require.NoError(t, err)

	// awaiting_payment sipariş → iade reddedilir
	_, err = svc.Refund(context.Background(), o.ID)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)

	// callback ile paid yap
	_, err = svc.ApplyCallback(context.Background(), payment.CallbackInput{
		MerchantOID: o.PaymentRef,
		Status:      "success",
		TotalAmount: "375000",
		Hash:        "gecerli-hash",
	}, []byte(`{}`))
	require.NoError(t, err)

	refunded, err := svc.Refund(context.Background(), o.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusRefunded, refunded.Status)
	assert.NotNil(t, refunded.RefundedAt)
}

// pay.Refund hata dönerse sipariş refunded yapılmaz (para gerçekte iade
// edilmemişken statü değiştirilirse esnaf/müşteri kaydı tutarsız kalır).
func TestService_Refund_SaglayiciHataVerirseStatuDegismez(t *testing.T) {
	pay := &fakePay{callbackOK: true, refundErr: fmt.Errorf("paytr: iade reddedildi")}
	svc, _, productID := setupServiceWithPay(t, pay)

	o, _, err := svc.Create(context.Background(), testCreateInput(productID), "127.0.0.1", nil)
	require.NoError(t, err)

	_, err = svc.ApplyCallback(context.Background(), payment.CallbackInput{
		MerchantOID: o.PaymentRef,
		Status:      "success",
		TotalAmount: "375000",
		Hash:        "gecerli-hash",
	}, []byte(`{}`))
	require.NoError(t, err)

	_, err = svc.Refund(context.Background(), o.ID)
	assert.Error(t, err)

	got, err := svc.Get(context.Background(), o.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPaid, got.Status)
}

// Elle statü ataması: yalnızca paid → delivered izinli.
func TestService_Update_AwaitingSiparisTeslimEdilemez(t *testing.T) {
	svc, _, productID := setupService(t)

	o, _, err := svc.Create(context.Background(), testCreateInput(productID), "127.0.0.1", nil)
	require.NoError(t, err)

	delivered := string(StatusDelivered)
	_, err = svc.Update(context.Background(), o.ID, &delivered, nil)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)

	got, err := svc.Get(context.Background(), o.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusAwaitingPayment, got.Status)
}

// paid sipariş elle delivered yapılabilir.
func TestService_Update_PaidSiparisTeslimEdilebilir(t *testing.T) {
	pay := &fakePay{callbackOK: true}
	svc, _, productID := setupServiceWithPay(t, pay)

	o, _, err := svc.Create(context.Background(), testCreateInput(productID), "127.0.0.1", nil)
	require.NoError(t, err)

	_, err = svc.ApplyCallback(context.Background(), payment.CallbackInput{
		MerchantOID: o.PaymentRef,
		Status:      "success",
		TotalAmount: "375000",
		Hash:        "gecerli-hash",
	}, []byte(`{}`))
	require.NoError(t, err)

	delivered := string(StatusDelivered)
	got, err := svc.Update(context.Background(), o.ID, &delivered, nil)
	require.NoError(t, err)
	assert.Equal(t, StatusDelivered, got.Status)
}

// awaiting_payment ve paid dışındaki statülerin elle atanması reddedilir
// (callback/refund akışıyla set edilirler).
func TestService_Update_OdemeStatuleriElleAtanamaz(t *testing.T) {
	svc, _, productID := setupService(t)

	o, _, err := svc.Create(context.Background(), testCreateInput(productID), "127.0.0.1", nil)
	require.NoError(t, err)

	for _, st := range []Status{StatusAwaitingPayment, StatusPaid, StatusRefunded} {
		s := string(st)
		_, err := svc.Update(context.Background(), o.ID, &s, nil)
		assert.ErrorIs(t, err, errorsx.ErrInvalidInput, "status=%s elle atanabilmemeli", st)
	}
}

// List default'ta awaiting_payment siparişleri gizler (esnaf görünümü) —
// ödeme başlatılıp tamamlanmamış sipariş çöp olarak listeye düşmesin.
func TestService_List_DefaultAwaitingPaymentGizler(t *testing.T) {
	pay := &fakePay{callbackOK: true}
	svc, _, productID := setupServiceWithPay(t, pay)

	oAwaiting, _, err := svc.Create(context.Background(), testCreateInput(productID), "127.0.0.1", nil)
	require.NoError(t, err)

	inPaid := testCreateInput(productID)
	oPaid, _, err := svc.Create(context.Background(), inPaid, "127.0.0.1", nil)
	require.NoError(t, err)
	_, err = svc.ApplyCallback(context.Background(), payment.CallbackInput{
		MerchantOID: oPaid.PaymentRef,
		Status:      "success",
		TotalAmount: "375000",
		Hash:        "gecerli-hash",
	}, []byte(`{}`))
	require.NoError(t, err)

	list, err := svc.List(context.Background(), "", 50, 0)
	require.NoError(t, err)

	var sawAwaiting, sawPaid bool
	for _, o := range list {
		if o.ID == oAwaiting.ID {
			sawAwaiting = true
		}
		if o.ID == oPaid.ID {
			sawPaid = true
		}
	}
	assert.False(t, sawAwaiting, "awaiting_payment sipariş default listede görünmemeli")
	assert.True(t, sawPaid, "paid sipariş listede görünmeli")

	// status filtresi verilince awaiting_payment de görünür (esnaf açıkça isterse)
	explicit, err := svc.List(context.Background(), string(StatusAwaitingPayment), 50, 0)
	require.NoError(t, err)
	var sawAwaitingExplicit bool
	for _, o := range explicit {
		if o.ID == oAwaiting.ID {
			sawAwaitingExplicit = true
		}
	}
	assert.True(t, sawAwaitingExplicit, "status=awaiting_payment açıkça istenirse görünmeli")
}
