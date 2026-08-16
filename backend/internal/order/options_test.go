package order_test

// Bu testler ayrı pakette (order_test) çünkü productoption'ı import
// ediyorlar ve productoption da order'ı import ediyor — aynı pakette
// olsalardı import döngüsü olurdu.

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/internal/payment"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/internal/productoption"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// optionTestEnv seçenek testleri için gerekli tüm servisleri ve test
// verisini bir arada tutar.
type optionTestEnv struct {
	ctx    context.Context
	pool   *pgxpool.Pool
	svc    *order.Service
	optSvc *productoption.Service

	productID int64
	pembeID   int64
	beyazID   int64

	// baskaUrunDegeriID ayrı bir ürüne bağlı, test ürününe AÇILMAMIŞ
	// grubun değeri — "ürüne kapalı grup" testinde kullanılır.
	baskaUrunDegeriID int64
}

// newOptionTestEnv gerçek DB ile ürün + "Ambalaj Rengi" grubu (zorunlu,
// Pembe #F0A6CA + Beyaz #FFFFFF) + ikinci bir ürüne bağlı ayrı bir grup
// kurar. OptionReader olarak gerçek productoption.Service geçirilir —
// doğrulama gerçek DB'ye karşı kanıtlanmalı, sahte değil.
func newOptionTestEnv(t *testing.T) *optionTestEnv {
	t.Helper()
	ctx := context.Background()
	pool := database.NewTestDB(t)

	var productID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ('51 Gül Buket', 'test', 1850.00, true) RETURNING id`).Scan(&productID))

	optSvc := productoption.NewService(productoption.NewStore(pool))

	grup, err := optSvc.CreateGroup(ctx, productoption.CreateGroupInput{
		Name: "Ambalaj Rengi", Kind: productoption.KindColor,
	})
	require.NoError(t, err)

	pembe, err := optSvc.CreateValue(ctx, productoption.CreateValueInput{
		GroupID: grup.ID, Name: "Pembe", SwatchHex: "#F0A6CA",
	})
	require.NoError(t, err)

	beyaz, err := optSvc.CreateValue(ctx, productoption.CreateValueInput{
		GroupID: grup.ID, Name: "Beyaz", SwatchHex: "#FFFFFF",
	})
	require.NoError(t, err)

	require.NoError(t, optSvc.SetProductGroups(ctx, productID, []productoption.ProductGroupLink{
		{GroupID: grup.ID},
	}))

	// İkinci ürün + ona bağlı ayrı grup/değer — test ürününe hiç açılmadı.
	var otherProductID int64
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ('Başka Ürün', 'test', 100.00, true) RETURNING id`).Scan(&otherProductID))

	otherGrup, err := optSvc.CreateGroup(ctx, productoption.CreateGroupInput{
		Name: "Kurdele Rengi", Kind: productoption.KindColor,
	})
	require.NoError(t, err)

	otherValue, err := optSvc.CreateValue(ctx, productoption.CreateValueInput{
		GroupID: otherGrup.ID, Name: "Kırmızı", SwatchHex: "#D93A34",
	})
	require.NoError(t, err)

	require.NoError(t, optSvc.SetProductGroups(ctx, otherProductID, []productoption.ProductGroupLink{
		{GroupID: otherGrup.ID},
	}))

	svc := order.NewService(order.NewStore(pool), product.NewStore(pool), optSvc,
		testOptionsDeliveryConfig(), &fakeOptionsPay{}, "https://example.com/ok", "https://example.com/fail")

	return &optionTestEnv{
		ctx:               ctx,
		pool:              pool,
		svc:               svc,
		optSvc:            optSvc,
		productID:         productID,
		pembeID:           pembe.ID,
		beyazID:           beyaz.ID,
		baskaUrunDegeriID: otherValue.ID,
	}
}

func testOptionsDeliveryConfig() order.DeliveryConfig {
	return order.DeliveryConfig{
		Fee:           "50",
		Slots:         []string{"09:00-12:00", "12:00-15:00", "15:00-18:00"},
		SameDayCutoff: "16:00",
		MaxDays:       30,
		Districts:     []string{"Ödemiş", "Tire", "Bayındır"},
		DistrictFees:  map[string]string{"Tire": "80"},
	}
}

// gecerliSiparis service_test.go'daki testCreateInput desenini izler,
// OptionValueIDs parametreyle set edilir.
func gecerliSiparis(productID int64, optionValueIDs []int64) order.CreateInput {
	return order.CreateInput{
		Items: []order.CreateItem{{
			ProductID:      productID,
			Quantity:       2,
			OptionValueIDs: optionValueIDs,
		}},
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

// fakeOptionsPay PaymentStarter test double'ı — bu dosyadaki testler ödeme
// akışını değil seçenek doğrulamasını hedefliyor.
type fakeOptionsPay struct{}

func (f *fakeOptionsPay) Start(_ context.Context, _ payment.StartInput) (payment.StartResult, error) {
	return payment.StartResult{Token: "tok"}, nil
}
func (f *fakeOptionsPay) VerifyCallback(in payment.CallbackInput) payment.CallbackResult {
	return payment.CallbackResult{OK: true, MerchantOID: in.MerchantOID}
}
func (f *fakeOptionsPay) Refund(_ context.Context, _ payment.RefundInput) error { return nil }

func TestCreate_SecimSiparise_Kopyalanir(t *testing.T) {
	env := newOptionTestEnv(t) // ürün + "Ambalaj Rengi" grubu + "Pembe" değeri

	o, _, err := env.svc.Create(env.ctx, gecerliSiparis(env.productID, []int64{env.pembeID}), "1.2.3.4", nil)

	require.NoError(t, err)
	require.Len(t, o.Items, 1)
	require.Len(t, o.Items[0].Options, 1)
	assert.Equal(t, "Ambalaj Rengi", o.Items[0].Options[0].GroupName)
	assert.Equal(t, "Pembe", o.Items[0].Options[0].ValueName)
	assert.Equal(t, "#F0A6CA", o.Items[0].Options[0].SwatchHex)
}

// Kopya olmanın anlamı: değer silinince eski sipariş bozulmamalı.
func TestCreate_DegerSilininceEskiSiparisBozulmaz(t *testing.T) {
	env := newOptionTestEnv(t)

	o, _, err := env.svc.Create(env.ctx, gecerliSiparis(env.productID, []int64{env.pembeID}), "1.2.3.4", nil)
	require.NoError(t, err)

	require.NoError(t, env.optSvc.DeleteValue(env.ctx, env.pembeID))

	tekrar, err := env.svc.Get(env.ctx, o.ID)

	require.NoError(t, err)
	require.Len(t, tekrar.Items[0].Options, 1)
	assert.Equal(t, "Pembe", tekrar.Items[0].Options[0].ValueName,
		"seçim kopya olduğu için değer silinse de kalmalı")
}

// Zorunluluk kavramı kaldırıldı (2026-08-16): müşteri sayfasında her grubun
// ilk değeri otomatik seçili geliyor, ama seçimsiz sipariş de reddedilmiyor —
// esnaf uygun olanı koyar. Eskiden bu senaryo hata veriyordu.
func TestCreate_SecimsizSiparisKabulEdilir(t *testing.T) {
	env := newOptionTestEnv(t)

	o, _, err := env.svc.Create(env.ctx, gecerliSiparis(env.productID, nil), "1.2.3.4", nil)

	require.NoError(t, err)
	require.Len(t, o.Items, 1)
	assert.Empty(t, o.Items[0].Options, "seçim yapılmadıysa kalem seçimsiz kalır")
}

func TestCreate_UruneKapaliGrubunDegeriReddedilir(t *testing.T) {
	env := newOptionTestEnv(t)

	_, _, err := env.svc.Create(env.ctx,
		gecerliSiparis(env.productID, []int64{env.baskaUrunDegeriID}), "1.2.3.4", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "geçersiz veya artık sunulmayan")
}

func TestCreate_PasifDegerReddedilir(t *testing.T) {
	env := newOptionTestEnv(t)

	yok := false
	_, err := env.optSvc.UpdateValue(env.ctx, env.pembeID, productoption.UpdateValueInput{IsActive: &yok})
	require.NoError(t, err)

	_, _, err = env.svc.Create(env.ctx, gecerliSiparis(env.productID, []int64{env.pembeID}), "1.2.3.4", nil)

	require.Error(t, err)
}

func TestCreate_AyniGruptanIkiDegerReddedilir(t *testing.T) {
	env := newOptionTestEnv(t)

	_, _, err := env.svc.Create(env.ctx,
		gecerliSiparis(env.productID, []int64{env.pembeID, env.beyazID}), "1.2.3.4", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "birden fazla seçim")
}

// N+1 regresyonu: çok kalemli sipariş listesinde seçimler tek sorguda
// çekilmeli. Bu test doğruluğu kilitler — her kalem KENDİ seçimlerini alır.
func TestList_HerKalemKendiSecimleriniAlir(t *testing.T) {
	env := newOptionTestEnv(t)

	_, _, err := env.svc.Create(env.ctx, gecerliSiparis(env.productID, []int64{env.pembeID}), "1.2.3.4", nil)
	require.NoError(t, err)
	_, _, err = env.svc.Create(env.ctx, gecerliSiparis(env.productID, []int64{env.beyazID}), "1.2.3.4", nil)
	require.NoError(t, err)

	// Bu testte siparişler ödeme callback'i almadığı için awaiting_payment'ta
	// kalır — List("") varsayılan olarak bu durumu esnaf görünümünden gizler
	// (ListVisible). Durumu açıkça istemek bu testin amacı değil (N+1
	// regresyonu), o yüzden filtre burada açıkça veriliyor.
	list, err := env.svc.List(env.ctx, string(order.StatusAwaitingPayment), 100, 0)

	require.NoError(t, err)
	require.Len(t, list, 2)
	for _, o := range list {
		require.Len(t, o.Items[0].Options, 1, "her kalem kendi seçimini almalı")
	}
}
