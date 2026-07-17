# Faz 2 — Sepet ve Sipariş Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Müşteri birden fazla ürünü sepete atıp tek siparişte gönderebilsin; esnaf siparişleri panelde görüp işleyebilsin.

**Architecture:** Sepet tarayıcıda (localStorage). Sipariş sunucuda — `orders` + `order_items`, tek transaction. Fiyat sepetten DEĞİL, sipariş anında DB'den okunur. Mevcut katman deseni: `handler` (HTTP) → `service` (iş mantığı) → `store` (SQL).

**Tech Stack:** Go + Fiber + pgx, Nuxt 4 (SSR), Vuetify admin panel, golang-migrate, testify, vitest.

**Spec:** `docs/superpowers/specs/2026-07-17-faz2-sepet-siparis-design.md`

**Önkoşul:** Plan 1-4 tamamlandı. 221 test geçiyor.

---

## Başlangıç Durumu — yeni bir oturumda uygulayacaksan ÖNCE OKU

**Repo:** `/Users/omerkoc/GolandProjects/cicekci`, branch `feat/backend-temeli`
(her şey bu branch'te — ayrı branch açma).

**Go kodu `backend/` altında** (kökte değil). Frontend: `frontend/app` (public,
Nuxt 4), `frontend/idare` (admin, Vuetify SPA).

**Ortam:** Go 1.25.4, Node 22, pnpm 10.24. Docker çalışıyor.

**Backend'i ayağa kaldırmak:**
```bash
cd /Users/omerkoc/GolandProjects/cicekci
export DATABASE_URL="postgres://cicekci:cicekci@localhost:5433/cicekci?sslmode=disable"
export TEST_DATABASE_URL="postgres://cicekci:cicekci@localhost:5434/cicekci_test?sslmode=disable"
export JWT_SECRET="local-development-secret-32-chars!"
export WHATSAPP_NUMBER="905551234567" SITE_URL="http://localhost:5173"
export APP_ENV=development STORAGE_DRIVER=local
make db-up && make migrate-up && make run
```

**Testler:** `make test` (`go test ./...` DEĞİL — paketler aynı test DB'sini
paylaşıyor, `-p 1` gerekli).

**Admin kullanıcısı:** `make seed` veya etkileşimsiz:
`cd backend && go run ./cmd/seed -username cicekci -password test-sifre-123`

**Örnek veri:** `cd backend && go run ./cmd/demoseed`

---

## Global Constraints

- **Fiyata asla güvenilmez.** `POST /api/orders` gövdesi fiyat içermez; sunucu
  her ürünün fiyatını DB'den okur. Bu tasarımın en kritik güvenlik kuralı.
- **Para `NUMERIC(10,2)`**, float değil. Go tarafında `pgtype.Numeric` veya
  string; hesaplama `shopspring/decimal` ile.
- **Sipariş yazımı tek transaction** — `orders` + `order_items` birlikte.
- **Arayüz dili Türkçe. Yorumlar Türkçe.**
- **Public uçlarda `is_active=false` filtresi store katmanında.**
- **Hata formatı:** `{"error": {"code": "...", "message": "..."}}`
- **Katmanlar:** `handler` HTTP'yi bilir, iş mantığını bilmez. `service` iş
  mantığını bilir, HTTP/SQL bilmez. `store` SQL'i bilir, iş mantığını bilmez.
- **Bu fazda YOK:** ödeme, üyelik/`customers`, stok, bildirim, `carts` tablosu.

---

## Dosya Yapısı

```
backend/
  migrations/
    000005_orders.up.sql / .down.sql        → YENİ
  internal/order/                           → YENİ paket
    model.go        → Order, OrderItem, CreateInput, Status
    orderno.go      → order_no üretimi (saf fonksiyon)
    orderno_test.go
    service.go      → doğrulama + fiyat okuma + tutar hesabı
    service_test.go
    store.go        → SQL, tek transaction
    store_test.go
  internal/api/app/
    order_handler.go / order_view.go        → YENİ (POST /orders, GET /delivery-config)
    router.go                               → değişecek
  internal/api/idare/
    order_handler.go / order_view.go        → YENİ (liste/detay/PATCH)
    router.go                               → değişecek
  pkg/config/config.go                      → teslimat ayarları eklenecek

frontend/app/app/
  composables/useCart.ts                    → YENİ (localStorage sepet)
  composables/useCart.test.ts               → YENİ
  composables/useOrders.ts                  → YENİ (POST /orders, delivery-config)
  types/api.ts                              → Order tipleri eklenecek
  components/TheCartDrawer.vue              → gerçek hale gelecek
  components/TheHeader.vue                  → sepet rozeti
  pages/urun/[slug].vue                     → "Sepete Ekle" gerçek
  pages/siparis/index.vue                   → YENİ (form)
  pages/siparis/tamam.vue                   → YENİ (teşekkür)

frontend/idare/src/
  model/order.ts                            → YENİ
  composables/useOrders.ts                  → YENİ
  pages/siparisler.vue                      → placeholder → gerçek liste
  pages/siparisler/[id].vue                 → YENİ (detay)
```

---

## Task 1: Migration + config

**Files:**
- Create: `backend/migrations/000005_orders.up.sql`, `backend/migrations/000005_orders.down.sql`
- Modify: `backend/pkg/config/config.go`
- Modify: `.env.example`, `.env.prod.example`
- Test: `backend/pkg/config/config_test.go`

**Interfaces:**
- Produces: `config.Config.DeliveryFee string`, `.DeliverySlots []string`,
  `.SameDayCutoff string`, `.MaxDeliveryDays int`

- [ ] **Step 1: Migration up dosyasını yaz**

`backend/migrations/000005_orders.up.sql`:
```sql
CREATE TABLE orders (
    id               BIGSERIAL PRIMARY KEY,
    order_no         TEXT NOT NULL UNIQUE,
    status           TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending','confirmed','delivered','cancelled')),

    buyer_name       TEXT NOT NULL,
    buyer_phone      TEXT NOT NULL,
    buyer_email      TEXT,

    recipient_name   TEXT NOT NULL,
    recipient_phone  TEXT NOT NULL,
    delivery_address TEXT NOT NULL,
    delivery_date    DATE NOT NULL,
    delivery_slot    TEXT NOT NULL,
    card_message     TEXT,

    items_total      NUMERIC(10,2) NOT NULL,
    delivery_fee     NUMERIC(10,2) NOT NULL,
    total            NUMERIC(10,2) NOT NULL,

    note             TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Panel listesi status filtresi + tarih sırasıyla okuyor
CREATE INDEX idx_orders_status_created ON orders (status, created_at DESC);

CREATE TABLE order_items (
    id             BIGSERIAL PRIMARY KEY,
    order_id       BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    -- Ürün silinirse sipariş ölmemeli; ad/fiyat kopya olduğu için okunabilir kalır
    product_id     BIGINT REFERENCES products(id) ON DELETE SET NULL,
    product_name   TEXT NOT NULL,
    price_at_order NUMERIC(10,2) NOT NULL,
    quantity       INT NOT NULL CHECK (quantity > 0)
);

CREATE INDEX idx_order_items_order ON order_items (order_id);
```

- [ ] **Step 2: Migration down dosyasını yaz**

`backend/migrations/000005_orders.down.sql`:
```sql
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
```

- [ ] **Step 3: Migration'ı çalıştır ve doğrula**

```bash
cd /Users/omerkoc/GolandProjects/cicekci
export DATABASE_URL="postgres://cicekci:cicekci@localhost:5433/cicekci?sslmode=disable"
make migrate-up
docker compose exec -T postgres psql -U cicekci -d cicekci -c "\d orders"
```
Expected: `orders` tablosu görünür, `order_no` UNIQUE, `status` CHECK'li.

Test DB'ye de uygula:
```bash
export TEST_DATABASE_URL="postgres://cicekci:cicekci@localhost:5434/cicekci_test?sslmode=disable"
make migrate-test-up
```

- [ ] **Step 4: Config testini yaz (RED)**

`backend/pkg/config/config_test.go` içine ekle:
```go
func TestLoad_DeliveryDefaults(t *testing.T) {
	setRequiredEnv(t)
	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "0", cfg.DeliveryFee)
	assert.Equal(t, []string{"09:00-12:00", "12:00-15:00", "15:00-18:00"}, cfg.DeliverySlots)
	assert.Equal(t, "16:00", cfg.SameDayCutoff)
	assert.Equal(t, 30, cfg.MaxDeliveryDays)
}

func TestLoad_DeliveryFromEnv(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DELIVERY_FEE", "50")
	t.Setenv("DELIVERY_SLOTS", "10:00-14:00,14:00-18:00")
	t.Setenv("SAME_DAY_CUTOFF", "15:30")
	t.Setenv("MAX_DELIVERY_DAYS", "14")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "50", cfg.DeliveryFee)
	assert.Equal(t, []string{"10:00-14:00", "14:00-18:00"}, cfg.DeliverySlots)
	assert.Equal(t, "15:30", cfg.SameDayCutoff)
	assert.Equal(t, 14, cfg.MaxDeliveryDays)
}
```

**NOT:** `setRequiredEnv` mevcut test dosyasında var mı bak. Yoksa mevcut
testlerin zorunlu env'i nasıl kurduğunu taklit et (`t.Setenv("DATABASE_URL", ...)`
vb.) ve aynı deseni kullan.

- [ ] **Step 5: Testi çalıştır, BAŞARISIZ olduğunu gör**

Run: `cd backend && go test ./pkg/config -run TestLoad_Delivery -v`
Expected: FAIL — `cfg.DeliveryFee` alanı yok (derleme hatası).

- [ ] **Step 6: Config'e alanları ekle**

`backend/pkg/config/config.go` — `Config` struct'ına ekle:
```go
	// Teslimat ayarları (spec §4). settings tablosu yerine config:
	// yılda bir değişir, ayarlar ekranı YAGNI (MVP §5.3 ile aynı gerekçe).
	// DEĞERLER ESNAFTAN ÖĞRENİLECEK — spec §7.
	DeliveryFee     string   // "50" — NUMERIC'e yazılacak, string tutuluyor (float precision)
	DeliverySlots   []string // ["09:00-12:00", ...]
	SameDayCutoff   string   // "16:00" — bu saatten sonra aynı güne sipariş yok
	MaxDeliveryDays int      // 30 — bugün + bu kadar güne kadar sipariş alınır
```

`Load()` içine, mevcut `loadStorage(cfg)` çağrısından önce ekle:
```go
	loadDelivery(cfg)
```

Dosyanın sonuna ekle:
```go
// loadDelivery teslimat ayarlarını okur. Hepsinin makul varsayılanı var —
// esnaftan gerçek değerler öğrenilene kadar sistem çalışır (spec §7).
func loadDelivery(cfg *Config) {
	cfg.DeliveryFee = os.Getenv("DELIVERY_FEE")
	if cfg.DeliveryFee == "" {
		cfg.DeliveryFee = "0"
	}

	slots := os.Getenv("DELIVERY_SLOTS")
	if slots == "" {
		slots = "09:00-12:00,12:00-15:00,15:00-18:00"
	}
	cfg.DeliverySlots = strings.Split(slots, ",")
	for i := range cfg.DeliverySlots {
		cfg.DeliverySlots[i] = strings.TrimSpace(cfg.DeliverySlots[i])
	}

	cfg.SameDayCutoff = os.Getenv("SAME_DAY_CUTOFF")
	if cfg.SameDayCutoff == "" {
		cfg.SameDayCutoff = "16:00"
	}

	cfg.MaxDeliveryDays = 30
	if v := os.Getenv("MAX_DELIVERY_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxDeliveryDays = n
		}
	}
}
```

`strconv` import'unu ekle.

- [ ] **Step 7: Testi çalıştır, GEÇTİĞİNİ gör**

Run: `cd backend && go test ./pkg/config -run TestLoad_Delivery -v`
Expected: PASS — 2 test.

- [ ] **Step 8: .env örneklerine ekle**

`.env.example` sonuna:
```
# Teslimat ayarları (Faz 2). DEĞERLER ESNAFTAN ÖĞRENİLECEK.
# Bölgeye göre ücret gerekirse config yetmez, settings tablosu gerekir.
DELIVERY_FEE=0
DELIVERY_SLOTS=09:00-12:00,12:00-15:00,15:00-18:00
SAME_DAY_CUTOFF=16:00
MAX_DELIVERY_DAYS=30
```

`.env.prod.example` sonuna aynısı (yorumla birlikte).

`docker-compose.prod.yml` → `backend` servisinin `environment` bloğuna:
```yaml
      DELIVERY_FEE: ${DELIVERY_FEE}
      DELIVERY_SLOTS: ${DELIVERY_SLOTS}
      SAME_DAY_CUTOFF: ${SAME_DAY_CUTOFF}
      MAX_DELIVERY_DAYS: ${MAX_DELIVERY_DAYS}
```

- [ ] **Step 9: Tüm testleri çalıştır**

Run: `make test 2>&1 | grep -cE "^--- PASS"; make test 2>&1 | grep -c FAIL`
Expected: PASS sayısı artmış, FAIL = 0.

- [ ] **Step 10: Commit**

```bash
git add backend/migrations/000005_orders.* backend/pkg/config/ .env.example .env.prod.example docker-compose.prod.yml
git commit -m "feat: orders şeması ve teslimat config'i

orders + order_items tabloları. price_at_order ve product_name kopya —
esnaf fiyat değiştirince eski sipariş bozulmasın (spec §3).
product_id ON DELETE SET NULL: ürün silinince sipariş ölmesin.

Teslimat ayarları config'den (spec §4): settings tablosu yerine .env,
yılda bir değişir. Değerler esnaftan öğrenilecek (spec §7)."
```

---

## Task 2: order_no üretimi

**Files:**
- Create: `backend/internal/order/orderno.go`, `backend/internal/order/orderno_test.go`

**Interfaces:**
- Produces: `func FormatOrderNo(t time.Time, seq int) string`

- [ ] **Step 1: Testi yaz (RED)**

`backend/internal/order/orderno_test.go`:
```go
package order

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatOrderNo(t *testing.T) {
	d := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)

	assert.Equal(t, "2607-0001", FormatOrderNo(d, 1))
	assert.Equal(t, "2607-0042", FormatOrderNo(d, 42))
	assert.Equal(t, "2607-9999", FormatOrderNo(d, 9999))
}

func TestFormatOrderNo_AyGunSifirDolgulu(t *testing.T) {
	d := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)

	// 5 Ocak → "0105", tek haneli ay/gün sıfırla dolgulanmalı
	assert.Equal(t, "0105-0001", FormatOrderNo(d, 1))
}

func TestFormatOrderNo_DortHaneyiAsarsa(t *testing.T) {
	d := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)

	// Günde 10000+ sipariş gerçekçi değil ama format bozulmamalı
	assert.Equal(t, "2607-10000", FormatOrderNo(d, 10000))
}
```

- [ ] **Step 2: Testi çalıştır, BAŞARISIZ olduğunu gör**

Run: `cd backend && go test ./internal/order -run TestFormatOrderNo -v`
Expected: FAIL — paket/fonksiyon yok.

- [ ] **Step 3: Implementasyonu yaz**

`backend/internal/order/orderno.go`:
```go
package order

import (
	"fmt"
	"time"
)

// FormatOrderNo müşteriye söylenen sipariş numarasını üretir: "AAGG-NNNN".
//
// Ay+gün, tire, o günün sıra numarası. Örnek: 26 Temmuz'un 42. siparişi →
// "2607-0042". Esnaf "bugünün 42. siparişi" diye okuyabilir.
//
// Neden id değil: id kaç sipariş alındığını dışarı sızdırır (spec §3).
func FormatOrderNo(t time.Time, seq int) string {
	return fmt.Sprintf("%02d%02d-%04d", int(t.Month()), t.Day(), seq)
}
```

- [ ] **Step 4: Testi çalıştır, GEÇTİĞİNİ gör**

Run: `cd backend && go test ./internal/order -run TestFormatOrderNo -v`
Expected: PASS — 3 test.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/order/
git commit -m "feat: order_no üretimi — AAGG-NNNN

Her gün 0001'den başlar. id yerine ayrı numara: id kaç sipariş
alındığını sızdırır (spec §3)."
```

---

## Task 3: Order modeli ve store

**Files:**
- Create: `backend/internal/order/model.go`, `backend/internal/order/store.go`
- Test: `backend/internal/order/store_test.go`

**Interfaces:**
- Consumes: `FormatOrderNo(t, seq)` (Task 2)
- Produces:
  - `type Status string`, sabitler: `StatusPending`, `StatusConfirmed`, `StatusDelivered`, `StatusCancelled`
  - `type Order struct`, `type OrderItem struct`
  - `type NewOrder struct` (store'a giden, tutarlar hesaplanmış)
  - `func NewStore(pool *pgxpool.Pool) *Store`
  - `func (s *Store) Create(ctx, in NewOrder) (*Order, error)`
  - `func (s *Store) List(ctx, status string, limit, offset int) ([]Order, error)`
  - `func (s *Store) GetByID(ctx, id int64) (*Order, error)`
  - `func (s *Store) Update(ctx, id int64, status *string, note *string) (*Order, error)`

- [ ] **Step 1: Model dosyasını yaz**

`backend/internal/order/model.go`:
```go
package order

import (
	"time"

	"github.com/shopspring/decimal"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusDelivered Status = "delivered"
	StatusCancelled Status = "cancelled"
)

// Valid statü geçerli mi. DB'de CHECK var ama hatayı service'te yakalayıp
// düzgün mesaj vermek için burada da kontrol ediliyor.
func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusConfirmed, StatusDelivered, StatusCancelled:
		return true
	}
	return false
}

type OrderItem struct {
	ID           int64           `json:"id"`
	ProductID    *int64          `json:"product_id"` // ürün silinmişse nil
	ProductName  string          `json:"product_name"`
	PriceAtOrder decimal.Decimal `json:"price_at_order"`
	Quantity     int             `json:"quantity"`
}

type Order struct {
	ID      int64  `json:"id"`
	OrderNo string `json:"order_no"`
	Status  Status `json:"status"`

	BuyerName  string `json:"buyer_name"`
	BuyerPhone string `json:"buyer_phone"`
	BuyerEmail string `json:"buyer_email"`

	RecipientName   string    `json:"recipient_name"`
	RecipientPhone  string    `json:"recipient_phone"`
	DeliveryAddress string    `json:"delivery_address"`
	DeliveryDate    time.Time `json:"delivery_date"`
	DeliverySlot    string    `json:"delivery_slot"`
	CardMessage     string    `json:"card_message"`

	ItemsTotal  decimal.Decimal `json:"items_total"`
	DeliveryFee decimal.Decimal `json:"delivery_fee"`
	Total       decimal.Decimal `json:"total"`

	Note      string      `json:"note"`
	Items     []OrderItem `json:"items"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// NewOrder store'a giden kayıt — tutarlar service'te hesaplanmış,
// fiyatlar DB'den okunmuş halde gelir.
type NewOrder struct {
	BuyerName  string
	BuyerPhone string
	BuyerEmail string

	RecipientName   string
	RecipientPhone  string
	DeliveryAddress string
	DeliveryDate    time.Time
	DeliverySlot    string
	CardMessage     string

	ItemsTotal  decimal.Decimal
	DeliveryFee decimal.Decimal
	Total       decimal.Decimal

	Items []NewOrderItem
}

type NewOrderItem struct {
	ProductID    int64
	ProductName  string
	PriceAtOrder decimal.Decimal
	Quantity     int
}
```

- [ ] **Step 2: Store testini yaz (RED)**

**NOT:** `shopspring/decimal` zaten `backend/go.mod`'da (v1.4.0) —
`product.Product.Price` onu kullanıyor. Ek bağımlılık gerekmiyor.

`backend/internal/order/store_test.go`:
```go
package order

import (
	"context"
	"testing"
	"time"

	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewOrder() NewOrder {
	return NewOrder{
		BuyerName:       "Ahmet Yılmaz",
		BuyerPhone:      "05551112233",
		BuyerEmail:      "ahmet@example.com",
		RecipientName:   "Ayşe Yılmaz",
		RecipientPhone:  "05554445566",
		DeliveryAddress: "Teşvikiye Cad. No:1, Şişli/İstanbul",
		DeliveryDate:    time.Now().AddDate(0, 0, 1),
		DeliverySlot:    "12:00-15:00",
		CardMessage:     "Doğum günün kutlu olsun",
		ItemsTotal:      decimal.RequireFromString("1850.00"),
		DeliveryFee:     decimal.RequireFromString("50.00"),
		Total:           decimal.RequireFromString("1900.00"),
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
```

**NOT:** `errorsx` import'unu ekle (son testte kullanılıyor).

- [ ] **Step 3: Testi çalıştır, BAŞARISIZ olduğunu gör**

Run: `cd backend && go test ./internal/order -v`
Expected: FAIL — `NewStore` yok.

- [ ] **Step 4: Store'u yaz**

`backend/internal/order/store.go`:
```go
package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/shopspring/decimal"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// orderNoMaxRetry aynı anda gelen siparişler aynı sırayı görebilir; UNIQUE
// constraint çakışmayı yakalar, bir sonraki sırayla tekrar denenir.
// Plan 1'de uniqueSlug'da aynı yarış vardı — orada hata kaba düşüyordu.
const orderNoMaxRetry = 5

const orderSelect = `
	SELECT id, order_no, status,
	       buyer_name, buyer_phone, COALESCE(buyer_email, ''),
	       recipient_name, recipient_phone, delivery_address,
	       delivery_date, delivery_slot, COALESCE(card_message, ''),
	       items_total, delivery_fee, total,
	       COALESCE(note, ''), created_at, updated_at
	FROM orders`

func scanOrder(row pgx.Row) (*Order, error) {
	var o Order
	var itemsTotal, deliveryFee, total pgtype.Numeric

	err := row.Scan(&o.ID, &o.OrderNo, &o.Status,
		&o.BuyerName, &o.BuyerPhone, &o.BuyerEmail,
		&o.RecipientName, &o.RecipientPhone, &o.DeliveryAddress,
		&o.DeliveryDate, &o.DeliverySlot, &o.CardMessage,
		&itemsTotal, &deliveryFee, &total,
		&o.Note, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}

	o.ItemsTotal = numericToDecimal(itemsTotal)
	o.DeliveryFee = numericToDecimal(deliveryFee)
	o.Total = numericToDecimal(total)

	return &o, nil
}

// numericToDecimal pgtype.Numeric → decimal.Decimal.
// pgx NUMERIC'i doğrudan decimal'e taramıyor, ara dönüşüm gerekiyor.
func numericToDecimal(n pgtype.Numeric) decimal.Decimal {
	if !n.Valid {
		return decimal.Zero
	}
	f, err := n.Value()
	if err != nil {
		return decimal.Zero
	}
	s, ok := f.(string)
	if !ok {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

// Create siparişi ve kalemlerini TEK transaction'da yazar.
// Yarısı yazılıp kalanı patlarsa tutarsız sipariş kalır — Plan 1'de slug
// atomikliğinde aynı ders alınmıştı.
func (s *Store) Create(ctx context.Context, in NewOrder) (*Order, error) {
	var lastErr error

	for attempt := 0; attempt < orderNoMaxRetry; attempt++ {
		o, err := s.createOnce(ctx, in)
		if err == nil {
			return o, nil
		}
		// order_no çakışması → bir sonraki sırayla tekrar dene
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" &&
			strings.Contains(pgErr.ConstraintName, "order_no") {
			lastErr = err
			continue
		}
		return nil, err
	}

	return nil, fmt.Errorf("sipariş numarası üretilemedi: %w", lastErr)
}

func (s *Store) createOnce(ctx context.Context, in NewOrder) (*Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Bugünün kaçıncı siparişi
	now := time.Now()
	var todayCount int
	err = tx.QueryRow(ctx,
		`SELECT count(*) FROM orders WHERE created_at::date = CURRENT_DATE`).Scan(&todayCount)
	if err != nil {
		return nil, err
	}
	orderNo := FormatOrderNo(now, todayCount+1)

	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (
			order_no, buyer_name, buyer_phone, buyer_email,
			recipient_name, recipient_phone, delivery_address,
			delivery_date, delivery_slot, card_message,
			items_total, delivery_fee, total
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id`,
		orderNo, in.BuyerName, in.BuyerPhone, nullIfEmpty(in.BuyerEmail),
		in.RecipientName, in.RecipientPhone, in.DeliveryAddress,
		in.DeliveryDate, in.DeliverySlot, nullIfEmpty(in.CardMessage),
		in.ItemsTotal.String(), in.DeliveryFee.String(), in.Total.String(),
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	for _, it := range in.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (order_id, product_id, product_name, price_at_order, quantity)
			VALUES ($1,$2,$3,$4,$5)`,
			id, it.ProductID, it.ProductName, it.PriceAtOrder.String(), it.Quantity)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.GetByID(ctx, id)
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Order, error) {
	o, err := scanOrder(s.pool.QueryRow(ctx, orderSelect+` WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsx.ErrNotFound
		}
		return nil, err
	}

	items, err := s.itemsOf(ctx, id)
	if err != nil {
		return nil, err
	}
	o.Items = items

	return o, nil
}

func (s *Store) itemsOf(ctx context.Context, orderID int64) ([]OrderItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, product_id, product_name, price_at_order, quantity
		FROM order_items WHERE order_id = $1 ORDER BY id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []OrderItem{}
	for rows.Next() {
		var it OrderItem
		var price pgtype.Numeric
		if err := rows.Scan(&it.ID, &it.ProductID, &it.ProductName, &price, &it.Quantity); err != nil {
			return nil, err
		}
		it.PriceAtOrder = numericToDecimal(price)
		items = append(items, it)
	}

	return items, rows.Err()
}

// List siparişleri en yeniden eskiye listeler. status boşsa hepsi.
func (s *Store) List(ctx context.Context, status string, limit, offset int) ([]Order, error) {
	q := orderSelect
	args := []any{}

	if status != "" {
		q += ` WHERE status = $1`
		args = append(args, status)
	}
	q += fmt.Sprintf(` ORDER BY created_at DESC LIMIT %d OFFSET %d`, limit, offset)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Liste ekranı kalemleri de gösteriyor (ürün adları)
	for i := range orders {
		items, err := s.itemsOf(ctx, orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].Items = items
	}

	return orders, nil
}

// Update status ve/veya note günceller. nil olan alan değişmez (PATCH semantiği).
func (s *Store) Update(ctx context.Context, id int64, status *string, note *string) (*Order, error) {
	ct, err := s.pool.Exec(ctx, `
		UPDATE orders SET
			status = COALESCE($2, status),
			note = COALESCE($3, note),
			updated_at = now()
		WHERE id = $1`, id, status, note)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, errorsx.ErrNotFound
	}

	return s.GetByID(ctx, id)
}
```

**NOT:** `pgtype`, `pgconn`, `strings` import'larını ekle:
`github.com/jackc/pgx/v5/pgtype`, `github.com/jackc/pgx/v5/pgconn`, `strings`.

`scanOrder` hem `pgx.Row` hem `pgx.Rows` ile çağrılıyor — ikisi de `Scan`
metoduna sahip, `pgx.Row` arayüzü yeterli.

- [ ] **Step 5: Testi çalıştır, GEÇTİĞİNİ gör**

Run: `cd backend && go test ./internal/order -v`
Expected: PASS — 5+ test. Atomiklik testi rollback'i kanıtlamalı.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/order/ backend/go.mod backend/go.sum
git commit -m "feat: order store — tek transaction, order_no retry

Create tek transaction: orders + order_items birlikte. Atomiklik testle
kanıtlandı (FK hatası → orders satırı da rollback).

order_no çakışmasında retry: eşzamanlı sipariş aynı sırayı görebilir,
UNIQUE yakalar, bir sonraki sıra denenir (spec §3)."
```

---

## Task 4: Order service — doğrulama ve fiyat okuma

**Files:**
- Create: `backend/internal/order/service.go`
- Test: `backend/internal/order/service_test.go`

**Interfaces:**
- Consumes: `Store` (Task 3), `product.Service`
- Produces:
  - `type DeliveryConfig struct { Fee string; Slots []string; SameDayCutoff string; MaxDays int }`
  - `type CreateInput struct` (handler'dan gelen ham girdi)
  - `func NewService(store *Store, prodStore ProductReader, cfg DeliveryConfig) *Service`
  - `type ProductReader interface { GetByID(ctx, id int64) (*product.Product, error) }`
  - `func (s *Service) Create(ctx, in CreateInput) (*Order, error)`
  - `func (s *Service) List/Get/Update`

**DOĞRULANMIŞ GERÇEKLER** (plan yazılırken kontrol edildi, varsayım değil):
- `product.Store.GetByID(ctx, id) (*Product, error)` ve
  `product.Service.GetByID(...)` ikisi de var. `ProductReader` ikisiyle de uyumlu.
- `product.Product.Price` tipi **`decimal.Decimal`** — string DEĞİL.
  Dolayısıyla `p.Price` doğrudan kullanılır, `decimal.NewFromString` gerekmez.
- `product.Product` alanları: `ID int64`, `Name string`, `Price decimal.Decimal`,
  `IsActive bool`.

- [ ] **Step 1: Service testini yaz (RED)**

`backend/internal/order/service_test.go`:
```go
package order

import (
	"context"
	"testing"
	"time"

	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDeliveryConfig() DeliveryConfig {
	return DeliveryConfig{
		Fee:           "50",
		Slots:         []string{"09:00-12:00", "12:00-15:00", "15:00-18:00"},
		SameDayCutoff: "16:00",
		MaxDays:       30,
	}
}

func testCreateInput(productID int64) CreateInput {
	return CreateInput{
		Items:           []CreateItem{{ProductID: productID, Quantity: 2}},
		BuyerName:       "Ahmet Yılmaz",
		BuyerPhone:      "05551112233",
		RecipientName:   "Ayşe Yılmaz",
		RecipientPhone:  "05554445566",
		DeliveryAddress: "Teşvikiye Cad. No:1",
		DeliveryDate:    time.Now().AddDate(0, 0, 2),
		DeliverySlot:    "12:00-15:00",
	}
}

// setupService gerçek DB ile service kurar ve bir aktif ürün oluşturur.
//
// pool'u da döndürüyor: NewTestDB TRUNCATE çalıştırdığı için testlerin
// ikinci kez çağırması veriyi siler — aynı pool paylaşılmalı.
func setupService(t *testing.T) (svc *Service, pool *pgxpool.Pool, productID int64) {
	t.Helper()
	pool = database.NewTestDB(t)
	t.Cleanup(pool.Close)

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

	// Ürün 1850, adet 2 → 3700 + 50 teslimat = 3750
	assert.Equal(t, "3700", o.ItemsTotal.String())
	assert.Equal(t, "50", o.DeliveryFee.String())
	assert.Equal(t, "3750", o.Total.String())
	assert.Equal(t, "1850", o.Items[0].PriceAtOrder.String())
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
```

**NOT:** `pgxpool` import'unu ekle: `github.com/jackc/pgx/v5/pgxpool`.

- [ ] **Step 2: Testi çalıştır, BAŞARISIZ olduğunu gör**

Run: `cd backend && go test ./internal/order -run TestService -v`
Expected: FAIL — `NewService` yok.

- [ ] **Step 3: Service'i yaz**

`backend/internal/order/service.go`:
```go
package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/shopspring/decimal"
)

// ProductReader service'in ürün okumak için ihtiyaç duyduğu tek şey.
// Dar arayüz: order paketi product'ın tamamına bağlanmasın.
type ProductReader interface {
	GetByID(ctx context.Context, id int64) (*product.Product, error)
}

// DeliveryConfig teslimat kuralları — config'den gelir (spec §4).
type DeliveryConfig struct {
	Fee           string
	Slots         []string
	SameDayCutoff string
	MaxDays       int
}

type CreateItem struct {
	ProductID int64
	Quantity  int
}

// CreateInput handler'dan gelen ham girdi. FİYAT YOK — sunucu DB'den okur.
type CreateInput struct {
	Items []CreateItem

	BuyerName  string
	BuyerPhone string
	BuyerEmail string

	RecipientName   string
	RecipientPhone  string
	DeliveryAddress string
	DeliveryDate    time.Time
	DeliverySlot    string
	CardMessage     string
}

type Service struct {
	store *Store
	prod  ProductReader
	cfg   DeliveryConfig
}

func NewService(store *Store, prod ProductReader, cfg DeliveryConfig) *Service {
	return &Service{store: store, prod: prod, cfg: cfg}
}

// maxQuantity absürt girdiye karşı duvar. UI'da limit YOK — 50 buket gerçek
// bir sipariş olabilir (spec §5). Bu sadece 999999999 gibi girdileri eler.
const maxQuantity = 1000

func (s *Service) Create(ctx context.Context, in CreateInput) (*Order, error) {
	if err := s.validateContact(&in); err != nil {
		return nil, err
	}
	if err := s.validateDelivery(in); err != nil {
		return nil, err
	}
	if len(in.Items) == 0 {
		return nil, fmt.Errorf("%w: sepet boş", errorsx.ErrInvalidInput)
	}

	// FİYAT DB'DEN OKUNUR — sepetten gelen fiyata asla güvenilmez (spec §2.2)
	items := make([]NewOrderItem, 0, len(in.Items))
	itemsTotal := decimal.Zero

	for _, ci := range in.Items {
		if ci.Quantity <= 0 || ci.Quantity > maxQuantity {
			return nil, fmt.Errorf("%w: geçersiz adet", errorsx.ErrInvalidInput)
		}

		p, err := s.prod.GetByID(ctx, ci.ProductID)
		if err != nil {
			return nil, fmt.Errorf("%w: ürün bulunamadı", errorsx.ErrInvalidInput)
		}
		if !p.IsActive {
			return nil, fmt.Errorf("%w: %q artık satışta değil", errorsx.ErrInvalidInput, p.Name)
		}

		// p.Price zaten decimal.Decimal — dönüşüm gerekmez
		itemsTotal = itemsTotal.Add(p.Price.Mul(decimal.NewFromInt(int64(ci.Quantity))))
		items = append(items, NewOrderItem{
			ProductID:    p.ID,
			ProductName:  p.Name,
			PriceAtOrder: p.Price,
			Quantity:     ci.Quantity,
		})
	}

	// Teslimat ücreti de siparişe kopyalanır — esnaf yarın değiştirirse
	// dünkü siparişin toplamı bozulmasın (spec §3)
	fee, err := decimal.NewFromString(s.cfg.Fee)
	if err != nil {
		return nil, fmt.Errorf("teslimat ücreti okunamadı: %w", err)
	}

	return s.store.Create(ctx, NewOrder{
		BuyerName:       in.BuyerName,
		BuyerPhone:      in.BuyerPhone,
		BuyerEmail:      in.BuyerEmail,
		RecipientName:   in.RecipientName,
		RecipientPhone:  in.RecipientPhone,
		DeliveryAddress: in.DeliveryAddress,
		DeliveryDate:    in.DeliveryDate,
		DeliverySlot:    in.DeliverySlot,
		CardMessage:     in.CardMessage,
		ItemsTotal:      itemsTotal,
		DeliveryFee:     fee,
		Total:           itemsTotal.Add(fee),
		Items:           items,
	})
}

func (s *Service) validateContact(in *CreateInput) error {
	in.BuyerName = strings.TrimSpace(in.BuyerName)
	in.BuyerPhone = strings.TrimSpace(in.BuyerPhone)
	in.BuyerEmail = strings.TrimSpace(in.BuyerEmail)
	in.RecipientName = strings.TrimSpace(in.RecipientName)
	in.RecipientPhone = strings.TrimSpace(in.RecipientPhone)
	in.DeliveryAddress = strings.TrimSpace(in.DeliveryAddress)
	in.CardMessage = strings.TrimSpace(in.CardMessage)

	switch {
	case in.BuyerName == "":
		return fmt.Errorf("%w: ad soyad gerekli", errorsx.ErrInvalidInput)
	case in.BuyerPhone == "":
		return fmt.Errorf("%w: telefon gerekli", errorsx.ErrInvalidInput)
	case in.RecipientName == "":
		return fmt.Errorf("%w: alıcı adı gerekli", errorsx.ErrInvalidInput)
	// Kurye alıcıyı arayamazsa teslimat başarısız olur (spec §3)
	case in.RecipientPhone == "":
		return fmt.Errorf("%w: alıcı telefonu gerekli", errorsx.ErrInvalidInput)
	case in.DeliveryAddress == "":
		return fmt.Errorf("%w: teslimat adresi gerekli", errorsx.ErrInvalidInput)
	}

	return nil
}

func (s *Service) validateDelivery(in CreateInput) error {
	today := time.Now().Truncate(24 * time.Hour)
	d := in.DeliveryDate.Truncate(24 * time.Hour)

	if d.Before(today) {
		return fmt.Errorf("%w: teslimat tarihi geçmişte olamaz", errorsx.ErrInvalidInput)
	}
	if d.After(today.AddDate(0, 0, s.cfg.MaxDays)) {
		return fmt.Errorf("%w: en fazla %d gün sonrasına sipariş verilebilir",
			errorsx.ErrInvalidInput, s.cfg.MaxDays)
	}

	if !slices.Contains(s.cfg.Slots, in.DeliverySlot) {
		return fmt.Errorf("%w: geçersiz teslimat saati", errorsx.ErrInvalidInput)
	}

	// Aynı gün + cutoff geçmiş → esnaf yetiştiremez
	if d.Equal(today) && s.pastCutoff(time.Now()) {
		return fmt.Errorf("%w: aynı gün siparişi için saat %s'ı geçti",
			errorsx.ErrInvalidInput, s.cfg.SameDayCutoff)
	}

	return nil
}

// pastCutoff şu an cutoff saatini geçti mi. Cutoff bozuksa kısıt uygulanmaz —
// yanlış config yüzünden tüm siparişleri reddetmektense kısıtsız çalış.
func (s *Service) pastCutoff(now time.Time) bool {
	cutoff, err := time.Parse("15:04", s.cfg.SameDayCutoff)
	if err != nil {
		return false
	}
	nowMin := now.Hour()*60 + now.Minute()
	cutMin := cutoff.Hour()*60 + cutoff.Minute()

	return nowMin > cutMin
}

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

	return s.store.List(ctx, status, limit, offset)
}

func (s *Service) Get(ctx context.Context, id int64) (*Order, error) {
	return s.store.GetByID(ctx, id)
}

// Update statü ve/veya not günceller. nil alan değişmez (PATCH semantiği).
func (s *Service) Update(ctx context.Context, id int64, status *string, note *string) (*Order, error) {
	if status != nil {
		if !Status(*status).Valid() {
			return nil, fmt.Errorf("%w: geçersiz durum", errorsx.ErrInvalidInput)
		}
	}

	return s.store.Update(ctx, id, status, note)
}
```

**NOT:** `slices` import'unu ekle (Go 1.21+ stdlib).

- [ ] **Step 4: Testi çalıştır, GEÇTİĞİNİ gör**

Run: `cd backend && go test ./internal/order -v`
Expected: PASS — tüm testler. Özellikle `TestService_Create_FiyatDBdenOkunur`
geçmeli: 1850×2+50 = 3750.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/order/
git commit -m "feat: order service — fiyat DB'den, doğrulama kuralları

EN KRİTİK: fiyat sepetten değil DB'den okunuyor. Müşteri localStorage'da
1850'yi 1 yapsa bile sunucu 1850 hesaplıyor — testle kanıtlandı.

Doğrulamalar: pasif ürün reddi, geçmiş/çok ileri tarih, geçersiz slot,
cutoff sonrası aynı gün, boş sepet, zorunlu alanlar.

delivery_fee siparişe kopyalanıyor: esnaf ücreti değiştirince dünkü
siparişin toplamı bozulmasın."
```

---

## Task 5: Public API — POST /orders, GET /delivery-config

**Files:**
- Create: `backend/internal/api/app/order_handler.go`, `backend/internal/api/app/order_view.go`
- Modify: `backend/internal/api/app/router.go`
- Modify: `backend/cmd/server/main.go`
- Test: `backend/internal/api/app/order_handler_test.go`

**Interfaces:**
- Consumes: `order.Service` (Task 4)
- Produces: `POST /api/orders`, `GET /api/delivery-config`

- [ ] **Step 1: View dosyasını yaz**

`backend/internal/api/app/order_view.go`:
```go
package app

import "github.com/omerkoc/cicekci/internal/order"

// createOrderRequest — FİYAT ALANI YOK. Sunucu fiyatı DB'den okur (spec §2.2).
type createOrderRequest struct {
	Items []struct {
		ProductID int64 `json:"product_id"`
		Quantity  int   `json:"quantity"`
	} `json:"items"`

	Buyer struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
		Email string `json:"email"`
	} `json:"buyer"`

	Recipient struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
	} `json:"recipient"`

	Delivery struct {
		Address string `json:"address"`
		Date    string `json:"date"` // "2026-07-20"
		Slot    string `json:"slot"`
	} `json:"delivery"`

	CardMessage string `json:"card_message"`
}

// createOrderResponse public yanıt — sipariş no ve toplam yeter.
// Müşteriye iç detay (id, statü) sızdırılmaz.
type createOrderResponse struct {
	OrderNo string `json:"order_no"`
	Total   string `json:"total"`
}

func toCreateOrderResponse(o *order.Order) createOrderResponse {
	return createOrderResponse{
		OrderNo: o.OrderNo,
		Total:   o.Total.StringFixed(2),
	}
}

// deliveryConfigResponse frontend'in saat/ücret hardcode etmemesi için.
// Sunucu ve frontend AYNI kaynaktan beslenmeli (spec §4).
type deliveryConfigResponse struct {
	Fee           string   `json:"fee"`
	Slots         []string `json:"slots"`
	SameDayCutoff string   `json:"same_day_cutoff"`
	MaxDays       int      `json:"max_days"`
}
```

- [ ] **Step 2: Handler'ı yaz**

`backend/internal/api/app/order_handler.go`:
```go
package app

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/order"
)

type orderHandler struct {
	svc *order.Service
	cfg order.DeliveryConfig
}

func (h *orderHandler) create(c *fiber.Ctx) error {
	var req createOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return api.WriteError(c, err)
	}

	date, err := time.Parse("2006-01-02", req.Delivery.Date)
	if err != nil {
		return api.WriteError(c, errInvalidDate)
	}

	in := order.CreateInput{
		BuyerName:       req.Buyer.Name,
		BuyerPhone:      req.Buyer.Phone,
		BuyerEmail:      req.Buyer.Email,
		RecipientName:   req.Recipient.Name,
		RecipientPhone:  req.Recipient.Phone,
		DeliveryAddress: req.Delivery.Address,
		DeliveryDate:    date,
		DeliverySlot:    req.Delivery.Slot,
		CardMessage:     req.CardMessage,
	}
	for _, it := range req.Items {
		in.Items = append(in.Items, order.CreateItem{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
		})
	}

	o, err := h.svc.Create(c.Context(), in)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toCreateOrderResponse(o))
}

func (h *orderHandler) deliveryConfig(c *fiber.Ctx) error {
	return c.JSON(deliveryConfigResponse{
		Fee:           h.cfg.Fee,
		Slots:         h.cfg.Slots,
		SameDayCutoff: h.cfg.SameDayCutoff,
		MaxDays:       h.cfg.MaxDays,
	})
}
```

`errInvalidDate` için dosyanın başına ekle:
```go
var errInvalidDate = fmt.Errorf("%w: geçersiz teslimat tarihi", errorsx.ErrInvalidInput)
```
(`fmt` ve `errorsx` import'ları gerekli.)

**NOT:** `api.WriteError` mevcut mu kontrol et: `grep -rn "func WriteError" backend/internal/api/`
İmzası farklıysa mevcut handler'ların hatayı nasıl yazdığını taklit et
(`backend/internal/api/app/product_handler.go`'ya bak).

- [ ] **Step 3: Router'a bağla**

`backend/internal/api/app/router.go` — `Register` imzasına `orderSvc` ve
`deliveryCfg` ekle:
```go
func Register(router fiber.Router, catSvc *category.Service,
	prodSvc *product.Service, imgSvc *image.Service, sliderSvc *slider.Service,
	orderSvc *order.Service, deliveryCfg order.DeliveryConfig) {
	// ... mevcut handler'lar
	oh := &orderHandler{svc: orderSvc, cfg: deliveryCfg}

	router.Post("/orders", oh.create)
	router.Get("/delivery-config", oh.deliveryConfig)
	// ... mevcut rotalar
}
```

- [ ] **Step 4: main.go'da bağla**

`backend/cmd/server/main.go` — mevcut servis kurulumlarının yanına:
```go
	deliveryCfg := order.DeliveryConfig{
		Fee:           cfg.DeliveryFee,
		Slots:         cfg.DeliverySlots,
		SameDayCutoff: cfg.SameDayCutoff,
		MaxDays:       cfg.MaxDeliveryDays,
	}
	orderSvc := order.NewService(order.NewStore(pool), product.NewStore(pool), deliveryCfg)
```

`app.Register(...)` çağrısına yeni parametreleri ekle. `order` import'unu ekle.

- [ ] **Step 5: Handler testini yaz**

`backend/internal/api/app/order_handler_test.go`:
```go
package app

import (
	"bytes"
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
	defer pool.Close()

	var productID int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ('51 Gül Buket', 'test', 1850.00, true) RETURNING id`).Scan(&productID)
	require.NoError(t, err)

	cfg := order.DeliveryConfig{
		Fee: "50", Slots: []string{"12:00-15:00"},
		SameDayCutoff: "16:00", MaxDays: 30,
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
		"delivery": {"address": "Test Cad. 1", "date": "%s", "slot": "12:00-15:00"}
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
}
```

`context` import'unu ekle.

- [ ] **Step 6: Testleri çalıştır**

Run: `cd backend && go test ./internal/api/app -run "TestCreateOrder|TestDeliveryConfig" -v`
Expected: PASS — 2 test.

- [ ] **Step 7: Canlı sunucuda doğrula**

```bash
cd /Users/omerkoc/GolandProjects/cicekci
# .env'e DELIVERY_FEE=50 ekle, sonra sunucuyu başlat (make run)

curl -s localhost:8080/api/delivery-config

# Ürün id'sini al
curl -s localhost:8080/api/products | head -c 200

# Sipariş oluştur (product_id'yi değiştir)
curl -s -X POST localhost:8080/api/orders -H 'Content-Type: application/json' -d '{
  "items": [{"product_id": 1, "quantity": 2}],
  "buyer": {"name": "Ahmet", "phone": "05551112233"},
  "recipient": {"name": "Ayşe", "phone": "05554445566"},
  "delivery": {"address": "Test Cad. 1", "date": "2026-07-25", "slot": "12:00-15:00"}
}'
```
Expected: `{"order_no":"...","total":"..."}` — toplam DB fiyatından hesaplanmış.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/api/app/ backend/cmd/server/main.go
git commit -m "feat: public sipariş uçları — POST /orders, GET /delivery-config

Gövdede fiyat alanı YOK; gönderilse bile yok sayılır, sunucu DB'den
okur. Testle kanıtlandı: gövdede price=1.00 gönderildi, sipariş 3750
oluştu.

delivery-config ayrı uç: sunucu ve frontend aynı kaynaktan beslenmeli,
yoksa frontend 50 gösterip sunucu 60 hesaplar (spec §4)."
```

---

## Task 6: Admin API — liste, detay, durum güncelleme

**Files:**
- Create: `backend/internal/api/idare/order_handler.go`, `backend/internal/api/idare/order_view.go`
- Modify: `backend/internal/api/idare/router.go`
- Modify: `backend/cmd/server/main.go`
- Test: `backend/internal/api/idare/order_handler_test.go`

**Interfaces:**
- Consumes: `order.Service` (Task 4)
- Produces: `GET /api/admin/orders`, `GET /api/admin/orders/:id`, `PATCH /api/admin/orders/:id`

- [ ] **Step 1: View dosyasını yaz**

`backend/internal/api/idare/order_view.go`:
```go
package idare

import (
	"time"

	"github.com/omerkoc/cicekci/internal/order"
)

// orderItemView admin görünümü — esnaf ne göndereceğini görmeli.
type orderItemView struct {
	ProductID    *int64 `json:"product_id"`
	ProductName  string `json:"product_name"`
	PriceAtOrder string `json:"price_at_order"`
	Quantity     int    `json:"quantity"`
}

// orderView admin tam görünüm — public'ten farklı: her şey görünür.
type orderView struct {
	ID      int64  `json:"id"`
	OrderNo string `json:"order_no"`
	Status  string `json:"status"`

	BuyerName  string `json:"buyer_name"`
	BuyerPhone string `json:"buyer_phone"`
	BuyerEmail string `json:"buyer_email"`

	RecipientName   string `json:"recipient_name"`
	RecipientPhone  string `json:"recipient_phone"`
	DeliveryAddress string `json:"delivery_address"`
	DeliveryDate    string `json:"delivery_date"`
	DeliverySlot    string `json:"delivery_slot"`
	CardMessage     string `json:"card_message"`

	ItemsTotal  string `json:"items_total"`
	DeliveryFee string `json:"delivery_fee"`
	Total       string `json:"total"`

	Note      string          `json:"note"`
	Items     []orderItemView `json:"items"`
	CreatedAt time.Time       `json:"created_at"`
}

func toOrderView(o *order.Order) orderView {
	items := make([]orderItemView, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, orderItemView{
			ProductID:    it.ProductID,
			ProductName:  it.ProductName,
			PriceAtOrder: it.PriceAtOrder.StringFixed(2),
			Quantity:     it.Quantity,
		})
	}

	return orderView{
		ID:              o.ID,
		OrderNo:         o.OrderNo,
		Status:          string(o.Status),
		BuyerName:       o.BuyerName,
		BuyerPhone:      o.BuyerPhone,
		BuyerEmail:      o.BuyerEmail,
		RecipientName:   o.RecipientName,
		RecipientPhone:  o.RecipientPhone,
		DeliveryAddress: o.DeliveryAddress,
		DeliveryDate:    o.DeliveryDate.Format("2006-01-02"),
		DeliverySlot:    o.DeliverySlot,
		CardMessage:     o.CardMessage,
		ItemsTotal:      o.ItemsTotal.StringFixed(2),
		DeliveryFee:     o.DeliveryFee.StringFixed(2),
		Total:           o.Total.StringFixed(2),
		Note:            o.Note,
		Items:           items,
		CreatedAt:       o.CreatedAt,
	}
}

func toOrderViews(list []order.Order) []orderView {
	out := make([]orderView, 0, len(list))
	for i := range list {
		out = append(out, toOrderView(&list[i]))
	}
	return out
}

// updateOrderRequest PATCH: nil alan değişmez.
type updateOrderRequest struct {
	Status *string `json:"status"`
	Note   *string `json:"note"`
}
```

- [ ] **Step 2: Handler'ı yaz**

`backend/internal/api/idare/order_handler.go`:
```go
package idare

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/order"
)

type orderHandler struct {
	svc *order.Service
}

func (h *orderHandler) list(c *fiber.Ctx) error {
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}

	list, err := h.svc.List(c.Context(), status, limit, (page-1)*limit)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(toOrderViews(list))
}

func (h *orderHandler) get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return api.WriteError(c, errInvalidID)
	}

	o, err := h.svc.Get(c.Context(), id)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(toOrderView(o))
}

func (h *orderHandler) update(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return api.WriteError(c, errInvalidID)
	}

	var req updateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return api.WriteError(c, err)
	}

	o, err := h.svc.Update(c.Context(), id, req.Status, req.Note)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(toOrderView(o))
}
```

**NOT:** `errInvalidID` mevcut idare paketinde var mı bak
(`grep -rn "errInvalidID\|geçersiz id" backend/internal/api/idare/`).
Yoksa mevcut handler'ların id parse hatasını nasıl döndürdüğünü taklit et.

- [ ] **Step 3: Router'a bağla**

`backend/internal/api/idare/router.go` — `Deps`'e ekle:
```go
	OrderSvc *order.Service
```

`Register` içine:
```go
	oh := &orderHandler{svc: d.OrderSvc}

	router.Get("/orders", oh.list)
	router.Get("/orders/:id", oh.get)
	router.Patch("/orders/:id", oh.update)
```

`main.go`'da `idare.Register(...)` çağrısındaki `Deps`'e `OrderSvc: orderSvc` ekle.

- [ ] **Step 4: Auth testini yaz**

`backend/internal/api/idare/order_handler_test.go`:
```go
package idare

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Admin uçları JWT ister — cookie'siz istek 401 dönmeli.
func TestOrders_AuthGerekli(t *testing.T) {
	f := fiber.New()
	// Mevcut testlerin auth middleware'i nasıl kurduğunu taklit et —
	// image_handler_test.go veya product_handler_test.go'ya bak.

	resp, err := f.Test(httptest.NewRequest(http.MethodGet, "/api/admin/orders", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}
```

**NOT:** Bu testi mevcut idare testlerinin desenine uydur. Mevcut testler
auth'u nasıl kuruyorsa aynısını yap — `backend/internal/api/idare/product_handler_test.go`
iyi bir örnek.

- [ ] **Step 5: Testleri çalıştır**

Run: `cd backend && go test ./internal/api/idare -v`
Expected: PASS.

- [ ] **Step 6: Canlı doğrula**

```bash
# Login ol
curl -s -X POST localhost:8080/api/admin/login -H 'Content-Type: application/json' \
  -d '{"username":"cicekci","password":"test-sifre-123"}' -c /tmp/c.txt -o /dev/null

# Liste
curl -s -b /tmp/c.txt localhost:8080/api/admin/orders

# Durum güncelle (id'yi değiştir)
curl -s -b /tmp/c.txt -X PATCH localhost:8080/api/admin/orders/1 \
  -H 'Content-Type: application/json' -d '{"status":"confirmed"}'

# Geçersiz durum reddedilmeli
curl -s -b /tmp/c.txt -X PATCH localhost:8080/api/admin/orders/1 \
  -H 'Content-Type: application/json' -d '{"status":"uydurma"}'
```
Expected: Liste dolu, durum `confirmed` oldu, uydurma durum 400 döndü.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/api/idare/ backend/cmd/server/main.go
git commit -m "feat: admin sipariş uçları — liste, detay, durum güncelleme

Sipariş oluşturma/silme admin ucu YOK: siparişi müşteri oluşturur,
esnaf sadece durumunu değiştirir. Silme yok — sipariş finansal iz,
cancelled var (spec §4)."
```

---

## Task 7: Sepet composable (frontend)

**Files:**
- Create: `frontend/app/app/composables/useCart.ts`
- Create: `frontend/app/app/composables/useCart.test.ts`
- Modify: `frontend/app/app/types/api.ts`

**Interfaces:**
- Produces:
  - `type CartItem = { product_id: number; name: string; slug: string; price: string; image: string; quantity: number }`
  - `useCart()` → `{ items, count, itemsTotal, add, remove, setQuantity, clear }`

- [ ] **Step 1: Tipleri ekle**

`frontend/app/app/types/api.ts` sonuna:
```ts
/** Sepet kalemi — localStorage'da yaşar. Fiyat GÖSTERİM için; sipariş
 *  anında sunucu DB'den okur, buradaki fiyata güvenilmez (spec §2.2). */
export interface CartItem {
  product_id: number
  name: string
  slug: string
  price: string
  image: string
  quantity: number
}

export interface DeliveryConfig {
  fee: string
  slots: string[]
  same_day_cutoff: string
  max_days: number
}

export interface CreateOrderInput {
  items: { product_id: number, quantity: number }[]
  buyer: { name: string, phone: string, email?: string }
  recipient: { name: string, phone: string }
  delivery: { address: string, date: string, slot: string }
  card_message?: string
}

export interface CreateOrderResult {
  order_no: string
  total: string
}
```

- [ ] **Step 2: Testi yaz (RED)**

`frontend/app/app/composables/useCart.test.ts`:
```ts
import { beforeEach, describe, expect, it } from 'vitest'
import { cartTotal, addItem, removeItem, setItemQuantity } from './cartLogic'
import type { CartItem } from '~/types/api'

function urun(id: number, price: string, qty = 1): CartItem {
  return {
    product_id: id,
    name: `Ürün ${id}`,
    slug: `urun-${id}`,
    price,
    image: '',
    quantity: qty,
  }
}

describe('cartTotal', () => {
  it('kalem fiyatlarını adetle çarpıp toplar', () => {
    const items = [urun(1, '1850.00', 2), urun(2, '500.50', 1)]
    expect(cartTotal(items)).toBe('4200.50')
  })

  it('boş sepette sıfır', () => {
    expect(cartTotal([])).toBe('0.00')
  })

  it('kuruşlu fiyatlarda yuvarlama hatası yapmaz', () => {
    const items = [urun(1, '0.10', 3)]
    expect(cartTotal(items)).toBe('0.30')
  })
})

describe('addItem', () => {
  it('yeni ürünü ekler', () => {
    const out = addItem([], urun(1, '100.00'))
    expect(out).toHaveLength(1)
    expect(out[0]!.quantity).toBe(1)
  })

  it('var olan ürünün adedini artırır, kopya oluşturmaz', () => {
    const out = addItem([urun(1, '100.00', 2)], urun(1, '100.00', 3))
    expect(out).toHaveLength(1)
    expect(out[0]!.quantity).toBe(5)
  })
})

describe('setItemQuantity', () => {
  it('adedi değiştirir', () => {
    const out = setItemQuantity([urun(1, '100.00', 1)], 1, 4)
    expect(out[0]!.quantity).toBe(4)
  })

  it('adet 0 veya altına inerse kalemi siler', () => {
    expect(setItemQuantity([urun(1, '100.00', 1)], 1, 0)).toHaveLength(0)
  })
})

describe('removeItem', () => {
  it('kalemi siler', () => {
    expect(removeItem([urun(1, '100.00'), urun(2, '200.00')], 1)).toHaveLength(1)
  })
})
```

- [ ] **Step 3: Testi çalıştır, BAŞARISIZ olduğunu gör**

Run: `cd frontend/app && pnpm test`
Expected: FAIL — `./cartLogic` modülü yok.

- [ ] **Step 4: Saf mantığı yaz**

`frontend/app/app/composables/cartLogic.ts`:
```ts
import type { CartItem } from '~/types/api'

/**
 * Sepet mantığı — Nuxt'tan bağımsız saf fonksiyonlar, test edilebilir.
 * useCart bunları sarmalayıp localStorage'a bağlar.
 *
 * Fiyat string olarak geliyor ("1850.00") — float precision sorunu olmasın
 * diye kuruş cinsinden integer'a çevrilip hesaplanıyor.
 */

/** "1850.50" → 185050 (kuruş). Float toplamada 0.1+0.2=0.30000000000000004 olur. */
function toKurus(price: string): number {
  return Math.round(Number.parseFloat(price) * 100)
}

export function cartTotal(items: CartItem[]): string {
  const kurus = items.reduce((sum, i) => sum + toKurus(i.price) * i.quantity, 0)

  return (kurus / 100).toFixed(2)
}

export function addItem(items: CartItem[], yeni: CartItem): CartItem[] {
  const mevcut = items.find(i => i.product_id === yeni.product_id)
  if (mevcut) {
    return items.map(i =>
      i.product_id === yeni.product_id
        ? { ...i, quantity: i.quantity + yeni.quantity }
        : i)
  }

  return [...items, yeni]
}

export function removeItem(items: CartItem[], productId: number): CartItem[] {
  return items.filter(i => i.product_id !== productId)
}

export function setItemQuantity(items: CartItem[], productId: number, qty: number): CartItem[] {
  if (qty <= 0)
    return removeItem(items, productId)

  return items.map(i => (i.product_id === productId ? { ...i, quantity: qty } : i))
}
```

- [ ] **Step 5: Testi çalıştır, GEÇTİĞİNİ gör**

Run: `cd frontend/app && pnpm test`
Expected: PASS — 8 test (mevcut 11 + yeni 8 = 19).

- [ ] **Step 6: useCart composable'ını yaz**

`frontend/app/app/composables/useCart.ts`:
```ts
import type { CartItem } from '~/types/api'
import { addItem, cartTotal, removeItem, setItemQuantity } from './cartLogic'

const STORAGE_KEY = 'cicekci_sepet'

/**
 * Sepet — localStorage'da yaşar, sunucuda carts tablosu YOK (spec §2.1).
 *
 * useState ile paylaşılan tek state: drawer, header rozeti ve sipariş
 * formu aynı sepeti görür.
 */
export function useCart() {
  const items = useState<CartItem[]>('sepet', () => [])

  // SSR'da localStorage yok; hydration sonrası okunur
  onMounted(() => {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw)
      return

    try {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed))
        items.value = parsed
    }
    catch {
      // Bozuk veri — sıfırla, patlatma
      localStorage.removeItem(STORAGE_KEY)
    }
  })

  function kaydet() {
    if (import.meta.client)
      localStorage.setItem(STORAGE_KEY, JSON.stringify(items.value))
  }

  const count = computed(() => items.value.reduce((s, i) => s + i.quantity, 0))
  const itemsTotal = computed(() => cartTotal(items.value))

  function add(item: CartItem) {
    items.value = addItem(items.value, item)
    kaydet()
  }

  function remove(productId: number) {
    items.value = removeItem(items.value, productId)
    kaydet()
  }

  function setQuantity(productId: number, qty: number) {
    items.value = setItemQuantity(items.value, productId, qty)
    kaydet()
  }

  function clear() {
    items.value = []
    kaydet()
  }

  return { items, count, itemsTotal, add, remove, setQuantity, clear }
}
```

- [ ] **Step 7: Commit**

```bash
git add frontend/app/app/composables/ frontend/app/app/types/api.ts
git commit -m "feat: sepet composable — localStorage, saf mantık ayrı

cartLogic.ts Nuxt'tan bağımsız: 8 vitest testi. Fiyat kuruş cinsinden
integer'a çevrilip hesaplanıyor — float toplamada 0.1+0.2 hatası olur.

Sepet localStorage'da, sunucuda carts tablosu yok (spec §2.1)."
```

---

## Task 8: Sepet drawer ve "Sepete Ekle" gerçek hale gelsin

**Files:**
- Modify: `frontend/app/app/components/TheCartDrawer.vue`
- Modify: `frontend/app/app/components/TheHeader.vue`
- Modify: `frontend/app/app/pages/urun/[slug].vue`

**Interfaces:**
- Consumes: `useCart()` (Task 7)

- [ ] **Step 1: Mevcut dosyaları oku**

```bash
cd /Users/omerkoc/GolandProjects/cicekci/frontend/app
cat app/components/TheCartDrawer.vue
grep -n "sepet\|cart\|Sepete Ekle" app/components/TheHeader.vue app/pages/urun/\[slug\].vue
```
Mevcut tasarım dilini (Tailwind sınıfları, `font-serif`, `text-primary`,
`btn-primary`) koru — DESIGN.md'ye sadık kal.

- [ ] **Step 2: TheCartDrawer'ı gerçek yap**

`frontend/app/app/components/TheCartDrawer.vue` — `<script setup>` bloğunu değiştir:
```ts
/**
 * Sepet drawer. Faz 2'de gerçek hale geldi — luxe redesign spec'i §2.1
 * "bu ekranlar atılacak değil, backend'e bağlanacak" demişti.
 */
import { formatPrice } from '~/utils/price'

const acik = defineModel<boolean>({ required: true })

const { items, itemsTotal, remove, setQuantity } = useCart()

watch(acik, (a) => {
  if (import.meta.client)
    document.body.style.overflow = a ? 'hidden' : ''
})

onBeforeUnmount(() => {
  if (import.meta.client)
    document.body.style.overflow = ''
})
```

Boş durum bloğunu (`<div class="flex flex-1 flex-col items-center justify-center...">`)
`v-if="!items.length"` ile koşulla, altına dolu durumu ekle:
```vue
        <!-- Dolu sepet -->
        <div v-else class="flex flex-1 flex-col overflow-hidden">
          <ul class="flex-1 overflow-y-auto px-6 py-4">
            <li
              v-for="item in items"
              :key="item.product_id"
              class="flex gap-4 border-b border-outline-variant/30 py-4"
            >
              <img
                v-if="item.image"
                :src="item.image"
                :alt="item.name"
                width="80"
                height="80"
                class="size-20 shrink-0 rounded-md object-cover"
              >
              <div
                v-else
                class="flex size-20 shrink-0 items-center justify-center rounded-md bg-surface-container-low"
              >
                <Icon name="material-symbols:local-florist-outline" size="24" class="text-outline-variant" />
              </div>

              <div class="min-w-0 flex-1">
                <NuxtLink
                  :to="`/urun/${item.slug}`"
                  class="line-clamp-2 text-body-md text-primary hover:underline"
                  @click="acik = false"
                >
                  {{ item.name }}
                </NuxtLink>
                <p class="mt-1 text-body-md font-medium text-primary">
                  {{ formatPrice(item.price) }}
                </p>

                <div class="mt-2 flex items-center gap-3">
                  <div class="flex items-center rounded border border-outline-variant/50">
                    <button
                      type="button"
                      class="px-2.5 py-1 text-on-surface-variant hover:text-primary"
                      :aria-label="`${item.name} adedini azalt`"
                      @click="setQuantity(item.product_id, item.quantity - 1)"
                    >
                      −
                    </button>
                    <span class="min-w-8 text-center text-body-md">{{ item.quantity }}</span>
                    <button
                      type="button"
                      class="px-2.5 py-1 text-on-surface-variant hover:text-primary"
                      :aria-label="`${item.name} adedini artır`"
                      @click="setQuantity(item.product_id, item.quantity + 1)"
                    >
                      +
                    </button>
                  </div>

                  <button
                    type="button"
                    class="text-xs text-on-surface-variant underline-offset-4 hover:text-primary hover:underline"
                    @click="remove(item.product_id)"
                  >
                    Kaldır
                  </button>
                </div>
              </div>
            </li>
          </ul>

          <div class="border-t border-outline-variant/30 px-6 py-5">
            <div class="flex items-center justify-between">
              <span class="text-body-md text-on-surface-variant">Ara Toplam</span>
              <span class="font-serif text-xl text-primary">{{ formatPrice(itemsTotal) }}</span>
            </div>
            <p class="mt-1 text-xs text-on-surface-variant">
              Teslimat ücreti sipariş adımında eklenir.
            </p>

            <NuxtLink
              to="/siparis"
              class="btn-primary text-label-caps mt-5 flex w-full items-center justify-center"
              @click="acik = false"
            >
              Siparişi Tamamla
            </NuxtLink>
          </div>
        </div>
```

- [ ] **Step 3: Header rozetini ekle**

`frontend/app/app/components/TheHeader.vue` — sepet butonuna rozet ekle.
Önce mevcut sepet butonunu bul (`@click="$emit('open-cart')"` olan), sonra:

```vue
<!-- Rozet artık gerçek: sepette ürün var. Redesign spec'i §2.1 rozeti
     "var olmayan içeriği iddia etmesin" diye kaldırmıştı — sepet gerçek
     olduğu için rozet artık yalan değil, bilgi. -->
<span
  v-if="cartCount > 0"
  class="absolute -right-1 -top-1 flex size-4 items-center justify-center rounded-full bg-secondary text-[10px] font-medium text-white"
>
  {{ cartCount > 9 ? '9+' : cartCount }}
</span>
```

Butonun `class`'ına `relative` ekle. `<script setup>`'a:
```ts
const { count: cartCount } = useCart()
```

- [ ] **Step 4: "Sepete Ekle"yi gerçek yap**

`frontend/app/app/pages/urun/[slug].vue` — inert butonu bul, `<script setup>`'a ekle:
```ts
const { add } = useCart()
const sepetAcik = inject<Ref<boolean>>('sepetAcik')

function sepeteEkle() {
  if (!product.value)
    return

  add({
    product_id: product.value.id,
    name: product.value.name,
    slug: product.value.slug,
    price: product.value.price,
    image: product.value.images?.[0]?.url_400 ?? '',
    quantity: 1,
  })

  if (sepetAcik)
    sepetAcik.value = true
}
```

Butonun `@click`'ini `sepeteEkle`'ye bağla ve inert yorumunu sil.

- [ ] **Step 5: Tarayıcıda doğrula**

```bash
cd /Users/omerkoc/GolandProjects/cicekci/frontend/app && pnpm dev
```
`http://localhost:3000/urun/<slug>` → "Sepete Ekle" → drawer açılmalı, ürün
görünmeli. Header'da rozet "1" olmalı. Adet +/− çalışmalı. Sayfayı yenile →
sepet korunmalı (localStorage).

- [ ] **Step 6: Build ve test**

Run: `cd frontend/app && pnpm test && pnpm build`
Expected: 19 test PASS, build başarılı.

- [ ] **Step 7: Commit**

```bash
git add frontend/app/app/components/ frontend/app/app/pages/urun/
git commit -m "feat: sepet drawer ve Sepete Ekle gerçek hale geldi

Redesign spec'i §2.1: 'bu ekranlar atılacak değil, backend'e bağlanacak'.
Söz tutuldu — tasarım dili korundu, kabuk gerçek oldu.

Rozet geri geldi: sepet artık gerçek, rozet yalan değil bilgi.
WhatsApp butonuna dokunulmadı (spec §2.3)."
```

---

## Task 9: Sipariş formu ve teşekkür sayfası

**Files:**
- Create: `frontend/app/app/composables/useOrders.ts`
- Create: `frontend/app/app/pages/siparis/index.vue`
- Create: `frontend/app/app/pages/siparis/tamam.vue`

**Interfaces:**
- Consumes: `useCart()` (Task 7), `POST /api/orders`, `GET /api/delivery-config` (Task 5)
- Produces: `useDeliveryConfig()`, `createOrder(input)`

- [ ] **Step 1: API composable'ını yaz**

`frontend/app/app/composables/useOrders.ts`:
```ts
import type { CreateOrderInput, CreateOrderResult, DeliveryConfig } from '~/types/api'

/**
 * Sipariş API'si. Çağrılar same-origin /api/go proxy'sinden geçer —
 * CORS'a takılmasın (Plan 3'te kurulan proxy).
 */
function apiBase(): string {
  return useRuntimeConfig().public.apiBase
}

export function useDeliveryConfig() {
  return useFetch<DeliveryConfig>(() => `${apiBase()}/delivery-config`, {
    key: 'delivery-config',
  })
}

/** Sipariş oluşturur. Hata mesajı backend'den gelir (Türkçe). */
export async function createOrder(input: CreateOrderInput): Promise<CreateOrderResult> {
  return await $fetch<CreateOrderResult>(`${apiBase()}/orders`, {
    method: 'POST',
    body: input,
  })
}
```

- [ ] **Step 2: Sipariş formunu yaz**

`frontend/app/app/pages/siparis/index.vue`:
```vue
<script setup lang="ts">
import { formatPrice } from '~/utils/price'
import { apiErrorMessage } from '~/composables/useApi'

const { items, itemsTotal, clear } = useCart()
const { data: cfg } = await useDeliveryConfig()
const router = useRouter()

// Sepet boşken bu sayfanın anlamı yok (spec §5 kenar durumlar)
onMounted(() => {
  if (!items.value.length)
    router.replace('/urunler')
})

const form = reactive({
  buyerName: '',
  buyerPhone: '',
  buyerEmail: '',
  aliciBenim: false,
  recipientName: '',
  recipientPhone: '',
  address: '',
  date: '',
  slot: '',
  cardMessage: '',
})

// "Alıcı benim" — form uzunluğu dönüşümü düşürür (MVP §3.3), tek tık kısayol
watch(() => form.aliciBenim, (v) => {
  if (v) {
    form.recipientName = form.buyerName
    form.recipientPhone = form.buyerPhone
  }
})

const gonderiliyor = ref(false)
const hata = ref('')

const bugun = new Date().toISOString().slice(0, 10)
const sonTarih = computed(() => {
  const d = new Date()
  d.setDate(d.getDate() + (cfg.value?.max_days ?? 30))

  return d.toISOString().slice(0, 10)
})

const toplam = computed(() => {
  const ara = Number.parseFloat(itemsTotal.value)
  const ucret = Number.parseFloat(cfg.value?.fee ?? '0')

  return (ara + ucret).toFixed(2)
})

async function gonder() {
  hata.value = ''
  gonderiliyor.value = true

  try {
    const sonuc = await createOrder({
      items: items.value.map(i => ({ product_id: i.product_id, quantity: i.quantity })),
      buyer: { name: form.buyerName, phone: form.buyerPhone, email: form.buyerEmail || undefined },
      recipient: { name: form.recipientName, phone: form.recipientPhone },
      delivery: { address: form.address, date: form.date, slot: form.slot },
      card_message: form.cardMessage || undefined,
    })

    // Başarılı → sepeti temizle, yoksa müşteri aynı siparişi tekrar verebilir
    clear()
    await router.push(`/siparis/tamam?no=${sonuc.order_no}`)
  }
  catch (e) {
    // Başarısız → sepet KORUNUR, müşterinin emeği silinmesin (spec §5)
    hata.value = apiErrorMessage(e)
  }
  finally {
    gonderiliyor.value = false
  }
}

useSeoMeta({
  title: 'Siparişi Tamamla | Gözde Tasarım Çiçekçilik',
  robots: 'noindex, nofollow',
})
</script>

<template>
  <div class="site-container py-14 md:py-20">
    <h1 class="font-serif text-3xl text-primary md:text-4xl">
      Siparişi Tamamla
    </h1>

    <form class="mt-10 grid gap-12 lg:grid-cols-[1fr_380px]" @submit.prevent="gonder">
      <div class="space-y-10">
        <!-- Sipariş veren -->
        <fieldset>
          <legend class="font-serif text-xl text-primary">
            Sipariş Veren
          </legend>
          <div class="mt-5 grid gap-5 sm:grid-cols-2">
            <label class="block">
              <span class="text-label-caps text-secondary">Ad Soyad *</span>
              <input v-model="form.buyerName" required class="siparis-input">
            </label>
            <label class="block">
              <span class="text-label-caps text-secondary">Telefon *</span>
              <input v-model="form.buyerPhone" required type="tel" class="siparis-input">
            </label>
            <label class="block sm:col-span-2">
              <span class="text-label-caps text-secondary">E-posta</span>
              <input v-model="form.buyerEmail" type="email" class="siparis-input">
            </label>
          </div>
        </fieldset>

        <!-- Alıcı -->
        <fieldset>
          <legend class="font-serif text-xl text-primary">
            Alıcı
          </legend>

          <label class="mt-4 flex items-center gap-2">
            <input v-model="form.aliciBenim" type="checkbox" class="size-4">
            <span class="text-body-md text-on-surface-variant">Alıcı benim</span>
          </label>

          <div class="mt-5 grid gap-5 sm:grid-cols-2">
            <label class="block">
              <span class="text-label-caps text-secondary">Alıcı Adı *</span>
              <input v-model="form.recipientName" required class="siparis-input">
            </label>
            <label class="block">
              <span class="text-label-caps text-secondary">Alıcı Telefonu *</span>
              <input v-model="form.recipientPhone" required type="tel" class="siparis-input">
            </label>
            <label class="block sm:col-span-2">
              <span class="text-label-caps text-secondary">Teslimat Adresi *</span>
              <textarea v-model="form.address" required rows="3" class="siparis-input" />
            </label>
          </div>
        </fieldset>

        <!-- Teslimat -->
        <fieldset>
          <legend class="font-serif text-xl text-primary">
            Teslimat
          </legend>
          <div class="mt-5 grid gap-5 sm:grid-cols-2">
            <label class="block">
              <span class="text-label-caps text-secondary">Tarih *</span>
              <input v-model="form.date" required type="date" :min="bugun" :max="sonTarih" class="siparis-input">
            </label>
            <label class="block">
              <span class="text-label-caps text-secondary">Saat *</span>
              <select v-model="form.slot" required class="siparis-input">
                <option value="" disabled>
                  Seçiniz
                </option>
                <option v-for="s in cfg?.slots ?? []" :key="s" :value="s">
                  {{ s }}
                </option>
              </select>
            </label>
            <label class="block sm:col-span-2">
              <span class="text-label-caps text-secondary">Kart Mesajı</span>
              <textarea
                v-model="form.cardMessage"
                rows="3"
                placeholder="Doğum günün kutlu olsun!"
                class="siparis-input"
              />
            </label>
          </div>
        </fieldset>
      </div>

      <!-- Özet -->
      <aside class="h-fit rounded-lg border border-outline-variant/40 bg-surface-container-low p-6">
        <h2 class="font-serif text-xl text-primary">
          Sipariş Özeti
        </h2>

        <ul class="mt-5 space-y-3">
          <li v-for="item in items" :key="item.product_id" class="flex justify-between gap-3 text-body-md">
            <span class="text-on-surface-variant">{{ item.name }} × {{ item.quantity }}</span>
            <span class="shrink-0 text-primary">{{ formatPrice(item.price) }}</span>
          </li>
        </ul>

        <div class="mt-5 space-y-2 border-t border-outline-variant/30 pt-4 text-body-md">
          <div class="flex justify-between">
            <span class="text-on-surface-variant">Ara Toplam</span>
            <span>{{ formatPrice(itemsTotal) }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-on-surface-variant">Teslimat</span>
            <span>{{ formatPrice(cfg?.fee ?? '0') }}</span>
          </div>
        </div>

        <div class="mt-4 flex justify-between border-t border-outline-variant/30 pt-4">
          <span class="font-serif text-lg text-primary">Toplam</span>
          <span class="font-serif text-lg text-primary">{{ formatPrice(toplam) }}</span>
        </div>

        <p v-if="hata" class="mt-4 text-body-md text-red-700" role="alert">
          {{ hata }}
        </p>

        <button
          type="submit"
          :disabled="gonderiliyor"
          class="btn-primary text-label-caps mt-6 w-full disabled:opacity-60"
        >
          {{ gonderiliyor ? 'Gönderiliyor...' : 'Siparişi Gönder' }}
        </button>

        <p class="mt-3 text-xs text-on-surface-variant">
          Siparişiniz alındıktan sonra sizinle iletişime geçilecektir.
        </p>
      </aside>
    </form>
  </div>
</template>

<style scoped>
.siparis-input {
  @apply mt-1.5 w-full rounded border border-outline-variant/50 bg-surface px-3 py-2.5
         text-body-md text-on-surface focus:border-secondary focus:outline-none;
}
</style>
```

- [ ] **Step 3: Teşekkür sayfasını yaz**

`frontend/app/app/pages/siparis/tamam.vue`:
```vue
<script setup lang="ts">
const route = useRoute()
const orderNo = computed(() => String(route.query.no ?? ''))

// Sipariş no yoksa buraya doğrudan gelinmiş — ana sayfaya gönder
onMounted(() => {
  if (!orderNo.value)
    navigateTo('/')
})

useSeoMeta({
  title: 'Siparişiniz Alındı | Gözde Tasarım Çiçekçilik',
  robots: 'noindex, nofollow',
})
</script>

<template>
  <div class="site-container py-20 text-center md:py-28">
    <span class="mx-auto flex size-16 items-center justify-center rounded-full bg-secondary/10">
      <Icon name="material-symbols:check" size="32" class="text-secondary" />
    </span>

    <h1 class="mt-8 font-serif text-3xl text-primary md:text-4xl">
      Siparişiniz Alındı
    </h1>

    <p v-if="orderNo" class="mt-4 text-body-lg text-on-surface-variant">
      Sipariş numaranız: <strong class="text-primary">{{ orderNo }}</strong>
    </p>

    <p class="mx-auto mt-4 max-w-md text-body-md text-on-surface-variant">
      En kısa sürede sizinle iletişime geçeceğiz. Sipariş numaranızı not
      almanızı öneririz.
    </p>

    <NuxtLink to="/urunler" class="btn-primary text-label-caps mt-10">
      Alışverişe Devam Et
    </NuxtLink>
  </div>
</template>
```

- [ ] **Step 4: Uçtan uca doğrula**

Backend + `pnpm dev` çalışırken:
1. Ürün detaydan sepete ekle
2. Drawer → "Siparişi Tamamla" → `/siparis`
3. Formu doldur, gönder
4. `/siparis/tamam?no=...` açılmalı, sipariş no görünmeli
5. Header rozeti sıfırlanmalı (sepet temizlendi)
6. `curl -s -b /tmp/c.txt localhost:8080/api/admin/orders` → sipariş görünmeli

Hata yolunu da dene: geçmiş tarih seç → Türkçe hata mesajı çıkmalı, sepet
korunmalı.

- [ ] **Step 5: Build**

Run: `cd frontend/app && pnpm build`
Expected: Başarılı.

- [ ] **Step 6: Commit**

```bash
git add frontend/app/app/composables/useOrders.ts frontend/app/app/pages/siparis/
git commit -m "feat: sipariş formu ve teşekkür sayfası

Tek sayfa form (çok adımlı sihirbaz değil), 'Alıcı benim' kısayolu —
form uzunluğu dönüşümü düşürür (MVP §3.3).

Kenar durumlar (spec §5): sepet boşken /urunler'e yönlendir, başarıda
sepet temizlenir (tekrar sipariş önlenir), başarısızlıkta sepet korunur.

noindex: sipariş sayfaları Google'da olmamalı."
```

---

## Task 10: Admin sipariş listesi ve detayı

**Files:**
- Create: `frontend/idare/src/model/order.ts`
- Create: `frontend/idare/src/composables/useOrders.ts`
- Create: `frontend/idare/src/pages/siparisler/[id].vue`
- Modify: `frontend/idare/src/pages/siparisler.vue`

**Interfaces:**
- Consumes: `GET/PATCH /api/admin/orders` (Task 6)

- [ ] **Step 1: Model dosyasını yaz**

`frontend/idare/src/model/order.ts`:
```ts
export type OrderStatus = 'pending' | 'confirmed' | 'delivered' | 'cancelled'

export interface OrderItem {
  product_id: number | null
  product_name: string
  price_at_order: string
  quantity: number
}

export interface Order {
  id: number
  order_no: string
  status: OrderStatus

  buyer_name: string
  buyer_phone: string
  buyer_email: string

  recipient_name: string
  recipient_phone: string
  delivery_address: string
  delivery_date: string
  delivery_slot: string
  card_message: string

  items_total: string
  delivery_fee: string
  total: string

  note: string
  items: OrderItem[]
  created_at: string
}

export interface OrderUpdate {
  status?: OrderStatus
  note?: string
}

export const STATUS_LABELS: Record<OrderStatus, string> = {
  pending: 'Yeni',
  confirmed: 'Onaylandı',
  delivered: 'Teslim Edildi',
  cancelled: 'İptal',
}

export const STATUS_COLORS: Record<OrderStatus, string> = {
  pending: 'warning',
  confirmed: 'info',
  delivered: 'success',
  cancelled: 'error',
}
```

- [ ] **Step 2: Composable'ı yaz**

`frontend/idare/src/composables/useOrders.ts`:
```ts
import ApiService from '@/services/ApiService'
import type { Order, OrderUpdate } from '@/model/order'

export function useOrders() {
  const list = (status?: string) => {
    const q = status ? `?status=${status}` : ''

    return ApiService.get<Order[]>(`admin/orders${q}`)
  }

  const get = (id: number) => ApiService.get<Order>(`admin/orders/${id}`)

  const update = (id: number, data: OrderUpdate) =>
    ApiService.patch<Order>(`admin/orders/${id}`, data)

  return { list, get, update }
}
```

**NOT:** `ApiService`'in imzasını kontrol et — mevcut `useCategories.ts` /
`useProducts.ts` desenine birebir uy.

- [ ] **Step 3: Listeyi yaz**

`frontend/idare/src/pages/siparisler.vue` — placeholder'ı tamamen değiştir:
```vue
<script setup lang="ts">
import { useOrders } from '@/composables/useOrders'
import type { Order, OrderStatus } from '@/model/order'
import { STATUS_COLORS, STATUS_LABELS } from '@/model/order'
import { ErrorPopup } from '@/utils/Popup'

const api = useOrders()
const router = useRouter()

const loading = ref(false)
const orders = ref<Order[]>([])
const statusFilter = ref<string>('')

const headers = [
  { title: 'Sipariş No', key: 'order_no', width: 130 },
  { title: 'Alıcı', key: 'recipient_name' },
  { title: 'Teslimat', key: 'delivery_date', width: 180 },
  { title: 'Tutar', key: 'total', width: 120 },
  { title: 'Durum', key: 'status', width: 130 },
]

const load = async () => {
  loading.value = true

  const [err, data] = await api.list(statusFilter.value)

  loading.value = false

  if (err)
    return ErrorPopup(err.message)

  orders.value = data ?? []
}

onMounted(load)
watch(statusFilter, load)

const tutar = (v: string) =>
  `${Number.parseFloat(v).toLocaleString('tr-TR', { minimumFractionDigits: 2 })} ₺`

const tarih = (d: string, slot: string) => {
  const [y, m, g] = d.split('-')

  return `${g}.${m}.${y} · ${slot}`
}
</script>

<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-6">
      <h4 class="text-h4">
        Siparişler
      </h4>

      <VBtnToggle
        v-model="statusFilter"
        density="compact"
        divided
      >
        <VBtn value="">
          Hepsi
        </VBtn>
        <VBtn value="pending">
          Yeni
        </VBtn>
        <VBtn value="confirmed">
          Onaylandı
        </VBtn>
        <VBtn value="delivered">
          Teslim
        </VBtn>
      </VBtnToggle>
    </div>

    <VCard>
      <VDataTable
        :headers="headers"
        :items="orders"
        :loading="loading"
        :items-per-page="-1"
        no-data-text="Sipariş yok"
        loading-text="Yükleniyor..."
        hover
        @click:row="(_: unknown, { item }: { item: Order }) => router.push(`/siparisler/${item.id}`)"
      >
        <template #item.order_no="{ item }">
          <code class="text-caption">{{ item.order_no }}</code>
        </template>

        <template #item.recipient_name="{ item }">
          <div class="font-weight-medium">
            {{ item.recipient_name }}
          </div>
          <div class="text-caption text-medium-emphasis">
            {{ item.items.map(i => `${i.product_name} ×${i.quantity}`).join(', ') }}
          </div>
        </template>

        <template #item.delivery_date="{ item }">
          {{ tarih(item.delivery_date, item.delivery_slot) }}
        </template>

        <template #item.total="{ item }">
          {{ tutar(item.total) }}
        </template>

        <template #item.status="{ item }">
          <VChip :color="STATUS_COLORS[item.status]" size="small">
            {{ STATUS_LABELS[item.status] }}
          </VChip>
        </template>
      </VDataTable>
    </VCard>
  </div>
</template>
```

- [ ] **Step 4: Detay sayfasını yaz**

`frontend/idare/src/pages/siparisler/[id].vue`:
```vue
<script setup lang="ts">
import { useOrders } from '@/composables/useOrders'
import type { Order, OrderStatus } from '@/model/order'
import { STATUS_COLORS, STATUS_LABELS } from '@/model/order'
import { ErrorPopup, SuccessToast } from '@/utils/Popup'

const route = useRoute('siparisler-id')
const api = useOrders()

const loading = ref(false)
const saving = ref(false)
const order = ref<Order | null>(null)
const note = ref('')

const load = async () => {
  loading.value = true

  const [err, data] = await api.get(Number(route.params.id))

  loading.value = false

  if (err)
    return ErrorPopup(err.message)

  order.value = data
  note.value = data.note
}

onMounted(load)

const setStatus = async (status: OrderStatus) => {
  saving.value = true

  const [err] = await api.update(Number(route.params.id), { status })

  saving.value = false

  if (err)
    return ErrorPopup(err.message)

  SuccessToast('Durum güncellendi')
  await load()
}

const saveNote = async () => {
  saving.value = true

  const [err] = await api.update(Number(route.params.id), { note: note.value })

  saving.value = false

  if (err)
    return ErrorPopup(err.message)

  SuccessToast('Not kaydedildi')
}

const tutar = (v: string) =>
  `${Number.parseFloat(v).toLocaleString('tr-TR', { minimumFractionDigits: 2 })} ₺`
</script>

<template>
  <div v-if="order">
    <div class="d-flex align-center gap-2 mb-6">
      <VBtn
        icon="tabler-arrow-left"
        variant="text"
        to="/siparisler"
      />
      <h4 class="text-h4">
        {{ order.order_no }}
      </h4>
      <VChip :color="STATUS_COLORS[order.status]">
        {{ STATUS_LABELS[order.status] }}
      </VChip>
    </div>

    <VRow>
      <VCol cols="12" md="7">
        <VCard class="mb-6">
          <VCardItem>
            <VCardTitle>Ürünler</VCardTitle>
          </VCardItem>
          <VCardText>
            <div
              v-for="item in order.items"
              :key="item.product_name"
              class="d-flex justify-space-between py-2 border-b"
            >
              <span>{{ item.product_name }} × {{ item.quantity }}</span>
              <span>{{ tutar(item.price_at_order) }}</span>
            </div>

            <div class="d-flex justify-space-between pt-4">
              <span class="text-medium-emphasis">Ara Toplam</span>
              <span>{{ tutar(order.items_total) }}</span>
            </div>
            <div class="d-flex justify-space-between">
              <span class="text-medium-emphasis">Teslimat</span>
              <span>{{ tutar(order.delivery_fee) }}</span>
            </div>
            <div class="d-flex justify-space-between text-h6 pt-2">
              <span>Toplam</span>
              <span>{{ tutar(order.total) }}</span>
            </div>
          </VCardText>
        </VCard>

        <VCard>
          <VCardItem>
            <VCardTitle>Teslimat</VCardTitle>
          </VCardItem>
          <VCardText>
            <p><strong>Alıcı:</strong> {{ order.recipient_name }} — {{ order.recipient_phone }}</p>
            <p><strong>Adres:</strong> {{ order.delivery_address }}</p>
            <p><strong>Tarih:</strong> {{ order.delivery_date }} · {{ order.delivery_slot }}</p>
            <VAlert
              v-if="order.card_message"
              type="info"
              variant="tonal"
              class="mt-4"
            >
              <strong>Kart mesajı:</strong> {{ order.card_message }}
            </VAlert>
          </VCardText>
        </VCard>
      </VCol>

      <VCol cols="12" md="5">
        <VCard class="mb-6">
          <VCardItem>
            <VCardTitle>Sipariş Veren</VCardTitle>
          </VCardItem>
          <VCardText>
            <p>{{ order.buyer_name }}</p>
            <p>{{ order.buyer_phone }}</p>
            <p v-if="order.buyer_email">
              {{ order.buyer_email }}
            </p>
          </VCardText>
        </VCard>

        <VCard class="mb-6">
          <VCardItem>
            <VCardTitle>Durum</VCardTitle>
          </VCardItem>
          <VCardText class="d-flex flex-column gap-2">
            <VBtn
              v-if="order.status === 'pending'"
              :loading="saving"
              @click="setStatus('confirmed')"
            >
              Onayla
            </VBtn>
            <VBtn
              v-if="order.status === 'confirmed'"
              :loading="saving"
              color="success"
              @click="setStatus('delivered')"
            >
              Teslim Edildi
            </VBtn>
            <VBtn
              v-if="order.status !== 'cancelled' && order.status !== 'delivered'"
              :loading="saving"
              color="error"
              variant="outlined"
              @click="setStatus('cancelled')"
            >
              İptal Et
            </VBtn>
          </VCardText>
        </VCard>

        <VCard>
          <VCardItem>
            <VCardTitle>Not</VCardTitle>
          </VCardItem>
          <VCardText>
            <VTextarea
              v-model="note"
              rows="3"
              placeholder="Kendi notunuz..."
            />
            <VBtn
              :loading="saving"
              class="mt-2"
              @click="saveNote"
            >
              Kaydet
            </VBtn>
          </VCardText>
        </VCard>
      </VCol>
    </VRow>
  </div>
</template>
```

- [ ] **Step 5: Navigasyonu kontrol et**

`frontend/idare/src/navigation/vertical/index.ts` içinde `siparisler` zaten var
mı bak. Varsa dokunma.

- [ ] **Step 6: Tarayıcıda doğrula**

```bash
cd frontend/idare && pnpm dev
```
`http://localhost:5173/siparisler` → sipariş listesi. Satıra tıkla → detay.
"Onayla" → durum değişmeli. Not yaz → kaydet. Filtre çalışmalı.

- [ ] **Step 7: Build ve lint**

Run: `cd frontend/idare && pnpm build && npx eslint src/pages/siparisler.vue "src/pages/siparisler/[id].vue" src/composables/useOrders.ts src/model/order.ts -c .eslintrc.cjs`
Expected: Build başarılı, lint temiz.

- [ ] **Step 8: Commit**

```bash
git add frontend/idare/src/
git commit -m "feat: admin sipariş listesi ve detayı

Placeholder gerçek oldu. Liste: durum filtresi, alıcı+ürünler, tutar.
Detay: ürünler, teslimat, kart mesajı, sipariş veren, durum butonları,
esnaf notu.

Durum butonları bağlama duyarlı: pending'de 'Onayla', confirmed'de
'Teslim Edildi'. Statü sayısı minimumda (spec §2.4)."
```

---

## Task 11: Uçtan uca doğrulama

- [ ] **Step 1: Tüm testler**

```bash
cd /Users/omerkoc/GolandProjects/cicekci
make test 2>&1 | grep -cE "^--- PASS"
make test 2>&1 | grep -c FAIL
cd frontend/app && pnpm test
```
Expected: Backend FAIL=0 (221'den fazla PASS), frontend 19 test PASS.

- [ ] **Step 2: Build'ler**

```bash
cd frontend/app && pnpm build
cd ../idare && pnpm build
cd ../../backend && go build ./...
```
Expected: Üçü de hatasız.

- [ ] **Step 3: MÜŞTERİ AKIŞI — uçtan uca**

Backend + iki frontend çalışırken:
1. `localhost:3000/urunler` → bir ürüne gir → "Sepete Ekle"
2. Başka bir ürün daha ekle → rozet "2" olmalı
3. Drawer'da adet artır → ara toplam güncellenmeli
4. Sayfayı yenile → sepet korunmalı
5. "Siparişi Tamamla" → form
6. "Alıcı benim" işaretle → alıcı alanları dolmalı
7. Gönder → `/siparis/tamam` → sipariş no görünmeli
8. Rozet sıfırlanmalı

- [ ] **Step 4: FİYAT MANİPÜLASYONU — güvenlik testi**

```bash
# Sepete ürün ekle, sonra DevTools console'da:
# localStorage.setItem('cicekci_sepet', JSON.stringify([{product_id:1,name:"x",slug:"x",price:"1.00",image:"",quantity:1}]))
# Sayfayı yenile, sipariş ver.
```
Sonra siparişi kontrol et:
```bash
curl -s -b /tmp/c.txt localhost:8080/api/admin/orders | head -c 400
```
**Sipariş tutarı GERÇEK fiyattan hesaplanmış olmalı, 1.00'dan değil.**
Bu testi geçemezse Faz 3'te para kaybı olur.

- [ ] **Step 5: ESNAF AKIŞI**

1. `localhost:5173/siparisler` → yeni sipariş "Yeni" durumunda görünmeli
2. Detaya gir → ürünler, alıcı, adres, kart mesajı doğru
3. "Onayla" → "Onaylandı"
4. "Teslim Edildi" → "Teslim Edildi"
5. Filtre "Yeni" → liste boş olmalı

- [ ] **Step 6: Kenar durumlar**

```bash
# Boş sepetle /siparis → /urunler'e yönlenmeli
# Geçmiş tarih → Türkçe hata, sepet korunmalı
curl -s -X POST localhost:8080/api/orders -H 'Content-Type: application/json' -d '{
  "items": [{"product_id": 1, "quantity": 1}],
  "buyer": {"name": "A", "phone": "1"},
  "recipient": {"name": "B", "phone": "2"},
  "delivery": {"address": "x", "date": "2020-01-01", "slot": "12:00-15:00"}
}'
# Expected: 400, "teslimat tarihi geçmişte olamaz"

# Pasif ürün
# Panelden bir ürünü pasif yap, sonra onu sipariş etmeyi dene
# Expected: 400, "artık satışta değil"
```

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat: Faz 2 tamamlandı — sepet ve sipariş uçtan uca doğrulandı

Müşteri akışı: sepet → form → sipariş. Esnaf akışı: liste → detay →
durum. Fiyat manipülasyonu testi geçti: localStorage'da 1.00 yazılsa
bile sunucu gerçek fiyattan hesapladı."
```

---

## Faz 2 Bitiş Kriterleri

- [ ] `make test` FAIL=0; `pnpm test` (app) geçiyor
- [ ] Üç build de hatasız (backend, app, idare)
- [ ] Sepet localStorage'da yaşıyor, sayfa yenilenince korunuyor
- [ ] Header rozeti gerçek sayıyı gösteriyor
- [ ] **Fiyat manipülasyonu sunucuda reddediliyor** (en kritik)
- [ ] Sipariş tek transaction — yarım sipariş kalmıyor (testle kanıtlı)
- [ ] Pasif ürün, geçmiş tarih, geçersiz slot, cutoff sonrası aynı gün reddediliyor
- [ ] `order_no` her gün 0001'den başlıyor, çakışmada retry çalışıyor
- [ ] Esnaf panelde siparişi görüyor, durumu değiştirebiliyor
- [ ] `price_at_order` kopya: ürün fiyatı değişince eski sipariş bozulmuyor
- [ ] Sipariş sonrası sepet temizleniyor; hatada korunuyor
- [ ] WhatsApp butonu bozulmadı (spec §2.3)
- [ ] `/hesabim/*` mock sayfalarına dokunulmadı

**Sonraki:** Faz 3 — ödeme entegrasyonu. Karar bekleyenler: provizyon mu direkt
çekim mi (MVP §3.2'nin açık sorusu), stok modeli, üyelik, bildirim, ETBİS.

**Esnaftan öğrenilecek** (spec §7): teslimat ücreti (bölgeye göre değişiyorsa
tasarım revize), saat dilimleri, aynı gün cutoff, `delivered` statüsü gerekli mi.
