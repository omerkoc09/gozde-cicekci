# Ürün Özelleştirme ("Buket Tasarla") Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Müşteri ürün sipariş ederken ambalaj/kurdele/kutu rengi gibi
seçenekleri seçebilsin; esnaf bu seçenek gruplarını ve renkleri panelden
kendisi tanımlayıp her ürüne ayrı ayrı açabilsin.

**Architecture:** Merkezi seçenek havuzu (`option_groups` + `option_values`)
ürünlere `product_option_groups` ile bağlanır. Sipariş anındaki seçim
`order_item_options`'a **kopyalanır** (isim + hex), referans tutulmaz —
`order_items.product_name` desenindeki gibi. Yeni `internal/productoption`
paketi mevcut `internal/category` paketinin desenini takip eder; `order`
paketi ona dar bir arayüzle (`OptionReader`) bağlanır.

**Tech Stack:** Go 1.x + Fiber v2 + pgx v5 (backend), Vuetify 3 + Vite
(admin panel `frontend/idare`), Nuxt 4 SSR (public site `frontend/app`),
Postgres 16, testify.

**Spec:** `docs/superpowers/specs/2026-08-16-urun-ozellestirme-design.md`

## Global Constraints

- **Test komutu `make test`** — `go test ./...` KULLANILMAZ. Tüm test
  paketleri aynı DB'yi paylaşıyor ve `NewTestDB` TRUNCATE çalıştırıyor;
  paralel paketler birbirini siler. Makefile `-p 1` kullanıyor.
- **Dev Postgres portu 5435** (5433 değil), test DB 5434.
- **Bilinen kırmızı testler:** `pkg/config`'deki `TestLoad_PayTRDefaultsUnconfigured`
  ve `TestLoad_PayTRUnconfiguredWhenPartiallySet` bu çalışmadan ÖNCE de
  kırıktı. Bunları düzeltmek bu planın kapsamı değil; başka bir test
  kırmızıya dönerse o gerçek bir regresyondur.
- **Arayüz dili Türkçe** — kullanıcıya görünen tüm metinler, hata mesajları
  ve kod yorumları Türkçe.
- **Fiber route sırası:** Sabit segmentli route'lar (`/reorder`) `:id`
  kalıplarından ÖNCE kaydedilir; Fiber sıralı eşleştirir.
- **Para/fiyat yok:** Bu özellik fiyatı ETKİLEMEZ. `itemsTotal`, PayTR
  sepeti ve callback tutar doğrulaması hiçbir görevde değişmez.
- **Sunucu tarayıcıya güvenmez:** Sipariş oluştururken tarayıcı yalnızca
  `option_value_ids` gönderir; `group_name`/`value_name`/`swatch_hex`
  DB'den okunur.
- **E2E doğrulama Nuxt proxy üzerinden** (`localhost:3000/api/go/*`),
  Go'ya (`:8080`) doğrudan curl atılmaz — proxy katmanı hatalarını gizler.
- **Migration numarası 10** (mevcut son: `000009_customers`).

---

### Task 1: Migration 10 — dört tablo + seed

**Files:**
- Create: `backend/migrations/000010_product_options.up.sql`
- Create: `backend/migrations/000010_product_options.down.sql`

**Interfaces:**
- Consumes: yok (ilk görev)
- Produces: `option_groups`, `option_values`, `product_option_groups`,
  `order_item_options` tabloları. Sonraki tüm görevler bunlara yazar/okur.

- [ ] **Step 1: up migration'ı yaz**

`backend/migrations/000010_product_options.up.sql`:

```sql
-- Seçenek grubu: "Ambalaj Rengi", "Kurdele Rengi", "Çiçek Rengi"...
-- Panelden eklenir; kodda gömülü liste YOK.
CREATE TABLE option_groups (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL UNIQUE,
    -- kind oluşturulduktan sonra DEĞİŞMEZ: color'dan text'e geçiş mevcut
    -- hex değerlerini anlamsız kılar (categories.axis ile aynı kural).
    kind       TEXT NOT NULL CHECK (kind IN ('color','text')),
    sort_order INT NOT NULL DEFAULT 0,
    is_active  BOOLEAN NOT NULL DEFAULT true
);

-- Gruba bağlı değerler: "Pembe" #F5A9C8
CREATE TABLE option_values (
    id         BIGSERIAL PRIMARY KEY,
    group_id   BIGINT NOT NULL REFERENCES option_groups(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    -- kind='text' olan grupta boş kalır.
    swatch_hex TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    is_active  BOOLEAN NOT NULL DEFAULT true
);

CREATE INDEX idx_option_values_group ON option_values(group_id, sort_order);

-- Hangi üründe hangi grup soruluyor.
CREATE TABLE product_option_groups (
    product_id  BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    group_id    BIGINT NOT NULL REFERENCES option_groups(id) ON DELETE CASCADE,
    is_required BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (product_id, group_id)
);

-- Sipariş anındaki seçim. Gruba/değere REFERANS YOK — isim ve hex
-- kopyalanır. Esnaf sonradan "Pembe"yi silerse veya rengini değiştirirse
-- eski siparişin ne olduğu bilgisi bozulmamalı
-- (order_items.product_name / price_at_order deseninin aynısı).
CREATE TABLE order_item_options (
    id            BIGSERIAL PRIMARY KEY,
    order_item_id BIGINT NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    group_name    TEXT NOT NULL,
    value_name    TEXT NOT NULL,
    swatch_hex    TEXT NOT NULL DEFAULT '',
    sort_order    INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_order_item_options_item
    ON order_item_options(order_item_id, sort_order);

-- Başlangıç verisi. Panelden düzenlenebilir/silinebilir — kodda gömülü
-- değil, esnaf "Çiçek Rengi" grubunu sonradan kendisi ekleyebilir.
INSERT INTO option_groups (name, slug, kind, sort_order) VALUES
    ('Ambalaj Rengi',  'ambalaj-rengi',  'color', 0),
    ('Kurdele Rengi',  'kurdele-rengi',  'color', 1),
    ('Kutu Rengi',     'kutu-rengi',     'color', 2);

INSERT INTO option_values (group_id, name, swatch_hex, sort_order)
SELECT g.id, v.name, v.hex, v.ord
FROM option_groups g
CROSS JOIN (VALUES
    ('Pembe',    '#F0A6CA', 0),
    ('Beyaz',    '#FFFFFF', 1),
    ('Gri',      '#C4C4C4', 2),
    ('Kırmızı',  '#D93A34', 3),
    ('Fuşya',    '#E0219A', 4),
    ('Mor',      '#7B2FF7', 5),
    ('Lacivert', '#41618A', 6),
    ('Siyah',    '#000000', 7),
    ('Hardal',   '#D9A441', 8),
    ('Yeşil',    '#8CD147', 9)
) AS v(name, hex, ord)
WHERE g.slug IN ('ambalaj-rengi', 'kurdele-rengi', 'kutu-rengi');
```

- [ ] **Step 2: down migration'ı yaz**

`backend/migrations/000010_product_options.down.sql`:

```sql
-- DİKKAT: order_item_options da düşer, geçmiş siparişlerin renk seçimleri
-- KALICI OLARAK kaybolur. Down yalnızca geliştirme içindir.
DROP TABLE IF EXISTS order_item_options;
DROP TABLE IF EXISTS product_option_groups;
DROP TABLE IF EXISTS option_values;
DROP TABLE IF EXISTS option_groups;
```

- [ ] **Step 3: Migration'ı dev ve test DB'sine uygula**

```bash
cd /Users/omerkoc/GolandProjects/cicekci
DATABASE_URL="postgres://cicekci:cicekci@localhost:5435/cicekci?sslmode=disable" make migrate-up
TEST_DATABASE_URL="postgres://cicekci:cicekci@localhost:5434/cicekci_test?sslmode=disable" make test-db-migrate
```

Beklenen: `10/u product_options` çıktısı, hata yok.

- [ ] **Step 4: Seed verisinin geldiğini doğrula**

```bash
docker compose exec -T postgres psql -U cicekci -d cicekci -c \
  "SELECT g.name, count(v.id) FROM option_groups g
   LEFT JOIN option_values v ON v.group_id=g.id GROUP BY g.name ORDER BY g.name;"
```

Beklenen: üç grup, her birinde 10 değer.

- [ ] **Step 5: Commit**

```bash
git add backend/migrations/000010_product_options.up.sql backend/migrations/000010_product_options.down.sql
git commit -m "feat(db): seçenek grupları/değerleri + sipariş kalemi seçimleri

order_item_options gruba referans TUTMAZ, isim+hex kopyalar — esnaf
sonradan rengi silerse eski sipariş bozulmasın (product_name deseni)."
```

---

### Task 2: productoption paketi — model + store (grup CRUD)

**Files:**
- Create: `backend/internal/productoption/model.go`
- Create: `backend/internal/productoption/store.go`
- Create: `backend/internal/productoption/store_test.go`

**Interfaces:**
- Consumes: Task 1'in tabloları
- Produces:
  - `type Kind string`; `KindColor Kind = "color"`, `KindText Kind = "text"`; `func (k Kind) Valid() bool`
  - `type Group struct { ID int64; Name, Slug string; Kind Kind; SortOrder int; IsActive bool; Values []Value }`
  - `type Value struct { ID, GroupID int64; Name, SwatchHex string; SortOrder int; IsActive bool }`
  - `type CreateGroupInput struct { Name string; Kind Kind }`
  - `type UpdateGroupInput struct { Name *string; IsActive *bool }`
  - `type CreateValueInput struct { GroupID int64; Name, SwatchHex string }`
  - `type UpdateValueInput struct { Name, SwatchHex *string; IsActive *bool }`
  - `func NewStore(pool *pgxpool.Pool) *Store`
  - `(*Store) CreateGroup(ctx, CreateGroupInput, slug string) (*Group, error)`
  - `(*Store) GroupSlugExists(ctx, slug string) (bool, error)`
  - `(*Store) GetGroup(ctx, id int64) (*Group, error)`
  - `(*Store) UpdateGroup(ctx, id int64, UpdateGroupInput) (*Group, error)`
  - `(*Store) DeleteGroup(ctx, id int64) error`
  - `(*Store) ListGroups(ctx, onlyActive bool) ([]Group, error)` — değerleri dahil, tek batch sorgu
  - `(*Store) MaxGroupSortOrder(ctx) (int, error)`

- [ ] **Step 1: model.go yaz**

```go
package productoption

// Kind bir seçenek grubunun değerlerinin nasıl gösterileceğini söyler.
// Oluşturulduktan sonra DEĞİŞMEZ — color'dan text'e geçiş mevcut hex
// değerlerini anlamsız kılar (categories.axis ile aynı kural).
type Kind string

const (
	KindColor Kind = "color"
	KindText  Kind = "text"
)

func (k Kind) Valid() bool {
	return k == KindColor || k == KindText
}

// Group bir seçenek grubu — "Ambalaj Rengi". Values her zaman doludur
// (grup değerleriyle birlikte okunur), boş grup boş slice döner.
type Group struct {
	ID        int64
	Name      string
	Slug      string
	Kind      Kind
	SortOrder int
	IsActive  bool
	Values    []Value
}

// Value gruba bağlı tek seçenek — "Pembe" #F0A6CA.
// SwatchHex yalnızca KindColor gruplarda dolu.
type Value struct {
	ID        int64
	GroupID   int64
	Name      string
	SwatchHex string
	SortOrder int
	IsActive  bool
}

type CreateGroupInput struct {
	Name string
	Kind Kind
}

// UpdateGroupInput pointer alanlar PATCH semantiği — nil değişmez.
// Kind burada bilinçli olarak YOK: tip güncellenemez.
type UpdateGroupInput struct {
	Name     *string
	IsActive *bool
}

type CreateValueInput struct {
	GroupID   int64
	Name      string
	SwatchHex string
}

type UpdateValueInput struct {
	Name      *string
	SwatchHex *string
	IsActive  *bool
}
```

- [ ] **Step 2: Failing test yaz**

`backend/internal/productoption/store_test.go`:

```go
package productoption

import (
	"context"
	"testing"

	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	return NewStore(database.NewTestDB(t)), context.Background()
}

func TestStore_CreateGroup_DegerleriBosBaslar(t *testing.T) {
	store, ctx := newTestStore(t)

	g, err := store.CreateGroup(ctx, CreateGroupInput{
		Name: "Ambalaj Rengi", Kind: KindColor,
	}, "ambalaj-rengi")

	require.NoError(t, err)
	assert.Equal(t, "Ambalaj Rengi", g.Name)
	assert.Equal(t, "ambalaj-rengi", g.Slug)
	assert.Equal(t, KindColor, g.Kind)
	assert.True(t, g.IsActive)
	assert.Empty(t, g.Values)
}

func TestStore_ListGroups_DegerleriDeGetirir(t *testing.T) {
	store, ctx := newTestStore(t)

	g, err := store.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindColor}, "ambalaj")
	require.NoError(t, err)

	_, err = store.CreateValue(ctx, CreateValueInput{
		GroupID: g.ID, Name: "Pembe", SwatchHex: "#F0A6CA",
	})
	require.NoError(t, err)

	list, err := store.ListGroups(ctx, false)

	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Len(t, list[0].Values, 1)
	assert.Equal(t, "Pembe", list[0].Values[0].Name)
	assert.Equal(t, "#F0A6CA", list[0].Values[0].SwatchHex)
}

// onlyActive=true public akış için: pasif grup DA pasif değer DE gelmemeli.
func TestStore_ListGroups_OnlyActive_PasifleriEler(t *testing.T) {
	store, ctx := newTestStore(t)

	aktif, err := store.CreateGroup(ctx, CreateGroupInput{Name: "Aktif", Kind: KindColor}, "aktif")
	require.NoError(t, err)
	pasifGrup, err := store.CreateGroup(ctx, CreateGroupInput{Name: "Pasif", Kind: KindColor}, "pasif")
	require.NoError(t, err)

	yok := false
	_, err = store.UpdateGroup(ctx, pasifGrup.ID, UpdateGroupInput{IsActive: &yok})
	require.NoError(t, err)

	_, err = store.CreateValue(ctx, CreateValueInput{GroupID: aktif.ID, Name: "Pembe", SwatchHex: "#F0A6CA"})
	require.NoError(t, err)
	pasifDeger, err := store.CreateValue(ctx, CreateValueInput{GroupID: aktif.ID, Name: "Eski", SwatchHex: "#000000"})
	require.NoError(t, err)
	_, err = store.UpdateValue(ctx, pasifDeger.ID, UpdateValueInput{IsActive: &yok})
	require.NoError(t, err)

	list, err := store.ListGroups(ctx, true)

	require.NoError(t, err)
	require.Len(t, list, 1, "pasif grup gelmemeli")
	require.Len(t, list[0].Values, 1, "pasif değer gelmemeli")
	assert.Equal(t, "Pembe", list[0].Values[0].Name)
}

func TestStore_DeleteGroup_DegerleriDeSiler(t *testing.T) {
	store, ctx := newTestStore(t)

	g, err := store.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindColor}, "ambalaj")
	require.NoError(t, err)
	_, err = store.CreateValue(ctx, CreateValueInput{GroupID: g.ID, Name: "Pembe", SwatchHex: "#F0A6CA"})
	require.NoError(t, err)

	require.NoError(t, store.DeleteGroup(ctx, g.ID))

	list, err := store.ListGroups(ctx, false)
	require.NoError(t, err)
	assert.Empty(t, list)
}
```

- [ ] **Step 3: Test'i çalıştır, kırmızı olduğunu gör**

```bash
cd backend && go vet ./internal/productoption/
```

Beklenen: FAIL — `NewStore`, `Store`, `CreateValue` tanımlı değil.

- [ ] **Step 4: store.go yaz**

```go
package productoption

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const groupColumns = `id, name, slug, kind, sort_order, is_active`
const valueColumns = `id, group_id, name, swatch_hex, sort_order, is_active`

func scanGroup(row pgx.Row) (*Group, error) {
	var g Group
	err := row.Scan(&g.ID, &g.Name, &g.Slug, &g.Kind, &g.SortOrder, &g.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("seçenek grubu scan: %w", err)
	}
	g.Values = []Value{}
	return &g, nil
}

func (s *Store) CreateGroup(ctx context.Context, in CreateGroupInput, slug string) (*Group, error) {
	max, err := s.MaxGroupSortOrder(ctx)
	if err != nil {
		return nil, err
	}
	return scanGroup(s.pool.QueryRow(ctx,
		`INSERT INTO option_groups (name, slug, kind, sort_order)
		 VALUES ($1, $2, $3, $4) RETURNING `+groupColumns,
		in.Name, slug, in.Kind, max+1,
	))
}

// GroupSlugExists çakışma kontrolü — service -2, -3 eki eklerken kullanır.
func (s *Store) GroupSlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM option_groups WHERE slug = $1)`, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("slug kontrol: %w", err)
	}
	return exists, nil
}

// MaxGroupSortOrder yeni grubu sona eklemek için. Hiç grup yoksa -1.
func (s *Store) MaxGroupSortOrder(ctx context.Context) (int, error) {
	var max int
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(max(sort_order), -1) FROM option_groups`).Scan(&max)
	if err != nil {
		return 0, fmt.Errorf("sıra oku: %w", err)
	}
	return max, nil
}

// GetGroup grubu değerleriyle birlikte döner.
func (s *Store) GetGroup(ctx context.Context, id int64) (*Group, error) {
	g, err := scanGroup(s.pool.QueryRow(ctx,
		`SELECT `+groupColumns+` FROM option_groups WHERE id = $1`, id))
	if err != nil {
		return nil, err
	}

	byGroup, err := s.valuesOfMany(ctx, []int64{id}, false)
	if err != nil {
		return nil, err
	}
	if v, ok := byGroup[id]; ok {
		g.Values = v
	}
	return g, nil
}

func (s *Store) UpdateGroup(ctx context.Context, id int64, in UpdateGroupInput) (*Group, error) {
	return scanGroup(s.pool.QueryRow(ctx,
		`UPDATE option_groups SET
		   name      = COALESCE($2, name),
		   is_active = COALESCE($3, is_active)
		 WHERE id = $1
		 RETURNING `+groupColumns,
		id, in.Name, in.IsActive,
	))
}

func (s *Store) DeleteGroup(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM option_groups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("seçenek grubu sil: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

// ListGroups grupları değerleriyle döner. onlyActive true ise hem pasif
// gruplar hem pasif değerler elenir (public akış).
//
// Değerler TEK batch sorguyla çekilir — grup başına ayrı sorgu açmak
// N+1 olurdu (order.Store.List'te aynı ders alınmıştı).
func (s *Store) ListGroups(ctx context.Context, onlyActive bool) ([]Group, error) {
	where := ``
	if onlyActive {
		where = ` WHERE is_active`
	}

	rows, err := s.pool.Query(ctx,
		`SELECT `+groupColumns+` FROM option_groups`+where+` ORDER BY sort_order, id`)
	if err != nil {
		return nil, fmt.Errorf("seçenek grupları listele: %w", err)
	}
	defer rows.Close()

	groups := make([]Group, 0)
	ids := make([]int64, 0)
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.Slug, &g.Kind, &g.SortOrder, &g.IsActive); err != nil {
			return nil, fmt.Errorf("seçenek grubu scan: %w", err)
		}
		g.Values = []Value{}
		groups = append(groups, g)
		ids = append(ids, g.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return groups, nil
	}

	byGroup, err := s.valuesOfMany(ctx, ids, onlyActive)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if v, ok := byGroup[groups[i].ID]; ok {
			groups[i].Values = v
		}
	}
	return groups, nil
}

// valuesOfMany birden çok grubun değerlerini tek sorguda çeker.
func (s *Store) valuesOfMany(ctx context.Context, groupIDs []int64, onlyActive bool) (map[int64][]Value, error) {
	where := `WHERE group_id = ANY($1)`
	if onlyActive {
		where += ` AND is_active`
	}

	rows, err := s.pool.Query(ctx,
		`SELECT `+valueColumns+` FROM option_values `+where+` ORDER BY sort_order, id`, groupIDs)
	if err != nil {
		return nil, fmt.Errorf("seçenek değerleri listele: %w", err)
	}
	defer rows.Close()

	out := make(map[int64][]Value, len(groupIDs))
	for rows.Next() {
		var v Value
		if err := rows.Scan(&v.ID, &v.GroupID, &v.Name, &v.SwatchHex, &v.SortOrder, &v.IsActive); err != nil {
			return nil, fmt.Errorf("seçenek değeri scan: %w", err)
		}
		out[v.GroupID] = append(out[v.GroupID], v)
	}
	return out, rows.Err()
}
```

- [ ] **Step 5: Değer CRUD'unu store.go'ya ekle**

Aynı dosyanın sonuna:

```go
// CreateValue değeri grubun sonuna ekler.
func (s *Store) CreateValue(ctx context.Context, in CreateValueInput) (*Value, error) {
	var max int
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(max(sort_order), -1) FROM option_values WHERE group_id = $1`,
		in.GroupID).Scan(&max)
	if err != nil {
		return nil, fmt.Errorf("sıra oku: %w", err)
	}

	return scanValue(s.pool.QueryRow(ctx,
		`INSERT INTO option_values (group_id, name, swatch_hex, sort_order)
		 VALUES ($1, $2, $3, $4) RETURNING `+valueColumns,
		in.GroupID, in.Name, in.SwatchHex, max+1,
	))
}

func scanValue(row pgx.Row) (*Value, error) {
	var v Value
	err := row.Scan(&v.ID, &v.GroupID, &v.Name, &v.SwatchHex, &v.SortOrder, &v.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("seçenek değeri scan: %w", err)
	}
	return &v, nil
}

func (s *Store) GetValue(ctx context.Context, id int64) (*Value, error) {
	return scanValue(s.pool.QueryRow(ctx,
		`SELECT `+valueColumns+` FROM option_values WHERE id = $1`, id))
}

func (s *Store) UpdateValue(ctx context.Context, id int64, in UpdateValueInput) (*Value, error) {
	return scanValue(s.pool.QueryRow(ctx,
		`UPDATE option_values SET
		   name       = COALESCE($2, name),
		   swatch_hex = COALESCE($3, swatch_hex),
		   is_active  = COALESCE($4, is_active)
		 WHERE id = $1
		 RETURNING `+valueColumns,
		id, in.Name, in.SwatchHex, in.IsActive,
	))
}

func (s *Store) DeleteValue(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM option_values WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("seçenek değeri sil: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}
```

- [ ] **Step 6: Testleri çalıştır, yeşil olduğunu gör**

```bash
cd backend && go test -p 1 ./internal/productoption/ -v
```

Beklenen: 4 test PASS.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/productoption/
git commit -m "feat(productoption): grup/değer store'u

Değerler tek batch sorguyla çekiliyor (grup başına sorgu N+1 olurdu —
order.Store.List'teki ders). onlyActive public akış için pasif grup VE
pasif değerleri birlikte eliyor."
```

---

### Task 3: productoption service — doğrulama + sıralama + ürün bağlama

**Files:**
- Create: `backend/internal/productoption/service.go`
- Create: `backend/internal/productoption/service_test.go`
- Modify: `backend/internal/productoption/store.go` (sıralama + ürün bağlama sorguları)

**Interfaces:**
- Consumes: Task 2'nin `Store` metotları
- Produces:
  - `func NewService(store *Store) *Service`
  - `(*Service) CreateGroup(ctx, CreateGroupInput) (*Group, error)` — slug üretir, `Kind` doğrular
  - `(*Service) UpdateGroup(ctx, id int64, UpdateGroupInput) (*Group, error)`
  - `(*Service) DeleteGroup(ctx, id int64) error`
  - `(*Service) ListGroups(ctx, onlyActive bool) ([]Group, error)`
  - `(*Service) ReorderGroups(ctx, ids []int64) error`
  - `(*Service) GroupProductCount(ctx, groupID int64) (int, error)`
  - `(*Service) CreateValue(ctx, CreateValueInput) (*Value, error)`
  - `(*Service) UpdateValue(ctx, id int64, UpdateValueInput) (*Value, error)`
  - `(*Service) DeleteValue(ctx, id int64) error`
  - `(*Service) ReorderValues(ctx, groupID int64, ids []int64) error`
  - `(*Service) SetProductGroups(ctx, productID int64, links []ProductGroupLink) error`
  - `(*Service) GroupsForProduct(ctx, productID int64, onlyActive bool) ([]ProductGroup, error)`
  - `type ProductGroupLink struct { GroupID int64; IsRequired bool }`
  - `type ProductGroup struct { Group; IsRequired bool }`

- [ ] **Step 1: Store'a sıralama ve ürün bağlama sorgularını ekle**

`backend/internal/productoption/store.go` sonuna:

```go
// GroupIDs tüm grup ID'lerini döner — ReorderGroups'un gelen listeyi
// mevcutların tamamıyla karşılaştırması için.
func (s *Store) GroupIDs(ctx context.Context) ([]int64, error) {
	return s.scanIDs(ctx, `SELECT id FROM option_groups`)
}

// ValueIDsOfGroup gruptaki tüm değer ID'lerini döner.
func (s *Store) ValueIDsOfGroup(ctx context.Context, groupID int64) ([]int64, error) {
	return s.scanIDs(ctx, `SELECT id FROM option_values WHERE group_id = $1`, groupID)
}

func (s *Store) scanIDs(ctx context.Context, query string, args ...any) ([]int64, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("id listele: %w", err)
	}
	defer rows.Close()

	out := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("id scan: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ReorderGroups ids sırasına göre sort_order'ı 0,1,2... yazar.
// Tek transaction: yarım kalırsa hiçbiri uygulanmaz.
func (s *Store) ReorderGroups(ctx context.Context, ids []int64) error {
	return s.reorder(ctx, `UPDATE option_groups SET sort_order = $2 WHERE id = $1`, ids)
}

func (s *Store) ReorderValues(ctx context.Context, ids []int64) error {
	return s.reorder(ctx, `UPDATE option_values SET sort_order = $2 WHERE id = $1`, ids)
}

func (s *Store) reorder(ctx context.Context, query string, ids []int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("sıralama tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for i, id := range ids {
		if _, err := tx.Exec(ctx, query, id, i); err != nil {
			return fmt.Errorf("sıra yaz: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("sıralama commit: %w", err)
	}
	return nil
}

// GroupProductCount grubu kaç ürünün kullandığı — silme öncesi uyarı için.
func (s *Store) GroupProductCount(ctx context.Context, groupID int64) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM product_option_groups WHERE group_id = $1`, groupID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("ürün say: %w", err)
	}
	return n, nil
}

// SetProductGroups ürünün seçenek gruplarını KOMPLE değiştirir (önce siler,
// sonra yazar) — tek transaction. Boş links tüm bağları kaldırır.
func (s *Store) SetProductGroups(ctx context.Context, productID int64, links []ProductGroupLink) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ürün seçenek tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM product_option_groups WHERE product_id = $1`, productID); err != nil {
		return fmt.Errorf("ürün seçenek sil: %w", err)
	}

	for _, l := range links {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_option_groups (product_id, group_id, is_required)
			 VALUES ($1, $2, $3)`, productID, l.GroupID, l.IsRequired); err != nil {
			return fmt.Errorf("ürün seçenek ekle: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ürün seçenek commit: %w", err)
	}
	return nil
}

// GroupsForProduct ürüne açık grupları değerleriyle döner.
// onlyActive true ise pasif grup/değer elenir (public akış).
func (s *Store) GroupsForProduct(ctx context.Context, productID int64, onlyActive bool) ([]ProductGroup, error) {
	where := `WHERE pog.product_id = $1`
	if onlyActive {
		where += ` AND g.is_active`
	}

	rows, err := s.pool.Query(ctx,
		`SELECT g.id, g.name, g.slug, g.kind, g.sort_order, g.is_active, pog.is_required
		 FROM product_option_groups pog
		 JOIN option_groups g ON g.id = pog.group_id `+where+`
		 ORDER BY g.sort_order, g.id`, productID)
	if err != nil {
		return nil, fmt.Errorf("ürün seçenek grupları: %w", err)
	}
	defer rows.Close()

	out := make([]ProductGroup, 0)
	ids := make([]int64, 0)
	for rows.Next() {
		var pg ProductGroup
		if err := rows.Scan(&pg.ID, &pg.Name, &pg.Slug, &pg.Kind,
			&pg.SortOrder, &pg.IsActive, &pg.IsRequired); err != nil {
			return nil, fmt.Errorf("ürün seçenek grubu scan: %w", err)
		}
		pg.Values = []Value{}
		out = append(out, pg)
		ids = append(ids, pg.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return out, nil
	}

	byGroup, err := s.valuesOfMany(ctx, ids, onlyActive)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if v, ok := byGroup[out[i].ID]; ok {
			out[i].Values = v
		}
	}
	return out, nil
}
```

- [ ] **Step 2: model.go'ya ürün bağlama tiplerini ekle**

`backend/internal/productoption/model.go` sonuna:

```go
// ProductGroupLink ürün formundan gelen bağ — hangi grup, zorunlu mu.
type ProductGroupLink struct {
	GroupID    int64
	IsRequired bool
}

// ProductGroup ürüne açık bir grup; Group'a is_required eklenmiş hali.
type ProductGroup struct {
	Group
	IsRequired bool
}
```

- [ ] **Step 3: Failing test yaz**

`backend/internal/productoption/service_test.go`:

```go
package productoption

import (
	"context"
	"testing"

	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*Service, context.Context) {
	t.Helper()
	return NewService(NewStore(database.NewTestDB(t))), context.Background()
}

func TestService_CreateGroup_SlugUretir(t *testing.T) {
	svc, ctx := newTestService(t)

	g, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj Rengi", Kind: KindColor})

	require.NoError(t, err)
	assert.Equal(t, "ambalaj-rengi", g.Slug)
}

func TestService_CreateGroup_AyniAdSlugCakismasi(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindColor})
	require.NoError(t, err)

	ikinci, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindText})

	require.NoError(t, err)
	assert.Equal(t, "ambalaj-2", ikinci.Slug)
}

func TestService_CreateGroup_GecersizKindReddedilir(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: "renk"})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_CreateGroup_BosAdReddedilir(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "   ", Kind: KindColor})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

// Renk grubunda hex zorunlu ve geçerli formatta olmalı; aksi halde
// müşteri sayfasında görünmez bir nokta çıkar.
func TestService_CreateValue_RenkGrubundaHexZorunlu(t *testing.T) {
	svc, ctx := newTestService(t)

	g, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindColor})
	require.NoError(t, err)

	_, err = svc.CreateValue(ctx, CreateValueInput{GroupID: g.ID, Name: "Pembe"})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_CreateValue_GecersizHexReddedilir(t *testing.T) {
	svc, ctx := newTestService(t)

	g, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindColor})
	require.NoError(t, err)

	for _, hex := range []string{"F0A6CA", "#XYZ", "#F0A6C", "pembe"} {
		_, err = svc.CreateValue(ctx, CreateValueInput{GroupID: g.ID, Name: "Pembe", SwatchHex: hex})
		assert.ErrorIs(t, err, errorsx.ErrInvalidInput, "hex %q reddedilmeliydi", hex)
	}
}

// Metin grubunda hex gönderilse bile yok sayılır — kind='text' değerinde
// hex saklamak anlamsız.
func TestService_CreateValue_MetinGrubundaHexTemizlenir(t *testing.T) {
	svc, ctx := newTestService(t)

	g, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Boy", Kind: KindText})
	require.NoError(t, err)

	v, err := svc.CreateValue(ctx, CreateValueInput{GroupID: g.ID, Name: "Orta", SwatchHex: "#FFFFFF"})

	require.NoError(t, err)
	assert.Empty(t, v.SwatchHex)
}

func TestService_ReorderGroups_YenidenNumaralar(t *testing.T) {
	svc, ctx := newTestService(t)

	ids := make([]int64, 0, 3)
	for _, ad := range []string{"Bir", "İki", "Üç"} {
		g, err := svc.CreateGroup(ctx, CreateGroupInput{Name: ad, Kind: KindColor})
		require.NoError(t, err)
		ids = append(ids, g.ID)
	}

	require.NoError(t, svc.ReorderGroups(ctx, []int64{ids[2], ids[0], ids[1]}))

	list, err := svc.ListGroups(ctx, false)
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, ids[2], list[0].ID)
	assert.Equal(t, ids[0], list[1].ID)
	assert.Equal(t, ids[1], list[2].ID)
}

func TestService_ReorderGroups_EksikIDReddedilir(t *testing.T) {
	svc, ctx := newTestService(t)

	g, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Bir", Kind: KindColor})
	require.NoError(t, err)
	_, err = svc.CreateGroup(ctx, CreateGroupInput{Name: "İki", Kind: KindColor})
	require.NoError(t, err)

	err = svc.ReorderGroups(ctx, []int64{g.ID})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_ReorderValues_BaskaGrubunDegeriReddedilir(t *testing.T) {
	svc, ctx := newTestService(t)

	g1, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Bir", Kind: KindColor})
	require.NoError(t, err)
	g2, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "İki", Kind: KindColor})
	require.NoError(t, err)

	v1, err := svc.CreateValue(ctx, CreateValueInput{GroupID: g1.ID, Name: "Pembe", SwatchHex: "#F0A6CA"})
	require.NoError(t, err)
	yabanci, err := svc.CreateValue(ctx, CreateValueInput{GroupID: g2.ID, Name: "Mavi", SwatchHex: "#0000FF"})
	require.NoError(t, err)

	err = svc.ReorderValues(ctx, g1.ID, []int64{v1.ID, yabanci.ID})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}
```

- [ ] **Step 4: Test'i çalıştır, kırmızı olduğunu gör**

```bash
cd backend && go vet ./internal/productoption/
```

Beklenen: FAIL — `NewService` tanımlı değil.

- [ ] **Step 5: service.go yaz**

```go
package productoption

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

// hexPattern "#RRGGBB" — kısa form (#FFF) kabul edilmiyor: tek biçim
// tutmak panelde ve müşteri sayfasında sürprizi önler.
var hexPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func (s *Service) CreateGroup(ctx context.Context, in CreateGroupInput) (*Group, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, fmt.Errorf("%w: seçenek grubu adı boş olamaz", errorsx.ErrInvalidInput)
	}
	if !in.Kind.Valid() {
		return nil, fmt.Errorf("%w: geçersiz seçenek tipi %q (color veya text olmalı)",
			errorsx.ErrInvalidInput, in.Kind)
	}

	slug, err := s.uniqueSlug(ctx, product.Slugify(in.Name))
	if err != nil {
		return nil, err
	}
	return s.store.CreateGroup(ctx, in, slug)
}

func (s *Service) uniqueSlug(ctx context.Context, base string) (string, error) {
	candidate := base
	for i := 2; ; i++ {
		exists, err := s.store.GroupSlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

// UpdateGroup adı ve aktifliği günceller. Kind ve slug DEĞİŞMEZ.
func (s *Service) UpdateGroup(ctx context.Context, id int64, in UpdateGroupInput) (*Group, error) {
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: seçenek grubu adı boş olamaz", errorsx.ErrInvalidInput)
		}
		in.Name = &trimmed
	}
	return s.store.UpdateGroup(ctx, id, in)
}

func (s *Service) DeleteGroup(ctx context.Context, id int64) error {
	return s.store.DeleteGroup(ctx, id)
}

func (s *Service) ListGroups(ctx context.Context, onlyActive bool) ([]Group, error) {
	return s.store.ListGroups(ctx, onlyActive)
}

func (s *Service) GetGroup(ctx context.Context, id int64) (*Group, error) {
	return s.store.GetGroup(ctx, id)
}

// GroupProductCount silme öncesi uyarı için. Grup yoksa ErrNotFound —
// count(*) aggregate olduğu için store tek başına ayırt edemez.
func (s *Service) GroupProductCount(ctx context.Context, groupID int64) (int, error) {
	if _, err := s.store.GetGroup(ctx, groupID); err != nil {
		return 0, err
	}
	return s.store.GroupProductCount(ctx, groupID)
}

// CreateValue değeri gruba ekler. Renk grubunda hex zorunlu ve
// "#RRGGBB" formatında; metin grubunda hex temizlenir.
func (s *Service) CreateValue(ctx context.Context, in CreateValueInput) (*Value, error) {
	g, err := s.store.GetGroup(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}

	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, fmt.Errorf("%w: seçenek adı boş olamaz", errorsx.ErrInvalidInput)
	}

	in.SwatchHex, err = normalizeHex(g.Kind, in.SwatchHex)
	if err != nil {
		return nil, err
	}

	return s.store.CreateValue(ctx, in)
}

func (s *Service) UpdateValue(ctx context.Context, id int64, in UpdateValueInput) (*Value, error) {
	cur, err := s.store.GetValue(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: seçenek adı boş olamaz", errorsx.ErrInvalidInput)
		}
		in.Name = &trimmed
	}

	if in.SwatchHex != nil {
		g, err := s.store.GetGroup(ctx, cur.GroupID)
		if err != nil {
			return nil, err
		}
		hex, err := normalizeHex(g.Kind, *in.SwatchHex)
		if err != nil {
			return nil, err
		}
		in.SwatchHex = &hex
	}

	return s.store.UpdateValue(ctx, id, in)
}

func (s *Service) DeleteValue(ctx context.Context, id int64) error {
	return s.store.DeleteValue(ctx, id)
}

// normalizeHex renk grubunda hex'i doğrular, metin grubunda temizler.
func normalizeHex(kind Kind, hex string) (string, error) {
	hex = strings.TrimSpace(hex)

	if kind == KindText {
		return "", nil // metin grubunda renk saklanmaz
	}

	if hex == "" {
		return "", fmt.Errorf("%w: renk seçeneğinde renk kodu zorunlu", errorsx.ErrInvalidInput)
	}
	if !hexPattern.MatchString(hex) {
		return "", fmt.Errorf("%w: geçersiz renk kodu %q (#RRGGBB olmalı, örn. #F0A6CA)",
			errorsx.ErrInvalidInput, hex)
	}
	return strings.ToUpper(hex), nil
}

// ReorderGroups grupları ids sırasına dizer. ids TÜM grupları içermeli —
// kısmi sıralama listede olmayan grupları sessizce 0'a düşürüp sırayı bozar.
func (s *Service) ReorderGroups(ctx context.Context, ids []int64) error {
	mevcut, err := s.store.GroupIDs(ctx)
	if err != nil {
		return err
	}
	if err := dogrulaSiralama(mevcut, ids, "grup"); err != nil {
		return err
	}
	return s.store.ReorderGroups(ctx, ids)
}

// ReorderValues gruptaki değerleri ids sırasına dizer. ids O GRUBUN tüm
// değerlerini içermeli.
func (s *Service) ReorderValues(ctx context.Context, groupID int64, ids []int64) error {
	mevcut, err := s.store.ValueIDsOfGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if err := dogrulaSiralama(mevcut, ids, "seçenek"); err != nil {
		return err
	}
	return s.store.ReorderValues(ctx, ids)
}

// dogrulaSiralama gelen listenin mevcutların TAM bir permütasyonu
// olduğunu doğrular: eksik, fazla, tekrarlı veya yabancı ID reddedilir.
func dogrulaSiralama(mevcut, gelen []int64, ad string) error {
	if len(gelen) != len(mevcut) {
		return fmt.Errorf("%w: sıralama listesi tüm %sları içermeli (%d bekleniyordu, %d geldi)",
			errorsx.ErrInvalidInput, ad, len(mevcut), len(gelen))
	}

	beklenen := make(map[int64]bool, len(mevcut))
	for _, id := range mevcut {
		beklenen[id] = true
	}

	gorulen := make(map[int64]bool, len(gelen))
	for _, id := range gelen {
		if gorulen[id] {
			return fmt.Errorf("%w: sıralama listesinde tekrar eden %s", errorsx.ErrInvalidInput, ad)
		}
		if !beklenen[id] {
			return fmt.Errorf("%w: sıralama listesinde buraya ait olmayan %s", errorsx.ErrInvalidInput, ad)
		}
		gorulen[id] = true
	}
	return nil
}

// SetProductGroups ürünün seçenek gruplarını komple değiştirir.
func (s *Service) SetProductGroups(ctx context.Context, productID int64, links []ProductGroupLink) error {
	gorulen := make(map[int64]bool, len(links))
	for _, l := range links {
		if gorulen[l.GroupID] {
			return fmt.Errorf("%w: aynı seçenek grubu iki kez gönderildi", errorsx.ErrInvalidInput)
		}
		gorulen[l.GroupID] = true

		if _, err := s.store.GetGroup(ctx, l.GroupID); err != nil {
			return fmt.Errorf("%w: seçenek grubu bulunamadı", errorsx.ErrInvalidInput)
		}
	}
	return s.store.SetProductGroups(ctx, productID, links)
}

func (s *Service) GroupsForProduct(ctx context.Context, productID int64, onlyActive bool) ([]ProductGroup, error) {
	return s.store.GroupsForProduct(ctx, productID, onlyActive)
}
```

- [ ] **Step 6: Testleri çalıştır, yeşil olduğunu gör**

```bash
cd backend && go test -p 1 ./internal/productoption/ -v
```

Beklenen: Task 2'nin 4 testi + bu görevin 10 testi = 14 PASS.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/productoption/
git commit -m "feat(productoption): service — hex doğrulama, sıralama, ürün bağlama

Renk grubunda #RRGGBB zorunlu (görünmez nokta çıkmasın), metin grubunda
hex temizlenir. Sıralama listenin TAMAMINI ister — kısmi liste olmayan
kayıtları sessizce 0'a düşürüp sırayı bozardı."
```

---

### Task 4: Admin uçları — grup/değer CRUD + sıralama

**Files:**
- Create: `backend/internal/api/idare/option_handler.go`
- Create: `backend/internal/api/idare/option_view.go`
- Create: `backend/internal/api/idare/option_handler_test.go`
- Modify: `backend/internal/api/idare/router.go`
- Modify: `backend/cmd/server/main.go` (servisi kur ve router'a geçir)

**Interfaces:**
- Consumes: Task 3'ün `productoption.Service`
- Produces: `/api/admin/option-groups*` ve `/api/admin/option-values/:id` uçları;
  `OptionGroupView`/`OptionValueView` JSON şekilleri (Task 7 ve 8 bunları tüketir)

- [ ] **Step 1: option_view.go yaz**

```go
package idare

import "github.com/omerkoc/cicekci/internal/productoption"

// OptionValueView panel seçenek değeri — is_active DAHİL.
type OptionValueView struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SwatchHex string `json:"swatch_hex"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
}

// OptionGroupView panel seçenek grubu, değerleriyle birlikte.
type OptionGroupView struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	Slug      string            `json:"slug"`
	Kind      string            `json:"kind"`
	SortOrder int               `json:"sort_order"`
	IsActive  bool              `json:"is_active"`
	Values    []OptionValueView `json:"values"`
}

func toOptionValueView(v productoption.Value) OptionValueView {
	return OptionValueView{
		ID:        v.ID,
		Name:      v.Name,
		SwatchHex: v.SwatchHex,
		SortOrder: v.SortOrder,
		IsActive:  v.IsActive,
	}
}

func toOptionGroupView(g productoption.Group) OptionGroupView {
	values := make([]OptionValueView, 0, len(g.Values))
	for _, v := range g.Values {
		values = append(values, toOptionValueView(v))
	}
	return OptionGroupView{
		ID:        g.ID,
		Name:      g.Name,
		Slug:      g.Slug,
		Kind:      string(g.Kind),
		SortOrder: g.SortOrder,
		IsActive:  g.IsActive,
		Values:    values,
	}
}

func toOptionGroupViews(list []productoption.Group) []OptionGroupView {
	out := make([]OptionGroupView, 0, len(list))
	for _, g := range list {
		out = append(out, toOptionGroupView(g))
	}
	return out
}
```

- [ ] **Step 2: option_handler.go yaz**

```go
package idare

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/productoption"
)

type optionHandler struct {
	svc *productoption.Service
}

func newOptionHandler(svc *productoption.Service) *optionHandler {
	return &optionHandler{svc: svc}
}

type createGroupRequest struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// updateGroupRequest — kind YOK: tip oluşturulduktan sonra değişmez.
// sort_order da YOK: sıra reorder ucundan değişir.
type updateGroupRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}

type createValueRequest struct {
	Name      string `json:"name"`
	SwatchHex string `json:"swatch_hex"`
}

type updateValueRequest struct {
	Name      *string `json:"name"`
	SwatchHex *string `json:"swatch_hex"`
	IsActive  *bool   `json:"is_active"`
}

type optionReorderRequest struct {
	IDs []int64 `json:"ids"`
}

// list GET /api/admin/option-groups — pasifler dahil, değerleriyle.
func (h *optionHandler) list(c *fiber.Ctx) error {
	list, err := h.svc.ListGroups(c.Context(), false)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toOptionGroupViews(list))
}

// createGroup POST /api/admin/option-groups
func (h *optionHandler) createGroup(c *fiber.Ctx) error {
	var req createGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	g, err := h.svc.CreateGroup(c.Context(), productoption.CreateGroupInput{
		Name: req.Name,
		Kind: productoption.Kind(req.Kind),
	})
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toOptionGroupView(*g))
}

// updateGroup PATCH /api/admin/option-groups/:id
func (h *optionHandler) updateGroup(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req updateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	g, err := h.svc.UpdateGroup(c.Context(), int64(id), productoption.UpdateGroupInput{
		Name:     req.Name,
		IsActive: req.IsActive,
	})
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toOptionGroupView(*g))
}

// deleteGroup DELETE /api/admin/option-groups/:id
func (h *optionHandler) deleteGroup(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	if err := h.svc.DeleteGroup(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// productCount GET /api/admin/option-groups/:id/product-count
// Silme öncesi uyarı için: "Bu grup N üründe kullanılıyor".
func (h *optionHandler) productCount(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	n, err := h.svc.GroupProductCount(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(fiber.Map{"product_count": n})
}

// reorderGroups PUT /api/admin/option-groups/reorder
// Body: {"ids":[3,1,2]} — TÜM grupları içermeli.
func (h *optionHandler) reorderGroups(c *fiber.Ctx) error {
	var req optionReorderRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	if err := h.svc.ReorderGroups(c.Context(), req.IDs); err != nil {
		return api.WriteError(c, err)
	}
	return h.list(c)
}

// createValue POST /api/admin/option-groups/:id/values
func (h *optionHandler) createValue(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req createValueRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	v, err := h.svc.CreateValue(c.Context(), productoption.CreateValueInput{
		GroupID:   int64(id),
		Name:      req.Name,
		SwatchHex: req.SwatchHex,
	})
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toOptionValueView(*v))
}

// reorderValues PUT /api/admin/option-groups/:id/values/reorder
func (h *optionHandler) reorderValues(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req optionReorderRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	if err := h.svc.ReorderValues(c.Context(), int64(id), req.IDs); err != nil {
		return api.WriteError(c, err)
	}
	return h.list(c)
}

// updateValue PATCH /api/admin/option-values/:id
func (h *optionHandler) updateValue(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req updateValueRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	v, err := h.svc.UpdateValue(c.Context(), int64(id), productoption.UpdateValueInput{
		Name:      req.Name,
		SwatchHex: req.SwatchHex,
		IsActive:  req.IsActive,
	})
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toOptionValueView(*v))
}

// deleteValue DELETE /api/admin/option-values/:id
func (h *optionHandler) deleteValue(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	if err := h.svc.DeleteValue(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
```

- [ ] **Step 3: Router'a kaydet**

`backend/internal/api/idare/router.go` — `Register` fonksiyonunun imzasına
`optSvc *productoption.Service` parametresi eklenir (mevcut servis
parametrelerinin yanına), gövdesinde handler kurulur:

```go
	oph := newOptionHandler(optSvc)
```

Route'lar `protected` grubuna, slider route'larının altına:

```go
	// reorder ":id" kalıplarından ÖNCE — Fiber sıralı eşleştirir.
	protected.Put("/option-groups/reorder", oph.reorderGroups)
	protected.Get("/option-groups", oph.list)
	protected.Post("/option-groups", oph.createGroup)
	protected.Patch("/option-groups/:id", oph.updateGroup)
	protected.Delete("/option-groups/:id", oph.deleteGroup)
	protected.Get("/option-groups/:id/product-count", oph.productCount)
	protected.Post("/option-groups/:id/values", oph.createValue)
	protected.Put("/option-groups/:id/values/reorder", oph.reorderValues)
	protected.Patch("/option-values/:id", oph.updateValue)
	protected.Delete("/option-values/:id", oph.deleteValue)
```

`backend/cmd/server/main.go`'da servis kurulur ve `idare.Register` çağrısına
eklenir:

```go
	optSvc := productoption.NewService(productoption.NewStore(pool))
```

- [ ] **Step 4: Failing test yaz**

`backend/internal/api/idare/option_handler_test.go` — mevcut
`product_handler_test.go`'daki test app kurulum desenini birebir takip et
(o dosyayı önce oku; `newTestApp` benzeri yardımcıyı aynı şekilde kur).

```go
package idare

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Route sırası regresyonu: "/option-groups/reorder" isteği ":id"
// route'una düşerse "reorder" geçersiz id sayılır ve 400 döner.
// Bu test o sıralamayı kilitler.
func TestOptionGroups_ReorderRoute_IDRoutunaDusmez(t *testing.T) {
	app, token := newTestAppWithAuth(t)

	// Önce iki grup oluştur
	g1 := createTestGroup(t, app, token, "Ambalaj", "color")
	g2 := createTestGroup(t, app, token, "Kurdele", "color")

	req := authedRequest(t, token, http.MethodPut, "/api/admin/option-groups/reorder",
		map[string]any{"ids": []int64{g2, g1}})

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"reorder ':id' route'una düşmemeli")
}

func TestOptionGroups_YetkisizErisimReddedilir(t *testing.T) {
	app, _ := newTestAppWithAuth(t)

	req := jsonRequest(t, http.MethodGet, "/api/admin/option-groups", nil)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestOptionValues_GecersizHex400Doner(t *testing.T) {
	app, token := newTestAppWithAuth(t)

	gid := createTestGroup(t, app, token, "Ambalaj", "color")

	req := authedRequest(t, token, http.MethodPost,
		"/api/admin/option-groups/"+itoa(gid)+"/values",
		map[string]any{"name": "Pembe", "swatch_hex": "F0A6CA"})

	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
```

> **Not:** `newTestAppWithAuth`, `authedRequest`, `jsonRequest`,
> `createTestGroup`, `itoa` yardımcıları mevcut
> `product_handler_test.go`/`order_handler_test.go`'da hangi isimlerle
> varsa onları kullan. Yoksa aynı dosyada tanımla — yeni bir test altyapısı
> KURMA, var olanı takip et.

- [ ] **Step 5: Test'i çalıştır, kırmızı olduğunu gör**

```bash
cd backend && go test -p 1 ./internal/api/idare/ -run "Option" -v
```

Beklenen: FAIL (derleme hatası veya 404).

- [ ] **Step 6: Testler yeşil olana kadar router/main bağlantısını tamamla**

```bash
cd backend && go build ./... && go test -p 1 ./internal/api/idare/ -run "Option" -v
```

Beklenen: 3 test PASS.

- [ ] **Step 7: Uçları canlı sunucuya karşı dene**

```bash
cd backend && go run ./cmd/seed -username=optiontest -password=Test12345
cd backend && go run ./cmd/server &
sleep 6
cd /tmp && curl -s -c c.txt -X POST http://localhost:8080/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"optiontest","password":"Test12345"}' -o /dev/null -w "login %{http_code}\n"
curl -s -b c.txt http://localhost:8080/api/admin/option-groups | head -c 400
```

Beklenen: `login 200`, ardından seed'deki üç grup ve değerleri.

- [ ] **Step 8: Temizlik + commit**

```bash
pkill -f "cmd/server"
docker compose exec -T postgres psql -U cicekci -d cicekci -c \
  "DELETE FROM admin_users WHERE username='optiontest';"
git add backend/internal/api/idare/ backend/cmd/server/main.go
git commit -m "feat(api): seçenek grubu/değer admin uçları

reorder route'u ':id' kalıplarından ÖNCE kaydedildi (Fiber sıralı
eşleştirir); regresyon testiyle kilitlendi."
```

---

### Task 5: Ürün ↔ seçenek grubu bağlantısı (admin + public ürün uçları)

**Files:**
- Modify: `backend/internal/api/idare/product_handler.go`
- Modify: `backend/internal/api/idare/product_view.go`
- Modify: `backend/internal/api/app/product_handler.go`
- Modify: `backend/internal/api/app/product_view.go`
- Modify: `backend/internal/api/idare/router.go`, `backend/internal/api/app/router.go`
- Test: `backend/internal/api/idare/product_handler_test.go`

**Interfaces:**
- Consumes: Task 3'ün `SetProductGroups`, `GroupsForProduct`
- Produces:
  - Admin `POST/PATCH /api/admin/products` gövdesinde
    `option_groups: [{group_id, is_required}]`
  - Admin `GET /api/admin/products/:id` yanıtında `option_groups`
  - Public `GET /api/products/:slug` yanıtında `option_groups`
    (yalnızca aktif grup + aktif değerler) — Task 8 bunu tüketir

- [ ] **Step 1: Admin ürün handler'ına option_groups ekle**

`product_handler.go`'daki create/update request struct'larına:

```go
	// OptionGroups nil ise gruplar DEĞİŞMEZ; boş dizi hepsini kaldırır
	// (CategoryIDs ile aynı PATCH semantiği).
	OptionGroups []optionGroupLinkRequest `json:"option_groups"`
```

Aynı dosyaya:

```go
type optionGroupLinkRequest struct {
	GroupID    int64 `json:"group_id"`
	IsRequired bool  `json:"is_required"`
}

func toGroupLinks(reqs []optionGroupLinkRequest) []productoption.ProductGroupLink {
	out := make([]productoption.ProductGroupLink, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, productoption.ProductGroupLink{
			GroupID: r.GroupID, IsRequired: r.IsRequired,
		})
	}
	return out
}
```

`productHandler` struct'ına `optSvc *productoption.Service` alanı eklenir.
`create` ve `update` handler'larında ürün kaydedildikten SONRA:

```go
	// Gruplar ürün kaydedildikten sonra bağlanır — ürün id'si gerekiyor.
	// nil ise dokunulmaz (PATCH semantiği).
	if req.OptionGroups != nil {
		if err := h.optSvc.SetProductGroups(c.Context(), p.ID, toGroupLinks(req.OptionGroups)); err != nil {
			return api.WriteError(c, err)
		}
	}
```

- [ ] **Step 2: Admin ürün view'ına option_groups ekle**

`product_view.go`'da `ProductView` struct'ına:

```go
	OptionGroups []ProductOptionGroupView `json:"option_groups"`
```

Aynı dosyaya:

```go
// ProductOptionGroupView ürüne açık bir seçenek grubu — is_required ile.
type ProductOptionGroupView struct {
	ID         int64             `json:"id"`
	Name       string            `json:"name"`
	Kind       string            `json:"kind"`
	IsRequired bool              `json:"is_required"`
	IsActive   bool              `json:"is_active"`
	Values     []OptionValueView `json:"values"`
}

func toProductOptionGroupViews(list []productoption.ProductGroup) []ProductOptionGroupView {
	out := make([]ProductOptionGroupView, 0, len(list))
	for _, g := range list {
		values := make([]OptionValueView, 0, len(g.Values))
		for _, v := range g.Values {
			values = append(values, toOptionValueView(v))
		}
		out = append(out, ProductOptionGroupView{
			ID:         g.ID,
			Name:       g.Name,
			Kind:       string(g.Kind),
			IsRequired: g.IsRequired,
			IsActive:   g.IsActive,
			Values:     values,
		})
	}
	return out
}
```

Admin `get` handler'ında ürün okunduktan sonra
`h.optSvc.GroupsForProduct(ctx, p.ID, false)` çağrılıp view'a konur
(panelde pasif gruplar da görünmeli ki esnaf durumu anlasın).

- [ ] **Step 3: Public ürün view'ına option_groups ekle**

`backend/internal/api/app/product_view.go`'ya, admin'dekinin **yalnızca
aktif** olan karşılığı:

```go
// PublicOptionValueView müşteriye görünen seçenek değeri.
// is_active YOK — pasif değerler bu uçtan zaten hiç gelmez.
type PublicOptionValueView struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SwatchHex string `json:"swatch_hex"`
}

// PublicOptionGroupView müşteriye görünen seçenek grubu.
type PublicOptionGroupView struct {
	ID         int64                   `json:"id"`
	Name       string                  `json:"name"`
	Kind       string                  `json:"kind"`
	IsRequired bool                    `json:"is_required"`
	Values     []PublicOptionValueView `json:"values"`
}

func toPublicOptionGroupViews(list []productoption.ProductGroup) []PublicOptionGroupView {
	out := make([]PublicOptionGroupView, 0, len(list))
	for _, g := range list {
		// Değeri kalmamış grup müşteriye gösterilmez — seçenek sunmayan
		// bir başlık kafa karıştırır.
		if len(g.Values) == 0 {
			continue
		}
		values := make([]PublicOptionValueView, 0, len(g.Values))
		for _, v := range g.Values {
			values = append(values, PublicOptionValueView{
				ID: v.ID, Name: v.Name, SwatchHex: v.SwatchHex,
			})
		}
		out = append(out, PublicOptionGroupView{
			ID:         g.ID,
			Name:       g.Name,
			Kind:       string(g.Kind),
			IsRequired: g.IsRequired,
			Values:     values,
		})
	}
	return out
}
```

Public `ProductDetailView`'a `OptionGroups []PublicOptionGroupView
\`json:"option_groups"\`` eklenir; handler `GroupsForProduct(ctx, p.ID, true)`
çağırır.

- [ ] **Step 4: Failing test yaz**

`backend/internal/api/idare/product_handler_test.go` sonuna:

```go
// option_groups nil gönderilirse mevcut bağlar KORUNUR (PATCH semantiği).
// Boş dizi gönderilirse hepsi kaldırılır.
func TestProduct_OptionGroups_PatchSemantigi(t *testing.T) {
	app, token := newTestAppWithAuth(t)

	gid := createTestGroup(t, app, token, "Ambalaj", "color")
	pid := createTestProduct(t, app, token, map[string]any{
		"name":  "Buket",
		"price": "100.00",
		"option_groups": []map[string]any{
			{"group_id": gid, "is_required": true},
		},
	})

	// option_groups GÖNDERİLMEDEN isim güncelle → bağ korunmalı
	patch := authedRequest(t, token, http.MethodPatch, "/api/admin/products/"+itoa(pid),
		map[string]any{"name": "Yeni Buket"})
	resp, err := app.Test(patch, -1)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	urun := getProduct(t, app, token, pid)
	require.Len(t, urun.OptionGroups, 1, "option_groups gönderilmediyse bağ korunmalı")
	assert.True(t, urun.OptionGroups[0].IsRequired)

	// Boş dizi → hepsi kalkmalı
	patch2 := authedRequest(t, token, http.MethodPatch, "/api/admin/products/"+itoa(pid),
		map[string]any{"option_groups": []any{}})
	resp2, err := app.Test(patch2, -1)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	urun2 := getProduct(t, app, token, pid)
	assert.Empty(t, urun2.OptionGroups, "boş dizi tüm bağları kaldırmalı")
}
```

> `createTestProduct`, `getProduct` yardımcıları dosyada yoksa aynı
> dosyada tanımla; mevcut yardımcı varsa onu kullan.

- [ ] **Step 5: Test'i çalıştır, kırmızı olduğunu gör**

```bash
cd backend && go test -p 1 ./internal/api/idare/ -run "OptionGroups_Patch" -v
```

Beklenen: FAIL.

- [ ] **Step 6: Yeşil olana kadar tamamla, sonra tüm paketi çalıştır**

```bash
cd backend && go test -p 1 ./internal/api/idare/ ./internal/api/app/ -v 2>&1 | tail -20
```

Beklenen: hepsi PASS.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/api/
git commit -m "feat(api): ürün ↔ seçenek grubu bağlantısı

option_groups nil ise bağlar DEĞİŞMEZ, boş dizi hepsini kaldırır —
category_ids ile aynı PATCH semantiği. Public uç yalnızca aktif grup ve
aktif değerleri döner; değeri kalmamış grup hiç gösterilmez."
```

---

### Task 6: Sipariş akışı — seçim doğrulama ve kopyalama

**Files:**
- Modify: `backend/internal/order/model.go`
- Modify: `backend/internal/order/service.go`
- Modify: `backend/internal/order/store.go`
- Create: `backend/internal/productoption/order_reader.go`
- Create: `backend/internal/order/options_test.go`
- Modify: `backend/internal/api/app/order_handler.go`, `order_view.go`
- Modify: `backend/internal/api/idare/order_view.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes: Task 3'ün servisi, Task 1'in `order_item_options` tablosu
- Produces:
  - `order.CreateItem.OptionValueIDs []int64`
  - `order.OrderItem.Options []OrderItemOption`
  - `type OrderItemOption struct { GroupName, ValueName, SwatchHex string; SortOrder int }`
  - `type OptionReader interface { ResolveForProduct(ctx, productID int64, valueIDs []int64) ([]OrderItemOption, error) }`
  - `productoption.(*Service).ResolveForProduct` — `OptionReader`'ı karşılar
  - Public sipariş gövdesi: `items[].option_value_ids`
  - Sipariş yanıtlarında `items[].options`

- [ ] **Step 1: order/model.go'ya tipleri ekle**

```go
// OrderItemOption sipariş anındaki seçim — KOPYA. Gruba/değere referans
// tutulmaz: esnaf sonradan "Pembe"yi silerse veya rengini değiştirirse
// eski siparişin ne olduğu bilgisi bozulmamalı (ProductName ile aynı kural).
type OrderItemOption struct {
	GroupName string `json:"group_name"`
	ValueName string `json:"value_name"`
	SwatchHex string `json:"swatch_hex"`
	SortOrder int    `json:"sort_order"`
}
```

`OrderItem` struct'ına:

```go
	Options []OrderItemOption `json:"options"`
```

`CreateItem` struct'ına:

```go
	// OptionValueIDs müşterinin seçtiği değerlerin id'leri. YALNIZCA ID —
	// isim ve renk sunucuda DB'den okunur, tarayıcıdan gelene güvenilmez
	// (fiyatla aynı kural).
	OptionValueIDs []int64
```

`NewOrderItem` struct'ına:

```go
	Options []OrderItemOption
```

- [ ] **Step 2: OptionReader arayüzünü order/service.go'ya ekle**

`ProductReader`'ın hemen altına:

```go
// OptionReader order paketinin seçenek doğrulaması için ihtiyaç duyduğu
// tek şey. Dar arayüz: order paketi productoption'ın tamamına bağlanmasın
// (ProductReader ile aynı gerekçe).
//
// Dönüş tipi OrderItemOption — order'ın KENDİ tipi. Böylece productoption
// order'ı import etmek zorunda kalmaz: bağımlılık tek yönlü kalır
// (main.go somut servisi arayüze bağlar).
type OptionReader interface {
	// ResolveForProduct valueIDs'i doğrular ve isimleriyle döner.
	// Hata verir: değer yok, pasif, bu ürüne kapalı gruba ait, aynı
	// gruptan birden çok değer, zorunlu grup eksik.
	ResolveForProduct(ctx context.Context, productID int64, valueIDs []int64) ([]OrderItemOption, error)
}
```

`Service` struct'ına `opt OptionReader` alanı, `NewService` imzasına
`opt OptionReader` parametresi eklenir (`prod ProductReader`'ın yanına).

- [ ] **Step 3: Create içinde çağır**

`service.go`'daki kalem döngüsünde, `items = append(...)` satırından ÖNCE:

```go
		// Seçimler DB'den doğrulanır — tarayıcıdan gelen isim/renk
		// kullanılmaz. Ürüne kapalı grup, pasif değer, aynı gruptan iki
		// değer veya eksik zorunlu grup burada reddedilir.
		opts, err := s.opt.ResolveForProduct(ctx, p.ID, ci.OptionValueIDs)
		if err != nil {
			return nil, "", err
		}
```

ve `items = append(items, NewOrderItem{...})` çağrısına `Options: opts,`
eklenir.

- [ ] **Step 4: Store'da yaz ve oku**

`store.go`'daki `createOnce` kalem döngüsünde `INSERT INTO order_items`
`RETURNING id` alacak şekilde değiştirilir ve hemen ardından:

```go
		var itemID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO order_items (order_id, product_id, product_name, price_at_order, quantity)
			VALUES ($1,$2,$3,$4,$5) RETURNING id`,
			id, it.ProductID, it.ProductName, it.PriceAtOrder, it.Quantity).Scan(&itemID)
		if err != nil {
			return nil, err
		}

		for _, o := range it.Options {
			if _, err := tx.Exec(ctx, `
				INSERT INTO order_item_options
					(order_item_id, group_name, value_name, swatch_hex, sort_order)
				VALUES ($1,$2,$3,$4,$5)`,
				itemID, o.GroupName, o.ValueName, o.SwatchHex, o.SortOrder); err != nil {
				return nil, err
			}
		}
```

Okuma tarafında `itemsOf` ve `itemsOfMany`, kalemler toplandıktan SONRA
seçimleri **tek batch sorguyla** çeker:

```go
// optionsOfMany kalem seçimlerini tek sorguda çeker. Kalem başına ayrı
// sorgu N+1 olurdu — Store.List'te aynı ders alınmıştı.
func (s *Store) optionsOfMany(ctx context.Context, itemIDs []int64) (map[int64][]OrderItemOption, error) {
	if len(itemIDs) == 0 {
		return map[int64][]OrderItemOption{}, nil
	}

	rows, err := s.pool.Query(ctx, `
		SELECT order_item_id, group_name, value_name, swatch_hex, sort_order
		FROM order_item_options
		WHERE order_item_id = ANY($1)
		ORDER BY order_item_id, sort_order, id`, itemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64][]OrderItemOption, len(itemIDs))
	for rows.Next() {
		var itemID int64
		var o OrderItemOption
		if err := rows.Scan(&itemID, &o.GroupName, &o.ValueName, &o.SwatchHex, &o.SortOrder); err != nil {
			return nil, err
		}
		out[itemID] = append(out[itemID], o)
	}
	return out, rows.Err()
}

// attachOptions kalemlere seçimlerini bağlar. Seçimi olmayan kalem boş
// slice alır (nil değil) — JSON'da null yerine [] çıksın.
func (s *Store) attachOptions(ctx context.Context, items []OrderItem) error {
	ids := make([]int64, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}

	byItem, err := s.optionsOfMany(ctx, ids)
	if err != nil {
		return err
	}

	for i := range items {
		if o, ok := byItem[items[i].ID]; ok {
			items[i].Options = o
		} else {
			items[i].Options = []OrderItemOption{}
		}
	}
	return nil
}
```

`itemsOf` sonunda `s.attachOptions(ctx, items)` çağrılır; `itemsOfMany`
sonunda tüm kalemler düz listeye toplanıp bir kez `attachOptions` çağrılır.

- [ ] **Step 5: ResolveForProduct'ı productoption'a yaz**

`backend/internal/productoption/order_reader.go`:

```go
package productoption

import (
	"context"
	"fmt"

	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// ResolveForProduct order.OptionReader'ı karşılar: müşterinin gönderdiği
// değer id'lerini doğrular ve isimleriyle döner.
//
// Reddedilen durumlar:
//   - değer yok veya pasif
//   - değerin grubu bu ürüne açık değil
//   - aynı gruptan birden fazla değer
//   - ürünün zorunlu grubu doldurulmamış
func (s *Service) ResolveForProduct(ctx context.Context, productID int64, valueIDs []int64) ([]order.OrderItemOption, error) {
	gruplar, err := s.store.GroupsForProduct(ctx, productID, true)
	if err != nil {
		return nil, err
	}

	// Ürüne açık aktif değerlerin dizini: valueID → (grup, değer)
	type kayit struct {
		grup  ProductGroup
		deger Value
	}
	dizin := make(map[int64]kayit)
	for _, g := range gruplar {
		for _, v := range g.Values {
			dizin[v.ID] = kayit{grup: g, deger: v}
		}
	}

	out := make([]order.OrderItemOption, 0, len(valueIDs))
	gorulenGrup := make(map[int64]bool, len(valueIDs))

	for _, vid := range valueIDs {
		k, ok := dizin[vid]
		if !ok {
			return nil, fmt.Errorf("%w: geçersiz veya artık sunulmayan seçenek", errorsx.ErrInvalidInput)
		}
		if gorulenGrup[k.grup.ID] {
			return nil, fmt.Errorf("%w: %q için birden fazla seçim gönderildi",
				errorsx.ErrInvalidInput, k.grup.Name)
		}
		gorulenGrup[k.grup.ID] = true

		out = append(out, order.OrderItemOption{
			GroupName: k.grup.Name,
			ValueName: k.deger.Name,
			SwatchHex: k.deger.SwatchHex,
			SortOrder: k.grup.SortOrder,
		})
	}

	// Zorunlu gruplar eksiksiz mi
	for _, g := range gruplar {
		if g.IsRequired && !gorulenGrup[g.ID] {
			return nil, fmt.Errorf("%w: %q seçilmeli", errorsx.ErrInvalidInput, g.Name)
		}
	}

	return out, nil
}
```

`main.go`'da `order.NewService(...)` çağrısına `optSvc` eklenir.

- [ ] **Step 6: Failing test yaz**

`backend/internal/order/options_test.go` — mevcut `service_test.go`'daki
kurulum desenini takip et (o dosyayı önce oku, `newTestService` benzeri
yardımcıyı aynı şekilde kullan; `OptionReader` için gerçek
`productoption.Service` geçirilir, sahte değil — doğrulama gerçek DB'ye
karşı kanıtlanmalı).

```go
package order_test

// Bu testler ayrı pakette (order_test) çünkü productoption'ı import
// ediyorlar ve productoption da order'ı import ediyor — aynı pakette
// olsalardı import döngüsü olurdu.

import (
	"testing"

	"github.com/omerkoc/cicekci/internal/productoption"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestCreate_ZorunluGrupEksikReddedilir(t *testing.T) {
	env := newOptionTestEnv(t) // Ambalaj Rengi is_required=true

	_, _, err := env.svc.Create(env.ctx, gecerliSiparis(env.productID, nil), "1.2.3.4", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Ambalaj Rengi")
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

	list, err := env.svc.List(env.ctx, "", 100, 0)

	require.NoError(t, err)
	require.Len(t, list, 2)
	for _, o := range list {
		require.Len(t, o.Items[0].Options, 1, "her kalem kendi seçimini almalı")
	}
}
```

> `newOptionTestEnv` ve `gecerliSiparis` yardımcılarını aynı dosyada
> tanımla: `database.NewTestDB` ile pool kur, `product`, `productoption`
> ve `order` servislerini gerçek store'larla oluştur, bir ürün + "Ambalaj
> Rengi" (is_required=true, Pembe #F0A6CA + Beyaz #FFFFFF) + ikinci bir
> ürüne bağlı ayrı grup oluştur. `gecerliSiparis` mevcut
> `service_test.go`'daki geçerli `CreateInput`'u üretir, `OptionValueIDs`
> parametreyle set edilir.

- [ ] **Step 7: Test'i çalıştır, kırmızı olduğunu gör**

```bash
cd backend && go test -p 1 ./internal/order/ -run "Secim|Zorunlu|Kapali|Pasif|AyniGrup|HerKalem" -v
```

Beklenen: FAIL.

- [ ] **Step 8: Handler'lara option_value_ids ekle**

`backend/internal/api/app/order_handler.go`'daki kalem request struct'ına:

```go
	OptionValueIDs []int64 `json:"option_value_ids"`
```

ve `order.CreateItem`'a geçirilir. `order_view.go` (app ve idare) kalem
view'larına:

```go
	Options []OrderItemOptionView `json:"options"`
```

```go
// OrderItemOptionView sipariş kalemindeki seçim — sipariş anındaki
// kopyadan gelir, güncel gruba bakılmaz.
type OrderItemOptionView struct {
	GroupName string `json:"group_name"`
	ValueName string `json:"value_name"`
	SwatchHex string `json:"swatch_hex"`
}
```

- [ ] **Step 9: Testler yeşil olana kadar tamamla**

```bash
cd backend && go build ./... && go test -p 1 ./internal/order/ ./internal/productoption/ ./internal/api/... -v 2>&1 | tail -25
```

Beklenen: hepsi PASS.

- [ ] **Step 10: Tüm suite + commit**

```bash
cd /Users/omerkoc/GolandProjects/cicekci && make test 2>&1 | tail -15
```

Beklenen: yalnızca bilinen 2 PayTR config testi kırmızı.

```bash
git add backend/
git commit -m "feat(order): sipariş kalemine seçenek seçimi

Sunucu tarayıcıdan YALNIZCA value_id alır; isim ve renk DB'den okunur
(fiyattaki kuralın aynısı). Ürüne kapalı grup, pasif değer, aynı gruptan
iki değer ve eksik zorunlu grup reddedilir.

Seçim siparişe KOPYALANIR — değer sonradan silinse de eski sipariş
bozulmuyor (regresyon testi var). Seçimler tek batch sorguyla okunuyor."
```

---

### Task 7: Admin paneli — Seçenek Yönetimi sayfası

**Files:**
- Create: `frontend/idare/src/model/option.ts`
- Create: `frontend/idare/src/composables/useOptions.ts`
- Create: `frontend/idare/src/pages/secenekler.vue`
- Modify: `frontend/idare/src/navigation/vertical/index.ts` (veya projedeki
  navigasyon dosyası — `kategoriler` girdisinin bulunduğu dosyayı bul)

**Interfaces:**
- Consumes: Task 4'ün `/api/admin/option-groups*` uçları
- Produces: `OptionGroup`, `OptionValue` tipleri; `useOptions()` composable —
  Task 8 ürün formunda `useOptions().list` kullanır

- [ ] **Step 1: model/option.ts yaz**

```ts
export type OptionKind = 'color' | 'text'

export const KIND_LABELS: Record<OptionKind, string> = {
  color: 'Renk',
  text: 'Metin',
}

export interface OptionValue {
  id: number
  name: string
  /** kind='text' grupta boş. */
  swatch_hex: string
  sort_order: number
  is_active: boolean
}

// sort_order YOK: yeni grup sona eklenir, sıra reorder ucundan değişir.
export interface OptionGroupCreate {
  name: string
  kind: OptionKind
}

// kind YOK: tip oluşturulduktan sonra değişmez.
export interface OptionGroupUpdate {
  name?: string
  is_active?: boolean
}

export interface OptionGroup {
  id: number
  name: string
  slug: string
  kind: OptionKind
  sort_order: number
  is_active: boolean
  values: OptionValue[]
}

export interface OptionValueCreate {
  name: string
  swatch_hex: string
}

export interface OptionValueUpdate {
  name?: string
  swatch_hex?: string
  is_active?: boolean
}
```

- [ ] **Step 2: composables/useOptions.ts yaz**

```ts
import ApiService from '@/services/ApiService'
import type {
  OptionGroup, OptionGroupCreate, OptionGroupUpdate,
  OptionValue, OptionValueCreate, OptionValueUpdate,
} from '@/model/option'

export function useOptions() {
  const list = () => ApiService.get<OptionGroup[]>('admin/option-groups')

  const createGroup = (data: OptionGroupCreate) =>
    ApiService.post<OptionGroup>('admin/option-groups', data)

  const updateGroup = (id: number, data: OptionGroupUpdate) =>
    ApiService.patch<OptionGroup>(`admin/option-groups/${id}`, data)

  const removeGroup = (id: number) =>
    ApiService.delete<void>(`admin/option-groups/${id}`)

  /** Silme öncesi uyarı: "Bu grup N üründe kullanılıyor". */
  const groupProductCount = (id: number) =>
    ApiService.get<{ product_count: number }>(`admin/option-groups/${id}/product-count`)

  /** ids TÜM grupları içermeli — backend eksik listeyi reddediyor. */
  const reorderGroups = (ids: number[]) =>
    ApiService.put<OptionGroup[]>('admin/option-groups/reorder', { ids })

  const createValue = (groupId: number, data: OptionValueCreate) =>
    ApiService.post<OptionValue>(`admin/option-groups/${groupId}/values`, data)

  const updateValue = (id: number, data: OptionValueUpdate) =>
    ApiService.patch<OptionValue>(`admin/option-values/${id}`, data)

  const removeValue = (id: number) =>
    ApiService.delete<void>(`admin/option-values/${id}`)

  /** ids O GRUBUN tüm değerlerini içermeli. */
  const reorderValues = (groupId: number, ids: number[]) =>
    ApiService.put<OptionGroup[]>(`admin/option-groups/${groupId}/values/reorder`, { ids })

  return {
    list, createGroup, updateGroup, removeGroup, groupProductCount, reorderGroups,
    createValue, updateValue, removeValue, reorderValues,
  }
}
```

- [ ] **Step 3: pages/secenekler.vue yaz**

`kategoriler.vue`'yu şablon olarak kullan (aynı `VDataTable` + `VDialog` +
`ConfirmPopup`/`ErrorPopup`/`SuccessToast` deseni). Sayfa iki sütun:

Sol sütun — gruplar tablosu:
- Kolonlar: Ad, Tip (`KIND_LABELS` rozeti), Seçenek sayısı, Aktif (switch),
  Sıra (▲▼ butonları — `kategoriler.vue`'daki `move()` deseninin aynısı,
  `reorderGroups` çağırır), İşlemler (düzenle/sil)
- Satıra tıklayınca sağ sütun o grubu gösterir (`seciliGrupId` ref'i)
- Silme onayı, ürün sayısıyla:
  ```
  n > 0
    ? `"${g.name}" grubu ${n} üründe kullanılıyor. Silerseniz bu ürünlerde artık sorulmayacak (geçmiş siparişler etkilenmez). Devam edilsin mi?`
    : `"${g.name}" grubu silinecek. Devam edilsin mi?`
  ```

Sağ sütun — seçili grubun değerleri:
- Kolonlar: Renk (yuvarlak nokta, `:style="{ background: v.swatch_hex }"` —
  yalnızca `kind === 'color'`), Ad, Aktif (switch), Sıra (▲▼ →
  `reorderValues`), Sil
- "Yeni Seçenek" formu: ad + (kind='color' ise) `<input type="color">` ve
  hex metin kutusu yan yana, birbirine bağlı `v-model`
- Grup seçili değilse: "Soldan bir grup seçin" boş durumu

Grup oluşturma diyaloğunda **tip radio'su yalnızca yeni kayıtta** görünür,
düzenlemede salt okunur — `kategoriler.vue`'daki `axis` alanının birebir
aynısı:

```vue
<VRadioGroup v-if="!editing" v-model="form.kind" label="Tip" inline>
  <VRadio label="Renk" value="color" />
  <VRadio label="Metin" value="text" />
</VRadioGroup>
<VTextField
  v-else
  :model-value="KIND_LABELS[form.kind]"
  label="Tip"
  readonly disabled
  hint="Tip değiştirilemez"
  persistent-hint
/>
```

Sayfa başına bilgi kutusu:

```vue
<VAlert type="info" variant="tonal" class="mb-6">
  Buradaki gruplar ürün formunda "Özelleştirme" bölümünde çıkar; her ürün
  için ayrı ayrı açılır. Pasif grup veya seçenek müşteriye gösterilmez,
  geçmiş siparişler etkilenmez. Sırayı değiştirmek için yukarı/aşağı
  oklarını kullanın.
</VAlert>
```

- [ ] **Step 4: Navigasyona ekle**

Navigasyon dosyasında `Kategori Yönetimi` girdisini bul, hemen altına:

```ts
{ title: 'Seçenek Yönetimi', to: 'secenekler', icon: { icon: 'tabler-palette' } },
```

- [ ] **Step 5: Tip kontrolü ve lint**

```bash
cd frontend/idare
npx vue-tsc --noEmit -p tsconfig.json 2>&1 | grep -v "@core\|@layouts"
npx eslint src/pages/secenekler.vue src/composables/useOptions.ts src/model/option.ts
```

Beklenen: her ikisi de çıktısız (temiz).

- [ ] **Step 6: Tarayıcıda doğrula**

```bash
cd backend && go run ./cmd/seed -username=paneltest -password=Test12345
cd backend && go run ./cmd/server &
cd frontend/idare && pnpm dev &
sleep 20
```

Playwright ile: giriş → `/secenekler` → yeni grup ("Çiçek Rengi", Renk) →
gruba değer ekle (Kırmızı #D93A34) → ▲▼ ile sırala → sayfayı yenile,
sıranın kalıcı olduğunu gör → grubu sil (onay diyaloğu ürün sayısını
gösteriyor) → konsol hatası yok.

Ekran görüntüsü al ve gözle kontrol et.

- [ ] **Step 7: Temizlik + commit**

```bash
pkill -f "cmd/server"; pkill -f vite
docker compose exec -T postgres psql -U cicekci -d cicekci -c \
  "DELETE FROM admin_users WHERE username='paneltest';"
git add frontend/idare/
git commit -m "feat(idare): Seçenek Yönetimi sayfası

Grup ve değer CRUD + ok butonlarıyla sıralama. Tip (renk/metin) yalnızca
oluştururken seçilir; sonradan değişmez (kategorideki axis kuralı).
Silme onayı grubu kaç ürünün kullandığını gösteriyor."
```

---

### Task 8: Admin paneli — ürün formuna "Özelleştirme" bölümü + sipariş detayında seçimler

**Files:**
- Create: `frontend/idare/src/components/ProductOptionPicker.vue`
- Modify: `frontend/idare/src/pages/urunler/[id].vue` (ürün formu — kesin
  yolu `ls frontend/idare/src/pages/urunler/` ile doğrula)
- Modify: `frontend/idare/src/model/product.ts`
- Modify: `frontend/idare/src/composables/useProducts.ts`
- Modify: `frontend/idare/src/model/order.ts`
- Modify: `frontend/idare/src/pages/siparisler/[id].vue`

**Interfaces:**
- Consumes: Task 5'in `option_groups` alanı, Task 6'nın `items[].options`
- Produces: ürün formunda seçilen grupların `POST/PATCH /admin/products`
  gövdesine `option_groups` olarak gitmesi

- [ ] **Step 1: model/product.ts'e tipleri ekle**

```ts
import type { OptionKind, OptionValue } from '@/model/option'

/** Ürüne açık seçenek grubu — panel görünümü, pasifler dahil. */
export interface ProductOptionGroup {
  id: number
  name: string
  kind: OptionKind
  is_required: boolean
  is_active: boolean
  values: OptionValue[]
}

/** Ürün formundan giden bağ. */
export interface ProductOptionGroupLink {
  group_id: number
  is_required: boolean
}
```

`Product` arayüzüne `option_groups: ProductOptionGroup[]`,
`ProductCreate`/`ProductUpdate` arayüzlerine
`option_groups?: ProductOptionGroupLink[]` eklenir.

> **Dikkat:** `option_groups` **gönderilmezse** backend mevcut bağları
> korur; **boş dizi** gönderilirse hepsini kaldırır. Form her kaydetmede
> mevcut seçimi (boş olsa bile) gönderdiği için bu doğru davranış.

- [ ] **Step 2: ProductOptionPicker.vue yaz**

```vue
<script setup lang="ts">
import { useOptions } from '@/composables/useOptions'
import type { OptionGroup } from '@/model/option'
import type { ProductOptionGroupLink } from '@/model/product'
import { ErrorPopup } from '@/utils/Popup'

const props = defineProps<{
  /** Ürünün mevcut bağları; yeni üründe boş dizi. */
  modelValue: ProductOptionGroupLink[]
}>()

const emit = defineEmits<{
  'update:modelValue': [ProductOptionGroupLink[]]
}>()

const api = useOptions()

const gruplar = ref<OptionGroup[]>([])
const loading = ref(false)

// Yalnızca aktif ve en az bir değeri olan gruplar seçilebilir — değeri
// olmayan grubu ürüne açmak müşteriye boş bir başlık gösterirdi.
const secilebilir = computed(() =>
  gruplar.value.filter(g => g.is_active && g.values.some(v => v.is_active)))

const load = async () => {
  loading.value = true

  const [err, data] = await api.list()

  loading.value = false

  if (err)
    return ErrorPopup(err.message)

  gruplar.value = data ?? []
}

onMounted(load)

const bagliMi = (groupId: number) =>
  props.modelValue.some(l => l.group_id === groupId)

const zorunluMu = (groupId: number) =>
  props.modelValue.find(l => l.group_id === groupId)?.is_required ?? false

const toggleGrup = (groupId: number, acik: boolean) => {
  emit('update:modelValue', acik
    ? [...props.modelValue, { group_id: groupId, is_required: false }]
    : props.modelValue.filter(l => l.group_id !== groupId))
}

const toggleZorunlu = (groupId: number, zorunlu: boolean) => {
  emit('update:modelValue', props.modelValue.map(l =>
    l.group_id === groupId ? { ...l, is_required: zorunlu } : l))
}
</script>

<template>
  <div>
    <p class="text-subtitle-1 mb-1">
      Özelleştirme
    </p>
    <p class="text-caption text-medium-emphasis mb-4">
      İşaretlenen gruplar müşteriye ürün sayfasında sorulur. "Zorunlu"
      işaretlenirse müşteri seçmeden sepete ekleyemez.
    </p>

    <VProgressLinear
      v-if="loading"
      indeterminate
      class="mb-4"
    />

    <VAlert
      v-else-if="!secilebilir.length"
      type="info"
      variant="tonal"
      density="compact"
    >
      Henüz kullanılabilir seçenek grubu yok. Seçenek Yönetimi sayfasından
      grup ve renk ekleyebilirsiniz.
    </VAlert>

    <div
      v-for="g in secilebilir"
      :key="g.id"
      class="d-flex align-center flex-wrap ga-4 py-2 border-b"
    >
      <VCheckbox
        :model-value="bagliMi(g.id)"
        :label="g.name"
        density="compact"
        hide-details
        style="min-inline-size: 200px;"
        @update:model-value="toggleGrup(g.id, $event as boolean)"
      />

      <VCheckbox
        :model-value="zorunluMu(g.id)"
        :disabled="!bagliMi(g.id)"
        label="Zorunlu"
        density="compact"
        hide-details
        @update:model-value="toggleZorunlu(g.id, $event as boolean)"
      />

      <!-- Grubun renkleri — salt okunur önizleme, esnaf ne sunulacağını görsün -->
      <div class="d-flex ga-1 flex-wrap">
        <VTooltip
          v-for="v in g.values.filter(x => x.is_active)"
          :key="v.id"
          :text="v.name"
          location="top"
        >
          <template #activator="{ props: tip }">
            <span
              v-if="g.kind === 'color'"
              v-bind="tip"
              class="d-inline-block rounded-circle border"
              :style="{ background: v.swatch_hex, inlineSize: '18px', blockSize: '18px' }"
            />
            <VChip
              v-else
              v-bind="tip"
              size="x-small"
              variant="tonal"
            >
              {{ v.name }}
            </VChip>
          </template>
        </VTooltip>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 3: Ürün formuna bağla**

Ürün formu sayfasında (`urunler/[id].vue`) kategori seçiminin altına yeni
bir `VCol cols="12"` içinde:

```vue
<ProductOptionPicker v-model="form.option_groups" />
```

`form` ref'ine `option_groups: [] as ProductOptionGroupLink[]` eklenir;
ürün yüklenirken mevcut bağlar doldurulur:

```ts
  form.value.option_groups = (p.option_groups ?? []).map(g => ({
    group_id: g.id,
    is_required: g.is_required,
  }))
```

Kaydetme çağrısında `option_groups: form.value.option_groups` gönderilir.

- [ ] **Step 4: Sipariş detayında seçimleri göster**

`model/order.ts`'teki `OrderItem` arayüzüne:

```ts
export interface OrderItemOption {
  group_name: string
  value_name: string
  /** kind='text' seçimde boş — o zaman nokta gösterilmez. */
  swatch_hex: string
}
```

ve `options: OrderItemOption[]` alanı.

`siparisler/[id].vue`'de kalem satırında ürün adının altına:

```vue
<div
  v-if="item.options?.length"
  class="d-flex flex-wrap ga-3 mt-1"
>
  <span
    v-for="(o, i) in item.options"
    :key="i"
    class="d-inline-flex align-center ga-1 text-caption text-medium-emphasis"
  >
    <span
      v-if="o.swatch_hex"
      class="d-inline-block rounded-circle border"
      :style="{ background: o.swatch_hex, inlineSize: '12px', blockSize: '12px' }"
    />
    {{ o.group_name }}: {{ o.value_name }}
  </span>
</div>
```

- [ ] **Step 5: Tip kontrolü ve lint**

```bash
cd frontend/idare
npx vue-tsc --noEmit -p tsconfig.json 2>&1 | grep -v "@core\|@layouts"
npx eslint src/components/ProductOptionPicker.vue src/pages/urunler/ src/pages/siparisler/
```

Beklenen: çıktısız.

- [ ] **Step 6: Tarayıcıda doğrula**

Panelde: ürün formunu aç → "Özelleştirme" bölümünde grupları gör → birini
işaretle + Zorunlu yap → kaydet → **sayfayı yenile** → seçimin korunduğunu
doğrula (bu adım kritik: "görsel bölümü kaydettikten sonra açılmıyordu"
hatası tam burada, kaydet-sonrası-yeniden-yükle akışında yaşanmıştı).

Sonra checkbox'ı kaldır → kaydet → yenile → gitmiş olduğunu doğrula.

- [ ] **Step 7: Commit**

```bash
git add frontend/idare/
git commit -m "feat(idare): ürün formunda Özelleştirme + sipariş detayında seçimler

Yalnızca aktif ve en az bir değeri olan gruplar seçilebilir — değersiz
grubu ürüne açmak müşteriye boş başlık gösterirdi. Sipariş detayında
seçimler renk noktasıyla, sipariş anındaki kopyadan okunuyor."
```

---

### Task 9: Public site — sepet mantığı (birleştirme anahtarı)

**Files:**
- Modify: `frontend/app/app/types/api.ts`
- Modify: `frontend/app/app/composables/cartLogic.ts`
- Modify: `frontend/app/app/composables/useCart.ts`
- Modify: `frontend/app/app/composables/useCart.test.ts`

**Interfaces:**
- Consumes: yok (saf frontend mantığı)
- Produces:
  - `CartItemOption { value_id, group_name, value_name, swatch_hex }`
  - `CartItem.options: CartItemOption[]`
  - `cartLineKey(item: CartItem): string`
  - `addItem/removeItem/setItemQuantity` artık `lineKey: string` alır
    (Task 10 bunları çağırır)

- [ ] **Step 1: types/api.ts'i güncelle**

```ts
/** Sepet kalemindeki tek seçim. value_id sunucuya gider; isim ve renk
 *  yalnızca GÖSTERİM için — sipariş anında sunucu DB'den okur. */
export interface CartItemOption {
  value_id: number
  group_name: string
  value_name: string
  swatch_hex: string
}

export interface CartItem {
  product_id: number
  name: string
  slug: string
  price: string
  image: string
  quantity: number
  /** Seçim yoksa boş dizi. Bu alandan ÖNCE kurulmuş sepetlerde
   *  undefined gelir — okuyan taraf boş dizi kabul eder. */
  options?: CartItemOption[]
}
```

Ayrıca public ürün tipine:

```ts
export interface ProductOptionValue {
  id: number
  name: string
  swatch_hex: string
}

export interface ProductOptionGroup {
  id: number
  name: string
  kind: 'color' | 'text'
  is_required: boolean
  values: ProductOptionValue[]
}
```

ve `Product` arayüzüne `option_groups?: ProductOptionGroup[]`.

`CreateOrderInput.items` şu hale gelir:

```ts
  items: { product_id: number, quantity: number, option_value_ids: number[] }[]
```

- [ ] **Step 2: Failing test yaz**

`frontend/app/app/composables/useCart.test.ts` sonuna:

```ts
import { cartLineKey } from './cartLogic'

const secim = (id: number, ad: string) => ({
  value_id: id, group_name: 'Ambalaj', value_name: ad, swatch_hex: '#FFFFFF',
})

describe('cartLineKey', () => {
  it('seçim sırası anahtarı değiştirmez', () => {
    const a = { ...urun(1, '100.00'), options: [secim(3, 'Pembe'), secim(7, 'Beyaz')] }
    const b = { ...urun(1, '100.00'), options: [secim(7, 'Beyaz'), secim(3, 'Pembe')] }

    expect(cartLineKey(a)).toBe(cartLineKey(b))
  })

  it('seçimsiz kalem (eski sepet) patlamaz', () => {
    const eski = urun(1, '100.00') // options alanı YOK

    expect(cartLineKey(eski)).toBe('1:')
  })

  it('farklı seçim farklı anahtar üretir', () => {
    const pembe = { ...urun(1, '100.00'), options: [secim(3, 'Pembe')] }
    const beyaz = { ...urun(1, '100.00'), options: [secim(7, 'Beyaz')] }

    expect(cartLineKey(pembe)).not.toBe(cartLineKey(beyaz))
  })
})

describe('addItem — seçimli', () => {
  it('aynı ürün farklı seçim AYRI kalem olur', () => {
    const pembe = { ...urun(1, '100.00'), options: [secim(3, 'Pembe')] }
    const beyaz = { ...urun(1, '100.00'), options: [secim(7, 'Beyaz')] }

    const out = addItem(addItem([], pembe), beyaz)

    expect(out).toHaveLength(2)
  })

  it('aynı ürün aynı seçim TEK kalemde birleşir', () => {
    const pembe = { ...urun(1, '100.00', 1), options: [secim(3, 'Pembe')] }

    const out = addItem(addItem([], pembe), { ...pembe, quantity: 2 })

    expect(out).toHaveLength(1)
    expect(out[0]!.quantity).toBe(3)
  })

  it('seçimsiz eski kalemler eskisi gibi birleşir', () => {
    const out = addItem([urun(1, '100.00', 2)], urun(1, '100.00', 3))

    expect(out).toHaveLength(1)
    expect(out[0]!.quantity).toBe(5)
  })
})

describe('removeItem / setItemQuantity — seçimli', () => {
  it('yalnızca hedef satırı siler, aynı ürünün diğer rengine dokunmaz', () => {
    const pembe = { ...urun(1, '100.00'), options: [secim(3, 'Pembe')] }
    const beyaz = { ...urun(1, '100.00'), options: [secim(7, 'Beyaz')] }
    const sepet = addItem(addItem([], pembe), beyaz)

    const out = removeItem(sepet, cartLineKey(pembe))

    expect(out).toHaveLength(1)
    expect(out[0]!.options![0]!.value_name).toBe('Beyaz')
  })

  it('adet değişimi yalnızca hedef satıra uygulanır', () => {
    const pembe = { ...urun(1, '100.00', 1), options: [secim(3, 'Pembe')] }
    const beyaz = { ...urun(1, '100.00', 1), options: [secim(7, 'Beyaz')] }
    const sepet = addItem(addItem([], pembe), beyaz)

    const out = setItemQuantity(sepet, cartLineKey(pembe), 5)

    expect(out.find(i => i.options![0]!.value_name === 'Pembe')!.quantity).toBe(5)
    expect(out.find(i => i.options![0]!.value_name === 'Beyaz')!.quantity).toBe(1)
  })
})
```

- [ ] **Step 3: Test'i çalıştır, kırmızı olduğunu gör**

```bash
cd frontend/app && npx vitest run app/composables/useCart.test.ts
```

Beklenen: FAIL — `cartLineKey` yok.

- [ ] **Step 4: cartLogic.ts'i güncelle**

```ts
/**
 * Sepet satırının kimliği: ürün + seçilen değerler.
 *
 * Seçimler SIRALANIR — müşteri renkleri hangi sırayla seçerse seçsin aynı
 * satır olmalı. Bu alandan önce kurulmuş sepetlerde options undefined
 * gelir; boş dizi kabul edilir, sepet sıfırlanmaz.
 */
export function cartLineKey(item: CartItem): string {
  const ids = (item.options ?? []).map(o => o.value_id).sort((a, b) => a - b)

  return `${item.product_id}:${ids.join(',')}`
}

export function addItem(items: CartItem[], yeni: CartItem): CartItem[] {
  const key = cartLineKey(yeni)
  const mevcut = items.find(i => cartLineKey(i) === key)
  if (mevcut) {
    return items.map(i =>
      cartLineKey(i) === key
        ? { ...i, quantity: i.quantity + yeni.quantity }
        : i)
  }

  return [...items, yeni]
}

export function removeItem(items: CartItem[], lineKey: string): CartItem[] {
  return items.filter(i => cartLineKey(i) !== lineKey)
}

export function setItemQuantity(items: CartItem[], lineKey: string, qty: number): CartItem[] {
  if (qty <= 0)
    return removeItem(items, lineKey)

  return items.map(i => (cartLineKey(i) === lineKey ? { ...i, quantity: qty } : i))
}
```

`cartTotal` değişmez.

- [ ] **Step 5: useCart.ts'i güncelle**

`remove` ve `setQuantity` sarmalayıcıları artık `lineKey: string` alır;
`cartLineKey` re-export edilir ki bileşenler `import { cartLineKey } from
'./cartLogic'` yerine composable'dan alsın. Sepet çekmecesi ve sipariş
sayfası bu fonksiyonları `cartLineKey(item)` ile çağıracak (Task 10).

- [ ] **Step 6: Testleri çalıştır, yeşil olduğunu gör**

```bash
cd frontend/app && npx vitest run app/composables/useCart.test.ts
```

Beklenen: eski testler + 8 yeni test PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend/app/app/composables/ frontend/app/app/types/api.ts
git commit -m "feat(sepet): birleştirme anahtarı ürün+seçim oldu

Pembe ambalajlı buket ile beyaz ambalajlı aynı buket artık ayrı satır.
removeItem/setItemQuantity de anahtar aldı — yoksa 'pembeyi sil' beyazı
da silerdi. Seçim id'leri sıralanıyor: seçim sırası satırı değiştirmemeli.

Eski sepetlerde options undefined gelir, boş dizi kabul ediliyor —
kimsenin sepeti sıfırlanmıyor."
```

---

### Task 10: Public site — ürün sayfasında seçim + sepet/sipariş gösterimi

**Files:**
- Create: `frontend/app/app/components/ProductOptionSelector.vue`
- Modify: `frontend/app/app/pages/urun/[slug].vue`
- Modify: `frontend/app/app/components/TheCartDrawer.vue`
- Modify: `frontend/app/app/pages/siparis/index.vue`
- Modify: `frontend/app/app/composables/useOrders.ts`

**Interfaces:**
- Consumes: Task 5'in public `option_groups`, Task 9'un `cartLineKey`
- Produces: uçtan uca çalışan özellik

- [ ] **Step 1: ProductOptionSelector.vue yaz**

```vue
<script setup lang="ts">
import type { CartItemOption, ProductOptionGroup } from '~/types/api'

const props = defineProps<{
  groups: ProductOptionGroup[]
  modelValue: CartItemOption[]
}>()

const emit = defineEmits<{
  'update:modelValue': [CartItemOption[]]
}>()

const secilenId = (groupId: number) => {
  const g = props.groups.find(x => x.id === groupId)

  return props.modelValue.find(o => g?.values.some(v => v.id === o.value_id))?.value_id
}

const sec = (group: ProductOptionGroup, valueId: number) => {
  const deger = group.values.find(v => v.id === valueId)
  if (!deger)
    return

  // Aynı gruptan önceki seçim düşer — grup başına tek değer.
  const digerGruplar = props.modelValue.filter(
    o => !group.values.some(v => v.id === o.value_id))

  emit('update:modelValue', [...digerGruplar, {
    value_id: deger.id,
    group_name: group.name,
    value_name: deger.name,
    swatch_hex: deger.swatch_hex,
  }])
}

/** Doldurulmamış zorunlu grupların adları — sepete ekle butonu bunu kullanır. */
const eksikZorunlular = computed(() =>
  props.groups.filter(g => g.is_required && secilenId(g.id) === undefined).map(g => g.name))

defineExpose({ eksikZorunlular })
</script>

<template>
  <div class="space-y-5">
    <div
      v-for="g in groups"
      :key="g.id"
    >
      <p class="mb-2 text-body-md">
        <span class="font-medium">{{ g.name }}</span>
        <span
          v-if="g.is_required"
          class="text-error"
          aria-hidden="true"
        > *</span>
      </p>

      <!--
        Renk tek başına bilgi taşımamalı: her seçenek aria-label ile
        rengin ADINI da taşıyor, seçili olan aria-checked ile bildiriliyor.
      -->
      <div
        role="radiogroup"
        :aria-label="g.name"
        :aria-required="g.is_required"
        class="flex flex-wrap gap-2"
      >
        <button
          v-for="v in g.values"
          :key="v.id"
          type="button"
          role="radio"
          :aria-checked="secilenId(g.id) === v.id"
          :aria-label="v.name"
          :title="v.name"
          class="transition"
          :class="g.kind === 'color'
            ? 'size-9 rounded-full border-2'
            : 'rounded-full border px-4 py-1.5 text-body-sm'"
          :style="g.kind === 'color'
            ? {
              background: v.swatch_hex,
              borderColor: secilenId(g.id) === v.id ? 'var(--color-primary, #7a5c3e)' : 'rgba(0,0,0,.15)',
              outline: secilenId(g.id) === v.id ? '2px solid var(--color-primary, #7a5c3e)' : 'none',
              outlineOffset: '2px',
            }
            : undefined"
          @click="sec(g, v.id)"
        >
          <span v-if="g.kind !== 'color'">{{ v.name }}</span>
        </button>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Ürün detay sayfasına bağla**

`urun/[slug].vue`'de galeri ile "Sepete Ekle" arasına:

```vue
<ProductOptionSelector
  v-if="urun.option_groups?.length"
  ref="secimRef"
  v-model="secimler"
  :groups="urun.option_groups"
  class="my-6"
/>
```

Script'te:

```ts
const secimler = ref<CartItemOption[]>([])

const eksikZorunlular = computed(() =>
  (urun.value?.option_groups ?? [])
    .filter(g => g.is_required && !secimler.value.some(
      o => g.values.some(v => v.id === o.value_id)))
    .map(g => g.name))

const sepeteEklenebilir = computed(() => eksikZorunlular.value.length === 0)
```

Sepete ekle butonu `:disabled="!sepeteEklenebilir"` alır ve altına uyarı:

```vue
<p
  v-if="eksikZorunlular.length"
  class="mt-2 text-body-sm text-error"
>
  {{ eksikZorunlular.join(', ') }} seçiniz.
</p>
```

Sepete ekleme çağrısına `options: secimler.value` eklenir.

- [ ] **Step 3: Sepet çekmecesinde seçimleri göster**

`TheCartDrawer.vue`'de kalem satırında ürün adının altına:

```vue
<div
  v-if="item.options?.length"
  class="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-1"
>
  <span
    v-for="o in item.options"
    :key="o.value_id"
    class="inline-flex items-center gap-1 text-body-sm text-on-surface-variant"
  >
    <span
      v-if="o.swatch_hex"
      class="inline-block size-3 rounded-full border border-outline-variant/50"
      :style="{ background: o.swatch_hex }"
      aria-hidden="true"
    />
    {{ o.value_name }}
  </span>
</div>
```

`remove` ve `setQuantity` çağrıları `cartLineKey(item)` geçirecek şekilde
güncellenir (Task 9'da imzalar değişti). Aynı güncelleme
`siparis/index.vue`'deki sepet özeti için de yapılır.

- [ ] **Step 4: Sipariş gönderiminde id'leri gönder**

`useOrders.ts`'te sipariş gövdesi kurulurken:

```ts
    items: sepet.map(i => ({
      product_id: i.product_id,
      quantity: i.quantity,
      // YALNIZCA id — isim ve renk sunucuda DB'den okunur.
      option_value_ids: (i.options ?? []).map(o => o.value_id),
    })),
```

- [ ] **Step 5: Birim testleri çalıştır**

```bash
cd frontend/app && npx vitest run
```

Beklenen: hepsi PASS.

- [ ] **Step 6: E2E — proxy üzerinden uçtan uca doğrula**

Servisleri kaldır:

```bash
cd backend && go run ./cmd/seed -username=e2etest -password=Test12345
cd backend && go run ./cmd/server &
cd frontend/idare && pnpm dev &
cd frontend/app && pnpm dev &
sleep 30
```

**Sipariş çağrıları `localhost:3000/api/go/*` üzerinden yapılır, Go'ya
(`:8080`) doğrudan curl ATILMAZ** — proxy katmanı hatalarını gizler.

Playwright ile tam zincir:
1. Panelde `/secenekler` → grup + renkler mevcut (seed'den)
2. Panelde ürün formunda "Ambalaj Rengi"ni aç, Zorunlu işaretle, kaydet
3. Public sitede ürün sayfası → renk noktaları görünüyor
4. Seçim yapmadan "Sepete Ekle" **pasif**, uyarı metni görünüyor
5. Pembe seç → buton aktif → sepete ekle
6. Geri dön, Beyaz seç → sepete ekle → **sepette 2 ayrı satır**
7. Pembe satırın adedini artır → beyaz satır etkilenmiyor
8. Pembe satırı sil → beyaz duruyor
9. Siparişi tamamla → `awaiting_payment` sipariş oluşuyor
10. Panelde sipariş detayı → iki kalem, her biri kendi renk noktasıyla
11. Konsol hatası yok

Ekran görüntüleri al (ürün sayfası, sepet, sipariş detayı) ve gözle kontrol et.

- [ ] **Step 7: Sunucu tarafı güvenlik doğrulaması**

Tarayıcının gönderdiğine güvenilmediğini kanıtla — proxy üzerinden
elle hazırlanmış gövde gönder:

```bash
# Ürüne KAPALI bir gruba ait value_id ile sipariş → reddedilmeli
curl -s -X POST http://localhost:3000/api/go/orders \
  -H "Content-Type: application/json" \
  -d '{"items":[{"product_id":<ID>,"quantity":1,"option_value_ids":[<YABANCI_ID>]}], ...}' \
  -w "\n%{http_code}\n"
```

Beklenen: 400 ve "geçersiz veya artık sunulmayan seçenek".

```bash
# Zorunlu grup boş → reddedilmeli
curl -s -X POST http://localhost:3000/api/go/orders \
  -H "Content-Type: application/json" \
  -d '{"items":[{"product_id":<ID>,"quantity":1,"option_value_ids":[]}], ...}' \
  -w "\n%{http_code}\n"
```

Beklenen: 400 ve grup adını içeren mesaj.

- [ ] **Step 8: Tüm suite**

```bash
cd /Users/omerkoc/GolandProjects/cicekci && make test 2>&1 | tail -15
cd frontend/app && npx vitest run
cd frontend/idare && npx vue-tsc --noEmit -p tsconfig.json 2>&1 | grep -v "@core\|@layouts"
```

Beklenen: yalnızca bilinen 2 PayTR config testi kırmızı, geri kalan temiz.

- [ ] **Step 9: Temizlik + commit**

```bash
pkill -f "cmd/server"; pkill -f vite; pkill -f "nuxt dev"
docker compose exec -T postgres psql -U cicekci -d cicekci -c \
  "DELETE FROM admin_users WHERE username='e2etest';"
git add frontend/app/
git commit -m "feat(site): ürün sayfasında buket özelleştirme

Renk noktaları radiogroup içinde, her biri aria-label ile rengin adını
taşıyor — renk tek başına bilgi taşımamalı. Zorunlu grup seçilmeden
sepete ekle pasif.

Sipariş gövdesinde YALNIZCA option_value_ids gidiyor; isim ve renk
sunucuda DB'den okunuyor."
```

---

### Task 11: Dokümantasyon

**Files:**
- Modify: `docs/DURUM.md`

**Interfaces:**
- Consumes: Task 1-10'un tamamı
- Produces: yok (son görev)

- [ ] **Step 1: DURUM.md'yi güncelle**

"Nerede kaldık" tablosuna satır ekle:

```markdown
| Ürün özelleştirme (buket tasarla) | ✅ **Uygulandı** — migration 10, seçenek grupları panelden yönetiliyor |
```

Yeni bölüm ekle (Yedekleme bölümünün üstüne):

```markdown
## Ürün Özelleştirme, 2026-08-16

Spec: `docs/superpowers/specs/2026-08-16-urun-ozellestirme-design.md`
Plan: `docs/superpowers/plans/2026-08-16-urun-ozellestirme.md`

Müşteri ürünü sipariş ederken ambalaj/kurdele/kutu rengi seçebiliyor.
Seçenek grupları **panelden** yönetiliyor (`/idare/secenekler`) — yeni bir
grup ("Çiçek Rengi" vb.) eklemek kod değişikliği veya migration
gerektirmiyor. Migration 10'daki üç grup yalnızca başlangıç verisi.

**Kararlar:** merkezi seçenek havuzu + ürün başına aç/kapa, grup başına
tip (`color`/`text`), fiyat farkı YOK, zorunluluk ürün başına, farklı
seçim = ayrı sepet kalemi.

**Kritik tasarım noktaları:**
- Seçim siparişe **kopyalanır** (`order_item_options`: isim + hex), gruba
  referans tutulmaz. Esnaf sonradan rengi silerse eski sipariş bozulmaz —
  `order_items.product_name` deseninin aynısı. Regresyon testi var.
- Sunucu tarayıcıdan **yalnızca `option_value_ids`** alır; isim ve renk
  DB'den okunur. Ürüne kapalı grup, pasif değer, aynı gruptan iki değer ve
  eksik zorunlu grup reddedilir. Fiyattaki "sepetten gelen değere güvenme"
  kuralının aynısı.
- **Sepet birleştirme anahtarı** `product_id` → `product_id + sıralı
  seçim id'leri` oldu. `addItem`/`removeItem`/`setItemQuantity` üçü birden
  değişti — yoksa "pembeyi sil" beyazı da silerdi.
- Bu alandan önce kurulmuş sepetlerde `options` undefined gelir, boş dizi
  kabul edilir — kimsenin sepeti sıfırlanmadı.

**Ödemeye etkisi yok:** fiyat farkı olmadığı için `itemsTotal`, PayTR
sepeti ve callback tutar doğrulaması değişmedi.
```

- [ ] **Step 2: Commit**

```bash
git add docs/DURUM.md
git commit -m "docs: ürün özelleştirme uygulandı — DURUM.md güncel"
```

---

## Self-Review

**1. Spec coverage:**

| Spec bölümü | Görev |
|---|---|
| §3 Veri modeli (4 tablo + seed) | Task 1 |
| §4 Backend paket (model/store/service) | Task 2, 3 |
| §4.1 Admin uçları | Task 4 |
| §4.1 Ürün formu uçları / §4.2 Public uç | Task 5 |
| §4.3 Sipariş güvenliği + `OptionReader` + N+1 | Task 6 |
| §5.1 Seçenekler sayfası | Task 7 |
| §5.2 Ürün formu bölümü / §5.3 Sipariş detayı | Task 8 |
| §6.2 Sepet birleştirme + geriye dönük uyum | Task 9 |
| §6.1 Ürün sayfası seçici / §6.3 Sipariş gönderimi | Task 10 |
| §7 Test planı | Her görevin test adımları + Task 10 E2E |
| §8 Kapsam dışı | Hiçbir görevde yok ✅ |

Boşluk yok.

**2. Placeholder scan:** "TBD"/"TODO"/"uygun hata yönetimi ekle" yok. Her
kod adımı gerçek kod içeriyor. Üç yerde "mevcut dosyadaki yardımcıyı
kullan" notu var (Task 4 Step 4, Task 6 Step 6, Task 8 Step 3) — bunlar
kasıtlı: test altyapısı zaten var, yenisini kurmak tekrar olurdu.

**3. Type consistency:**
- `Kind`/`KindColor`/`KindText` — Task 2'de tanımlı, Task 3/4/5'te aynı
- `ProductGroupLink`/`ProductGroup` — Task 3 Step 2'de tanımlı, Task 3/5'te aynı
- `OptionReader` — Task 6 Step 2'de `order` paketinde tanımlı, Task 6
  Step 5'te `productoption` karşılıyor. Dönüş tipi `order.OrderItemOption`
  (ayrı bir `ResolvedOption` tipi YOK — tek tip, tek dönüşüm noktası) ✅
- `OrderItemOption` — Go tarafı Task 6, panel tarafı Task 8, public tarafı
  Task 9; alan adları (`group_name`/`value_name`/`swatch_hex`) üçünde de aynı
- `cartLineKey` — Task 9'da tanımlı, Task 10'da çağrılıyor
- `OptionValueView` — Task 4'te tanımlı, Task 5'te tekrar kullanılıyor ✅

**Import döngüsü kontrolü** (mevcut kodda doğrulandı):

```
productoption → order    (OrderItemOption tipi için)
productoption → product  (Slugify için)
order         → product
product       → (hiçbir iç paket — yaprak)
```

`order` paketi `productoption`'ı **import etmiyor**: `OptionReader`
arayüzü tüketici tarafta tanımlı, somut servisi `main.go` bağlıyor.
Döngü yok.

Ancak `order`'ın kendi test dosyaları `productoption`'ı import ederse
döngü oluşur (test dosyaları da paketin parçası). Bu yüzden Task 6'nın
testleri `order_test` harici paketinde — Step 6'da not edildi.

---

## Execution Handoff

**Plan complete and saved to `docs/superpowers/plans/2026-08-16-urun-ozellestirme.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
