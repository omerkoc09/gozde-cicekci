# Faz 3 — Ödeme Entegrasyonu (PayTR) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Müşterinin site üzerinden PayTR ile gerçek ödeme yaparak sipariş vermesini, esnafın ödenmiş siparişleri panelde görüp gerektiğinde tek tıkla iade etmesini sağlamak.

**Architecture:** PayTR iFrame API. Müşteri "Öde"ye basınca sipariş `awaiting_payment` olarak kaydedilir, backend PayTR'den token alır, frontend iframe açar. Ödemenin gerçekliğine yalnızca PayTR'nin sunucu-sunucu callback'i (hash doğrulamalı) karar verir — sipariş `paid` olur. PayTR'ye özel her şey (`internal/payment/`) izole; `order` paketi yalnızca bir `PaymentProvider` arayüzü görür.

**Tech Stack:** Go 1.25 + Fiber v2, PostgreSQL (pgx), `shopspring/decimal`; Nuxt 4 SSR (public), Vuetify 3 (admin). PayTR iFrame API (HMAC-SHA256 + base64 hash).

## Global Constraints

- **Test komutu:** `make test` (`-p 1`). **`go test ./...` KULLANILMAZ** — tüm test paketleri aynı DB'yi paylaşır, paralel TRUNCATE birbirini siler (DURUM.md).
- **Para:** `decimal.Decimal`, asla float. PayTR kuruş bekler: tutar × 100, tam sayı.
- **Fiyata güvenilmez:** Sipariş oluşturulurken fiyat DB'den okunur (mevcut `service.go` kuralı korunur).
- **Ödeme kararı yalnızca callback'ten:** `merchant_ok_url`'e (müşteri yönlendirmesi) asla güvenilmez.
- **Callback idempotent:** Aynı `merchant_oid` için `callback_ok` işlenmişse tekrar iş yapılmaz, yine `OK` döner.
- **Hata formatı:** `{"error": {"code": "...", "message": "..."}}` — `api.WriteError` ile.
- **Durum seti:** `awaiting_payment`, `paid`, `delivered`, `refunded`. Eski `pending`/`confirmed`/`cancelled` kaldırılır.
- **Dil:** Kullanıcıya dönen tüm mesajlar Türkçe.
- **PayTR resmi formülleri (doğrulandı — dev.paytr.com):**
  - Token endpoint: `POST https://www.paytr.com/odeme/api/get-token`
  - Token hash string: `merchant_id + user_ip + merchant_oid + email + payment_amount + user_basket + no_installment + max_installment + currency + test_mode + merchant_salt`
  - Callback hash string: `merchant_oid + merchant_salt + status + total_amount`
  - İade endpoint: `POST https://www.paytr.com/odeme/iade`
  - İade hash string: `merchant_id + merchant_oid + return_amount + merchant_salt`
  - Her hash: `base64( HMAC-SHA256( hashString, merchant_key ) )`
  - Callback yanıtı: düz metin `OK` (aksi halde PayTR tekrar gönderir)

---

## File Structure

**Yeni backend paketi `internal/payment/`:**
- `provider.go` — `PaymentProvider` arayüzü + ortak tipler (`StartInput`, `StartResult`, `CallbackInput`, `CallbackResult`, `RefundInput`). Sağlayıcıdan bağımsız.
- `paytr.go` — `PayTRProvider`: token isteği, callback hash doğrulama, iade çağrısı, kuruş dönüşümü, sepet (`user_basket`) kodlama. PayTR'ye özel HER ŞEY burada.
- `paytr_test.go` — hash üretimi/doğrulaması, kuruş dönüşümü testleri (HTTP'siz, saf).
- `mock.go` — `MockProvider`: gerçek anahtar olmadan geliştirme/test.

**Değişen backend:**
- `internal/order/model.go` — yeni statüler, `NewOrder`/`Order`'a ödeme alanları.
- `internal/order/store.go` — statü CHECK'e uyum, `SetPaid`, `SetRefunded`, `payment_ref` güncelleme, `payment_events` yazma, awaiting_payment filtresi.
- `internal/order/service.go` — `Create` ödeme başlatır (IP + provider), `ApplyCallback`, `Refund`, statü geçiş doğrulaması.
- `internal/order/payment_provider.go` — `order` paketinin gördüğü dar `PaymentProvider` arayüzü (import döngüsünü önlemek için order kendi arayüzünü tanımlar).
- `internal/api/app/order_handler.go` — `create` IP geçirir + token döner; yeni `paymentCallback` handler.
- `internal/api/app/order_view.go` — `createOrderResponse`'a `paytr_token`.
- `internal/api/app/router.go` — `POST /payment/callback`.
- `internal/api/idare/order_handler.go` — `refund` handler; `list` default awaiting_payment gizler.
- `internal/api/idare/order_view.go` — `orderView`'e `paid_at`/`refunded_at`/`payment_ref`.
- `internal/api/idare/router.go` — `POST /orders/:id/refund`.
- `pkg/config/config.go` — PayTR ayarları.
- `cmd/server/main.go` — provider kurulumu ve wiring.
- `backend/migrations/000007_payment.up.sql` / `.down.sql`.

**Değişen frontend:**
- `frontend/app/app/pages/siparis/index.vue` — submit sonrası iframe akışı.
- `frontend/app/app/pages/siparis/hata.vue` — YENİ.
- `frontend/app/app/pages/siparis/tamam.vue` — metin: "ödeme onaylanıyor".
- `frontend/app/app/composables/useOrders.ts` + `types/api.ts` — `paytr_token`.
- `frontend/idare/` sipariş listesi/detay ekranları — yeni statü rozetleri + İade butonu.

---

## Task 1: Migration — ödeme şeması

**Files:**
- Create: `backend/migrations/000007_payment.up.sql`
- Create: `backend/migrations/000007_payment.down.sql`

**Interfaces:**
- Produces: `orders` tablosunda `paid_at TIMESTAMPTZ`, `refunded_at TIMESTAMPTZ`, `payment_ref TEXT` kolonları; `payment_events` tablosu; statü CHECK yeni set.

- [ ] **Step 1: up migration yaz**

`backend/migrations/000007_payment.up.sql`:

```sql
-- Statü seti Faz 3 ile değişiyor: awaiting_payment / paid / delivered / refunded.
-- Canlıda henüz gerçek sipariş yok (Faz 2+3 birlikte yayınlanacaktı), veri kaybı yok.
ALTER TABLE orders ALTER COLUMN status SET DEFAULT 'awaiting_payment';
ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('awaiting_payment','paid','delivered','refunded'));

ALTER TABLE orders
    ADD COLUMN paid_at     TIMESTAMPTZ,
    ADD COLUMN refunded_at TIMESTAMPTZ,
    ADD COLUMN payment_ref TEXT;

CREATE TABLE payment_events (
    id          BIGSERIAL PRIMARY KEY,
    order_id    BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    event_type  TEXT NOT NULL,   -- 'token_requested','callback_ok','callback_fail','refund'
    raw_payload JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_payment_events_order ON payment_events (order_id);
```

- [ ] **Step 2: down migration yaz**

`backend/migrations/000007_payment.down.sql`:

```sql
DROP TABLE payment_events;

ALTER TABLE orders
    DROP COLUMN payment_ref,
    DROP COLUMN refunded_at,
    DROP COLUMN paid_at;

ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending','confirmed','delivered','cancelled'));
ALTER TABLE orders ALTER COLUMN status SET DEFAULT 'pending';
```

- [ ] **Step 3: Migration'ı uygula ve doğrula**

Run: `cd backend && make migrate-up` (yoksa: `migrate -path migrations -database "$DATABASE_URL" up`)
Sonra: `psql "$DATABASE_URL" -c "\d orders"` → `paid_at`, `refunded_at`, `payment_ref` görünmeli; `\d payment_events` tabloyu göstermeli.
Expected: Hata yok, kolonlar ve tablo mevcut.

- [ ] **Step 4: Commit**

```bash
git add backend/migrations/000007_payment.up.sql backend/migrations/000007_payment.down.sql
git commit -m "feat(db): ödeme şeması — statü seti, ödeme alanları, payment_events"
```

---

## Task 2: Payment provider arayüzü ve ortak tipler

**Files:**
- Create: `backend/internal/payment/provider.go`
- Create: `backend/internal/payment/mock.go`
- Test: `backend/internal/payment/mock_test.go`

**Interfaces:**
- Produces:
  - Tip `StartInput struct { MerchantOID string; UserIP string; Email string; AmountKurus int64; Basket []BasketItem; OkURL string; FailURL string }`
  - Tip `BasketItem struct { Name string; PriceKurus int64; Quantity int }`
  - Tip `StartResult struct { Token string }`
  - Tip `CallbackInput struct { MerchantOID string; Status string; TotalAmount string; Hash string }`
  - Tip `CallbackResult struct { OK bool; MerchantOID string }` (OK=hash geçerli ve status=success)
  - Tip `RefundInput struct { MerchantOID string; ReturnAmountKurus int64 }`
  - Arayüz `Provider interface { Start(ctx, StartInput) (StartResult, error); VerifyCallback(CallbackInput) CallbackResult; Refund(ctx, RefundInput) error }`
  - `MockProvider` bu arayüzü sağlar.

- [ ] **Step 1: provider.go yaz**

`backend/internal/payment/provider.go`:

```go
package payment

import "context"

// BasketItem PayTR user_basket için tek satır.
type BasketItem struct {
	Name       string
	PriceKurus int64
	Quantity   int
}

// StartInput ödeme başlatmak için gereken her şey (sağlayıcıdan bağımsız).
type StartInput struct {
	MerchantOID string
	UserIP      string
	Email       string
	AmountKurus int64
	Basket      []BasketItem
	OkURL       string
	FailURL     string
}

type StartResult struct {
	Token string
}

// CallbackInput sağlayıcının bildirim POST'undan gelen ham alanlar.
type CallbackInput struct {
	MerchantOID string
	Status      string
	TotalAmount string
	Hash        string
}

// CallbackResult VerifyCallback sonucu. OK yalnızca hash geçerli VE status=success.
type CallbackResult struct {
	OK          bool
	MerchantOID string
}

type RefundInput struct {
	MerchantOID       string
	ReturnAmountKurus int64
}

// Provider ödeme sağlayıcısı arayüzü. PayTR bunu paytr.go'da,
// test/geliştirme mock.go'da sağlar.
type Provider interface {
	Start(ctx context.Context, in StartInput) (StartResult, error)
	VerifyCallback(in CallbackInput) CallbackResult
	Refund(ctx context.Context, in RefundInput) error
}
```

- [ ] **Step 2: mock.go yaz**

`backend/internal/payment/mock.go`:

```go
package payment

import "context"

// MockProvider gerçek PayTR anahtarları olmadan geliştirme/test için.
// Start sabit bir token döner; VerifyCallback status=="success" ise OK der
// (hash kontrolü yok — mock). Refund her zaman başarılı.
type MockProvider struct{}

func NewMockProvider() *MockProvider { return &MockProvider{} }

func (m *MockProvider) Start(_ context.Context, in StartInput) (StartResult, error) {
	return StartResult{Token: "mock-token-" + in.MerchantOID}, nil
}

func (m *MockProvider) VerifyCallback(in CallbackInput) CallbackResult {
	return CallbackResult{OK: in.Status == "success", MerchantOID: in.MerchantOID}
}

func (m *MockProvider) Refund(_ context.Context, _ RefundInput) error {
	return nil
}
```

- [ ] **Step 3: Failing test yaz**

`backend/internal/payment/mock_test.go`:

```go
package payment

import (
	"context"
	"testing"
)

func TestMockProvider_Start_TokenDoner(t *testing.T) {
	m := NewMockProvider()
	res, err := m.Start(context.Background(), StartInput{MerchantOID: "abc"})
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if res.Token != "mock-token-abc" {
		t.Fatalf("token = %q", res.Token)
	}
}

func TestMockProvider_VerifyCallback_SuccessOK(t *testing.T) {
	m := NewMockProvider()
	if !m.VerifyCallback(CallbackInput{MerchantOID: "abc", Status: "success"}).OK {
		t.Fatal("success için OK bekleniyordu")
	}
	if m.VerifyCallback(CallbackInput{MerchantOID: "abc", Status: "failed"}).OK {
		t.Fatal("failed için OK olmamalı")
	}
}
```

- [ ] **Step 4: Testi çalıştır — geçmeli**

Run: `cd backend && go test ./internal/payment/ -run TestMock -v`
Expected: PASS (kod zaten yazıldı — bu task saf Go, DB yok).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/payment/provider.go backend/internal/payment/mock.go backend/internal/payment/mock_test.go
git commit -m "feat(payment): sağlayıcı arayüzü + mock provider"
```

---

## Task 3: PayTR provider — hash, token, callback, iade

**Files:**
- Create: `backend/internal/payment/paytr.go`
- Test: `backend/internal/payment/paytr_test.go`

**Interfaces:**
- Consumes: `Provider`, `StartInput`, `CallbackInput`, `RefundInput`, `BasketItem` (Task 2).
- Produces:
  - `PayTRConfig struct { MerchantID, MerchantKey, MerchantSalt string; TestMode bool; HTTPClient *http.Client }`
  - `func NewPayTR(cfg PayTRConfig) *PayTRProvider`
  - `PayTRProvider` `Provider` arayüzünü sağlar.
  - `func KurusFromDecimal(d decimal.Decimal) int64` (tutar × 100, tam sayı) — Task 4/service bunu kullanır.

- [ ] **Step 1: Failing test yaz (hash + kuruş — saf, HTTP yok)**

`backend/internal/payment/paytr_test.go`:

```go
package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/shopspring/decimal"
)

// beklenenCallbackHash test yardımcı — dokümandaki formülü bağımsız üretir.
func beklenenCallbackHash(oid, salt, status, total, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(oid + salt + status + total))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestPayTR_VerifyCallback_DogruHashKabul(t *testing.T) {
	p := NewPayTR(PayTRConfig{MerchantID: "m", MerchantKey: "key", MerchantSalt: "salt"})
	hash := beklenenCallbackHash("oid1", "salt", "success", "18500", "key")

	res := p.VerifyCallback(CallbackInput{
		MerchantOID: "oid1", Status: "success", TotalAmount: "18500", Hash: hash,
	})
	if !res.OK {
		t.Fatal("doğru hash + success için OK bekleniyordu")
	}
}

func TestPayTR_VerifyCallback_YanlisHashRed(t *testing.T) {
	p := NewPayTR(PayTRConfig{MerchantID: "m", MerchantKey: "key", MerchantSalt: "salt"})
	res := p.VerifyCallback(CallbackInput{
		MerchantOID: "oid1", Status: "success", TotalAmount: "18500", Hash: "SAHTE",
	})
	if res.OK {
		t.Fatal("yanlış hash reddedilmeliydi — bedava sipariş riski")
	}
}

func TestPayTR_VerifyCallback_FailedStatusOKDegil(t *testing.T) {
	p := NewPayTR(PayTRConfig{MerchantID: "m", MerchantKey: "key", MerchantSalt: "salt"})
	hash := beklenenCallbackHash("oid1", "salt", "failed", "0", "key")
	res := p.VerifyCallback(CallbackInput{
		MerchantOID: "oid1", Status: "failed", TotalAmount: "0", Hash: hash,
	})
	if res.OK {
		t.Fatal("failed status için OK olmamalı (hash doğru olsa bile)")
	}
}

func TestKurusFromDecimal(t *testing.T) {
	cases := map[string]int64{
		"1850.00": 185000,
		"1850.50": 185050,
		"0.01":    1,
		"1234.56": 123456,
	}
	for in, want := range cases {
		d := decimal.RequireFromString(in)
		if got := KurusFromDecimal(d); got != want {
			t.Errorf("KurusFromDecimal(%s) = %d, beklenen %d", in, got, want)
		}
	}
}
```

- [ ] **Step 2: Testi çalıştır — derlenmemeli/fail**

Run: `cd backend && go test ./internal/payment/ -run 'TestPayTR|TestKurus' -v`
Expected: FAIL — `NewPayTR`, `PayTRConfig`, `KurusFromDecimal` tanımlı değil.

- [ ] **Step 3: paytr.go yaz**

`backend/internal/payment/paytr.go`:

```go
package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	tokenURL  = "https://www.paytr.com/odeme/api/get-token"
	refundURL = "https://www.paytr.com/odeme/iade"
)

type PayTRConfig struct {
	MerchantID   string
	MerchantKey  string
	MerchantSalt string
	TestMode     bool
	HTTPClient   *http.Client
}

type PayTRProvider struct {
	cfg    PayTRConfig
	client *http.Client
}

func NewPayTR(cfg PayTRConfig) *PayTRProvider {
	c := cfg.HTTPClient
	if c == nil {
		c = &http.Client{Timeout: 20 * time.Second}
	}
	return &PayTRProvider{cfg: cfg, client: c}
}

// KurusFromDecimal tutarı kuruşa çevirir (× 100, tam sayı). PayTR kuruş bekler.
func KurusFromDecimal(d decimal.Decimal) int64 {
	return d.Mul(decimal.NewFromInt(100)).Round(0).IntPart()
}

func (p *PayTRProvider) hmacBase64(s string) string {
	mac := hmac.New(sha256.New, []byte(p.cfg.MerchantKey))
	mac.Write([]byte(s))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (p *PayTRProvider) testModeStr() string {
	if p.cfg.TestMode {
		return "1"
	}
	return "0"
}

// encodeBasket PayTR user_basket formatı: [["ad","fiyat_kurus","adet"],...] → base64(JSON).
func encodeBasket(items []BasketItem) string {
	rows := make([][]any, 0, len(items))
	for _, it := range items {
		rows = append(rows, []any{it.Name, strconv.FormatInt(it.PriceKurus, 10), it.Quantity})
	}
	b, _ := json.Marshal(rows)
	return base64.StdEncoding.EncodeToString(b)
}

func (p *PayTRProvider) Start(ctx context.Context, in StartInput) (StartResult, error) {
	basket := encodeBasket(in.Basket)
	amount := strconv.FormatInt(in.AmountKurus, 10)
	noInstallment := "0"
	maxInstallment := "0"
	currency := "TL"

	// Token hash string (doküman sırası):
	// merchant_id + user_ip + merchant_oid + email + payment_amount +
	// user_basket + no_installment + max_installment + currency + test_mode + merchant_salt
	hashStr := p.cfg.MerchantID + in.UserIP + in.MerchantOID + in.Email + amount +
		basket + noInstallment + maxInstallment + currency + p.testModeStr() + p.cfg.MerchantSalt
	token := p.hmacBase64(hashStr)

	form := url.Values{
		"merchant_id":       {p.cfg.MerchantID},
		"user_ip":           {in.UserIP},
		"merchant_oid":      {in.MerchantOID},
		"email":             {in.Email},
		"payment_amount":    {amount},
		"paytr_token":       {token},
		"user_basket":       {basket},
		"no_installment":    {noInstallment},
		"max_installment":   {maxInstallment},
		"currency":          {currency},
		"test_mode":         {p.testModeStr()},
		"merchant_ok_url":   {in.OkURL},
		"merchant_fail_url": {in.FailURL},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return StartResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return StartResult{}, err
	}
	defer resp.Body.Close()

	var out struct {
		Status string `json:"status"`
		Token  string `json:"token"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return StartResult{}, fmt.Errorf("paytr yanıtı okunamadı: %w", err)
	}
	if out.Status != "success" {
		return StartResult{}, fmt.Errorf("paytr token reddetti: %s", out.Reason)
	}

	return StartResult{Token: out.Token}, nil
}

func (p *PayTRProvider) VerifyCallback(in CallbackInput) CallbackResult {
	// Callback hash string: merchant_oid + merchant_salt + status + total_amount
	expected := p.hmacBase64(in.MerchantOID + p.cfg.MerchantSalt + in.Status + in.TotalAmount)
	if !hmac.Equal([]byte(expected), []byte(in.Hash)) {
		return CallbackResult{OK: false, MerchantOID: in.MerchantOID}
	}
	return CallbackResult{OK: in.Status == "success", MerchantOID: in.MerchantOID}
}

func (p *PayTRProvider) Refund(ctx context.Context, in RefundInput) error {
	// return_amount tam TL değil kuruş değil — PayTR iade "TL cinsinden ondalık"
	// bekler (ör. "18.50"). Kuruşu 100'e bölerek string üretiyoruz.
	amount := decimal.New(in.ReturnAmountKurus, -2).StringFixed(2)

	// İade hash string: merchant_id + merchant_oid + return_amount + merchant_salt
	token := p.hmacBase64(p.cfg.MerchantID + in.MerchantOID + amount + p.cfg.MerchantSalt)

	form := url.Values{
		"merchant_id":   {p.cfg.MerchantID},
		"merchant_oid":  {in.MerchantOID},
		"return_amount": {amount},
		"paytr_token":   {token},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, refundURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out struct {
		Status string `json:"status"`
		ErrMsg string `json:"err_msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("paytr iade yanıtı okunamadı: %w", err)
	}
	if out.Status != "success" {
		return fmt.Errorf("paytr iade reddetti: %s", out.ErrMsg)
	}
	return nil
}
```

- [ ] **Step 4: Testi çalıştır — geçmeli**

Run: `cd backend && go test ./internal/payment/ -v`
Expected: PASS (tüm hash/kuruş testleri; Start/Refund'ın HTTP dalları bu testte çağrılmıyor).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/payment/paytr.go backend/internal/payment/paytr_test.go
git commit -m "feat(payment): PayTR provider — token, callback doğrulama, iade"
```

---

## Task 4: Order model + store — ödeme alanları ve statü işlemleri

**Files:**
- Modify: `backend/internal/order/model.go`
- Modify: `backend/internal/order/store.go`
- Test: `backend/internal/order/store_test.go` (ekleme)

**Interfaces:**
- Consumes: mevcut `Order`, `NewOrder`, `Store` (order paketi).
- Produces:
  - `Order`'a alanlar: `PaidAt *time.Time`, `RefundedAt *time.Time`, `PaymentRef string`.
  - Statü sabitleri: `StatusAwaitingPayment`, `StatusPaid`, `StatusDelivered`, `StatusRefunded` (eski `StatusPending`/`Confirmed`/`Cancelled` silinir).
  - `Store.SetPaid(ctx, merchantOID string) (*Order, error)` — payment_ref eşleşen siparişi `paid` yapar, `paid_at=now()`.
  - `Store.SetRefunded(ctx, id int64) (*Order, error)` — `refunded`, `refunded_at=now()`.
  - `Store.SetPaymentRef(ctx, id int64, ref string) error`.
  - `Store.GetByPaymentRef(ctx, ref string) (*Order, error)`.
  - `Store.AddPaymentEvent(ctx, orderID int64, eventType string, payload []byte) error`.
  - `Store.HasPaymentEvent(ctx, orderID int64, eventType string) (bool, error)` — idempotency.

- [ ] **Step 1: model.go güncelle**

`backend/internal/order/model.go` içinde statü bloğunu değiştir:

```go
const (
	StatusAwaitingPayment Status = "awaiting_payment"
	StatusPaid            Status = "paid"
	StatusDelivered       Status = "delivered"
	StatusRefunded        Status = "refunded"
)

func (s Status) Valid() bool {
	switch s {
	case StatusAwaitingPayment, StatusPaid, StatusDelivered, StatusRefunded:
		return true
	}
	return false
}
```

`Order` struct'a (Total'dan sonra) ekle:

```go
	PaidAt     *time.Time `json:"paid_at,omitempty"`
	RefundedAt *time.Time `json:"refunded_at,omitempty"`
	PaymentRef string     `json:"payment_ref,omitempty"`
```

- [ ] **Step 2: store.go — scanOrder ve orderSelect'i güncelle**

`orderSelect`'e yeni kolonlar (note'tan önce ekle, sıra scanOrder ile eşleşmeli):

```go
const orderSelect = `
	SELECT id, order_no, status,
	       buyer_name, buyer_phone, COALESCE(buyer_email, ''),
	       recipient_name, recipient_phone, delivery_address, delivery_district,
	       delivery_date, delivery_slot, COALESCE(card_message, ''),
	       items_total, delivery_fee, total,
	       paid_at, refunded_at, COALESCE(payment_ref, ''),
	       COALESCE(note, ''), created_at, updated_at
	FROM orders`
```

`scanOrder` Scan çağrısına ekle (total'dan sonra, note'tan önce):

```go
	err := row.Scan(&o.ID, &o.OrderNo, &o.Status,
		&o.BuyerName, &o.BuyerPhone, &o.BuyerEmail,
		&o.RecipientName, &o.RecipientPhone, &o.DeliveryAddress, &o.DeliveryDistrict,
		&o.DeliveryDate, &o.DeliverySlot, &o.CardMessage,
		&o.ItemsTotal, &o.DeliveryFee, &o.Total,
		&o.PaidAt, &o.RefundedAt, &o.PaymentRef,
		&o.Note, &o.CreatedAt, &o.UpdatedAt)
```

- [ ] **Step 3: store.go — yeni metodları ekle**

Dosya sonuna:

```go
// SetPaymentRef siparişe PayTR merchant_oid yazar (token isteğinden sonra).
func (s *Store) SetPaymentRef(ctx context.Context, id int64, ref string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE orders SET payment_ref = $2, updated_at = now() WHERE id = $1`, id, ref)
	return err
}

// GetByPaymentRef callback'te merchant_oid ile siparişi bulur.
func (s *Store) GetByPaymentRef(ctx context.Context, ref string) (*Order, error) {
	o, err := scanOrder(s.pool.QueryRow(ctx, orderSelect+` WHERE payment_ref = $1`, ref))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsx.ErrNotFound
		}
		return nil, err
	}
	items, err := s.itemsOf(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return o, nil
}

// SetPaid siparişi paid yapar (yalnızca awaiting_payment'tan). Zaten paid ise
// dokunmaz — idempotency callback'te AddPaymentEvent kontrolüyle sağlanır ama
// bu koşul çift güvenlik.
func (s *Store) SetPaid(ctx context.Context, id int64) (*Order, error) {
	ct, err := s.pool.Exec(ctx, `
		UPDATE orders SET status = 'paid', paid_at = now(), updated_at = now()
		WHERE id = $1 AND status = 'awaiting_payment'`, id)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, errorsx.ErrNotFound
	}
	return s.GetByID(ctx, id)
}

// SetRefunded siparişi refunded yapar (paid veya delivered'dan).
func (s *Store) SetRefunded(ctx context.Context, id int64) (*Order, error) {
	ct, err := s.pool.Exec(ctx, `
		UPDATE orders SET status = 'refunded', refunded_at = now(), updated_at = now()
		WHERE id = $1 AND status IN ('paid','delivered')`, id)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, errorsx.ErrNotFound
	}
	return s.GetByID(ctx, id)
}

// AddPaymentEvent denetim izi kaydı ekler.
func (s *Store) AddPaymentEvent(ctx context.Context, orderID int64, eventType string, payload []byte) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO payment_events (order_id, event_type, raw_payload) VALUES ($1,$2,$3)`,
		orderID, eventType, payload)
	return err
}

// HasPaymentEvent bu tip olay bu sipariş için işlenmiş mi (idempotency).
func (s *Store) HasPaymentEvent(ctx context.Context, orderID int64, eventType string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM payment_events WHERE order_id=$1 AND event_type=$2)`,
		orderID, eventType).Scan(&exists)
	return exists, err
}
```

- [ ] **Step 4: Failing test yaz — SetPaid / idempotency**

`backend/internal/order/store_test.go`'a ekle (mevcut test helper'larını kullan — dosyadaki `newTestStore`/`seedProduct` benzeri kurulum neyse ona uy):

```go
func TestStore_SetPaid_AwaitingdanPaid(t *testing.T) {
	store := newTestStore(t) // dosyadaki mevcut helper
	o := createTestOrder(t, store) // awaiting_payment döner

	paid, err := store.SetPaid(context.Background(), o.ID)
	if err != nil {
		t.Fatalf("SetPaid: %v", err)
	}
	if paid.Status != StatusPaid {
		t.Fatalf("status = %s, beklenen paid", paid.Status)
	}
	if paid.PaidAt == nil {
		t.Fatal("paid_at set edilmeliydi")
	}
}

func TestStore_HasPaymentEvent_Idempotency(t *testing.T) {
	store := newTestStore(t)
	o := createTestOrder(t, store)
	ctx := context.Background()

	has, _ := store.HasPaymentEvent(ctx, o.ID, "callback_ok")
	if has {
		t.Fatal("başlangıçta olay olmamalı")
	}
	if err := store.AddPaymentEvent(ctx, o.ID, "callback_ok", []byte(`{}`)); err != nil {
		t.Fatalf("AddPaymentEvent: %v", err)
	}
	has, _ = store.HasPaymentEvent(ctx, o.ID, "callback_ok")
	if !has {
		t.Fatal("olay eklendikten sonra true olmalı")
	}
}
```

> Not: `newTestStore` ve `createTestOrder` mevcut `store_test.go`'daki gerçek yardımcı adlarıyla değiştirilecek. Dosyayı önce oku, kullanılan kurulum desenini birebir kullan.

- [ ] **Step 5: Testi çalıştır**

Run: `cd backend && make test` (ya da tek paket için testi izole edemiyorsan tümü — go test ./... YASAK)
Alternatif tek paket: `cd backend && go test ./internal/order/ -run 'TestStore_SetPaid|TestStore_HasPaymentEvent' -v` (tek paket, güvenli)
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/order/model.go backend/internal/order/store.go backend/internal/order/store_test.go
git commit -m "feat(order): ödeme alanları, SetPaid/SetRefunded/payment_events store metodları"
```

---

## Task 5: Order service — Create ödeme başlatır, ApplyCallback, Refund

**Files:**
- Create: `backend/internal/order/payment_provider.go`
- Modify: `backend/internal/order/service.go`
- Test: `backend/internal/order/service_test.go` (ekleme)

**Interfaces:**
- Consumes: `Store` yeni metodları (Task 4), `product.Product` (mevcut).
- Produces:
  - Arayüz `PaymentStarter interface { Start(ctx, payment.StartInput) (payment.StartResult, error); VerifyCallback(payment.CallbackInput) payment.CallbackResult; Refund(ctx, payment.RefundInput) error }` — order paketinin gördüğü.
  - `NewService` imzası: `NewService(store *Store, prod ProductReader, cfg DeliveryConfig, pay PaymentStarter, okURL, failURL string)`.
  - `Service.Create(ctx, in, userIP string) (*Order, string, error)` — dönüş: sipariş, paytr_token, hata. (İmza değişti: userIP eklendi, token döndü.)
  - `Service.ApplyCallback(ctx, in payment.CallbackInput, rawPayload []byte) (accepted bool, err error)` — hash doğrular, paid yapar, idempotent.
  - `Service.Refund(ctx, id int64) (*Order, error)`.
  - `Service.Update` — statü geçiş doğrulaması (paid→delivered izinli; refunded'dan çıkış yok).

> **Import notu:** `order` paketi `payment` paketini import EDER (tipleri için) ama `payment` `order`'ı import etmez → döngü yok. Arayüz order'da tanımlı, somut PayTR provider main'de enjekte edilir.

- [ ] **Step 1: payment_provider.go yaz**

`backend/internal/order/payment_provider.go`:

```go
package order

import (
	"context"

	"github.com/omerkoc/cicekci/internal/payment"
)

// PaymentStarter order paketinin ödeme sağlayıcısından ihtiyaç duyduğu davranış.
// Somut implementasyon (PayTR/mock) main'de enjekte edilir.
type PaymentStarter interface {
	Start(ctx context.Context, in payment.StartInput) (payment.StartResult, error)
	VerifyCallback(in payment.CallbackInput) payment.CallbackResult
	Refund(ctx context.Context, in payment.RefundInput) error
}
```

- [ ] **Step 2: service.go — Service struct ve NewService güncelle**

```go
type Service struct {
	store   *Store
	prod    ProductReader
	cfg     DeliveryConfig
	pay     PaymentStarter
	okURL   string
	failURL string
}

func NewService(store *Store, prod ProductReader, cfg DeliveryConfig,
	pay PaymentStarter, okURL, failURL string) *Service {
	return &Service{store: store, prod: prod, cfg: cfg, pay: pay, okURL: okURL, failURL: failURL}
}
```

- [ ] **Step 3: service.go — Create'i güncelle (userIP + ödeme başlat)**

`Create` imzasını ve sonunu değiştir. Sipariş kaydedildikten SONRA ödeme başlat:

```go
func (s *Service) Create(ctx context.Context, in CreateInput, userIP string) (*Order, string, error) {
	if err := s.validateContact(&in); err != nil {
		return nil, "", err
	}
	if err := s.validateDelivery(in); err != nil {
		return nil, "", err
	}
	if len(in.Items) == 0 {
		return nil, "", fmt.Errorf("%w: sepet boş", errorsx.ErrInvalidInput)
	}

	items := make([]NewOrderItem, 0, len(in.Items))
	basket := make([]payment.BasketItem, 0, len(in.Items))
	itemsTotal := decimal.Zero

	for _, ci := range in.Items {
		if ci.Quantity <= 0 || ci.Quantity > maxQuantity {
			return nil, "", fmt.Errorf("%w: geçersiz adet", errorsx.ErrInvalidInput)
		}
		p, err := s.prod.GetByID(ctx, ci.ProductID)
		if err != nil {
			return nil, "", fmt.Errorf("%w: ürün bulunamadı", errorsx.ErrInvalidInput)
		}
		if !p.IsActive {
			return nil, "", fmt.Errorf("%w: %q artık satışta değil", errorsx.ErrInvalidInput, p.Name)
		}
		itemsTotal = itemsTotal.Add(p.Price.Mul(decimal.NewFromInt(int64(ci.Quantity))))
		items = append(items, NewOrderItem{
			ProductID: p.ID, ProductName: p.Name, PriceAtOrder: p.Price, Quantity: ci.Quantity,
		})
		basket = append(basket, payment.BasketItem{
			Name: p.Name, PriceKurus: payment.KurusFromDecimal(p.Price), Quantity: ci.Quantity,
		})
	}

	feeStr := s.cfg.Fee
	if districtFee, ok := s.cfg.DistrictFees[in.DeliveryDistrict]; ok {
		feeStr = districtFee
	}
	fee, err := decimal.NewFromString(feeStr)
	if err != nil {
		return nil, "", fmt.Errorf("teslimat ücreti okunamadı: %w", err)
	}
	total := itemsTotal.Add(fee)

	o, err := s.store.Create(ctx, NewOrder{
		BuyerName: in.BuyerName, BuyerPhone: in.BuyerPhone, BuyerEmail: in.BuyerEmail,
		RecipientName: in.RecipientName, RecipientPhone: in.RecipientPhone,
		DeliveryAddress: in.DeliveryAddress, DeliveryDistrict: in.DeliveryDistrict,
		DeliveryDate: in.DeliveryDate, DeliverySlot: in.DeliverySlot, CardMessage: in.CardMessage,
		ItemsTotal: itemsTotal, DeliveryFee: fee, Total: total, Items: items,
	})
	if err != nil {
		return nil, "", err
	}

	// merchant_oid PayTR için ALFANÜMERİK olmalı (tire kabul etmez). order_no
	// "2607-0042" → tireyi at + tekillik için sipariş id'si ekle.
	merchantOID := strings.ReplaceAll(o.OrderNo, "-", "") + "x" + strconv.FormatInt(o.ID, 10)
	if err := s.store.SetPaymentRef(ctx, o.ID, merchantOID); err != nil {
		return nil, "", err
	}

	email := in.BuyerEmail
	if email == "" {
		email = "noemail@example.com" // PayTR email zorunlu; müşteri girmediyse placeholder
	}

	res, err := s.pay.Start(ctx, payment.StartInput{
		MerchantOID: merchantOID,
		UserIP:      userIP,
		Email:       email,
		AmountKurus: payment.KurusFromDecimal(total),
		Basket:      basket,
		OkURL:       s.okURL,
		FailURL:     s.failURL,
	})
	if err != nil {
		return nil, "", fmt.Errorf("ödeme başlatılamadı: %w", err)
	}

	_ = s.store.AddPaymentEvent(ctx, o.ID, "token_requested", nil)
	o.PaymentRef = merchantOID
	return o, res.Token, nil
}
```

> `service.go` import bloğuna `strconv` ekle (`strings` zaten var).

- [ ] **Step 4: service.go — ApplyCallback ve Refund ekle**

```go
// ApplyCallback PayTR bildirimini işler. Dönüş accepted → PayTR'ye "OK"
// dönülüp dönülmeyeceği.
//
// Güvenlik kuralı: siparişi paid yapan TEK koşul res.OK (hash geçerli VE
// status=success). Bu yüzden sahte hash asla SetPaid'e ulaşmaz.
//
// accepted mantığı:
//   - Sipariş bulunamadı (GetByPaymentRef hata) → accepted=false. Var olmayan
//     oid; PayTR'ye OK DÖNME (sahte/eski istek).
//   - res.OK=false (hash geçersiz VEYA status=failed) → callback_fail izi bırak,
//     accepted=true. Gerekçe: GetByPaymentRef siparişi buldu → oid meşru; failed
//     ödemede de PayTR geçerli callback gönderir ve OK bekler. Para hareketi yok,
//     sipariş awaiting_payment kalır. OK dönmezsek PayTR aynı failed'i döngüde
//     tekrar gönderir.
//   - res.OK=true → idempotent şekilde paid yap, accepted=true.
func (s *Service) ApplyCallback(ctx context.Context, in payment.CallbackInput, rawPayload []byte) (bool, error) {
	res := s.pay.VerifyCallback(in)

	o, err := s.store.GetByPaymentRef(ctx, in.MerchantOID)
	if err != nil {
		return false, err
	}

	if !res.OK {
		_ = s.store.AddPaymentEvent(ctx, o.ID, "callback_fail", rawPayload)
		return true, nil
	}

	// Hash geçerli + success. Idempotency: zaten işlenmişse tekrar etme.
	already, err := s.store.HasPaymentEvent(ctx, o.ID, "callback_ok")
	if err != nil {
		return false, err
	}
	if already {
		return true, nil // no-op, ama OK dön
	}

	if _, err := s.store.SetPaid(ctx, o.ID); err != nil {
		return false, err
	}
	_ = s.store.AddPaymentEvent(ctx, o.ID, "callback_ok", rawPayload)
	return true, nil
}

// Refund PayTR iadesini çağırır, başarılıysa siparişi refunded yapar.
func (s *Service) Refund(ctx context.Context, id int64) (*Order, error) {
	o, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if o.Status != StatusPaid && o.Status != StatusDelivered {
		return nil, fmt.Errorf("%w: yalnızca ödenmiş sipariş iade edilebilir", errorsx.ErrInvalidInput)
	}
	if o.PaymentRef == "" {
		return nil, fmt.Errorf("%w: ödeme referansı yok", errorsx.ErrInvalidInput)
	}

	err = s.pay.Refund(ctx, payment.RefundInput{
		MerchantOID:       o.PaymentRef,
		ReturnAmountKurus: payment.KurusFromDecimal(o.Total),
	})
	if err != nil {
		return nil, fmt.Errorf("iade başarısız: %w", err)
	}

	refunded, err := s.store.SetRefunded(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = s.store.AddPaymentEvent(ctx, id, "refund", nil)
	return refunded, nil
}
```

> **Sadeleştirme kararı:** `hashValid` yukarıda gereksiz karmaşık. Basitleştir: `ApplyCallback` içinde `GetByPaymentRef` başarılıysa (oid meşru) callback_fail durumunda da `accepted=true` dön (para hareketi yok, sadece iz). Sahte hash + var olmayan oid zaten `GetByPaymentRef` ile elenir. Gerçek koruma: **`SetPaid` yalnızca `res.OK` (hash geçerli + success) olduğunda çağrılır.** Bu yüzden `hashValid` metodunu SİL, `!res.OK` dalında doğrudan `return true, nil` (iz bırakıp OK dön). Aşağıdaki test bunu doğrular.

- [ ] **Step 5: service.go — Update statü geçiş doğrulaması**

`Update`'i güncelle (geçersiz geçişi reddet):

```go
func (s *Service) Update(ctx context.Context, id int64, status *string, note *string) (*Order, error) {
	if status != nil {
		st := Status(*status)
		if !st.Valid() {
			return nil, fmt.Errorf("%w: geçersiz durum", errorsx.ErrInvalidInput)
		}
		// Elle izin verilen tek geçiş: paid → delivered. Ödeme/iade statüleri
		// (awaiting_payment/paid/refunded) callback ve refund akışıyla set edilir.
		if st != StatusDelivered {
			return nil, fmt.Errorf("%w: bu durum elle atanamaz", errorsx.ErrInvalidInput)
		}
		cur, err := s.store.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if cur.Status != StatusPaid {
			return nil, fmt.Errorf("%w: yalnızca ödenmiş sipariş teslim edilebilir", errorsx.ErrInvalidInput)
		}
	}
	return s.store.Update(ctx, id, status, note)
}
```

- [ ] **Step 6: service.go — List default'u awaiting_payment gizlesin**

`List`'i güncelle — status boşsa awaiting_payment hariç tut. En temiz yol: store.List'e "hariç tutulacak statü" yerine, status boşken store'un awaiting_payment'ı filtrelemesi. Basit çözüm: service'te status boşsa store'a özel bir sorgu; ama mevcut store.List imzasını korumak için store'a `ListVisible` ekle.

`store.go`'ya ekle:

```go
// ListVisible awaiting_payment HARİÇ tüm siparişleri listeler (esnaf görünümü).
func (s *Store) ListVisible(ctx context.Context, limit, offset int) ([]Order, error) {
	return s.listWhere(ctx, `status <> 'awaiting_payment'`, nil, limit, offset)
}
```

> Bu, mevcut `List`'i `listWhere` ortak yardımcısına refactor etmeni gerektirir. `List(status)` → status varsa `status = $1`, yoksa `listWhere` ile hepsini çeker. Refactor sırasında `itemsOfMany` batch mantığını koru. **Alternatif (daha az refactor):** service `List`'te status boşsa `store` üzerinde yeni `ListVisible` çağır, doluysa mevcut `List(status)`. Bunu tercih et — daha küçük değişiklik:

`service.go` `List`:

```go
func (s *Service) List(ctx context.Context, status string, limit, offset int) ([]Order, error) {
	if status != "" && !Status(status).Valid() {
		return nil, fmt.Errorf("%w: geçersiz durum", errorsx.ErrInvalidInput)
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	if status == "" {
		return s.store.ListVisible(ctx, limit, offset)
	}
	return s.store.List(ctx, status, limit, offset)
}
```

Ve `store.go`'da `ListVisible`'ı `List`'in gövdesini paylaşacak şekilde yaz (itemsOfMany dahil). En basiti: `List`'i `status <> 'awaiting_payment'` destekleyecek biçimde küçük bir iç fonksiyona çıkarmak. Uygulayıcı `List` ve `ListVisible`'ı tek `listQuery(ctx, whereClause, args, limit, offset)` yardımcısı üzerinden yazsın; kalem doldurma (itemsOfMany) tek yerde kalsın.

- [ ] **Step 7: Failing test yaz — callback + refund + geçiş**

`backend/internal/order/service_test.go`'a ekle. Mevcut test kurulumunu (fake product reader, test store) kullan; `PaymentStarter` için test double yaz:

```go
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

func TestService_ApplyCallback_SuccessPaidYapar(t *testing.T) {
	// svc'yi fakePay{callbackOK:true} ile kur, bir sipariş oluştur (Create),
	// payment_ref al, ApplyCallback çağır, sipariş paid olmalı.
	// ... (mevcut newTestService helper'ına pay parametresi eklenerek)
}

func TestService_ApplyCallback_Idempotent(t *testing.T) {
	// Aynı callback iki kez → sipariş bir kez paid, ikinci çağrı accepted=true no-op.
}

func TestService_ApplyCallback_HashGecersizPaidYapmaz(t *testing.T) {
	// fakePay{callbackOK:false} → sipariş awaiting_payment kalır, SetPaid çağrılmaz.
}

func TestService_Refund_YalnizPaidVeyaDelivered(t *testing.T) {
	// awaiting_payment sipariş → Refund ErrInvalidInput.
	// paid sipariş → Refund başarılı, status refunded.
}

func TestService_Update_AwaitingSiparisTeslimEdilemez(t *testing.T) {
	// awaiting_payment → Update(status=delivered) reddedilmeli.
}
```

> Test gövdelerini mevcut `service_test.go`'daki gerçek helper adlarıyla doldur. `newTestService` benzeri kurulum `pay` parametresi alacak şekilde güncellenmeli.

- [ ] **Step 8: Testi çalıştır**

Run: `cd backend && go test ./internal/order/ -v`
Expected: PASS (tek paket, güvenli).

- [ ] **Step 9: Commit**

```bash
git add backend/internal/order/
git commit -m "feat(order): ödeme başlatma, callback işleme, iade, statü geçiş doğrulaması"
```

---

## Task 6: Public API handler — token dönüşü + callback ucu

**Files:**
- Modify: `backend/internal/api/app/order_handler.go`
- Modify: `backend/internal/api/app/order_view.go`
- Modify: `backend/internal/api/app/router.go`
- Test: `backend/internal/api/app/order_handler_test.go` (ekleme)

**Interfaces:**
- Consumes: `order.Service.Create(ctx, in, userIP)` → `(*Order, token, err)`; `order.Service.ApplyCallback`.
- Produces: `POST /api/orders` yanıtı `{order_no, total, paytr_token}`; `POST /api/payment/callback` düz `OK`.

- [ ] **Step 1: order_view.go — response'a token ekle**

```go
type createOrderResponse struct {
	OrderNo   string `json:"order_no"`
	Total     string `json:"total"`
	PaytrToken string `json:"paytr_token"`
}

func toCreateOrderResponse(o *order.Order, token string) createOrderResponse {
	return createOrderResponse{
		OrderNo:    o.OrderNo,
		Total:      o.Total.StringFixed(2),
		PaytrToken: token,
	}
}
```

- [ ] **Step 2: order_handler.go — create IP geçir + token dön; callback handler ekle**

`create` sonunu değiştir:

```go
	o, token, err := h.svc.Create(c.Context(), in, c.IP())
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toCreateOrderResponse(o, token))
```

Yeni handler (aynı dosyaya):

```go
// paymentCallback PayTR sunucu-sunucu bildirimi. Yanıt DÜZ METİN "OK" olmalı,
// yoksa PayTR tekrar tekrar gönderir. Ödeme kararı YALNIZCA burada verilir.
func (h *orderHandler) paymentCallback(c *fiber.Ctx) error {
	in := order.CallbackFromForm(
		c.FormValue("merchant_oid"),
		c.FormValue("status"),
		c.FormValue("total_amount"),
		c.FormValue("hash"),
	)
	raw := c.Body() // ham gövde denetim izi için

	accepted, err := h.svc.ApplyCallback(c.Context(), in, raw)
	if err != nil || !accepted {
		// Hata/geçersiz → PayTR'ye OK DÖNME (tekrar denesin / sahte reddedilsin).
		// PayTR "OK dışı" yanıtı başarısız sayar.
		return c.Status(fiber.StatusBadRequest).SendString("FAIL")
	}
	return c.SendString("OK")
}
```

> `order.CallbackFromForm` küçük bir yardımcı: `payment.CallbackInput` üretir ama handler `payment` paketini import etmesin diye `order` paketinde ince bir wrapper. **Alternatif:** handler doğrudan `payment.CallbackInput{...}` kursun (app paketi zaten `order`'ı import ediyor, `payment`'ı da import edebilir — döngü yok). Bunu tercih et, `CallbackFromForm` gereksiz:

```go
	in := payment.CallbackInput{
		MerchantOID: c.FormValue("merchant_oid"),
		Status:      c.FormValue("status"),
		TotalAmount: c.FormValue("total_amount"),
		Hash:        c.FormValue("hash"),
	}
```

`order_handler.go` import'una `"github.com/omerkoc/cicekci/internal/payment"` ekle.

- [ ] **Step 3: router.go — callback rotası**

`Register` içine (auth yok — PayTR sunucusu çağırır):

```go
	router.Post("/orders", oh.create)
	router.Post("/payment/callback", oh.paymentCallback)
	router.Get("/delivery-config", oh.deliveryConfig)
```

- [ ] **Step 4: Test — callback OK/FAIL yanıtı**

`order_handler_test.go`'a ekle (mevcut Fiber test app kurulumunu kullan). Fake service yerine gerçek service + mock pay + test DB (mevcut desen neyse). En az: callback başarılı senaryo düz `OK` döner; hash geçersiz senaryo `OK` DÖNMEZ.

```go
func TestPaymentCallback_BasariliOKDoner(t *testing.T) {
	// mock provider ile: geçerli callback → gövde "OK", status 200.
}
func TestPaymentCallback_GecersizFAILDoner(t *testing.T) {
	// var olmayan merchant_oid → gövde "OK" DEĞİL.
}
```

- [ ] **Step 5: Testi çalıştır**

Run: `cd backend && go test ./internal/api/app/ -run TestPaymentCallback -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/api/app/
git commit -m "feat(api): ödeme token dönüşü + PayTR callback ucu"
```

---

## Task 7: Admin API — iade ucu + görünüm

**Files:**
- Modify: `backend/internal/api/idare/order_handler.go`
- Modify: `backend/internal/api/idare/order_view.go`
- Modify: `backend/internal/api/idare/router.go`
- Test: `backend/internal/api/idare/order_handler_test.go` (ekleme)

**Interfaces:**
- Consumes: `order.Service.Refund(ctx, id)`.
- Produces: `POST /api/admin/orders/:id/refund`; `orderView`'de `paid_at`/`refunded_at`/`payment_ref`.

- [ ] **Step 1: order_view.go — ödeme alanları**

`orderView` struct'a (Total'dan sonra) ekle:

```go
	PaidAt      *time.Time `json:"paid_at"`
	RefundedAt  *time.Time `json:"refunded_at"`
	PaymentRef  string     `json:"payment_ref"`
```

`toOrderView` içinde doldur:

```go
		PaidAt:     o.PaidAt,
		RefundedAt: o.RefundedAt,
		PaymentRef: o.PaymentRef,
```

- [ ] **Step 2: order_handler.go — refund handler**

```go
func (h *orderHandler) refund(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}
	o, err := h.svc.Refund(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toOrderView(o))
}
```

- [ ] **Step 3: router.go — refund rotası**

`protected` grubuna:

```go
	protected.Get("/orders", oh.list)
	protected.Get("/orders/:id", oh.get)
	protected.Patch("/orders/:id", oh.update)
	protected.Post("/orders/:id/refund", oh.refund)
```

- [ ] **Step 4: Test — auth + iade**

`order_handler_test.go`'a ekle (mevcut desen: çoğu handler testi auth kontrolüdür):

```go
func TestRefund_AuthGerekli(t *testing.T) {
	// token'sız POST /orders/1/refund → 401.
}
func TestRefund_PaidSiparisRefunded(t *testing.T) {
	// mock pay ile: paid sipariş → refund → status refunded (200).
}
```

- [ ] **Step 5: Testi çalıştır**

Run: `cd backend && go test ./internal/api/idare/ -run TestRefund -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/api/idare/
git commit -m "feat(api): admin iade ucu + ödeme bilgisi görünümü"
```

---

## Task 8: Config + wiring (main.go)

**Files:**
- Modify: `backend/pkg/config/config.go`
- Modify: `backend/cmd/server/main.go`
- Modify: `.env.example`, `.env.prod.example`
- Test: `backend/pkg/config/config_test.go` (varsa ekleme)

**Interfaces:**
- Consumes: `payment.NewPayTR`, `payment.NewMockProvider`, `order.NewService` yeni imza.
- Produces: PayTR config alanları; provider seçimi (test/mock vs gerçek).

- [ ] **Step 1: config.go — PayTR alanları**

`Config` struct'a:

```go
	// PayTR ödeme (Faz 3). Anahtarlar boşsa mock provider kullanılır.
	PayTRMerchantID   string
	PayTRMerchantKey  string
	PayTRMerchantSalt string
	PayTRTestMode     bool
	PaymentCallbackURL string
```

`Load` içinde (loadDelivery çağrısından sonra) `loadPayment(cfg)` çağır ve fonksiyonu yaz:

```go
func loadPayment(cfg *Config) {
	cfg.PayTRMerchantID = os.Getenv("PAYTR_MERCHANT_ID")
	cfg.PayTRMerchantKey = os.Getenv("PAYTR_MERCHANT_KEY")
	cfg.PayTRMerchantSalt = os.Getenv("PAYTR_MERCHANT_SALT")
	cfg.PayTRTestMode = os.Getenv("PAYTR_TEST_MODE") != "0" // varsayılan test modu açık
	cfg.PaymentCallbackURL = os.Getenv("PAYMENT_CALLBACK_URL")
	if cfg.PaymentCallbackURL == "" {
		cfg.PaymentCallbackURL = cfg.SiteURL + "/api/payment/callback"
	}
}

// PayTRConfigured gerçek PayTR anahtarları var mı.
func (c *Config) PayTRConfigured() bool {
	return c.PayTRMerchantID != "" && c.PayTRMerchantKey != "" && c.PayTRMerchantSalt != ""
}
```

- [ ] **Step 2: main.go — provider seçimi ve wiring**

`orderSvc` kurulumunu değiştir:

```go
	var payProvider order.PaymentStarter
	if cfg.PayTRConfigured() {
		payProvider = payment.NewPayTR(payment.PayTRConfig{
			MerchantID:   cfg.PayTRMerchantID,
			MerchantKey:  cfg.PayTRMerchantKey,
			MerchantSalt: cfg.PayTRMerchantSalt,
			TestMode:     cfg.PayTRTestMode,
		})
		log.Println("ödeme: PayTR provider aktif (test_mode:", cfg.PayTRTestMode, ")")
	} else {
		payProvider = payment.NewMockProvider()
		log.Println("ödeme: MOCK provider (PayTR anahtarları yok)")
	}

	// Public site ödeme sonrası buralara döner.
	okURL := cfg.SiteURL + "/siparis/tamam"
	failURL := cfg.SiteURL + "/siparis/hata"

	orderSvc := order.NewService(order.NewStore(pool), product.NewStore(pool),
		deliveryCfg, payProvider, okURL, failURL)
```

`main.go` import'una `"github.com/omerkoc/cicekci/internal/payment"` ekle.

- [ ] **Step 3: .env.example ve .env.prod.example güncelle**

Her ikisine ekle:

```
# PayTR ödeme (Faz 3). Anahtarlar boşsa mock provider (geliştirme) kullanılır.
PAYTR_MERCHANT_ID=
PAYTR_MERCHANT_KEY=
PAYTR_MERCHANT_SALT=
PAYTR_TEST_MODE=1
# Boşsa SITE_URL + /api/payment/callback olur. PayTR panelinde bildirim URL'i buna eşit olmalı.
PAYMENT_CALLBACK_URL=
```

- [ ] **Step 4: Derleme + tüm testler**

Run: `cd backend && go build ./... && make test`
Expected: Derleme temiz, tüm testler PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/pkg/config/config.go backend/cmd/server/main.go .env.example .env.prod.example
git commit -m "feat(config): PayTR ayarları + provider wiring (anahtar yoksa mock)"
```

---

## Task 9: Public frontend — iframe ödeme akışı

**Files:**
- Modify: `frontend/app/app/composables/useOrders.ts`
- Modify: `frontend/app/app/types/api.ts`
- Modify: `frontend/app/app/pages/siparis/index.vue`
- Create: `frontend/app/app/pages/siparis/hata.vue`
- Modify: `frontend/app/app/pages/siparis/tamam.vue`

**Interfaces:**
- Consumes: `POST /api/orders` yanıtı `{order_no, total, paytr_token}`.
- Produces: iframe ödeme akışı; `/siparis/hata` sayfası.

- [ ] **Step 1: types/api.ts — CreateOrderResult'a token**

`CreateOrderResult` tipine ekle:

```ts
export interface CreateOrderResult {
  order_no: string
  total: string
  paytr_token: string
}
```

- [ ] **Step 2: siparis/index.vue — submit sonrası iframe aç**

`gonder` fonksiyonundaki başarılı dalı değiştir. Sepeti HEMEN temizleme — ödeme henüz olmadı. Bunun yerine token ile PayTR iframe'ini aç:

```ts
  try {
    const sonuc = await createOrder({ /* ...mevcut... */ })

    // Sepet burada TEMİZLENMEZ — ödeme henüz yapılmadı. Müşteri iframe'de
    // öderse PayTR merchant_ok_url ile /siparis/tamam'a döner; orada temizlenir.
    // İframe'i token ile aç:
    await odemeIframiAc(sonuc.paytr_token, sonuc.order_no)
  }
  catch (e) {
    hata.value = apiErrorMessage(e)
  }
  finally {
    gonderiliyor.value = false
  }
```

Aynı `<script setup>` içine iframe açma yardımcısı:

```ts
// PayTR iframe'ini token ile gömer. iframeResizer scriptini bir kez yükler.
async function odemeIframiAc(token: string, orderNo: string) {
  // order_no'yu tamam sayfası için sakla — PayTR redirect'inde query taşımıyoruz.
  sessionStorage.setItem('bekleyen_siparis_no', orderNo)

  await yukleIframeResizer()
  odemeToken.value = token
  odemeAcik.value = true
  await nextTick()
  // @ts-expect-error iFrameResize global (PayTR scripti)
  window.iFrameResize({}, '#paytriframe')
}

function yukleIframeResizer(): Promise<void> {
  return new Promise((resolve) => {
    if (document.getElementById('paytr-iframe-resizer')) {
      resolve()
      return
    }
    const s = document.createElement('script')
    s.id = 'paytr-iframe-resizer'
    s.src = 'https://www.paytr.com/js/iframeResizer.min.js'
    s.onload = () => resolve()
    document.head.appendChild(s)
  })
}
```

Reaktif state (script setup üstüne):

```ts
const odemeAcik = ref(false)
const odemeToken = ref('')
```

Template'e iframe bloğu (form'dan sonra, `odemeAcik` true olunca form yerine göster ya da modal):

```html
<div v-if="odemeAcik" class="mt-8">
  <iframe
    id="paytriframe"
    :src="`https://www.paytr.com/odeme/guvenli/${odemeToken}`"
    frameborder="0"
    scrolling="no"
    style="width: 100%;"
  />
</div>
```

- [ ] **Step 3: siparis/tamam.vue — metin + sepet temizleme**

Sepet burada temizlenir (ödeme başarıyla döndü) ve mesaj "ödeme onaylanıyor" olur:

```ts
const { clear } = useCart()
const orderNo = computed(() =>
  String(route.query.no ?? sessionStorage.getItem('bekleyen_siparis_no') ?? ''))

onMounted(() => {
  if (!orderNo.value) {
    navigateTo('/')
    return
  }
  clear() // ödeme başarılı döndü, sepeti temizle
  sessionStorage.removeItem('bekleyen_siparis_no')
})
```

Template'te başlık altındaki açıklamayı değiştir (ödemenin kesinliği callback'te onaylandığı için):

```html
<p class="mt-4 text-body-lg text-on-surface-variant">
  Ödemeniz alındı, siparişiniz hazırlanıyor.
  <span v-if="orderNo">Sipariş numaranız: <strong class="text-primary">{{ orderNo }}</strong></span>
</p>
```

- [ ] **Step 4: siparis/hata.vue — YENİ**

```vue
<script setup lang="ts">
useSeoMeta({
  title: 'Ödeme Tamamlanamadı | Gözde Tasarım Çiçekçilik',
  robots: 'noindex, nofollow',
})
</script>

<template>
  <div class="site-container py-20 text-center md:py-28">
    <span class="mx-auto flex size-16 items-center justify-center rounded-full bg-error/10">
      <Icon name="material-symbols:close" size="32" class="text-error" />
    </span>
    <h1 class="mt-8 font-serif text-3xl text-primary md:text-4xl">
      Ödeme Tamamlanamadı
    </h1>
    <p class="mt-4 text-body-lg text-on-surface-variant">
      Ödemeniz alınamadı. Sepetiniz korundu — dilerseniz tekrar deneyebilirsiniz.
    </p>
    <NuxtLink to="/siparis" class="mt-8 inline-block rounded bg-secondary px-6 py-3 text-on-secondary">
      Siparişe Dön
    </NuxtLink>
  </div>
</template>
```

> Not: Sepet `/siparis/hata`'da temizlenmez (korunur). `/siparis/tamam`'da temizlenir. `bekleyen_siparis_no` sessionStorage'ı hata durumunda kalabilir; zararsız, sonraki başarılı siparişte üzerine yazılır.

- [ ] **Step 5: CSP notu — Nitro/Caddy**

PayTR iframe + script harici origin. `frontend/app/nuxt.config.ts` veya `deploy/Caddyfile`'da CSP varsa `frame-src https://www.paytr.com` ve `script-src https://www.paytr.com` eklenmeli. **Önce kontrol et:** mevcut CSP header'ı var mı?

Run: `grep -rn "Content-Security-Policy\|frame-src\|script-src" frontend/app/nuxt.config.ts deploy/Caddyfile`
- CSP yoksa: adım atlanır (tarayıcı iframe'i engellemez).
- CSP varsa: `www.paytr.com` origin'i `frame-src` ve `script-src`'ye eklenir.

- [ ] **Step 6: Frontend build + mevcut testler**

Run: `cd frontend/app && pnpm test && pnpm build`
Expected: `useCart` testleri PASS, build başarılı.

- [ ] **Step 7: Commit**

```bash
git add frontend/app/
git commit -m "feat(public): PayTR iframe ödeme akışı + hata sayfası"
```

---

## Task 10: Admin frontend — statü rozetleri + İade butonu

**Files:**
- Modify: `frontend/idare/` sipariş listesi ekranı (mevcut `/siparisler` sayfası)
- Modify: `frontend/idare/` sipariş detay ekranı

**Interfaces:**
- Consumes: `GET /api/admin/orders` (awaiting_payment gizli), `orderView` yeni alanlar; `POST /api/admin/orders/:id/refund`.
- Produces: Yeni statü rozetleri (Ödendi/Teslim Edildi/İade Edildi), ödeme bilgi bloğu, İade butonu.

> **Önce keşfet:** `frontend/idare/` içinde sipariş ekranlarının gerçek dosya yollarını ve mevcut statü/rozet kod desenini bul. Faz 2'de bu ekranlar yazıldı; onların yapısını birebir takip et (mevcut `Currency.ts`, statü çeviri haritası vb.).

Run: `grep -rln "siparis\|order\|pending\|confirmed" frontend/idare/src --include=*.vue --include=*.ts | grep -iv node_modules`

- [ ] **Step 1: Statü etiket haritasını güncelle**

Mevcut statü→Türkçe etiket ve renk haritasını bul; eski `pending/confirmed/delivered/cancelled` yerine yeni set:

```ts
const STATU_ETIKET: Record<string, string> = {
  paid: 'Ödendi',
  delivered: 'Teslim Edildi',
  refunded: 'İade Edildi',
  // awaiting_payment listede görünmez ama detayda olabilir:
  awaiting_payment: 'Ödeme Bekliyor',
}
```

Renk haritası da aynı anahtarlarla güncellenir (mevcut renk desenini takip et).

- [ ] **Step 2: Sipariş detayına ödeme bilgi bloğu**

Detay ekranına, mevcut alanların yanına ödeme bilgisi göster: durum, `paid_at` (varsa, TR formatı — mevcut `Currency.ts`/tarih formatlayıcı neyse), `payment_ref`.

- [ ] **Step 3: İade butonu**

Sipariş `paid` veya `delivered` ise "İade Et" butonu göster. Tıklanınca onay diyaloğu (mevcut dialog deseni), onaylanırsa `POST /api/admin/orders/:id/refund`. Başarılıysa listeyi/detayı yenile. Hata mesajı backend'den (Türkçe) gösterilir.

```ts
async function iadeEt(id: number) {
  // onay diyaloğu... sonra:
  await api.post(`/orders/${id}/refund`)
  // listeyi yenile
}
```

- [ ] **Step 4: "Teslim Edildi" butonu yeni sete uysun**

`paid` siparişte "Teslim Edildi" butonu `PATCH /orders/:id` body `{status:'delivered'}` gönderir (mevcut update deseni; eski `confirmed` adımı kalktı).

- [ ] **Step 5: Admin build**

Run: `cd frontend/idare && pnpm build`
Expected: Build başarılı, tip hatası yok (yazılan panel kodu tipli kalmalı).

- [ ] **Step 6: Commit**

```bash
git add frontend/idare/
git commit -m "feat(idare): ödeme statü rozetleri, ödeme bilgisi, tek tıkla iade"
```

---

## Task 11: Uçtan uca doğrulama (mock provider)

**Files:** (yeni dosya yok — manuel/E2E doğrulama)

**Interfaces:** Tüm akış mock provider ile çalışmalı.

- [ ] **Step 1: Backend'i mock provider ile başlat**

`.env`'de PayTR anahtarları boş bırak (mock aktif olur). `make run` (veya docker compose).
Log'da `ödeme: MOCK provider` görünmeli.

- [ ] **Step 2: Sipariş oluştur — token dönüyor mu**

Run:
```bash
curl -s -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -d '{"items":[{"product_id":1,"quantity":1}],
       "buyer":{"name":"Test","phone":"5551112233"},
       "recipient":{"name":"Alıcı","phone":"5559998877"},
       "delivery":{"address":"Adres","district":"Ödemiş","date":"<yarın>","slot":"12:00-15:00"}}'
```
Expected: `{"order_no":"...","total":"...","paytr_token":"mock-token-..."}` (product_id gerçek aktif bir ürün olmalı).

- [ ] **Step 2b: Sipariş awaiting_payment mı**

Run: `psql "$DATABASE_URL" -c "SELECT order_no,status,payment_ref FROM orders ORDER BY id DESC LIMIT 1;"`
Expected: `status = awaiting_payment`, `payment_ref` dolu.

- [ ] **Step 3: Mock callback gönder — paid oluyor mu**

merchant_oid'i yukarıdaki payment_ref'ten al. Mock provider hash'e bakmaz, status=success yeter:

```bash
curl -s -X POST http://localhost:8080/api/payment/callback \
  -d "merchant_oid=<payment_ref>&status=success&total_amount=<kurus>&hash=x"
```
Expected: gövde `OK`. Sonra:
`psql "$DATABASE_URL" -c "SELECT status,paid_at FROM orders WHERE payment_ref='<payment_ref>';"`
→ `status = paid`, `paid_at` dolu.

- [ ] **Step 4: Idempotency — callback'i tekrar gönder**

Aynı curl'ü tekrar çalıştır. Expected: yine `OK`, sipariş hâlâ tek `paid` (çift işlem yok). `payment_events`'te tek `callback_ok`:
`psql "$DATABASE_URL" -c "SELECT event_type,count(*) FROM payment_events GROUP BY event_type;"`

- [ ] **Step 5: Admin listesi awaiting_payment gizliyor mu**

İkinci bir sipariş oluştur (callback GÖNDERME → awaiting kalsın). Admin token'la:
Run: `curl -s http://localhost:8080/api/admin/orders -H "Cookie: <auth>"` (veya login akışı)
Expected: Yalnızca `paid` sipariş görünür, awaiting_payment görünmez.

- [ ] **Step 6: İade — refunded oluyor mu**

Run: `curl -s -X POST http://localhost:8080/api/admin/orders/<paid_id>/refund -H "Cookie: <auth>"`
Expected: `status=refunded`, `refunded_at` dolu. `payment_events`'te `refund` kaydı.

- [ ] **Step 7: Tarayıcıda public akış**

Public siteyi çalıştır (`cd frontend/app && pnpm dev`), sepete ürün ekle, /siparis doldur, "Öde"ye bas. Mock modda iframe PayTR'ye gider (gerçek ödeme sandbox anahtarı gerektirir) — bu adımda yalnızca **token alındığını ve iframe src'sinin oluştuğunu** doğrula (Network sekmesi: `/api/orders` 201 + paytr_token). Gerçek iframe ödemesi Task 12'de (sandbox anahtarları gelince).

- [ ] **Step 8: DURUM.md güncelle**

`docs/DURUM.md`'ye Faz 3 satırı ekle: mock ile uçtan uca doğrulandı, gerçek PayTR sandbox testi bekliyor.

- [ ] **Step 9: Commit**

```bash
git add docs/DURUM.md
git commit -m "docs: Faz 3 mock uçtan uca doğrulama sonuçları"
```

---

## Task 12: Gerçek PayTR sandbox doğrulaması (anahtarlar gelince — ERTELENİR)

**Files:** `.env` (gerçek sandbox anahtarları — commit EDİLMEZ).

> Bu task PayTR sandbox anahtarları geldiğinde çalıştırılır. Plan tamamlanması için bloklamaz.

- [ ] `.env`'e gerçek `PAYTR_MERCHANT_ID/KEY/SALT` gir, `PAYTR_TEST_MODE=1`.
- [ ] PayTR panelinde **Bildirim URL**'ini `PAYMENT_CALLBACK_URL` ile eşitle (canlı/tünel URL — localhost'a PayTR callback atamaz; test için ngrok/deploy gerekir).
- [ ] Backend'i başlat — log `PayTR provider aktif (test_mode: true)` demeli.
- [ ] Public sitede gerçek test kartıyla ödeme yap (PayTR test kartları). iframe açılmalı, ödeme sonrası `/siparis/tamam`.
- [ ] Callback'in geldiğini ve siparişin `paid` olduğunu doğrula (log + DB).
- [ ] Panelden iade dene — PayTR sandbox iadeyi kabul etmeli, sipariş `refunded`.
- [ ] Gerçek (canlı) anahtarlara geçmeden ETBİS kaydının tamamlandığını teyit et (kullanıcı işi — kod değil).

---

## Self-Review

**Spec coverage (spec bölümleri → task):**
- §1 kararlar → tüm plana yayılı (provider=PayTR T3/8, model=direkt çekim T5, sıra=önce kaydet T5, tek adım onay T5, iade T7, awaiting gizli T5/6, bildirim yok — hiçbir task mail eklemiyor ✓).
- §2.1 akış → T5 (Create) + T6 (callback) + T9 (iframe).
- §2.2 sadece callback karar verir → T5 ApplyCallback + T6 handler (OK/FAIL). ✓
- §2.3 idempotency → T4 HasPaymentEvent + T5 ApplyCallback + T11 Step 4. ✓
- §2.4 izolasyon → T2/3 payment paketi + T5 PaymentStarter arayüzü. ✓
- §3.1 durum makinesi → T1 CHECK + T4 sabitler + T5 Update geçiş kontrolü. ✓
- §3.2 orders alanları → T1 migration + T4 model/store. ✓
- §3.3 payment_events → T1 + T4. ✓
- §3.4 merchant_oid tekilliği → T5 Create (`order_no`+id). ✓
- §4.1 POST /orders + callback → T6. ✓
- §4.2 admin uçları + refund → T7. ✓
- §4.3 config → T8. ✓
- §5 kod organizasyonu → T2/3/5 dosya yapısı. ✓
- §6 frontend → T9 (public) + T10 (admin). ✓
- §7 test stratejisi → hash(T3), idempotency(T4/T11), kuruş(T3), fiyat(mevcut korunur T5), iade(T5/T7), geçiş(T5). ✓
- §8 kabul kriteri → T11 uçtan uca. ✓
- §9 ETBİS/devreden → T12 son adım (kullanıcı işi). ✓

**Placeholder scan:** Test gövdelerinde "mevcut helper adıyla değiştir" notları var — bunlar gerçek kod isimlerinin task yazımında bilinmemesinden (order_test.go helper'ları okunmadı). Uygulayıcı ilgili `_test.go`'yu okuyup birebir helper adını kullanmalı; test senaryoları (ne doğrulanacağı) tam yazılı. Bu bilinçli — test kurulumu mevcut desene bağlı, uydurmak yanlış olurdu.

**Type consistency:** `PaymentStarter` (order) ile `Provider` (payment) aynı üç metod imzasına sahip (Start/VerifyCallback/Refund) — order kendi arayüzünü tanımlar ama payment tiplerini kullanır, PayTR/Mock ikisini de sağlar. `Create` yeni imzası `(ctx, in, userIP) → (*Order, string, error)` handler(T6) ve test(T5) ile tutarlı. `KurusFromDecimal` T3'te tanımlı, T5'te kullanılıyor. ✓

**Callback güvenlik özeti (T5):** Siparişi `paid` yapan tek koşul `res.OK` (hash geçerli + status=success) — sahte hash asla `SetPaid`'e ulaşmaz. `accepted` (PayTR'ye OK/FAIL dönüşü) ödeme kararından ayrı: var olmayan oid → FAIL, meşru ama failed → OK (iz bırakılır, sipariş awaiting kalır), success → paid + OK. Bu ayrım kod yorumlarında açıkça belgelendi.
