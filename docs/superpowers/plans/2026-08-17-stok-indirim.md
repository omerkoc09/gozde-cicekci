# Stok Yönetimi ve İndirimli Ürünler Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ürünlere isteğe bağlı stok takibi ve kota bazlı indirim ekle; stok bitince satışı engelle, indirimli ürünleri ayrı sayfada göster.

**Architecture:** `products` tablosuna 6 kolon + yeni `stock_movements` tablosu. Stok, sipariş anında `stock_reserved` ile rezerve edilir (tek atomik `UPDATE ... WHERE`), PayTR callback'inde kesinleşir, 20 dk'da süpürücü ile serbest kalır. İndirim `discount_sold < discount_quota` koşuluyla türetilir — kota dolunca kendiliğinden söner, ayrı bir iş çalışmaz. "İndirimli Ürünler" sanal kategori: `categories` tablosuna kayıt eklenmez.

**Tech Stack:** Go 1.2x (pgx/v5, shopspring/decimal, testify), PostgreSQL, golang-migrate, Nuxt 4 (public site), Vue 3 + Vuetify (idare paneli), Vitest.

**Spec:** `docs/superpowers/specs/2026-08-17-stok-indirim-design.md`

## Global Constraints

- **Dil:** Kod yorumları, commit mesajları ve kullanıcıya görünen tüm metinler **Türkçe**. Hata mesajları müşteriye gösterilir — anlaşılır Türkçe olmalı.
- **Testler:** `make test` ile çalıştırılır (`go test -p 1`). `go test ./...` KULLANMA — paketler aynı test DB'sini paylaşıyor ve paralel koşarsa birbirlerinin verisini siler.
- **Dev Postgres portu 5435**, test DB portu 5434. `TEST_DATABASE_URL` yoksa testler skip olur; `make db-up` çalıştırılmalı.
- **Fiyat tipi:** Go tarafında `decimal.Decimal`, JSON'a `StringFixed(2)` ile string olarak gider. Float KULLANMA.
- **Mevcut davranış korunur:** `track_stock` varsayılanı `false` — migration sonrası hiçbir ürün "tükendi" görünmemeli, site aynen çalışmalı.
- **Sipariş anındaki fiyat bağlayıcıdır:** `order_items.price_at_order`'a yazılan fiyat sonradan değişmez.
- **Yarış korumasında "oku → karar ver → yaz" YASAK.** Stok kontrolü daima tek `UPDATE ... WHERE` ifadesinde, koşul `WHERE` içinde.
- **E2E doğrulama Nuxt proxy üzerinden** yapılır — Go'ya doğrudan curl atmak proxy hatalarını gizler.

---

## File Structure

**Backend — yeni dosyalar:**
- `backend/migrations/000011_stock_discount.up.sql` / `.down.sql` — şema
- `backend/internal/product/stock.go` — stok/indirim saf mantığı (`Available`, `DiscountActive`, `EffectivePrice`)
- `backend/internal/product/stock_test.go` — saf mantık testleri (DB'siz)
- `backend/internal/product/stock_store.go` — `Reserve`, `Release`, `Commit`, `ManualAdjust`, `ListMovements`
- `backend/internal/product/stock_store_test.go` — yarış durumu dâhil DB testleri
- `backend/internal/product/sweeper.go` — süresi geçen rezervasyonları serbest bırakan goroutine
- `backend/internal/product/sweeper_test.go`

**Backend — değişen dosyalar:**
- `backend/internal/product/model.go` — `Product`'a stok/indirim alanları, `Filter`'a `DiscountedOnly`
- `backend/internal/product/store.go` — `productSelect` + `scanProduct` + `ListPublic` filtresi
- `backend/internal/product/service.go` — stok/indirim güncelleme, manuel hareket
- `backend/internal/order/service.go` — rezervasyon, kesinleşme, iade
- `backend/internal/order/store.go` — `SetPaidTx` (tx-aware varyant)
- `backend/internal/api/app/product_view.go` — `old_price`, `in_stock`, `discount_remaining`
- `backend/internal/api/idare/product_view.go` + `product_handler.go` — panel alanları, stok uçları
- `backend/internal/api/idare/router.go`, `backend/internal/api/app/product_handler.go`
- `backend/pkg/database/testdb.go` — TRUNCATE listesine `stock_movements`
- `backend/cmd/server/main.go` — süpürücüyü başlat

**Frontend — public (`frontend/app/app/`):**
- `types/api.ts`, `components/ProductCard.vue`, `pages/urun/[slug].vue`, `pages/indirimli.vue` (yeni), `components/TheHeader.vue`, `utils/whatsapp.ts`, `composables/useCart.ts`

**Frontend — idare (`frontend/idare/src/`):**
- `pages/urunler/index.vue` (stok sütunu + `+`/`−`), `pages/urunler/[id].vue` (stok/indirim bölümleri + Hareketler sekmesi), `components/StokDusurDialog.vue` (yeni)

---

## Task 1: Migration — şema

**Files:**
- Create: `backend/migrations/000011_stock_discount.up.sql`
- Create: `backend/migrations/000011_stock_discount.down.sql`
- Modify: `backend/pkg/database/testdb.go:44-52`

**Interfaces:**
- Consumes: mevcut `products`, `orders`, `admin_users` tabloları
- Produces: `products.track_stock/stock_quantity/stock_reserved/discount_price/discount_quota/discount_sold` kolonları, `stock_movements` tablosu

- [ ] **Step 1: Migration up dosyasını yaz**

`backend/migrations/000011_stock_discount.up.sql`:

```sql
-- Stok takibi ürün başına İSTEĞE BAĞLI. Varsayılan false: mevcut ürünler
-- aynen çalışmaya devam eder, hiçbiri "tükendi" görünmez.
ALTER TABLE products
    ADD COLUMN track_stock     BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN stock_quantity  INT     NOT NULL DEFAULT 0,
    -- Ödeme bekleyen adet. Satılabilir = stock_quantity - stock_reserved.
    ADD COLUMN stock_reserved  INT     NOT NULL DEFAULT 0,
    ADD COLUMN discount_price  NUMERIC(10,2),
    ADD COLUMN discount_quota  INT,
    ADD COLUMN discount_sold   INT     NOT NULL DEFAULT 0;

ALTER TABLE products
    ADD CONSTRAINT products_stock_nonneg
        CHECK (stock_quantity >= 0 AND stock_reserved >= 0),
    ADD CONSTRAINT products_discount_sold_nonneg
        CHECK (discount_sold >= 0),
    -- Kotasız indirim süresiz indirimdir; bu özelliğin amacı değil.
    ADD CONSTRAINT products_discount_pair
        CHECK ((discount_price IS NULL AND discount_quota IS NULL)
            OR (discount_price IS NOT NULL AND discount_quota > 0));

-- /indirimli sayfası bu koşulla okuyor
CREATE INDEX idx_products_discount_active ON products (id)
    WHERE discount_price IS NOT NULL;

-- Her stok değişiminin izi. İki soruyu cevaplar: "bu ay WhatsApp'tan kaç
-- sattım" ve "stok neden bu sayıda".
CREATE TABLE stock_movements (
    id             BIGSERIAL PRIMARY KEY,
    product_id     BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    delta          INT NOT NULL,          -- negatif: düşüş, pozitif: giriş
    reason         TEXT NOT NULL CHECK (reason IN
                     ('siparis','whatsapp_satisi','sayim_duzeltme',
                      'yeni_parti','iptal_iade','rezervasyon_iptal')),
    -- Sipariş silinirse hareket kaydı ölmemeli (order_items.product_id deseni)
    order_id       BIGINT REFERENCES orders(id) ON DELETE SET NULL,
    -- İndirim kotasının WhatsApp satışlarını da sayabilmesi için
    was_discounted BOOLEAN NOT NULL DEFAULT false,
    note           TEXT NOT NULL DEFAULT '',
    -- Hareketi yapan panel kullanıcısı; kullanıcı silinse de kayıt kalır
    admin_user_id  BIGINT REFERENCES admin_users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_stock_movements_product
    ON stock_movements (product_id, created_at DESC);
```

- [ ] **Step 2: Migration down dosyasını yaz**

`backend/migrations/000011_stock_discount.down.sql`:

```sql
DROP TABLE IF EXISTS stock_movements;

DROP INDEX IF EXISTS idx_products_discount_active;

ALTER TABLE products
    DROP CONSTRAINT IF EXISTS products_stock_nonneg,
    DROP CONSTRAINT IF EXISTS products_discount_sold_nonneg,
    DROP CONSTRAINT IF EXISTS products_discount_pair;

ALTER TABLE products
    DROP COLUMN IF EXISTS track_stock,
    DROP COLUMN IF EXISTS stock_quantity,
    DROP COLUMN IF EXISTS stock_reserved,
    DROP COLUMN IF EXISTS discount_price,
    DROP COLUMN IF EXISTS discount_quota,
    DROP COLUMN IF EXISTS discount_sold;
```

- [ ] **Step 3: Test DB temizlik listesine yeni tabloyu ekle**

`backend/pkg/database/testdb.go` içindeki `truncateAll` fonksiyonunda TRUNCATE listesine `stock_movements` ekle. Eklenmezse testler birbirinin verisini görür:

```go
	_, err := pool.Exec(context.Background(), `
		TRUNCATE products, product_slugs, product_images,
		         categories, product_categories, admin_users, slides,
		         orders, order_items, customers,
		         option_groups, option_values, product_option_groups,
		         order_item_options, stock_movements
		RESTART IDENTITY CASCADE
	`)
```

- [ ] **Step 4: Migration'ı test DB'sine uygula ve doğrula**

```bash
make db-up
make test-db-migrate
```

Beklenen: hata yok. Kolonların geldiğini doğrula:

```bash
docker compose exec -T postgres_test psql -U cicekci -d cicekci_test -c "\d products" | grep -E "track_stock|discount_"
```

Beklenen: 6 satır (track_stock, stock_quantity, stock_reserved, discount_price, discount_quota, discount_sold).

- [ ] **Step 5: Down migration'ın çalıştığını doğrula, sonra tekrar up**

```bash
migrate -path backend/migrations -database "$TEST_DATABASE_URL" down 1
migrate -path backend/migrations -database "$TEST_DATABASE_URL" up
```

Beklenen: iki komut da hatasız. (Down bozuksa geri alma imkânı kalmaz — şimdi öğrenmek gerekir.)

- [ ] **Step 6: Commit**

```bash
git add backend/migrations/000011_stock_discount.up.sql \
        backend/migrations/000011_stock_discount.down.sql \
        backend/pkg/database/testdb.go
git commit -m "feat(stok): migration 11 — stok ve indirim kolonları, hareket tablosu"
```

---

## Task 2: Saf mantık — satılabilirlik ve indirim

**Files:**
- Create: `backend/internal/product/stock.go`
- Create: `backend/internal/product/stock_test.go`
- Modify: `backend/internal/product/model.go`

**Interfaces:**
- Consumes: Task 1'in kolonları
- Produces:
  - `Product` alanları: `TrackStock bool`, `StockQuantity int`, `StockReserved int`, `DiscountPrice *decimal.Decimal`, `DiscountQuota *int`, `DiscountSold int`
  - `func (p Product) Available() (int, bool)` — (adet, sınırlı mı). `sınırlı=false` → takipsiz, sonsuz
  - `func (p Product) InStock() bool`
  - `func (p Product) DiscountActive() bool`
  - `func (p Product) EffectivePrice() decimal.Decimal`
  - `func (p Product) OldPrice() *decimal.Decimal` — indirim aktifse normal fiyat, değilse nil
  - `func (p Product) DiscountRemaining() *int`

- [ ] **Step 1: Model alanlarını ekle**

`backend/internal/product/model.go` içinde `Product` struct'ına ekle (mevcut alanlar korunur):

```go
type Product struct {
	ID          int64
	Name        string
	Slug        string
	Description string
	Price       decimal.Decimal
	IsActive    bool
	IsFeatured  bool
	CategoryIDs []int64
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// Stok takibi ürün başına isteğe bağlı. TrackStock false ise ürün
	// sınırsız satılır ve diğer stok alanları anlamsızdır.
	TrackStock    bool
	StockQuantity int
	// StockReserved ödeme bekleyen adet — satılabilir hesabından düşülür.
	StockReserved int

	// DiscountPrice nil ise indirim yok. Kota dolduğunda indirim
	// kendiliğinden söner (DiscountActive), alan temizlenmez.
	DiscountPrice *decimal.Decimal
	DiscountQuota *int
	DiscountSold  int
}
```

Aynı dosyada `Filter` struct'ına ekle:

```go
	// DiscountedOnly true ise yalnızca indirimi AKTİF ürünler döner —
	// /indirimli sayfası bunu kullanıyor (FeaturedOnly ile aynı desen).
	DiscountedOnly bool
```

Ayrıca `UpdateInput`'a stok/indirim alanları ekle (PATCH semantiği, nil = değişmez):

```go
	TrackStock    *bool
	StockQuantity *int
	// DiscountPrice ve DiscountQuota BİRLİKTE set edilir. İkisi de nil
	// değilse indirim açılır; ClearDiscount true ise indirim kaldırılır.
	DiscountPrice *decimal.Decimal
	DiscountQuota *int
	ClearDiscount bool
```

- [ ] **Step 2: Failing testleri yaz**

`backend/internal/product/stock_test.go` (DB gerektirmez — saf mantık):

```go
package product

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

func intPtr(i int) *int { return &i }

func decPtr(s string) *decimal.Decimal {
	d := dec(s)
	return &d
}

func TestAvailable_TakipsizUrunSinirsiz(t *testing.T) {
	p := Product{TrackStock: false, StockQuantity: 0, StockReserved: 0}

	_, sinirli := p.Available()

	assert.False(t, sinirli, "takipsiz ürün sınırsız olmalı")
	assert.True(t, p.InStock(), "takipsiz ürün stokta 0 olsa bile satılabilir")
}

func TestAvailable_RezerveDusulur(t *testing.T) {
	p := Product{TrackStock: true, StockQuantity: 10, StockReserved: 3}

	adet, sinirli := p.Available()

	assert.True(t, sinirli)
	assert.Equal(t, 7, adet)
	assert.True(t, p.InStock())
}

func TestAvailable_TumuRezerveyseTukendi(t *testing.T) {
	p := Product{TrackStock: true, StockQuantity: 2, StockReserved: 2}

	adet, _ := p.Available()

	assert.Equal(t, 0, adet)
	assert.False(t, p.InStock(), "hepsi rezerveyse tükendi sayılmalı")
}

func TestAvailable_NegatifeDusmez(t *testing.T) {
	// Elle düzeltme sonrası rezerve stoktan büyük kalabilir; müşteriye
	// negatif adet göstermek yerine 0 döner.
	p := Product{TrackStock: true, StockQuantity: 1, StockReserved: 3}

	adet, _ := p.Available()

	assert.Equal(t, 0, adet)
}

func TestDiscountActive_KotaVarkenAktif(t *testing.T) {
	p := Product{Price: dec("1850.00"), DiscountPrice: decPtr("1450.00"),
		DiscountQuota: intPtr(10), DiscountSold: 3}

	assert.True(t, p.DiscountActive())
	assert.Equal(t, "1450", p.EffectivePrice().String())
	assert.Equal(t, "1850", p.OldPrice().String())
	assert.Equal(t, 7, *p.DiscountRemaining())
}

func TestDiscountActive_KotaDolunca_Soner(t *testing.T) {
	p := Product{Price: dec("1850.00"), DiscountPrice: decPtr("1450.00"),
		DiscountQuota: intPtr(10), DiscountSold: 10}

	assert.False(t, p.DiscountActive(), "kota dolunca indirim sönmeli")
	assert.Equal(t, "1850", p.EffectivePrice().String(), "normal fiyata dönmeli")
	assert.Nil(t, p.OldPrice(), "indirim yoksa eski fiyat gösterilmez")
	assert.Nil(t, p.DiscountRemaining())
}

func TestDiscountActive_IndirimYok(t *testing.T) {
	p := Product{Price: dec("1850.00")}

	assert.False(t, p.DiscountActive())
	assert.Equal(t, "1850", p.EffectivePrice().String())
	assert.Nil(t, p.OldPrice())
}
```

- [ ] **Step 3: Testin başarısız olduğunu doğrula**

Run: `cd backend && go test ./internal/product/ -run 'TestAvailable|TestDiscount' -v`
Expected: FAIL — `p.Available undefined`, `p.DiscountActive undefined`.

- [ ] **Step 4: Saf mantığı yaz**

`backend/internal/product/stock.go`:

```go
package product

import "github.com/shopspring/decimal"

// Available satılabilir adedi ve bu sayının anlamlı olup olmadığını döner.
// sinirli=false → stok takibi kapalı, ürün sınırsız satılır (adet 0 döner
// ama anlamı yoktur).
//
// Rezerve edilmiş (ödeme bekleyen) adet düşülür: müşteri ödeme ekranındayken
// aynı ürünü başkası satın alamamalı.
func (p Product) Available() (adet int, sinirli bool) {
	if !p.TrackStock {
		return 0, false
	}
	kalan := p.StockQuantity - p.StockReserved
	// Elle düzeltme sonrası rezerve stoktan büyük kalabilir — negatif
	// göstermek yerine tükendi say.
	if kalan < 0 {
		return 0, true
	}
	return kalan, true
}

// InStock ürün satın alınabilir mi. Takipsiz ürün her zaman true.
func (p Product) InStock() bool {
	adet, sinirli := p.Available()
	return !sinirli || adet > 0
}

// DiscountActive indirim yürürlükte mi. Kota dolduğunda kendiliğinden
// false döner — ayrıca "indirimi kapat" işi çalıştırmaya gerek yok.
func (p Product) DiscountActive() bool {
	return p.DiscountPrice != nil && p.DiscountQuota != nil &&
		p.DiscountSold < *p.DiscountQuota
}

// EffectivePrice müşterinin ödeyeceği fiyat.
func (p Product) EffectivePrice() decimal.Decimal {
	if p.DiscountActive() {
		return *p.DiscountPrice
	}
	return p.Price
}

// OldPrice indirim aktifse üstü çizili gösterilecek normal fiyat.
func (p Product) OldPrice() *decimal.Decimal {
	if !p.DiscountActive() {
		return nil
	}
	fiyat := p.Price
	return &fiyat
}

// DiscountRemaining kalan indirimli adet; indirim yoksa nil.
func (p Product) DiscountRemaining() *int {
	if !p.DiscountActive() {
		return nil
	}
	kalan := *p.DiscountQuota - p.DiscountSold
	return &kalan
}
```

- [ ] **Step 5: Testlerin geçtiğini doğrula**

Run: `cd backend && go test ./internal/product/ -run 'TestAvailable|TestDiscount' -v`
Expected: PASS (7 test).

- [ ] **Step 6: Commit**

```bash
git add backend/internal/product/stock.go backend/internal/product/stock_test.go \
        backend/internal/product/model.go
git commit -m "feat(stok): satılabilirlik ve indirim saf mantığı"
```

---

## Task 3: Store — okuma katmanı ve indirim filtresi

**Files:**
- Modify: `backend/internal/product/store.go:22-46` (productSelect, scanProduct)
- Modify: `backend/internal/product/store.go:262-316` (ListPublic)
- Modify: `backend/internal/product/store.go:170-227` (Update)
- Test: `backend/internal/product/store_test.go`

**Interfaces:**
- Consumes: Task 2'nin `Product` alanları, `Filter.DiscountedOnly`
- Produces: Stok/indirim alanları dolu dönen `productSelect`; `ListPublic` içinde `DiscountedOnly` filtresi; `Update` ile stok/indirim yazımı

- [ ] **Step 1: Failing testi yaz**

`backend/internal/product/store_test.go` sonuna ekle:

```go
// setStock test için doğrudan stok/indirim alanlarını yazar — service
// katmanına bağımlı olmadan store davranışını sınamak için.
func setStock(t *testing.T, pool *pgxpool.Pool, id int64,
	track bool, qty, reserved int) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`UPDATE products SET track_stock=$2, stock_quantity=$3, stock_reserved=$4
		 WHERE id=$1`, id, track, qty, reserved)
	require.NoError(t, err)
}

func setDiscount(t *testing.T, pool *pgxpool.Pool, id int64,
	price string, quota, sold int) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`UPDATE products SET discount_price=$2, discount_quota=$3, discount_sold=$4
		 WHERE id=$1`, id, price, quota, sold)
	require.NoError(t, err)
}

func TestStore_GetByID_StokAlanlariOkunur(t *testing.T) {
	store, pool, ctx := newTestStore(t)

	p, err := store.Create(ctx, CreateInput{
		Name: "Gül Buketi", Price: price(t, "1850.00"), IsActive: true,
	}, "gul-buketi")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 10, 3)
	setDiscount(t, pool, p.ID, "1450.00", 10, 2)

	got, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err)

	assert.True(t, got.TrackStock)
	assert.Equal(t, 10, got.StockQuantity)
	assert.Equal(t, 3, got.StockReserved)
	require.NotNil(t, got.DiscountPrice)
	assert.Equal(t, "1450", got.DiscountPrice.String())
	require.NotNil(t, got.DiscountQuota)
	assert.Equal(t, 10, *got.DiscountQuota)
	assert.Equal(t, 2, got.DiscountSold)
	assert.True(t, got.DiscountActive())
}

func TestStore_GetByID_IndirimsizUrun_NilDoner(t *testing.T) {
	store, _, ctx := newTestStore(t)

	p, err := store.Create(ctx, CreateInput{
		Name: "Orkide", Price: price(t, "900.00"), IsActive: true,
	}, "orkide")
	require.NoError(t, err)

	got, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err)

	assert.False(t, got.TrackStock, "varsayılan takipsiz olmalı")
	assert.Nil(t, got.DiscountPrice)
	assert.Nil(t, got.DiscountQuota)
	assert.False(t, got.DiscountActive())
}

func TestStore_ListPublic_DiscountedOnly(t *testing.T) {
	store, pool, ctx := newTestStore(t)

	indirimli, err := store.Create(ctx, CreateInput{
		Name: "İndirimli Buket", Price: price(t, "1850.00"), IsActive: true,
	}, "indirimli-buket")
	require.NoError(t, err)
	setDiscount(t, pool, indirimli.ID, "1450.00", 10, 3)

	// Kotası dolmuş ürün indirimli listede GÖRÜNMEMELİ
	kotasiDolmus, err := store.Create(ctx, CreateInput{
		Name: "Kotası Dolmuş", Price: price(t, "1200.00"), IsActive: true,
	}, "kotasi-dolmus")
	require.NoError(t, err)
	setDiscount(t, pool, kotasiDolmus.ID, "1000.00", 5, 5)

	_, err = store.Create(ctx, CreateInput{
		Name: "Normal Ürün", Price: price(t, "700.00"), IsActive: true,
	}, "normal-urun")
	require.NoError(t, err)

	list, err := store.ListPublic(ctx, Filter{DiscountedOnly: true, Limit: 50})
	require.NoError(t, err)

	require.Len(t, list, 1, "yalnızca indirimi AKTİF ürün dönmeli")
	assert.Equal(t, "İndirimli Buket", list[0].Name)
}

func TestStore_Update_StokVeIndirimYazilir(t *testing.T) {
	store, _, ctx := newTestStore(t)

	p, err := store.Create(ctx, CreateInput{
		Name: "Lilyum", Price: price(t, "600.00"), IsActive: true,
	}, "lilyum")
	require.NoError(t, err)

	track := true
	qty := 25
	dp := price(t, "450.00")
	quota := 8
	updated, err := store.Update(ctx, p.ID, UpdateInput{
		TrackStock: &track, StockQuantity: &qty,
		DiscountPrice: &dp, DiscountQuota: &quota,
	}, "")
	require.NoError(t, err)

	assert.True(t, updated.TrackStock)
	assert.Equal(t, 25, updated.StockQuantity)
	require.NotNil(t, updated.DiscountPrice)
	assert.Equal(t, "450", updated.DiscountPrice.String())
	assert.Equal(t, 8, *updated.DiscountQuota)
}

func TestStore_Update_ClearDiscount_SayaciSifirlar(t *testing.T) {
	store, pool, ctx := newTestStore(t)

	p, err := store.Create(ctx, CreateInput{
		Name: "Papatya", Price: price(t, "300.00"), IsActive: true,
	}, "papatya")
	require.NoError(t, err)
	setDiscount(t, pool, p.ID, "250.00", 5, 4)

	updated, err := store.Update(ctx, p.ID, UpdateInput{ClearDiscount: true}, "")
	require.NoError(t, err)

	assert.Nil(t, updated.DiscountPrice)
	assert.Nil(t, updated.DiscountQuota)
	// Sıfırlanmazsa aynı ürüne ikinci indirim girildiğinde kota baştan
	// dolu görünür (spec §5.2).
	assert.Equal(t, 0, updated.DiscountSold, "sayaç sıfırlanmalı")
}
```

- [ ] **Step 2: Testin başarısız olduğunu doğrula**

Run: `cd backend && go test ./internal/product/ -run 'TestStore_GetByID_Stok|TestStore_ListPublic_Discounted|TestStore_Update_Stok|TestStore_Update_Clear' -v`
Expected: FAIL — scan hatası / bilinmeyen alan.

- [ ] **Step 3: productSelect ve scanProduct'ı güncelle**

`backend/internal/product/store.go` — `productSelect` sabitine yeni kolonları ekle (sıra `scanProduct` ile birebir aynı olmalı):

```go
const productSelect = `
	SELECT p.id, p.name,
	       COALESCE(ps.slug, ''),
	       p.description, p.price, p.is_active, p.is_featured,
	       COALESCE(
	         (SELECT array_agg(pc.category_id ORDER BY pc.category_id)
	          FROM product_categories pc WHERE pc.product_id = p.id),
	         '{}'
	       ),
	       p.created_at, p.updated_at,
	       p.track_stock, p.stock_quantity, p.stock_reserved,
	       p.discount_price, p.discount_quota, p.discount_sold
	FROM products p
	LEFT JOIN product_slugs ps ON ps.product_id = p.id AND ps.is_current
`
```

`scanProduct` fonksiyonunu güncelle:

```go
func scanProduct(row pgx.Row) (*Product, error) {
	var p Product
	err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price,
		&p.IsActive, &p.IsFeatured, &p.CategoryIDs, &p.CreatedAt, &p.UpdatedAt,
		&p.TrackStock, &p.StockQuantity, &p.StockReserved,
		&p.DiscountPrice, &p.DiscountQuota, &p.DiscountSold)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("ürün scan: %w", err)
	}
	return &p, nil
}
```

- [ ] **Step 4: ListPublic'e indirim filtresini ekle**

`ListPublic` içinde, `FeaturedOnly` bloğunun hemen altına ekle (sabit koşul, argüman almıyor — aynı desen):

```go
	// İndirimi AKTİF ürünler: kotası dolmuş indirim sayılmaz (spec §6.2).
	if f.DiscountedOnly {
		query += ` AND p.discount_price IS NOT NULL
		           AND p.discount_sold < p.discount_quota`
	}
```

- [ ] **Step 5: Update'e stok/indirim alanlarını ekle**

`backend/internal/product/store.go` içindeki `Update` fonksiyonunda, mevcut `COALESCE` deseniyle aynı şekilde yeni alanları ekle. `ClearDiscount` ayrı ele alınır çünkü NULL yazmak COALESCE ile ifade edilemez:

```go
	// ClearDiscount NULL yazar — COALESCE deseni NULL'ı "değişme" olarak
	// yorumladığı için ayrı ifade gerekiyor. discount_sold da sıfırlanır:
	// aksi halde ürüne ikinci kez indirim girildiğinde kota baştan dolu
	// görünür (spec §5.2).
	if in.ClearDiscount {
		if _, err := tx.Exec(ctx,
			`UPDATE products
			    SET discount_price = NULL, discount_quota = NULL,
			        discount_sold = 0, updated_at = now()
			  WHERE id = $1`, id); err != nil {
			return nil, fmt.Errorf("indirim kaldır: %w", err)
		}
	} else if in.DiscountPrice != nil && in.DiscountQuota != nil {
		// Yeni indirim: sayaç sıfırdan başlar.
		if _, err := tx.Exec(ctx,
			`UPDATE products
			    SET discount_price = $2, discount_quota = $3,
			        discount_sold = 0, updated_at = now()
			  WHERE id = $1`, id, *in.DiscountPrice, *in.DiscountQuota); err != nil {
			return nil, fmt.Errorf("indirim yaz: %w", err)
		}
	}
```

Stok alanları normal COALESCE deseniyle mevcut UPDATE ifadesine eklenir:

```go
		   track_stock    = COALESCE($N, track_stock),
		   stock_quantity = COALESCE($N+1, stock_quantity),
```

**DİKKAT:** `$N` numaralarını mevcut sorgudaki son parametre numarasından devam ettir; `in.TrackStock` ve `in.StockQuantity` args listesine aynı sırayla eklenmeli. `stock_reserved` buradan YAZILMAZ — yalnızca Task 4'ün atomik ifadeleri değiştirir.

- [ ] **Step 6: Testlerin geçtiğini doğrula**

Run: `cd backend && go test ./internal/product/ -v`
Expected: PASS — yeni testler dâhil, mevcut store testleri de geçmeli (regresyon yok).

- [ ] **Step 7: Commit**

```bash
git add backend/internal/product/store.go backend/internal/product/store_test.go
git commit -m "feat(stok): store stok/indirim alanlarını okur, indirim filtresi"
```

---

## Task 4: Rezervasyon ve hareket kaydı — yarış durumu

**Files:**
- Create: `backend/internal/product/stock_store.go`
- Create: `backend/internal/product/stock_store_test.go`

**Interfaces:**
- Consumes: Task 1 şeması, Task 2 model alanları
- Produces:
  - `type Reason string` + sabitler: `ReasonSiparis`, `ReasonWhatsApp`, `ReasonSayim`, `ReasonYeniParti`, `ReasonIptalIade`, `ReasonRezervasyonIptal`
  - `type Movement struct{ ID, ProductID int64; Delta int; Reason Reason; OrderID *int64; WasDiscounted bool; Note string; AdminUserID *int64; CreatedAt time.Time }`
  - `type StockError struct{ ProductID int64; ProductName string; Available int }` + `Error() string`, `errors.Is(err, errorsx.ErrInvalidInput)` uyumu
  - `func (s *Store) Reserve(ctx, tx pgx.Tx, productID int64, qty int) error`
  - `func (s *Store) Release(ctx context.Context, productID int64, qty int, orderID *int64) error`
  - `func (s *Store) CommitReservation(ctx, tx pgx.Tx, productID int64, qty int, orderID int64, discounted bool) error`
  - `func (s *Store) RestoreStock(ctx context.Context, productID int64, qty int, orderID *int64) error`
  - `func (s *Store) ManualAdjust(ctx context.Context, in ManualAdjustInput) (*Product, error)`
  - `func (s *Store) ListMovements(ctx context.Context, productID int64, limit int) ([]Movement, error)`

- [ ] **Step 1: Failing testi yaz — yarış durumu dâhil**

`backend/internal/product/stock_store_test.go`:

```go
package product

import (
	"context"
	"sync"
	"testing"

	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reserveTx Reserve'i kendi transaction'ında çalıştırır — üretimde sipariş
// transaction'ına katılıyor, testte tek başına sınanıyor.
func reserveTx(t *testing.T, store *Store, productID int64, qty int) error {
	t.Helper()
	ctx := context.Background()
	tx, err := store.pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	if err := store.Reserve(ctx, tx, productID, qty); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func TestReserve_TakipsizUrun_HerZamanBasarili(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Takipsiz", Price: price(t, "100.00"), IsActive: true,
	}, "takipsiz")
	require.NoError(t, err)

	require.NoError(t, reserveTx(t, store, p.ID, 1000))

	got, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, got.StockReserved, "takipsiz üründe rezerve artmamalı")
	assert.Equal(t, 0, got.StockQuantity)
	_ = pool
}

func TestReserve_YeterliStok_RezerveArtar(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Gül", Price: price(t, "100.00"), IsActive: true,
	}, "gul")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 5, 0)

	require.NoError(t, reserveTx(t, store, p.ID, 3))

	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 3, got.StockReserved)
	assert.Equal(t, 5, got.StockQuantity, "fiziksel stok rezervasyonda değişmez")
}

func TestReserve_YetersizStok_HataDoner(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Orkide", Price: price(t, "100.00"), IsActive: true,
	}, "orkide")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 2, 0)

	err = reserveTx(t, store, p.ID, 3)

	require.Error(t, err)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)

	var se *StockError
	require.ErrorAs(t, err, &se, "müşteriye kaç adet kaldığı söylenebilmeli")
	assert.Equal(t, 2, se.Available)
	assert.Equal(t, "Orkide", se.ProductName)

	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 0, got.StockReserved, "başarısız rezervasyon iz bırakmamalı")
}

// TestReserve_YarisDurumu tasarımın EN KRİTİK iddiasını kanıtlar:
// son ürün asla iki kişiye satılamaz.
func TestReserve_YarisDurumu_TekKazanan(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Son Ürün", Price: price(t, "100.00"), IsActive: true,
	}, "son-urun")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 1, 0)

	const esZamanli = 20
	var wg sync.WaitGroup
	sonuc := make([]error, esZamanli)

	for i := 0; i < esZamanli; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sonuc[idx] = reserveTx(t, store, p.ID, 1)
		}(i)
	}
	wg.Wait()

	basarili := 0
	for _, err := range sonuc {
		if err == nil {
			basarili++
		}
	}

	assert.Equal(t, 1, basarili, "1 adetlik stoktan tam olarak 1 rezervasyon geçmeli")

	got, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.StockReserved, "rezerve stoğu aşamaz")
	adet, _ := got.Available()
	assert.Equal(t, 0, adet)
}

func TestCommitReservation_StokVeKotaDuser(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "100.00"), IsActive: true,
	}, "buket")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 10, 0)
	setDiscount(t, pool, p.ID, "80.00", 5, 0)
	require.NoError(t, reserveTx(t, store, p.ID, 2))

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, store.CommitReservation(ctx, tx, p.ID, 2, 0, true))
	require.NoError(t, tx.Commit(ctx))

	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 8, got.StockQuantity, "fiziksel stok düşmeli")
	assert.Equal(t, 0, got.StockReserved, "rezerve serbest kalmalı")
	assert.Equal(t, 2, got.DiscountSold, "indirimli satış kotayı tüketmeli")
}

func TestCommitReservation_TakipsizUrun_KotaYineTuketilir(t *testing.T) {
	// Stok ve indirim bağımsız kavramlar (spec §2): stok takibi kapalı olsa
	// da indirim kotası tüketilmeli, yoksa esnaf sınırsız indirimli satar.
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Takipsiz İndirimli", Price: price(t, "100.00"), IsActive: true,
	}, "takipsiz-indirimli")
	require.NoError(t, err)
	setDiscount(t, pool, p.ID, "80.00", 5, 0)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, store.CommitReservation(ctx, tx, p.ID, 2, 0, true))
	require.NoError(t, tx.Commit(ctx))

	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 2, got.DiscountSold, "takipsiz üründe de kota tüketilmeli")
	assert.Equal(t, 0, got.StockQuantity, "takipsiz üründe stok alanları değişmemeli")
	assert.Equal(t, 0, got.StockReserved)
}

func TestRelease_RezerveSerbestKalir(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Lale", Price: price(t, "100.00"), IsActive: true,
	}, "lale")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 5, 0)
	require.NoError(t, reserveTx(t, store, p.ID, 2))

	require.NoError(t, store.Release(ctx, p.ID, 2, nil))

	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 0, got.StockReserved)
	assert.Equal(t, 5, got.StockQuantity, "serbest bırakma fiziksel stoğu değiştirmez")

	hareketler, err := store.ListMovements(ctx, p.ID, 10)
	require.NoError(t, err)
	require.Len(t, hareketler, 1, "serbest bırakma iz bırakmalı")
	assert.Equal(t, ReasonRezervasyonIptal, hareketler[0].Reason)
}

func TestManualAdjust_WhatsAppSatisi(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Karanfil", Price: price(t, "100.00"), IsActive: true,
	}, "karanfil")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 12, 0)
	setDiscount(t, pool, p.ID, "80.00", 10, 0)

	got, err := store.ManualAdjust(ctx, ManualAdjustInput{
		ProductID: p.ID, Delta: -1,
		Reason: ReasonWhatsApp, WasDiscounted: true,
	})
	require.NoError(t, err)

	assert.Equal(t, 11, got.StockQuantity)
	assert.Equal(t, 1, got.DiscountSold, "WhatsApp satışı da kotayı tüketir (spec §4.4)")

	hareketler, err := store.ListMovements(ctx, p.ID, 10)
	require.NoError(t, err)
	require.Len(t, hareketler, 1)
	assert.Equal(t, -1, hareketler[0].Delta)
	assert.Equal(t, ReasonWhatsApp, hareketler[0].Reason)
	assert.True(t, hareketler[0].WasDiscounted)
}

func TestManualAdjust_StokAltinaDusemez(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Zambak", Price: price(t, "100.00"), IsActive: true,
	}, "zambak")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 2, 0)

	_, err = store.ManualAdjust(ctx, ManualAdjustInput{
		ProductID: p.ID, Delta: -5, Reason: ReasonWhatsApp,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)

	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 2, got.StockQuantity, "başarısız düzeltme stoğu bozmamalı")
}

func TestManualAdjust_YeniParti_StokArtar(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Şebboy", Price: price(t, "100.00"), IsActive: true,
	}, "sebboy")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 3, 0)

	got, err := store.ManualAdjust(ctx, ManualAdjustInput{
		ProductID: p.ID, Delta: 20, Reason: ReasonYeniParti,
	})
	require.NoError(t, err)

	assert.Equal(t, 23, got.StockQuantity)
}
```

- [ ] **Step 2: Testin başarısız olduğunu doğrula**

Run: `cd backend && go test ./internal/product/ -run 'TestReserve|TestCommit|TestRelease|TestManualAdjust' -v`
Expected: FAIL — `store.Reserve undefined`, `StockError undefined`.

- [ ] **Step 3: stock_store.go'yu yaz**

`backend/internal/product/stock_store.go`:

```go
package product

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// Reason stok hareketinin sebebi. DB'deki CHECK ile birebir aynı olmalı.
type Reason string

const (
	ReasonSiparis          Reason = "siparis"
	ReasonWhatsApp         Reason = "whatsapp_satisi"
	ReasonSayim            Reason = "sayim_duzeltme"
	ReasonYeniParti        Reason = "yeni_parti"
	ReasonIptalIade        Reason = "iptal_iade"
	ReasonRezervasyonIptal Reason = "rezervasyon_iptal"
)

func (r Reason) Valid() bool {
	switch r {
	case ReasonSiparis, ReasonWhatsApp, ReasonSayim,
		ReasonYeniParti, ReasonIptalIade, ReasonRezervasyonIptal:
		return true
	}
	return false
}

// Movement tek bir stok hareketi.
type Movement struct {
	ID            int64     `json:"id"`
	ProductID     int64     `json:"product_id"`
	Delta         int       `json:"delta"`
	Reason        Reason    `json:"reason"`
	OrderID       *int64    `json:"order_id,omitempty"`
	WasDiscounted bool      `json:"was_discounted"`
	Note          string    `json:"note"`
	AdminUserID   *int64    `json:"admin_user_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// StockError stok yetmediğinde döner. Kaç adet kaldığını taşır — müşteri
// sepetini düzeltebilsin diye mesajda bu bilgi geçiyor (spec §8).
type StockError struct {
	ProductID   int64
	ProductName string
	Available   int
}

func (e *StockError) Error() string {
	if e.Available == 0 {
		return fmt.Sprintf("%q tükendi", e.ProductName)
	}
	return fmt.Sprintf("%q için yalnızca %d adet kaldı", e.ProductName, e.Available)
}

// Unwrap sayesinde errors.Is(err, errorsx.ErrInvalidInput) çalışır ve
// handler katmanı bu hatayı 400 olarak döner (mevcut httperr deseni).
func (e *StockError) Unwrap() error { return errorsx.ErrInvalidInput }

// Reserve stoğu rezerve eder. Sipariş transaction'ına KATILIR (tx parametresi)
// — sipariş yazılamazsa rezervasyon da geri alınır.
//
// Kontrol TEK ifadede: koşul WHERE içinde olduğu için Postgres satırı kilitler
// ve eşzamanlı iki istekten yalnızca biri kazanır. "Önce oku, sonra karar ver,
// sonra yaz" yapılırsa okuma ile yazma arasında başkası araya girer ve son
// ürün iki kişiye satılır.
func (s *Store) Reserve(ctx context.Context, tx pgx.Tx, productID int64, qty int) error {
	ct, err := tx.Exec(ctx, `
		UPDATE products
		   SET stock_reserved = stock_reserved + $2
		 WHERE id = $1
		   AND (NOT track_stock OR stock_quantity - stock_reserved >= $2)`,
		productID, qty)
	if err != nil {
		return fmt.Errorf("stok rezerve: %w", err)
	}
	if ct.RowsAffected() == 1 {
		return nil
	}

	// Satır güncellenmedi: ya ürün yok ya stok yetmiyor. Hangisi olduğunu
	// ve kaç adet kaldığını öğren — müşteriye anlamlı mesaj vermek için.
	var name string
	var qtyDB, reserved int
	var track bool
	err = tx.QueryRow(ctx,
		`SELECT name, track_stock, stock_quantity, stock_reserved
		   FROM products WHERE id = $1`, productID).
		Scan(&name, &track, &qtyDB, &reserved)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsx.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("stok kontrol: %w", err)
	}

	kalan := qtyDB - reserved
	if kalan < 0 {
		kalan = 0
	}
	return &StockError{ProductID: productID, ProductName: name, Available: kalan}
}

// CommitReservation rezervasyonu kesin düşüşe çevirir. Ödemeyi paid yapan
// transaction'a KATILIR — para hareketi ile stok asla ayrışmamalı (spec §8).
//
// discounted true ise indirim kotası da tüketilir.
func (s *Store) CommitReservation(ctx context.Context, tx pgx.Tx,
	productID int64, qty int, orderID int64, discounted bool) error {
	// TEK ifade: stok ve kota bağımsız kavramlar (spec §2) ama ikisi de aynı
	// satırda. Takipsiz üründe stok alanlarına dokunulmaz, kota yine tüketilir
	// — bu yüzden koşullar SET içinde, WHERE'de değil.
	//
	// GREATEST(...,0): elle düzeltme sonrası rezerve beklenenden az olabilir;
	// negatife düşüp CHECK constraint'i patlatmasın.
	_, err := tx.Exec(ctx, `
		UPDATE products
		   SET stock_reserved = CASE WHEN track_stock
		                             THEN GREATEST(stock_reserved - $2, 0)
		                             ELSE stock_reserved END,
		       stock_quantity = CASE WHEN track_stock
		                             THEN GREATEST(stock_quantity - $2, 0)
		                             ELSE stock_quantity END,
		       discount_sold  = discount_sold + CASE WHEN $3 THEN $2 ELSE 0 END,
		       updated_at     = now()
		 WHERE id = $1`,
		productID, qty, discounted)
	if err != nil {
		return fmt.Errorf("stok kesinleştir: %w", err)
	}

	var oid *int64
	if orderID > 0 {
		oid = &orderID
	}
	return s.addMovementTx(ctx, tx, Movement{
		ProductID: productID, Delta: -qty, Reason: ReasonSiparis,
		OrderID: oid, WasDiscounted: discounted,
	})
}

// Release rezervasyonu serbest bırakır (ödeme gelmedi). Fiziksel stok
// değişmez — ürün hiç satılmadı.
func (s *Store) Release(ctx context.Context, productID int64, qty int, orderID *int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE products
		   SET stock_reserved = GREATEST(stock_reserved - $2, 0), updated_at = now()
		 WHERE id = $1 AND track_stock`, productID, qty); err != nil {
		return fmt.Errorf("rezervasyon serbest: %w", err)
	}

	// delta 0: fiziksel stok değişmedi, ama izi kalmalı — sessizce kaybolan
	// bir şey olmasın (spec §4.3).
	if err := s.addMovementTx(ctx, tx, Movement{
		ProductID: productID, Delta: 0, Reason: ReasonRezervasyonIptal,
		OrderID: orderID, Note: fmt.Sprintf("%d adet rezervasyon serbest bırakıldı", qty),
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RestoreStock iade sonrası ürünü rafa geri koyar.
func (s *Store) RestoreStock(ctx context.Context, productID int64, qty int, orderID *int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE products
		   SET stock_quantity = stock_quantity + $2, updated_at = now()
		 WHERE id = $1 AND track_stock`, productID, qty); err != nil {
		return fmt.Errorf("stok iade: %w", err)
	}
	if err := s.addMovementTx(ctx, tx, Movement{
		ProductID: productID, Delta: qty, Reason: ReasonIptalIade, OrderID: orderID,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ManualAdjustInput panelden yapılan elle stok düzeltmesi.
type ManualAdjustInput struct {
	ProductID     int64
	Delta         int
	Reason        Reason
	WasDiscounted bool
	Note          string
	AdminUserID   *int64
}

// ManualAdjust elle stok düzeltmesi yapar (WhatsApp satışı, sayım, yeni parti).
// Stoğun altına düşen düzeltme reddedilir.
func (s *Store) ManualAdjust(ctx context.Context, in ManualAdjustInput) (*Product, error) {
	if in.Delta == 0 {
		return nil, fmt.Errorf("%w: değişim miktarı sıfır olamaz", errorsx.ErrInvalidInput)
	}
	if !in.Reason.Valid() {
		return nil, fmt.Errorf("%w: geçersiz sebep", errorsx.ErrInvalidInput)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	// Stok yetmiyorsa satır güncellenmez — Reserve ile aynı atomik desen.
	ct, err := tx.Exec(ctx, `
		UPDATE products
		   SET stock_quantity = stock_quantity + $2,
		       discount_sold  = discount_sold +
		           CASE WHEN $3 AND $2 < 0 THEN -$2 ELSE 0 END,
		       updated_at     = now()
		 WHERE id = $1 AND stock_quantity + $2 >= 0`,
		in.ProductID, in.Delta, in.WasDiscounted)
	if err != nil {
		return nil, fmt.Errorf("stok düzelt: %w", err)
	}
	if ct.RowsAffected() == 0 {
		var name string
		var qty int
		err = tx.QueryRow(ctx, `SELECT name, stock_quantity FROM products WHERE id = $1`,
			in.ProductID).Scan(&name, &qty)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsx.ErrNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("stok kontrol: %w", err)
		}
		return nil, fmt.Errorf("%w: %q stoğu %d, %d adet düşülemez",
			errorsx.ErrInvalidInput, name, qty, -in.Delta)
	}

	if err := s.addMovementTx(ctx, tx, Movement{
		ProductID: in.ProductID, Delta: in.Delta, Reason: in.Reason,
		WasDiscounted: in.WasDiscounted, Note: in.Note, AdminUserID: in.AdminUserID,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx commit: %w", err)
	}
	return s.GetByID(ctx, in.ProductID)
}

func (s *Store) addMovementTx(ctx context.Context, tx pgx.Tx, m Movement) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO stock_movements
		  (product_id, delta, reason, order_id, was_discounted, note, admin_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		m.ProductID, m.Delta, m.Reason, m.OrderID, m.WasDiscounted, m.Note, m.AdminUserID)
	if err != nil {
		return fmt.Errorf("hareket kaydı: %w", err)
	}
	return nil
}

// ListMovements ürünün stok hareketlerini yeniden eskiye döner.
func (s *Store) ListMovements(ctx context.Context, productID int64, limit int) ([]Movement, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, product_id, delta, reason, order_id, was_discounted,
		       note, admin_user_id, created_at
		  FROM stock_movements
		 WHERE product_id = $1
		 ORDER BY created_at DESC, id DESC
		 LIMIT $2`, productID, limit)
	if err != nil {
		return nil, fmt.Errorf("hareket listele: %w", err)
	}
	defer rows.Close()

	out := make([]Movement, 0)
	for rows.Next() {
		var m Movement
		if err := rows.Scan(&m.ID, &m.ProductID, &m.Delta, &m.Reason, &m.OrderID,
			&m.WasDiscounted, &m.Note, &m.AdminUserID, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("hareket scan: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
```

- [ ] **Step 4: Testlerin geçtiğini doğrula**

Run: `cd backend && go test ./internal/product/ -run 'TestReserve|TestCommit|TestRelease|TestManualAdjust' -v`
Expected: PASS. Özellikle `TestReserve_YarisDurumu_TekKazanan` — 20 goroutine'den tam olarak 1'i başarılı.

- [ ] **Step 5: Yarış testini defalarca koştur**

Yarış hataları tek koşuda saklanabilir:

```bash
cd backend && go test ./internal/product/ -run TestReserve_YarisDurumu -count=10 -v
```

Expected: 10/10 PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/product/stock_store.go backend/internal/product/stock_store_test.go
git commit -m "feat(stok): atomik rezervasyon, kesinleşme ve hareket kaydı"
```

---

## Task 5: Sipariş akışı — rezervasyon, kesinleşme, iade

**Files:**
- Modify: `backend/internal/order/service.go` (Create, ApplyCallback, Refund)
- Modify: `backend/internal/order/store.go` (SetPaidTx ekle, Create'e tx erişimi)
- Test: `backend/internal/order/service_test.go`

**Interfaces:**
- Consumes: Task 4'ün `Reserve`, `CommitReservation`, `RestoreStock`; Task 2'nin `EffectivePrice()`, `DiscountActive()`
- Produces:
  - `order.StockManager` arayüzü (dar arayüz — mevcut `ProductReader` deseni):
    ```go
    type StockManager interface {
        Reserve(ctx context.Context, tx pgx.Tx, productID int64, qty int) error
        CommitReservation(ctx context.Context, tx pgx.Tx, productID int64, qty int, orderID int64, discounted bool) error
        RestoreStock(ctx context.Context, productID int64, qty int, orderID *int64) error
    }
    ```
  - `func (s *Store) SetPaidTx(ctx context.Context, tx pgx.Tx, id int64) error`
  - `func (s *Store) BeginTx(ctx context.Context) (pgx.Tx, error)`
  - `func (s *Store) ItemsForStock(ctx context.Context, orderID int64) ([]StockLine, error)` — `StockLine{ProductID int64; Quantity int; WasDiscounted bool}`
  - `NewService` imzasına `stock StockManager` parametresi eklenir

- [ ] **Step 1: Failing testi yaz**

`backend/internal/order/service_test.go` sonuna ekle. Mevcut dosyadaki sahte (fake) ProductReader desenini kullan — dosyadaki mevcut yardımcıları incele ve aynı stille yaz:

```go
// fakeStock StockManager'ı taklit eder — çağrıları kaydeder.
type fakeStock struct {
	rezerveEdilen map[int64]int
	kesinlesen    map[int64]int
	iadeEdilen    map[int64]int
	indirimli     map[int64]bool
	reserveErr    error
}

func newFakeStock() *fakeStock {
	return &fakeStock{
		rezerveEdilen: map[int64]int{},
		kesinlesen:    map[int64]int{},
		iadeEdilen:    map[int64]int{},
		indirimli:     map[int64]bool{},
	}
}

func (f *fakeStock) Reserve(ctx context.Context, tx pgx.Tx, productID int64, qty int) error {
	if f.reserveErr != nil {
		return f.reserveErr
	}
	f.rezerveEdilen[productID] += qty
	return nil
}

func (f *fakeStock) CommitReservation(ctx context.Context, tx pgx.Tx,
	productID int64, qty int, orderID int64, discounted bool) error {
	f.kesinlesen[productID] += qty
	f.indirimli[productID] = discounted
	return nil
}

func (f *fakeStock) RestoreStock(ctx context.Context, productID int64, qty int, orderID *int64) error {
	f.iadeEdilen[productID] += qty
	return nil
}

func TestCreate_StokRezerveEdilir(t *testing.T) {
	// Mevcut testlerdeki kurulum desenini izle: servis + sahte ürün okuyucu.
	// Ürün: id=1, stok takipli, fiyat 100.00
	// Sipariş: 2 adet
	// Beklenen: fakeStock.rezerveEdilen[1] == 2
}

func TestCreate_StokYetmezse_SiparisOlusmaz(t *testing.T) {
	// fakeStock.reserveErr = &product.StockError{ProductName: "Gül", Available: 1}
	// Beklenen: Create hata döner, errorsx.ErrInvalidInput ile eşleşir,
	// mesajda "1 adet kaldı" geçer, DB'ye sipariş YAZILMAZ.
}

func TestApplyCallback_OdemeOnayinda_StokKesinlesir(t *testing.T) {
	// Başarılı callback sonrası fakeStock.kesinlesen[1] == 2
}

func TestApplyCallback_AyniCallbackIkiKez_StokBirKezDuser(t *testing.T) {
	// Aynı callback iki kez işlenir.
	// Beklenen: fakeStock.kesinlesen[1] == 2 (4 DEĞİL) — idempotency.
}

func TestCreate_IndirimliFiyatSiparise_Yazilir(t *testing.T) {
	// Ürün: price=1850.00, discount_price=1450.00, quota=10, sold=3
	// Beklenen: order_items.price_at_order == 1450.00,
	//           items_total indirimli fiyattan hesaplanmış.
}

func TestRefund_StokIadeEdilir(t *testing.T) {
	// Ödenmiş sipariş iade edilir.
	// Beklenen: fakeStock.iadeEdilen[1] == 2
}
```

**NOT:** Yukarıdaki gövdeler yorum olarak bırakıldı çünkü mevcut `service_test.go` içindeki kurulum yardımcılarının (sahte PaymentStarter, ProductReader, test DB kurulumu) tam imzaları dosyada duruyor. **Uygulayan kişi önce `backend/internal/order/service_test.go` dosyasını okumalı** ve yeni testleri oradaki mevcut desenle birebir aynı şekilde yazmalı. Testin ne doğrulaması gerektiği yorumlarda tam olarak belirtilmiştir.

- [ ] **Step 2: Testin başarısız olduğunu doğrula**

Run: `cd backend && go test ./internal/order/ -run 'TestCreate_Stok|TestApplyCallback_.*Stok|TestRefund_Stok|TestCreate_Indirimli' -v`
Expected: FAIL — derleme hatası (`StockManager` yok) veya assertion hatası.

- [ ] **Step 3: Store'a tx-aware yardımcıları ekle**

`backend/internal/order/store.go`:

```go
// BeginTx sipariş ve stok işlemlerinin aynı transaction'da yürümesi için.
func (s *Store) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.pool.Begin(ctx)
}

// SetPaidTx SetPaid'in transaction'a katılan varyantı. Stok kesinleşmesi
// ile ödeme aynı transaction'da olmalı — para hareketi ile stok ayrışamaz
// (spec §8).
func (s *Store) SetPaidTx(ctx context.Context, tx pgx.Tx, id int64) error {
	ct, err := tx.Exec(ctx, `
		UPDATE orders SET status = 'paid', paid_at = now(), updated_at = now()
		WHERE id = $1 AND status = 'awaiting_payment'`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

// StockLine stok işlemleri için sipariş kalemi özeti.
type StockLine struct {
	ProductID     int64
	Quantity      int
	WasDiscounted bool
}

// ItemsForStock siparişin stok etkileyen kalemlerini döner. Ürünü silinmiş
// kalem (product_id NULL) atlanır — düşülecek stok yok.
//
// was_discounted: sipariş anında indirimli fiyattan mı satıldı. price_at_order
// ile ürünün normal fiyatı karşılaştırılamaz (fiyat sonradan değişmiş
// olabilir), bu yüzden sipariş anındaki indirim durumu order_items'a
// yazılıyor (Task 5 Step 4).
func (s *Store) ItemsForStock(ctx context.Context, orderID int64) ([]StockLine, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT product_id, quantity, was_discounted
		  FROM order_items
		 WHERE order_id = $1 AND product_id IS NOT NULL`, orderID)
	if err != nil {
		return nil, fmt.Errorf("stok kalemleri: %w", err)
	}
	defer rows.Close()

	out := make([]StockLine, 0)
	for rows.Next() {
		var l StockLine
		if err := rows.Scan(&l.ProductID, &l.Quantity, &l.WasDiscounted); err != nil {
			return nil, fmt.Errorf("stok kalemi scan: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
```

- [ ] **Step 4: order_items'a was_discounted kolonu ekle**

Task 1'in migration dosyasına (henüz deploy edilmediyse) veya `000011` içine ekle:

```sql
-- Sipariş anında indirimli satıldı mı. Sonradan ürünün fiyatı değişebileceği
-- için price_at_order karşılaştırmasıyla anlaşılamaz; o anki durum kopyalanır
-- (product_name / price_at_order deseninin aynısı).
ALTER TABLE order_items
    ADD COLUMN was_discounted BOOLEAN NOT NULL DEFAULT false;
```

Down dosyasına da ekle:

```sql
ALTER TABLE order_items DROP COLUMN IF EXISTS was_discounted;
```

Migration'ı yeniden uygula:

```bash
migrate -path backend/migrations -database "$TEST_DATABASE_URL" down 1
migrate -path backend/migrations -database "$TEST_DATABASE_URL" up
```

`NewOrderItem` struct'ına (`backend/internal/order/model.go`) alan ekle:

```go
	// WasDiscounted sipariş anında indirimli fiyattan satıldı mı — kota
	// muhasebesi için gerekli.
	WasDiscounted bool
```

`store.go` içindeki `order_items` INSERT ifadesine bu kolonu ekle.

- [ ] **Step 5: Service'e stok akışını bağla**

`backend/internal/order/service.go`:

`StockManager` arayüzünü ve `Service.stock` alanını ekle; `NewService` imzasına parametre ekle. `Create` içinde, ürün doğrulama döngüsünde fiyatı `EffectivePrice()` ile oku:

```go
		// İndirim aktifse indirimli fiyat kullanılır; sipariş anındaki fiyat
		// bağlayıcıdır (spec §4.4).
		birimFiyat := p.EffectivePrice()
		indirimli := p.DiscountActive()

		itemsTotal = itemsTotal.Add(birimFiyat.Mul(decimal.NewFromInt(int64(ci.Quantity))))
		items = append(items, NewOrderItem{
			ProductID:     p.ID,
			ProductName:   p.Name,
			PriceAtOrder:  birimFiyat,
			Quantity:      ci.Quantity,
			Options:       opts,
			WasDiscounted: indirimli,
		})
		basket = append(basket, payment.BasketItem{
			Name: p.Name, PriceKurus: payment.KurusFromDecimal(birimFiyat),
			Quantity: ci.Quantity,
		})
```

Sipariş yazımını ve rezervasyonu tek transaction'a al. `s.store.Create` çağrısından ÖNCE rezervasyon yapılır ve **ürünler `ProductID` sırasına dizilir** (deadlock önlemi):

```go
	// NOT: bu blok "cmp" paketini kullanıyor — dosyanın import listesine
	// eklenmeli ("slices" zaten var).
	//
	// Deadlock önlemi: iki müşteri aynı iki ürünü ters sırayla sepete
	// koyarsa kilitler çaprazlanır. Sabit sıra (product_id artan) bunu
	// imkânsız kılar (spec §4.1).
	rezerveSirasi := make([]NewOrderItem, len(items))
	copy(rezerveSirasi, items)
	slices.SortFunc(rezerveSirasi, func(a, b NewOrderItem) int {
		return cmp.Compare(a.ProductID, b.ProductID)
	})

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, it := range rezerveSirasi {
		if err := s.stock.Reserve(ctx, tx, it.ProductID, it.Quantity); err != nil {
			// StockError müşteriye "kaç adet kaldı" bilgisini taşır.
			return nil, "", err
		}
	}
```

`s.store.Create` bu transaction'ı kullanacak şekilde `createOnceTx(ctx, tx, in)` varyantına yönlendirilir ve sonunda `tx.Commit(ctx)` çağrılır. Mevcut `createOnce` gövdesi tx parametresi alacak şekilde ayrıştırılır (order_no retry mantığı korunur).

`ApplyCallback` içinde, `SetPaid` çağrısını transaction'lı akışla değiştir:

```go
	// Stok kesinleşmesi ödeme ile AYNI transaction'da: biri yazılıp diğeri
	// yazılmazsa para hareketi ile stok ayrışır (spec §8).
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	if err := s.store.SetPaidTx(ctx, tx, o.ID); err != nil {
		return false, err
	}

	lines, err := s.store.ItemsForStock(ctx, o.ID)
	if err != nil {
		return false, err
	}
	for _, l := range lines {
		if err := s.stock.CommitReservation(ctx, tx, l.ProductID, l.Quantity,
			o.ID, l.WasDiscounted); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
```

`Refund` içinde, `SetRefunded` başarılı olduktan sonra stok iadesi:

```go
	// Stok iadesi başarısız olsa bile iade GEÇERLİdir — para geri gitti,
	// geri alınamaz. Stoğu esnaf panelden düzeltebilir (spec §8).
	lines, lerr := s.store.ItemsForStock(ctx, id)
	if lerr != nil {
		log.Printf("KRİTİK: iade sonrası stok kalemleri okunamadı (order=%d): %v", id, lerr)
	} else {
		for _, l := range lines {
			if err := s.stock.RestoreStock(ctx, l.ProductID, l.Quantity, &id); err != nil {
				log.Printf("KRİTİK: iade sonrası stok geri eklenemedi (order=%d, product=%d): %v",
					id, l.ProductID, err)
			}
		}
	}
```

`backend/cmd/server/main.go` içinde `order.NewService(...)` çağrısına `productStore` (StockManager'ı karşılıyor) parametresini ekle.

- [ ] **Step 6: Testlerin geçtiğini doğrula**

Run: `cd backend && go test ./internal/order/ -v`
Expected: PASS — yeni testler + mevcut sipariş testleri (regresyon yok).

- [ ] **Step 7: Tüm backend testlerini koştur**

Run: `make test`
Expected: tüm paketler PASS.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/order/ backend/cmd/server/main.go backend/migrations/
git commit -m "feat(stok): sipariş akışına rezervasyon, kesinleşme ve iade bağlandı"
```

---

## Task 6: Süpürücü — süresi geçen rezervasyonlar

**Files:**
- Create: `backend/internal/product/sweeper.go`
- Create: `backend/internal/product/sweeper_test.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes: Task 4'ün `Release`
- Produces:
  - `func NewSweeper(store *Store, ttl, interval time.Duration) *Sweeper`
  - `func (s *Sweeper) Run(ctx context.Context)` — ctx iptal olunca durur
  - `func (s *Sweeper) SweepOnce(ctx context.Context) (int, error)` — serbest bırakılan sipariş sayısı

- [ ] **Step 1: Failing testi yaz**

`backend/internal/product/sweeper_test.go`:

```go
package product

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insertAwaitingOrder test için ödeme bekleyen sipariş oluşturur.
// order paketine bağımlılık yaratmamak için doğrudan SQL (insertCategory
// deseninin aynısı).
func insertAwaitingOrder(t *testing.T, pool *pgxpool.Pool, productID int64,
	qty int, yas time.Duration) int64 {
	t.Helper()
	ctx := context.Background()
	var orderID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO orders (order_no, status, buyer_name, buyer_phone,
		   recipient_name, recipient_phone, delivery_address, delivery_district,
		   delivery_date, delivery_slot, items_total, delivery_fee, total, created_at)
		VALUES ($1, 'awaiting_payment', 'Test Alıcı', '5551112233',
		   'Test Gönderilen', '5554445566', 'Adres', 'Konak',
		   CURRENT_DATE, '09:00-12:00', 100, 0, 100, now() - $2::interval)
		RETURNING id`,
		"TEST-"+time.Now().Format("150405.000000"),
		yas.String()).Scan(&orderID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO order_items (order_id, product_id, product_name,
		                         price_at_order, quantity)
		VALUES ($1, $2, 'Test Ürün', 100, $3)`, orderID, productID, qty)
	require.NoError(t, err)
	return orderID
}

func TestSweeper_SuresiGecenRezervasyonSerbestKalir(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Gül", Price: price(t, "100.00"), IsActive: true,
	}, "gul-sweep")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 5, 2)
	insertAwaitingOrder(t, pool, p.ID, 2, 30*time.Minute)

	sweeper := NewSweeper(store, 20*time.Minute, time.Minute)
	sayi, err := sweeper.SweepOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, sayi)
	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 0, got.StockReserved, "rezervasyon serbest kalmalı")
	assert.Equal(t, 5, got.StockQuantity, "fiziksel stok değişmemeli")
}

func TestSweeper_SuresiGecmeyeniDokunmaz(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Orkide", Price: price(t, "100.00"), IsActive: true,
	}, "orkide-sweep")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 5, 2)
	// 19 dakikalık sipariş — henüz süresi geçmedi
	insertAwaitingOrder(t, pool, p.ID, 2, 19*time.Minute)

	sweeper := NewSweeper(store, 20*time.Minute, time.Minute)
	sayi, err := sweeper.SweepOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, 0, sayi)
	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 2, got.StockReserved, "süresi geçmemiş rezervasyon durmalı")
}

func TestSweeper_AyniSiparisIkiKezSupurulmez(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Lale", Price: price(t, "100.00"), IsActive: true,
	}, "lale-sweep")
	require.NoError(t, err)
	setStock(t, pool, p.ID, true, 5, 2)
	insertAwaitingOrder(t, pool, p.ID, 2, 30*time.Minute)

	sweeper := NewSweeper(store, 20*time.Minute, time.Minute)
	_, err = sweeper.SweepOnce(ctx)
	require.NoError(t, err)

	sayi, err := sweeper.SweepOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, 0, sayi, "ikinci süpürme aynı siparişi tekrar işlememeli")
	got, _ := store.GetByID(ctx, p.ID)
	assert.Equal(t, 0, got.StockReserved, "rezerve negatife düşmemeli")
}
```

**NOT:** `setStock`, `setDiscount` ve `price` yardımcıları Task 3/Task 2'de aynı pakette (`product`) tanımlandı — yeniden yazma, doğrudan kullan.

- [ ] **Step 2: Testin başarısız olduğunu doğrula**

Run: `cd backend && go test ./internal/product/ -run TestSweeper -v`
Expected: FAIL — `NewSweeper undefined`.

- [ ] **Step 3: Süpürücüyü yaz**

`backend/internal/product/sweeper.go`:

```go
package product

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Sweeper ödemesi tamamlanmayan siparişlerin rezervasyonlarını serbest
// bırakır. Yarım kalan ödeme stoğu sonsuza kadar tutmamalı (spec §4.3).
type Sweeper struct {
	store    *Store
	ttl      time.Duration
	interval time.Duration
}

func NewSweeper(store *Store, ttl, interval time.Duration) *Sweeper {
	return &Sweeper{store: store, ttl: ttl, interval: interval}
}

// Run ctx iptal edilene kadar periyodik olarak süpürür.
func (s *Sweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.SweepOnce(ctx); err != nil {
				log.Printf("stok süpürücü hatası: %v", err)
			}
		}
	}
}

// SweepOnce tek tur süpürür, serbest bırakılan sipariş sayısını döner.
//
// stock_swept işareti: aynı sipariş iki kez süpürülürse rezerve negatife
// düşer. İşaret orders tablosunda tutuluyor.
func (s *Sweeper) SweepOnce(ctx context.Context) (int, error) {
	rows, err := s.store.pool.Query(ctx, `
		SELECT o.id, oi.product_id, oi.quantity
		  FROM orders o
		  JOIN order_items oi ON oi.order_id = o.id
		 WHERE o.status = 'awaiting_payment'
		   AND NOT o.stock_swept
		   AND o.created_at < now() - $1::interval
		   AND oi.product_id IS NOT NULL`,
		s.ttl.String())
	if err != nil {
		return 0, fmt.Errorf("süresi geçen siparişler: %w", err)
	}

	type satir struct {
		orderID   int64
		productID int64
		qty       int
	}
	var satirlar []satir
	for rows.Next() {
		var r satir
		if err := rows.Scan(&r.orderID, &r.productID, &r.qty); err != nil {
			rows.Close()
			return 0, fmt.Errorf("süpürme scan: %w", err)
		}
		satirlar = append(satirlar, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	siparisler := map[int64]bool{}
	for _, r := range satirlar {
		// Tek satırdaki hata tüm süpürmeyi durdurmamalı — logla, devam et.
		// Bir sonraki tur tekrar dener (spec §8).
		if err := s.store.Release(ctx, r.productID, r.qty, &r.orderID); err != nil {
			log.Printf("rezervasyon serbest bırakılamadı (order=%d, product=%d): %v",
				r.orderID, r.productID, err)
			continue
		}
		siparisler[r.orderID] = true
	}

	for orderID := range siparisler {
		if _, err := s.store.pool.Exec(ctx,
			`UPDATE orders SET stock_swept = true WHERE id = $1`, orderID); err != nil {
			log.Printf("süpürme işareti yazılamadı (order=%d): %v", orderID, err)
		}
	}

	return len(siparisler), nil
}
```

- [ ] **Step 4: stock_swept kolonunu migration'a ekle**

`backend/migrations/000011_stock_discount.up.sql` sonuna:

```sql
-- Süpürülmüş sipariş işareti: aynı sipariş iki kez süpürülürse rezerve
-- negatife düşer (spec §4.3).
ALTER TABLE orders
    ADD COLUMN stock_swept BOOLEAN NOT NULL DEFAULT false;

-- Süpürücü bu koşulla tarıyor
CREATE INDEX idx_orders_sweep ON orders (created_at)
    WHERE status = 'awaiting_payment' AND NOT stock_swept;
```

Down dosyasına:

```sql
DROP INDEX IF EXISTS idx_orders_sweep;
ALTER TABLE orders DROP COLUMN IF EXISTS stock_swept;
```

Migration'ı yenile:

```bash
migrate -path backend/migrations -database "$TEST_DATABASE_URL" down 1
migrate -path backend/migrations -database "$TEST_DATABASE_URL" up
```

- [ ] **Step 5: Testlerin geçtiğini doğrula**

Run: `cd backend && go test ./internal/product/ -run TestSweeper -v`
Expected: PASS (3 test).

- [ ] **Step 6: Süpürücüyü sunucuda başlat**

`backend/cmd/server/main.go` içinde, sunucu dinlemeye başlamadan önce:

```go
	// Ödemesi yarım kalan siparişlerin stok rezervasyonlarını serbest bırakır.
	sweeper := product.NewSweeper(productStore, 20*time.Minute, 5*time.Minute)
	go sweeper.Run(ctx)
```

`ctx` sunucunun kapanışta iptal ettiği context olmalı (mevcut graceful shutdown context'i; dosyada nasıl kurulduğuna bak).

- [ ] **Step 7: Tüm testleri koştur**

Run: `make test`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/product/sweeper.go backend/internal/product/sweeper_test.go \
        backend/cmd/server/main.go backend/migrations/
git commit -m "feat(stok): süresi geçen rezervasyonları serbest bırakan süpürücü"
```

---

## Task 7: API — public ve panel uçları

**Files:**
- Modify: `backend/internal/api/app/product_view.go`
- Modify: `backend/internal/api/app/product_handler.go`
- Modify: `backend/internal/api/idare/product_view.go`
- Modify: `backend/internal/api/idare/product_handler.go`
- Modify: `backend/internal/api/idare/router.go`
- Modify: `backend/internal/product/service.go`
- Test: `backend/internal/api/app/product_handler_test.go`, `backend/internal/api/idare/product_handler_test.go`

**Interfaces:**
- Consumes: Task 2 (`OldPrice`, `InStock`, `DiscountRemaining`, `EffectivePrice`), Task 4 (`ManualAdjust`, `ListMovements`)
- Produces:
  - Public `ProductView` alanları: `price` (geçerli fiyat), `old_price *string`, `in_stock bool`, `stock_quantity *int`, `discount_remaining *int`
  - Panel `ProductView` ek alanları: `track_stock`, `stock_quantity`, `stock_reserved`, `discount_price`, `discount_quota`, `discount_sold`
  - `GET /api/products?discounted=true`
  - `POST /api/idare/products/:id/stock` — gövde: `{delta, reason, was_discounted, note}`
  - `GET /api/idare/products/:id/movements`
  - `product.Service.AdjustStock(ctx, in ManualAdjustInput) (*Product, error)`
  - `product.Service.Movements(ctx, productID int64, limit int) ([]Movement, error)`

- [ ] **Step 1: Failing testi yaz**

`backend/internal/api/app/product_handler_test.go` sonuna (mevcut test kurulum desenini izle):

```go
func TestListProducts_IndirimliUrun_EskiVeYeniFiyat(t *testing.T) {
	// Ürün: price=1850.00, discount_price=1450.00, quota=10, sold=3
	// GET /api/products
	// Beklenen JSON: price == "1450.00", old_price == "1850.00",
	//                discount_remaining == 7, in_stock == true
}

func TestListProducts_KotasiDolmus_EskiFiyatGosterilmez(t *testing.T) {
	// Ürün: discount sold == quota
	// Beklenen: price == "1850.00", old_price == null, discount_remaining == null
}

func TestListProducts_TukenenUrun_InStockFalse(t *testing.T) {
	// Ürün: track_stock=true, quantity=2, reserved=2
	// Beklenen: in_stock == false, ürün listede GÖRÜNÜR (gizlenmez)
}

func TestListProducts_TakipsizUrun_InStockTrue(t *testing.T) {
	// Ürün: track_stock=false, quantity=0
	// Beklenen: in_stock == true, stock_quantity == null
}

func TestListProducts_DiscountedFiltresi(t *testing.T) {
	// GET /api/products?discounted=true
	// Beklenen: yalnızca indirimi aktif ürünler
}
```

`backend/internal/api/idare/product_handler_test.go` sonuna:

```go
func TestAdjustStock_WhatsAppSatisi(t *testing.T) {
	// POST /api/idare/products/:id/stock
	// Gövde: {"delta": -1, "reason": "whatsapp_satisi", "was_discounted": true}
	// Beklenen: 200, stock_quantity 1 azalmış, discount_sold 1 artmış
}

func TestAdjustStock_GecersizSebep_400(t *testing.T) {
	// Gövde: {"delta": -1, "reason": "hatali_sebep"}
	// Beklenen: 400
}

func TestAdjustStock_StokAltinda_400(t *testing.T) {
	// stok=2, delta=-5
	// Beklenen: 400, mesajda mevcut stok geçer
}

func TestListMovements_HareketlerDoner(t *testing.T) {
	// İki hareket eklenir, GET /api/idare/products/:id/movements
	// Beklenen: 2 kayıt, yeniden eskiye sıralı
}
```

**NOT:** Gövdeler yorum olarak verildi. Uygulayan kişi **önce mevcut handler test dosyalarını okumalı** (auth token kurulumu, router kurulumu, JSON assert deseni oradadır) ve yeni testleri o desenle yazmalı.

- [ ] **Step 2: Testin başarısız olduğunu doğrula**

Run: `cd backend && go test ./internal/api/... -v`
Expected: FAIL.

- [ ] **Step 3: Public view'ı güncelle**

`backend/internal/api/app/product_view.go`:

```go
type ProductView struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	// Price GEÇERLİ fiyat — indirim aktifse indirimli fiyat. Sepet ve
	// toplam hesapları bu alanı kullanır, değişiklik gerektirmez.
	Price string `json:"price"`
	// OldPrice indirim aktifse üstü çizili gösterilecek normal fiyat.
	OldPrice          *string                 `json:"old_price"`
	InStock           bool                    `json:"in_stock"`
	StockQuantity     *int                    `json:"stock_quantity"`
	DiscountRemaining *int                    `json:"discount_remaining"`
	CategoryIDs       []int64                 `json:"category_ids"`
	Images            []ImageView             `json:"images"`
	OptionGroups      []PublicOptionGroupView `json:"option_groups"`
}
```

`toProductView` fonksiyonunu güncelle:

```go
func toProductView(p product.Product, imgSvc *image.Service, imgs []image.ProductImage) ProductView {
	v := ProductView{
		ID:                p.ID,
		Name:              p.Name,
		Slug:              p.Slug,
		Description:       p.Description,
		Price:             p.EffectivePrice().StringFixed(2),
		InStock:           p.InStock(),
		DiscountRemaining: p.DiscountRemaining(),
		CategoryIDs:       p.CategoryIDs,
		Images:            toImageViews(imgSvc, imgs),
	}
	if old := p.OldPrice(); old != nil {
		s := old.StringFixed(2)
		v.OldPrice = &s
	}
	// Takipsiz üründe adet anlamsız — null gider.
	if adet, sinirli := p.Available(); sinirli {
		v.StockQuantity = &adet
	}
	return v
}
```

- [ ] **Step 4: Public handler'a discounted filtresini ekle**

`backend/internal/api/app/product_handler.go` içinde, mevcut query parametresi okuma deseniyle:

```go
	// ?discounted=true → indirimi aktif ürünler (/indirimli sayfası)
	if r.URL.Query().Get("discounted") == "true" {
		f.DiscountedOnly = true
	}
```

- [ ] **Step 5: Panel view ve uçlarını ekle**

`backend/internal/api/idare/product_view.go` — panel `ProductView`'una ekle:

```go
	TrackStock    bool    `json:"track_stock"`
	StockQuantity int     `json:"stock_quantity"`
	StockReserved int     `json:"stock_reserved"`
	DiscountPrice *string `json:"discount_price"`
	DiscountQuota *int    `json:"discount_quota"`
	DiscountSold  int     `json:"discount_sold"`
```

Dönüştürücüde doldur (`DiscountPrice` için `StringFixed(2)`).

`backend/internal/product/service.go`'ye ekle:

```go
// AdjustStock elle stok düzeltmesi (panel).
func (s *Service) AdjustStock(ctx context.Context, in ManualAdjustInput) (*Product, error) {
	return s.store.ManualAdjust(ctx, in)
}

// Movements ürünün stok hareket geçmişi.
func (s *Service) Movements(ctx context.Context, productID int64, limit int) ([]Movement, error) {
	return s.store.ListMovements(ctx, productID, limit)
}
```

`backend/internal/api/idare/product_handler.go`'ye iki handler ekle (mevcut handler desenini izle — `httperr` ile hata dönüşü, `chi.URLParam` ile id okuma):

```go
type adjustStockRequest struct {
	Delta         int    `json:"delta"`
	Reason        string `json:"reason"`
	WasDiscounted bool   `json:"was_discounted"`
	Note          string `json:"note"`
}
```

Handler `product.ManualAdjustInput` kurup `AdjustStock` çağırır, güncel ürünü panel view'ı olarak döner. Oturumdaki admin kullanıcı id'si varsa `AdminUserID` alanına konur (mevcut auth middleware'in context'e ne koyduğuna bak).

`backend/internal/api/idare/router.go`'ye rotaları ekle:

```go
		r.Post("/products/{id}/stock", h.AdjustStock)
		r.Get("/products/{id}/movements", h.ListMovements)
```

- [ ] **Step 6: Testlerin geçtiğini doğrula**

Run: `make test`
Expected: tüm paketler PASS.

- [ ] **Step 7: Uçları elle doğrula**

```bash
make run
```

Başka bir terminalde:

```bash
curl -s localhost:8080/api/products | head -c 500
curl -s 'localhost:8080/api/products?discounted=true' | head -c 300
```

Beklenen: `in_stock`, `old_price`, `discount_remaining` alanları görünür; indirimsiz üründe `old_price` null.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/api/ backend/internal/product/service.go
git commit -m "feat(stok): public ve panel API uçları — stok, indirim, hareketler"
```

---

## Task 8: Panel arayüzü (idare)

**Files:**
- Modify: `frontend/idare/src/pages/urunler/index.vue`
- Modify: `frontend/idare/src/pages/urunler/[id].vue`
- Create: `frontend/idare/src/components/StokDusurDialog.vue`

**Interfaces:**
- Consumes: Task 7 uçları (`POST /api/idare/products/:id/stock`, `GET .../movements`), panel `ProductView` alanları
- Produces: panel UI — başka task buna bağlı değil

- [ ] **Step 1: StokDusurDialog bileşenini yaz**

`frontend/idare/src/components/StokDusurDialog.vue` — props: `product` (panel ürün nesnesi), `modelValue` (açık mı). Emit: `saved`.

İçerik:
- Adet girişi (varsayılan 1)
- Sebep seçimi: `whatsapp_satisi` (varsayılan, etiket "WhatsApp satışı"), `sayim_duzeltme` ("Sayım düzeltme"), `yeni_parti` ("Yeni parti")
- `was_discounted` onay kutusu — **yalnızca ürünün indirimi aktifse görünür** (`discount_price != null && discount_sold < discount_quota`), aktifse varsayılan işaretli
- Not alanı (isteğe bağlı)
- Kaydet → `POST /api/idare/products/:id/stock` gövde `{delta, reason, was_discounted, note}`. Düşürmede `delta` negatif, "Yeni parti"de pozitif.

Mevcut panel bileşenlerindeki (`ProductOptionPicker.vue`) Vuetify dialog desenini ve API çağrısı stilini izle.

- [ ] **Step 2: Ürün listesine stok sütunu ekle**

`frontend/idare/src/pages/urunler/index.vue` — tabloya "Stok" sütunu:

- `track_stock === false` → `—` göster, düğme yok
- `track_stock === true` → `[−] {stock_quantity} [+]` düğmeleri
- `−` tıklanınca `StokDusurDialog` açılır (delta negatif)
- `+` tıklanınca aynı dialog "Yeni parti" sebebiyle açılır (delta pozitif)
- `stock_reserved > 0` ise adet yanında küçük gri not: `(2 rezerve)`
- Kaydedince listeyi tazele

- [ ] **Step 3: Ürün düzenleme sayfasına stok ve indirim bölümleri**

`frontend/idare/src/pages/urunler/[id].vue`:

```
Stok
  [✓] Stok takibi yap          ← v-switch, track_stock
      Stok adedi: [ 12 ]       ← track_stock true ise görünür
      Rezerve: 2 (ödeme bekliyor)   ← salt okunur, stock_reserved > 0 ise

İndirim
  Normal fiyat: 1.850 TL       ← salt okunur
  İndirimli fiyat: [ 1.450 ] TL
  Kaç adet indirimli: [ 10 ]
  Satılan: 3 · Kalan: 7        ← salt okunur
  [İndirimi kaldır]            ← indirim varsa görünür
```

Kaydetme mevcut PATCH ucunu kullanır; `clear_discount: true` gönderirse indirim kaldırılır. İndirimli fiyat normal fiyattan büyükse kaydetmeden önce uyar ("İndirimli fiyat normal fiyattan yüksek olamaz").

- [ ] **Step 4: Hareketler sekmesini ekle**

Aynı sayfada, mevcut sekme desenini (seçenek grubu sayfasındaki "kullanan ürünler" sekmesi) izleyerek "Hareketler" sekmesi ekle. `GET /api/idare/products/:id/movements` çağırır, tablo:

| Tarih | Değişim | Sebep | Not |
|---|---|---|---|
| 17.08.2026 14:22 | −1 | WhatsApp satışı | |
| 17.08.2026 09:10 | +20 | Yeni parti | |

Sebep kodları Türkçe etiketlere çevrilir: `siparis` → "Sipariş", `whatsapp_satisi` → "WhatsApp satışı", `sayim_duzeltme` → "Sayım düzeltme", `yeni_parti` → "Yeni parti", `iptal_iade` → "İptal/İade", `rezervasyon_iptal` → "Rezervasyon iptali". `was_discounted` true ise satırda küçük "indirimli" rozeti.

- [ ] **Step 5: Panel derlemesini doğrula**

```bash
cd frontend/idare && npm run build
```

Expected: hata yok.

- [ ] **Step 6: Panelde elle doğrula**

Paneli aç, bir ürüne stok takibi aç, 12 adet gir, kaydet. Listede `[−] 12 [+]` görünmeli. `−` ile 1 düşür (WhatsApp satışı), Hareketler sekmesinde kaydın göründüğünü doğrula.

- [ ] **Step 7: Commit**

```bash
git add frontend/idare/src/
git commit -m "feat(idare): stok yönetimi ve indirim paneli, hareket geçmişi"
```

---

## Task 9: Site arayüzü (public)

**Files:**
- Modify: `frontend/app/app/types/api.ts`
- Modify: `frontend/app/app/components/ProductCard.vue`
- Modify: `frontend/app/app/pages/urun/[slug].vue`
- Create: `frontend/app/app/pages/indirimli.vue`
- Modify: `frontend/app/app/components/TheHeader.vue`
- Modify: `frontend/app/app/utils/whatsapp.ts`
- Test: `frontend/app/app/utils/whatsapp.test.ts`

**Interfaces:**
- Consumes: Task 7 public alanları (`price`, `old_price`, `in_stock`, `discount_remaining`)
- Produces: müşteriye görünen arayüz — son task

- [ ] **Step 1: Tip tanımlarını güncelle**

`frontend/app/app/types/api.ts` — `Product` arayüzüne ekle:

```ts
export interface Product {
  id: number
  name: string
  slug: string
  description: string
  /** "1850.00" — GEÇERLİ fiyat; indirim aktifse indirimli fiyat */
  price: string
  /** İndirim aktifse üstü çizili gösterilecek normal fiyat, yoksa null */
  old_price: string | null
  /** Stok takibi kapalı ürünlerde her zaman true */
  in_stock: boolean
  /** Takipsiz üründe null */
  stock_quantity: number | null
  /** Kalan indirimli adet; indirim yoksa null */
  discount_remaining: number | null
  category_ids: number[]
  images: ProductImage[]
  option_groups?: ProductOptionGroup[]
}
```

- [ ] **Step 2: WhatsApp "tükendi" mesajı için failing test yaz**

`frontend/app/app/utils/whatsapp.test.ts` sonuna:

```ts
describe('buildOutOfStockMessage', () => {
  it('tükenen ürün için ne zaman geleceğini sorar', () => {
    const p = { name: 'Gül Buketi', slug: 'gul-buketi', price: '1850.00' } as Product
    const msg = buildOutOfStockMessage(p, 'https://site.com')

    expect(msg).toContain('Gül Buketi')
    expect(msg).toContain('tükenmiş')
    expect(msg).toContain('https://site.com/urun/gul-buketi')
  })
})
```

- [ ] **Step 3: Testin başarısız olduğunu doğrula**

```bash
cd frontend/app && npx vitest run app/utils/whatsapp.test.ts
```

Expected: FAIL — `buildOutOfStockMessage is not defined`.

- [ ] **Step 4: Mesaj fonksiyonunu ekle**

`frontend/app/app/utils/whatsapp.ts`:

```ts
/**
 * Tükenen ürün için WhatsApp mesajı. Ürün sitede görünür kalıyor (spec §6.1)
 * — müşteri "ne zaman gelir" diye sorabilsin, satış fırsatı kaybolmasın.
 */
export function buildOutOfStockMessage(product: Product, siteUrl: string): string {
  return [
    'Merhaba, bu ürün tükenmiş görünüyor:',
    product.name,
    `${siteUrl}/urun/${product.slug}`,
    'Ne zaman tekrar gelir?',
  ].join('\n')
}

/** Tükenen ürün için wa.me linki. */
export function buildOutOfStockUrl(
  phoneNumber: string,
  product: Product,
  siteUrl: string,
): string {
  return `https://wa.me/${phoneNumber}?text=${encodeURIComponent(
    buildOutOfStockMessage(product, siteUrl))}`
}
```

- [ ] **Step 5: Testin geçtiğini doğrula**

```bash
cd frontend/app && npx vitest run app/utils/whatsapp.test.ts
```

Expected: PASS.

- [ ] **Step 6: ProductCard'a rozetleri ekle**

`frontend/app/app/components/ProductCard.vue`:

- `old_price` doluysa: fiyat satırında eski fiyat üstü çizili + soluk, yeni fiyat vurgulu. İndirim yüzdesi rozeti: `Math.round((1 - Number(price) / Number(old_price)) * 100)` → `%22 İNDİRİM`
- `in_stock === false` ise: "TÜKENDİ" rozeti, "Sepete Ekle" düğmesi `disabled`, altına "WhatsApp'tan sor" düğmesi (`buildOutOfStockUrl`)
- İki rozet aynı anda görünebilir — üst üste binmeyecek şekilde konumlandır (indirim sol üst, tükendi sağ üst)
- Ürün listede **gizlenmez** — sadece satın alma kapanır

- [ ] **Step 7: Ürün detay sayfasını güncelle**

`frontend/app/app/pages/urun/[slug].vue` — aynı kurallar: eski/yeni fiyat, tükendi durumunda sepete ekleme kapalı + WhatsApp düğmesi. İndirim aktifse "Son {discount_remaining} adet bu fiyata" notu göster (`discount_remaining` null değilse).

- [ ] **Step 8: İndirimli Ürünler sayfasını oluştur**

`frontend/app/app/pages/indirimli.vue` — mevcut kategori listeleme sayfasının desenini izle, `GET /api/products?discounted=true` çağırır. Başlık "İndirimli Ürünler". Ürün yoksa: "Şu anda indirimli ürün bulunmuyor." mesajı.

`frontend/app/app/components/TheHeader.vue` — menüye "İndirimli Ürünler" linki ekle (`/indirimli`).

- [ ] **Step 9: Sepette stok uyarısını göster**

`frontend/app/app/composables/useCart.ts` / sipariş gönderim akışında: sunucudan gelen stok hatası (400 + mesaj) kullanıcıya gösterilir. Mesaj zaten Türkçe ve ürün adı + kalan adet içeriyor (Task 4 `StockError`), olduğu gibi gösterilir. Ödeme başlatılamaz, müşteri sepeti düzeltip tekrar dener.

- [ ] **Step 10: Frontend testlerini ve derlemeyi doğrula**

```bash
cd frontend/app && npx vitest run && npm run build
```

Expected: testler PASS, derleme hatasız.

- [ ] **Step 11: E2E — proxy üzerinden doğrula**

**Nuxt proxy üzerinden** (doğrudan Go'ya değil — proxy hatalarını gizler):

```bash
# Backend ve frontend çalışır durumda olmalı
curl -s localhost:3000/api/products | head -c 400
curl -s 'localhost:3000/api/products?discounted=true' | head -c 300
```

Tarayıcıda doğrula:
1. Bir ürüne panelden stok 1 gir → sitede normal görünür
2. Sepete ekle, ödeme ekranına git (tamamlama) → başka bir tarayıcıda aynı ürün "TÜKENDİ" görünmeli (rezerve edildi)
3. Ödemeyi tamamlama, 20 dk bekle (veya süpürücü aralığını test için kısalt) → ürün tekrar satılabilir olmalı
4. Panelden indirim gir → `/indirimli` sayfasında görünmeli, kartta eski/yeni fiyat

- [ ] **Step 12: Commit**

```bash
git add frontend/app/app/
git commit -m "feat(site): tükendi rozeti, indirimli fiyat gösterimi, indirimli ürünler sayfası"
```

---

## Self-Review Notları

Bu plan yazıldıktan sonra spec'e karşı kontrol edildi:

- **Spec §3.1 (kolonlar)** → Task 1. `order_items.was_discounted` ve `orders.stock_swept` planlama sırasında gerekli olduğu anlaşılıp Task 5/6'da aynı migration'a eklendi.
- **Spec §3.3 (türetilen kavramlar)** → Task 2, tek yerde tanımlı.
- **Spec §4.1 (atomik rezervasyon)** → Task 4, `TestReserve_YarisDurumu_TekKazanan` ile kanıtlanıyor.
- **Spec §4.2 (üç çıkış)** → Task 5 (kesinleşme, iade) + Task 6 (süpürme).
- **Spec §4.4 (kota)** → Task 4 `CommitReservation`/`ManualAdjust`, Task 5 `EffectivePrice`.
- **Spec §5 (panel)** → Task 8. **Spec §6 (site)** → Task 9. **Spec §7 (API)** → Task 7.
- **Spec §8 (hata yönetimi)** → Task 4 (`StockError`), Task 5 (iade logu), Task 6 (süpürücü devam eder).
- **Spec §9 (testler)** → Task 2 (saf mantık), Task 4 (yarış, manuel), Task 5 (idempotency, indirimli fiyat), Task 6 (süpürücü), Task 7 (API), Task 9 (E2E proxy).
