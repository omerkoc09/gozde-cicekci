# Plan 1 — Backend Temeli Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Çalışan bir Go API — admin auth, kategori CRUD, ürün CRUD (slug geçmişi ve iki eksenli filtre dahil). Görsel yükleme Plan 2'de.

**Architecture:** Fiber + PostgreSQL. Domain dışta (`internal/product/`, `internal/category/`, `internal/auth/`), HTTP ayrımı içeride (`internal/api/app/` public, `internal/api/idare/` admin). Her domain üç katman: `handler` (HTTP'yi bilir), `service` (iş mantığını bilir), `store` (SQL'i bilir).

**Tech Stack:** Go 1.23+, Fiber v2, `pgx/v5`, `golang-migrate`, `golang-jwt/jwt/v5`, `bcrypt`, `testify`, Docker Compose (test DB).

**Spec:** `docs/superpowers/specs/2026-07-15-cicekci-mvp-design.md` — bu plan §4.1, §4.2, §4.3, §4.5, §4.6, §6 ve §7'nin 1-4. adımlarını kapsar.

## Global Constraints

- **Para `NUMERIC(10,2)`, float asla.** Go tarafında `shopspring/decimal`.
- **`is_active=false` filtresi store katmanında**, handler'da değil. Public store metodları pasif kaydı hiç görmez.
- **Public viewmodel'e `is_active` alanı konmaz.** Admin viewmodel'inde vardır.
- **`internal/`, `pkg/` değil.** `pkg/` sadece config, database, log, errorsx için.
- **`service` ve `store` katmanları Fiber'i import etmez.** Fiber sadece `internal/api/` altında.
- **Hata formatı:** `{"error": {"code": "...", "message": "..."}}`. Public'te iç detay sızmaz.
- **Migration ile şema.** Elle DDL yok.
- **Türkçe slug dönüşümü:** `İ→i`, `ı→i`, `ş→s`, `ğ→g`, `ü→u`, `ö→o`, `ç→c`.

---

## Dosya Yapısı

```
go.mod
docker-compose.yml               → test/geliştirme Postgres
.env.example
Makefile                         → make test, make migrate-up, make seed

cmd/
  server/main.go                 → Fiber başlatma, route bağlama
  seed/main.go                   → ilk admin kullanıcısı

migrations/
  000001_init.up.sql / .down.sql

pkg/
  config/config.go               → .env okuma
  database/database.go           → pgxpool bağlantısı
  database/testdb.go             → test helper (temiz DB)
  errorsx/errors.go              → domain hata tipleri

internal/
  auth/
    model.go                     → AdminUser
    store.go                     → FindByUsername, Create
    service.go                   → Login, HashPassword
    jwt.go                       → token üret/doğrula
    middleware.go                → Fiber JWT middleware
  category/
    model.go                     → Category, Axis
    store.go                     → CRUD + ProductCount
    service.go                   → iş mantığı
  product/
    model.go                     → Product, ProductFilter
    slug.go                      → Slugify, saf fonksiyon
    store.go                     → CRUD + filtre + slug geçmişi
    service.go                   → iş mantığı
  api/
    app/
      router.go                  → public rotalar
      category_handler.go
      category_view.go
      product_handler.go
      product_view.go
    idare/
      router.go                  → admin rotalar
      auth_handler.go
      category_handler.go
      category_view.go
      product_handler.go
      product_view.go
```

**Neden bu bölme:** Ürüne alan eklerken `internal/product/` içinde kalınır. `product/service.go` tektir — hem `api/app` hem `api/idare` onu çağırır, ama farklı viewmodel'lere dönüştürür.

---

## Task 1: Proje iskeleti, Docker Compose, config

**Files:**
- Create: `go.mod`, `docker-compose.yml`, `.env.example`, `.gitignore`, `Makefile`
- Create: `pkg/config/config.go`
- Test: `pkg/config/config_test.go`

**Interfaces:**
- Produces: `config.Config` struct, `config.Load() (*Config, error)`

- [ ] **Step 1: Go modülü ve bağımlılıklar**

```bash
cd /Users/omerkoc/GolandProjects/cicekci
go mod init github.com/omerkoc/cicekci
go get github.com/gofiber/fiber/v2
go get github.com/jackc/pgx/v5
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-jwt/jwt/v5
go get github.com/shopspring/decimal
go get golang.org/x/crypto/bcrypt
go get github.com/joho/godotenv
go get github.com/stretchr/testify
```

- [ ] **Step 2: docker-compose.yml oluştur**

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: cicekci
      POSTGRES_PASSWORD: cicekci
      POSTGRES_DB: cicekci
    ports:
      - "5433:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U cicekci"]
      interval: 2s
      timeout: 3s
      retries: 15

  postgres_test:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: cicekci
      POSTGRES_PASSWORD: cicekci
      POSTGRES_DB: cicekci_test
    ports:
      - "5434:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U cicekci"]
      interval: 2s
      timeout: 3s
      retries: 15
```

Port 5433/5434 seçildi — lokalde başka bir Postgres 5432'de olabilir, çakışmasın.

- [ ] **Step 3: .env.example ve .gitignore**

`.env.example`:
```
DATABASE_URL=postgres://cicekci:cicekci@localhost:5433/cicekci?sslmode=disable
TEST_DATABASE_URL=postgres://cicekci:cicekci@localhost:5434/cicekci_test?sslmode=disable
JWT_SECRET=change-me-in-production-min-32-chars
PORT=8080
WHATSAPP_NUMBER=905551234567
SITE_URL=http://localhost:3000
```

`.gitignore`:
```
.env
/tmp
/bin
*.test
```

- [ ] **Step 4: Config testini yaz**

`pkg/config/config_test.go`:
```go
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ReadsEnvVars(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")
	t.Setenv("PORT", "9999")
	t.Setenv("WHATSAPP_NUMBER", "905551234567")
	t.Setenv("SITE_URL", "https://example.com")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "postgres://x/y", cfg.DatabaseURL)
	assert.Equal(t, "9999", cfg.Port)
	assert.Equal(t, "905551234567", cfg.WhatsAppNumber)
}

func TestLoad_DefaultsPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")
	t.Setenv("WHATSAPP_NUMBER", "905551234567")
	t.Setenv("SITE_URL", "https://example.com")
	t.Setenv("PORT", "")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
}

func TestLoad_FailsWithoutDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")

	_, err := Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL")
}

func TestLoad_FailsWithShortJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "short")

	_, err := Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}
```

- [ ] **Step 5: Testi çalıştır, başarısız olduğunu gör**

Run: `go test ./pkg/config/ -v`
Expected: FAIL — `undefined: Load`

- [ ] **Step 6: config.go yaz**

`pkg/config/config.go`:
```go
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL    string
	JWTSecret      string
	Port           string
	WhatsAppNumber string
	SiteURL        string
}

// Load .env dosyasını okur (varsa) ve ortam değişkenlerinden Config üretir.
// .env yoksa hata değil — production'da değişkenler platformdan gelir.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		Port:           os.Getenv("PORT"),
		WhatsAppNumber: os.Getenv("WHATSAPP_NUMBER"),
		SiteURL:        os.Getenv("SITE_URL"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL zorunlu")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET en az 32 karakter olmalı")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg, nil
}
```

- [ ] **Step 7: Testi çalıştır, geçtiğini gör**

Run: `go test ./pkg/config/ -v`
Expected: PASS — 4 test

- [ ] **Step 8: Makefile yaz**

```makefile
.PHONY: db-up db-down test migrate-up migrate-down seed run

db-up:
	docker compose up -d
	@echo "Postgres hazır bekleniyor..."
	@until docker compose exec -T postgres pg_isready -U cicekci >/dev/null 2>&1; do sleep 1; done
	@until docker compose exec -T postgres_test pg_isready -U cicekci >/dev/null 2>&1; do sleep 1; done
	@echo "Hazır."

db-down:
	docker compose down

migrate-up:
	migrate -path migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path migrations -database "$$DATABASE_URL" down 1

test:
	go test ./... -v

seed:
	go run ./cmd/seed

run:
	go run ./cmd/server
```

- [ ] **Step 9: Commit**

```bash
git add go.mod go.sum docker-compose.yml .env.example .gitignore Makefile pkg/config/
git commit -m "feat: proje iskeleti, docker compose, config yükleme"
```

---

## Task 2: Migration — şema

**Files:**
- Create: `migrations/000001_init.up.sql`, `migrations/000001_init.down.sql`
- Create: `pkg/database/database.go`

**Interfaces:**
- Produces: `database.Connect(ctx, url) (*pgxpool.Pool, error)`

- [ ] **Step 1: golang-migrate CLI kur**

```bash
brew install golang-migrate
migrate -version
```

- [ ] **Step 2: up migration yaz**

`migrations/000001_init.up.sql` — spec §4.1'in birebir karşılığı:
```sql
CREATE TABLE products (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price       NUMERIC(10,2) NOT NULL CHECK (price >= 0),
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE product_slugs (
    slug       TEXT PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    is_current BOOLEAN NOT NULL DEFAULT true
);

CREATE INDEX idx_product_slugs_product_id ON product_slugs(product_id);
CREATE UNIQUE INDEX idx_product_slugs_one_current
    ON product_slugs(product_id) WHERE is_current;

CREATE TABLE product_images (
    id         BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    image_key  TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_product_images_product_id ON product_images(product_id, sort_order);

CREATE TABLE categories (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    axis        TEXT NOT NULL CHECK (axis IN ('occasion','type')),
    is_active   BOOLEAN NOT NULL DEFAULT true,
    is_featured BOOLEAN NOT NULL DEFAULT false,
    sort_order  INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_categories_axis ON categories(axis);

CREATE TABLE product_categories (
    product_id  BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, category_id)
);

CREATE INDEX idx_product_categories_category_id ON product_categories(category_id);

CREATE TABLE admin_users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL
);
```

`idx_product_slugs_one_current` partial unique index önemli: bir ürünün aynı anda birden fazla güncel slug'ı olamaz — veritabanı garantiliyor.

- [ ] **Step 3: down migration yaz**

`migrations/000001_init.down.sql`:
```sql
DROP TABLE IF EXISTS admin_users;
DROP TABLE IF EXISTS product_categories;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS product_images;
DROP TABLE IF EXISTS product_slugs;
DROP TABLE IF EXISTS products;
```

- [ ] **Step 4: database.go yaz**

`pkg/database/database.go`:
```go
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("db url parse: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}

	return pool, nil
}
```

- [ ] **Step 5: Migration'ı çalıştır ve doğrula**

```bash
make db-up
export DATABASE_URL="postgres://cicekci:cicekci@localhost:5433/cicekci?sslmode=disable"
make migrate-up
docker compose exec -T postgres psql -U cicekci -d cicekci -c "\dt"
```
Expected: 6 tablo listelenir — admin_users, categories, product_categories, product_images, product_slugs, products

- [ ] **Step 6: Down migration'ı da doğrula**

```bash
make migrate-down
docker compose exec -T postgres psql -U cicekci -d cicekci -c "\dt"
make migrate-up
```
Expected: down sonrası sadece `schema_migrations` kalır, up sonrası 6 tablo geri gelir.

- [ ] **Step 7: Commit**

```bash
git add migrations/ pkg/database/
git commit -m "feat: veritabanı şeması ve bağlantı"
```

---

## Task 3: Test DB helper

**Files:**
- Create: `pkg/database/testdb.go`

**Interfaces:**
- Produces: `database.NewTestDB(t *testing.T) *pgxpool.Pool` — her testte temiz DB

- [ ] **Step 1: testdb.go yaz**

`pkg/database/testdb.go`:
```go
package database

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// NewTestDB test veritabanına bağlanır ve tüm tabloları temizler.
// TEST_DATABASE_URL yoksa test skip edilir.
func NewTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://cicekci:cicekci@localhost:5434/cicekci_test?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := Connect(ctx, url)
	if err != nil {
		t.Skipf("test DB yok, skip: %v (make db-up çalıştırdın mı?)", err)
	}

	truncateAll(t, pool)

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func truncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE products, product_slugs, product_images,
		         categories, product_categories, admin_users
		RESTART IDENTITY CASCADE
	`)
	require.NoError(t, err, "test DB temizlenemedi — migration çalıştı mı?")
}
```

- [ ] **Step 2: Test DB'ye migration çalıştır**

```bash
export TEST_DATABASE_URL="postgres://cicekci:cicekci@localhost:5434/cicekci_test?sslmode=disable"
migrate -path migrations -database "$TEST_DATABASE_URL" up
```
Expected: `1/u init (...)`

- [ ] **Step 3: Makefile'a test-db-migrate ekle**

```makefile
test-db-migrate:
	migrate -path migrations -database "$$TEST_DATABASE_URL" up
```

- [ ] **Step 4: Commit**

```bash
git add pkg/database/testdb.go Makefile
git commit -m "test: test veritabanı helper'ı"
```

---

## Task 4: Slug üretimi (saf fonksiyon)

Spec §6'ya göre en çok bug çıkacak yer. Saf fonksiyon, DB'siz test.

**Files:**
- Create: `internal/product/slug.go`
- Test: `internal/product/slug_test.go`

**Interfaces:**
- Produces: `product.Slugify(name string) string`

- [ ] **Step 1: Testi yaz**

`internal/product/slug_test.go`:
```go
package product

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"basit", "Buket", "buket"},
		{"bosluk", "51 Gül Buket", "51-gul-buket"},
		{"turkce karakterler", "Çiçek Şöleni Güzel", "cicek-soleni-guzel"},
		{"buyuk I", "İstanbul Lalesi", "istanbul-lalesi"},
		{"noktali i", "Ilık Bahar", "ilik-bahar"},
		{"noktalama", "Gül & Papatya (Özel!)", "gul-papatya-ozel"},
		{"coklu bosluk", "Kırmızı   Gül", "kirmizi-gul"},
		{"bas son bosluk", "  Orkide  ", "orkide"},
		{"tire zaten var", "Mini-Buket", "mini-buket"},
		{"sadece rakam", "51", "51"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Slugify(tt.input))
		})
	}
}

func TestSlugify_TurkishCharsFully(t *testing.T) {
	assert.Equal(t, "cgiosu-cgiosu", Slugify("çğıöşü ÇĞİÖŞÜ"))
}

func TestSlugify_EmptyFallback(t *testing.T) {
	assert.Equal(t, "urun", Slugify(""))
	assert.Equal(t, "urun", Slugify("!!!"))
}
```

- [ ] **Step 2: Testi çalıştır, başarısız olduğunu gör**

Run: `go test ./internal/product/ -run TestSlugify -v`
Expected: FAIL — `undefined: Slugify`

- [ ] **Step 3: slug.go yaz**

`internal/product/slug.go`:
```go
package product

import (
	"regexp"
	"strings"
)

var turkishReplacer = strings.NewReplacer(
	"ç", "c", "Ç", "c",
	"ğ", "g", "Ğ", "g",
	"ı", "i", "I", "i",
	"İ", "i", "i", "i",
	"ö", "o", "Ö", "o",
	"ş", "s", "Ş", "s",
	"ü", "u", "Ü", "u",
)

var (
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)
	multipleDashes  = regexp.MustCompile(`-{2,}`)
)

// Slugify bir ürün adını URL slug'ına çevirir.
// Türkçe karakterler ASCII karşılıklarına dönüşür.
// Sonuç boş kalırsa "urun" döner.
func Slugify(name string) string {
	s := turkishReplacer.Replace(name)
	s = strings.ToLower(s)
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = multipleDashes.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	if s == "" {
		return "urun"
	}
	return s
}
```

`strings.ToLower` Türkçe `I`'yı doğru çeviremez (Go locale bilmez) — bu yüzden replacer `ToLower`'dan **önce** çalışıyor ve büyük harfleri de kapsıyor.

- [ ] **Step 4: Testi çalıştır, geçtiğini gör**

Run: `go test ./internal/product/ -run TestSlugify -v`
Expected: PASS — tüm alt testler

- [ ] **Step 5: Commit**

```bash
git add internal/product/slug.go internal/product/slug_test.go
git commit -m "feat: Türkçe karakter destekli slug üretimi"
```

---

## Task 5: Auth — model, store, bcrypt

**Files:**
- Create: `internal/auth/model.go`, `internal/auth/store.go`, `internal/auth/service.go`
- Test: `internal/auth/service_test.go`

**Interfaces:**
- Consumes: `database.NewTestDB`
- Produces:
  - `auth.AdminUser{ID int64, Username string, PasswordHash string}`
  - `auth.NewStore(pool *pgxpool.Pool) *Store`
  - `(*Store).FindByUsername(ctx, username) (*AdminUser, error)`
  - `(*Store).Create(ctx, username, passwordHash string) (*AdminUser, error)`
  - `auth.NewService(store *Store, jwtSecret string) *Service`
  - `(*Service).Login(ctx, username, password string) (string, error)` → JWT token
  - `(*Service).CreateAdmin(ctx, username, password string) error`

- [ ] **Step 1: errorsx paketini yaz**

`pkg/errorsx/errors.go`:
```go
package errorsx

import "errors"

var (
	ErrNotFound       = errors.New("kayıt bulunamadı")
	ErrInvalidInput   = errors.New("geçersiz girdi")
	ErrUnauthorized   = errors.New("yetkisiz")
	ErrConflict       = errors.New("çakışma")
)
```

- [ ] **Step 2: model.go yaz**

`internal/auth/model.go`:
```go
package auth

type AdminUser struct {
	ID           int64
	Username     string
	PasswordHash string
}
```

- [ ] **Step 3: Testi yaz**

`internal/auth/service_test.go`:
```go
package auth

import (
	"context"
	"testing"

	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-that-is-long-enough-32"

func TestService_CreateAdmin_ThenLogin(t *testing.T) {
	pool := database.NewTestDB(t)
	svc := NewService(NewStore(pool), testSecret)
	ctx := context.Background()

	err := svc.CreateAdmin(ctx, "cicekci", "gizli-sifre-123")
	require.NoError(t, err)

	token, err := svc.Login(ctx, "cicekci", "gizli-sifre-123")

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestService_Login_WrongPassword(t *testing.T) {
	pool := database.NewTestDB(t)
	svc := NewService(NewStore(pool), testSecret)
	ctx := context.Background()

	require.NoError(t, svc.CreateAdmin(ctx, "cicekci", "dogru-sifre-123"))

	_, err := svc.Login(ctx, "cicekci", "yanlis-sifre")

	require.ErrorIs(t, err, errorsx.ErrUnauthorized)
}

func TestService_Login_UnknownUser(t *testing.T) {
	pool := database.NewTestDB(t)
	svc := NewService(NewStore(pool), testSecret)

	_, err := svc.Login(context.Background(), "yok-boyle", "sifre")

	require.ErrorIs(t, err, errorsx.ErrUnauthorized)
}

func TestService_CreateAdmin_DuplicateUsername(t *testing.T) {
	pool := database.NewTestDB(t)
	svc := NewService(NewStore(pool), testSecret)
	ctx := context.Background()

	require.NoError(t, svc.CreateAdmin(ctx, "cicekci", "sifre-123456"))
	err := svc.CreateAdmin(ctx, "cicekci", "baska-sifre-123")

	require.ErrorIs(t, err, errorsx.ErrConflict)
}

func TestService_CreateAdmin_ShortPassword(t *testing.T) {
	pool := database.NewTestDB(t)
	svc := NewService(NewStore(pool), testSecret)

	err := svc.CreateAdmin(context.Background(), "cicekci", "kisa")

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_PasswordIsHashed(t *testing.T) {
	pool := database.NewTestDB(t)
	store := NewStore(pool)
	svc := NewService(store, testSecret)
	ctx := context.Background()

	require.NoError(t, svc.CreateAdmin(ctx, "cicekci", "gizli-sifre-123"))

	user, err := store.FindByUsername(ctx, "cicekci")

	require.NoError(t, err)
	assert.NotEqual(t, "gizli-sifre-123", user.PasswordHash)
	assert.Contains(t, user.PasswordHash, "$2a$")
}
```

- [ ] **Step 4: Testi çalıştır, başarısız olduğunu gör**

Run: `go test ./internal/auth/ -v`
Expected: FAIL — `undefined: NewService`

- [ ] **Step 5: store.go yaz**

`internal/auth/store.go`:
```go
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) FindByUsername(ctx context.Context, username string) (*AdminUser, error) {
	var u AdminUser
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash FROM admin_users WHERE username = $1`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("admin ara: %w", err)
	}
	return &u, nil
}

func (s *Store) Create(ctx context.Context, username, passwordHash string) (*AdminUser, error) {
	var u AdminUser
	err := s.pool.QueryRow(ctx,
		`INSERT INTO admin_users (username, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, username, password_hash`,
		username, passwordHash,
	).Scan(&u.ID, &u.Username, &u.PasswordHash)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, errorsx.ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("admin oluştur: %w", err)
	}
	return &u, nil
}
```

- [ ] **Step 6: service.go yaz**

`internal/auth/service.go`:
```go
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/omerkoc/cicekci/pkg/errorsx"
	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 8

type Service struct {
	store     *Store
	jwtSecret string
}

func NewService(store *Store, jwtSecret string) *Service {
	return &Service{store: store, jwtSecret: jwtSecret}
}

// CreateAdmin yeni bir admin kullanıcısı oluşturur. cmd/seed tarafından çağrılır.
func (s *Service) CreateAdmin(ctx context.Context, username, password string) error {
	if username == "" {
		return fmt.Errorf("%w: kullanıcı adı boş olamaz", errorsx.ErrInvalidInput)
	}
	if len(password) < minPasswordLength {
		return fmt.Errorf("%w: şifre en az %d karakter olmalı", errorsx.ErrInvalidInput, minPasswordLength)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("şifre hashle: %w", err)
	}

	if _, err := s.store.Create(ctx, username, string(hash)); err != nil {
		return err
	}
	return nil
}

// Login kullanıcı adı ve şifreyi doğrular, başarılıysa JWT token döner.
// Kullanıcı yok ile şifre yanlış aynı hatayı döner — bilgi sızdırmamak için.
func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.store.FindByUsername(ctx, username)
	if errors.Is(err, errorsx.ErrNotFound) {
		return "", errorsx.ErrUnauthorized
	}
	if err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", errorsx.ErrUnauthorized
	}

	return GenerateToken(user.ID, user.Username, s.jwtSecret)
}
```

- [ ] **Step 7: jwt.go yaz (Login'in bağımlılığı)**

`internal/auth/jwt.go`:
```go
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

const TokenTTL = 7 * 24 * time.Hour

type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"usr"`
	jwt.RegisteredClaims
}

func GenerateToken(userID int64, username, secret string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("token imzala: %w", err)
	}
	return signed, nil
}

func ParseToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("beklenmeyen imza yöntemi: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errorsx.ErrUnauthorized
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errorsx.ErrUnauthorized
	}
	return claims, nil
}
```

- [ ] **Step 8: Testi çalıştır, geçtiğini gör**

Run: `go test ./internal/auth/ -v`
Expected: PASS — 6 test

- [ ] **Step 9: JWT testini yaz**

`internal/auth/jwt_test.go`:
```go
package auth

import (
	"testing"

	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndParseToken(t *testing.T) {
	token, err := GenerateToken(42, "cicekci", testSecret)
	require.NoError(t, err)

	claims, err := ParseToken(token, testSecret)

	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, "cicekci", claims.Username)
}

func TestParseToken_WrongSecret(t *testing.T) {
	token, err := GenerateToken(42, "cicekci", testSecret)
	require.NoError(t, err)

	_, err = ParseToken(token, "baska-bir-secret-uzunlugu-yeterli")

	require.ErrorIs(t, err, errorsx.ErrUnauthorized)
}

func TestParseToken_Garbage(t *testing.T) {
	_, err := ParseToken("bu-token-degil", testSecret)

	require.ErrorIs(t, err, errorsx.ErrUnauthorized)
}
```

- [ ] **Step 10: Testi çalıştır**

Run: `go test ./internal/auth/ -v`
Expected: PASS — 9 test

- [ ] **Step 11: Commit**

```bash
git add pkg/errorsx/ internal/auth/
git commit -m "feat: admin auth — bcrypt, JWT, login"
```

---

## Task 6: Seed komutu

**Files:**
- Create: `cmd/seed/main.go`

**Interfaces:**
- Consumes: `config.Load`, `database.Connect`, `auth.NewStore`, `auth.NewService`, `(*auth.Service).CreateAdmin`

- [ ] **Step 1: main.go yaz**

`cmd/seed/main.go`:
```go
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/pkg/config"
	"github.com/omerkoc/cicekci/pkg/database"
	"golang.org/x/term"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("veritabanı: %v", err)
	}
	defer pool.Close()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Admin kullanıcı adı: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("okuma: %v", err)
	}
	username = strings.TrimSpace(username)

	fmt.Print("Şifre (en az 8 karakter): ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalf("şifre okuma: %v", err)
	}
	fmt.Println()
	password := strings.TrimSpace(string(passwordBytes))

	svc := auth.NewService(auth.NewStore(pool), cfg.JWTSecret)
	if err := svc.CreateAdmin(ctx, username, password); err != nil {
		log.Fatalf("admin oluşturulamadı: %v", err)
	}

	fmt.Printf("Admin kullanıcısı oluşturuldu: %s\n", username)
}
```

- [ ] **Step 2: term paketini ekle**

```bash
go get golang.org/x/term
```

- [ ] **Step 3: Elle test et**

```bash
export DATABASE_URL="postgres://cicekci:cicekci@localhost:5433/cicekci?sslmode=disable"
export JWT_SECRET="local-development-secret-32-chars!"
export WHATSAPP_NUMBER="905551234567"
export SITE_URL="http://localhost:3000"
make seed
```
Girdi: `cicekci` / `test-sifre-123`
Expected: `Admin kullanıcısı oluşturuldu: cicekci`

Doğrula:
```bash
docker compose exec -T postgres psql -U cicekci -d cicekci -c "SELECT username, left(password_hash, 7) FROM admin_users;"
```
Expected: `cicekci | $2a$10$` — şifre hash'lenmiş, düz metin değil.

- [ ] **Step 4: Aynı kullanıcıyı tekrar oluşturmayı dene**

```bash
make seed
```
Girdi: `cicekci` / `baska-sifre-123`
Expected: `admin oluşturulamadı: çakışma`

- [ ] **Step 5: Commit**

```bash
git add cmd/seed/ go.mod go.sum
git commit -m "feat: seed komutu — ilk admin kullanıcısı"
```

---

## Task 7: Kategori — model, store, service

**Files:**
- Create: `internal/category/model.go`, `internal/category/store.go`, `internal/category/service.go`
- Test: `internal/category/service_test.go`

**Interfaces:**
- Consumes: `database.NewTestDB`, `errorsx`
- Produces:
  - `category.Axis` (string tipi), sabitler: `category.AxisOccasion = "occasion"`, `category.AxisType = "type"`
  - `category.Category{ID int64, Name string, Slug string, Axis Axis, IsActive bool, IsFeatured bool, SortOrder int}`
  - `category.CreateInput{Name string, Axis Axis, IsActive bool, IsFeatured bool, SortOrder int}`
  - `category.UpdateInput{Name *string, IsActive *bool, IsFeatured *bool, SortOrder *int}`
  - `category.NewStore(pool) *Store`, `category.NewService(store) *Service`
  - `(*Service).Create(ctx, CreateInput) (*Category, error)`
  - `(*Service).Update(ctx, id int64, UpdateInput) (*Category, error)`
  - `(*Service).Delete(ctx, id int64) error`
  - `(*Service).ListPublic(ctx, axis *Axis) ([]Category, error)` — sadece `is_active=true`
  - `(*Service).ListFeatured(ctx) ([]Category, error)` — `is_active AND is_featured`
  - `(*Service).ListAdmin(ctx) ([]Category, error)` — hepsi
  - `(*Service).GetPublicBySlug(ctx, slug string) (*Category, error)`
  - `(*Service).ProductCount(ctx, id int64) (int, error)` — silme uyarısı için

- [ ] **Step 1: model.go yaz**

`internal/category/model.go`:
```go
package category

type Axis string

const (
	AxisOccasion Axis = "occasion"
	AxisType     Axis = "type"
)

func (a Axis) Valid() bool {
	return a == AxisOccasion || a == AxisType
}

type Category struct {
	ID         int64
	Name       string
	Slug       string
	Axis       Axis
	IsActive   bool
	IsFeatured bool
	SortOrder  int
}

type CreateInput struct {
	Name       string
	Axis       Axis
	IsActive   bool
	IsFeatured bool
	SortOrder  int
}

// UpdateInput alanları pointer — nil olan alan değiştirilmez (PATCH semantiği).
type UpdateInput struct {
	Name       *string
	IsActive   *bool
	IsFeatured *bool
	SortOrder  *int
}
```

`Axis` güncellenemez — kategori "occasion" olarak yaratıldıysa öyle kalır. Eksen değiştirmek ürün ilişkilerini anlamsız hale getirir; gerekirse sil-yeniden oluştur.

- [ ] **Step 2: Testi yaz**

`internal/category/service_test.go`:
```go
package category

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
	pool := database.NewTestDB(t)
	return NewService(NewStore(pool)), context.Background()
}

func TestService_Create(t *testing.T) {
	svc, ctx := newTestService(t)

	c, err := svc.Create(ctx, CreateInput{
		Name: "Doğum Günü", Axis: AxisOccasion, IsActive: true, IsFeatured: true,
	})

	require.NoError(t, err)
	assert.Equal(t, "Doğum Günü", c.Name)
	assert.Equal(t, "dogum-gunu", c.Slug)
	assert.Equal(t, AxisOccasion, c.Axis)
	assert.True(t, c.IsFeatured)
}

func TestService_Create_InvalidAxis(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.Create(ctx, CreateInput{Name: "Test", Axis: "gecersiz"})

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_EmptyName(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.Create(ctx, CreateInput{Name: "  ", Axis: AxisType})

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_DuplicateSlugGetsSuffix(t *testing.T) {
	svc, ctx := newTestService(t)

	first, err := svc.Create(ctx, CreateInput{Name: "Buket", Axis: AxisType})
	require.NoError(t, err)
	second, err := svc.Create(ctx, CreateInput{Name: "Buket", Axis: AxisOccasion})
	require.NoError(t, err)

	assert.Equal(t, "buket", first.Slug)
	assert.Equal(t, "buket-2", second.Slug)
}

func TestService_ListPublic_HidesInactive(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{Name: "Doğum Günü", Axis: AxisOccasion, IsActive: true})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{Name: "Anneler Günü", Axis: AxisOccasion, IsActive: false})
	require.NoError(t, err)

	list, err := svc.ListPublic(ctx, nil)

	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Doğum Günü", list[0].Name)
}

func TestService_ListPublic_FiltersByAxis(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{Name: "Doğum Günü", Axis: AxisOccasion, IsActive: true})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{Name: "Buket", Axis: AxisType, IsActive: true})
	require.NoError(t, err)

	occasion := AxisOccasion
	list, err := svc.ListPublic(ctx, &occasion)

	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Doğum Günü", list[0].Name)
}

// Spec §4.1: is_active=false her şeyi ezer — pasif kategori featured olsa bile görünmez.
func TestService_ListFeatured_InactiveOverridesFeatured(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{
		Name: "Anneler Günü", Axis: AxisOccasion, IsActive: false, IsFeatured: true,
	})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{
		Name: "Doğum Günü", Axis: AxisOccasion, IsActive: true, IsFeatured: true,
	})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{
		Name: "Taziye", Axis: AxisOccasion, IsActive: true, IsFeatured: false,
	})
	require.NoError(t, err)

	list, err := svc.ListFeatured(ctx)

	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Doğum Günü", list[0].Name)
}

func TestService_ListAdmin_ShowsInactive(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{Name: "Aktif", Axis: AxisType, IsActive: true})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{Name: "Pasif", Axis: AxisType, IsActive: false})
	require.NoError(t, err)

	list, err := svc.ListAdmin(ctx)

	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestService_Update_PartialFields(t *testing.T) {
	svc, ctx := newTestService(t)
	c, err := svc.Create(ctx, CreateInput{
		Name: "Buket", Axis: AxisType, IsActive: true, IsFeatured: false,
	})
	require.NoError(t, err)

	featured := true
	updated, err := svc.Update(ctx, c.ID, UpdateInput{IsFeatured: &featured})

	require.NoError(t, err)
	assert.Equal(t, "Buket", updated.Name, "isim değişmemeli")
	assert.True(t, updated.IsFeatured)
	assert.True(t, updated.IsActive, "is_active değişmemeli")
}

// Spec §4.2: kategori slug'ı isim değişince güncellenmez — kategori URL'leri sabit kalır.
func TestService_Update_NameDoesNotChangeSlug(t *testing.T) {
	svc, ctx := newTestService(t)
	c, err := svc.Create(ctx, CreateInput{Name: "Buket", Axis: AxisType, IsActive: true})
	require.NoError(t, err)

	newName := "Gül Buketi"
	updated, err := svc.Update(ctx, c.ID, UpdateInput{Name: &newName})

	require.NoError(t, err)
	assert.Equal(t, "Gül Buketi", updated.Name)
	assert.Equal(t, "buket", updated.Slug, "slug sabit kalmalı")
}

func TestService_Update_NotFound(t *testing.T) {
	svc, ctx := newTestService(t)

	name := "Yok"
	_, err := svc.Update(ctx, 9999, UpdateInput{Name: &name})

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestService_Delete(t *testing.T) {
	svc, ctx := newTestService(t)
	c, err := svc.Create(ctx, CreateInput{Name: "Silinecek", Axis: AxisType})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(ctx, c.ID))

	_, err = svc.GetPublicBySlug(ctx, "silinecek")
	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestService_Delete_NotFound(t *testing.T) {
	svc, ctx := newTestService(t)

	err := svc.Delete(ctx, 9999)

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestService_GetPublicBySlug_HidesInactive(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{Name: "Pasif", Axis: AxisType, IsActive: false})
	require.NoError(t, err)

	_, err = svc.GetPublicBySlug(ctx, "pasif")

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestService_ProductCount_Empty(t *testing.T) {
	svc, ctx := newTestService(t)
	c, err := svc.Create(ctx, CreateInput{Name: "Boş", Axis: AxisType})
	require.NoError(t, err)

	count, err := svc.ProductCount(ctx, c.ID)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
```

- [ ] **Step 3: Testi çalıştır, başarısız olduğunu gör**

Run: `go test ./internal/category/ -v`
Expected: FAIL — `undefined: NewService`

- [ ] **Step 4: store.go yaz**

`internal/category/store.go`:
```go
package category

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

const categoryColumns = `id, name, slug, axis, is_active, is_featured, sort_order`

func scanCategory(row pgx.Row) (*Category, error) {
	var c Category
	err := row.Scan(&c.ID, &c.Name, &c.Slug, &c.Axis, &c.IsActive, &c.IsFeatured, &c.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("kategori scan: %w", err)
	}
	return &c, nil
}

func (s *Store) Create(ctx context.Context, in CreateInput, slug string) (*Category, error) {
	row := s.pool.QueryRow(ctx,
		`INSERT INTO categories (name, slug, axis, is_active, is_featured, sort_order)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+categoryColumns,
		in.Name, slug, in.Axis, in.IsActive, in.IsFeatured, in.SortOrder,
	)
	return scanCategory(row)
}

// SlugExists slug çakışma kontrolü için — service -2, -3 eki eklerken kullanır.
func (s *Store) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM categories WHERE slug = $1)`, slug,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("slug kontrol: %w", err)
	}
	return exists, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Category, error) {
	return scanCategory(s.pool.QueryRow(ctx,
		`SELECT `+categoryColumns+` FROM categories WHERE id = $1`, id))
}

// GetPublicBySlug sadece aktif kategoriyi döner — is_active filtresi store'da.
func (s *Store) GetPublicBySlug(ctx context.Context, slug string) (*Category, error) {
	return scanCategory(s.pool.QueryRow(ctx,
		`SELECT `+categoryColumns+` FROM categories WHERE slug = $1 AND is_active`, slug))
}

func (s *Store) Update(ctx context.Context, id int64, in UpdateInput) (*Category, error) {
	row := s.pool.QueryRow(ctx,
		`UPDATE categories SET
		   name        = COALESCE($2, name),
		   is_active   = COALESCE($3, is_active),
		   is_featured = COALESCE($4, is_featured),
		   sort_order  = COALESCE($5, sort_order)
		 WHERE id = $1
		 RETURNING `+categoryColumns,
		id, in.Name, in.IsActive, in.IsFeatured, in.SortOrder,
	)
	return scanCategory(row)
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("kategori sil: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

func (s *Store) list(ctx context.Context, where string, args ...any) ([]Category, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+categoryColumns+` FROM categories `+where+
			` ORDER BY sort_order, name`, args...)
	if err != nil {
		return nil, fmt.Errorf("kategori listele: %w", err)
	}
	defer rows.Close()

	out := make([]Category, 0)
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Axis,
			&c.IsActive, &c.IsFeatured, &c.SortOrder); err != nil {
			return nil, fmt.Errorf("kategori scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListPublic sadece aktif kategorileri döner. axis nil ise iki eksen de gelir.
func (s *Store) ListPublic(ctx context.Context, axis *Axis) ([]Category, error) {
	if axis == nil {
		return s.list(ctx, `WHERE is_active`)
	}
	return s.list(ctx, `WHERE is_active AND axis = $1`, *axis)
}

// ListFeatured — is_active=false her şeyi ezer, featured olsa bile görünmez.
func (s *Store) ListFeatured(ctx context.Context) ([]Category, error) {
	return s.list(ctx, `WHERE is_active AND is_featured`)
}

func (s *Store) ListAdmin(ctx context.Context) ([]Category, error) {
	return s.list(ctx, ``)
}

func (s *Store) ProductCount(ctx context.Context, id int64) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM product_categories WHERE category_id = $1`, id,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ürün say: %w", err)
	}
	return count, nil
}
```

- [ ] **Step 5: service.go yaz**

`internal/category/service.go`:
```go
package category

import (
	"context"
	"fmt"
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

func (s *Service) Create(ctx context.Context, in CreateInput) (*Category, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, fmt.Errorf("%w: kategori adı boş olamaz", errorsx.ErrInvalidInput)
	}
	if !in.Axis.Valid() {
		return nil, fmt.Errorf("%w: geçersiz eksen %q (occasion veya type olmalı)",
			errorsx.ErrInvalidInput, in.Axis)
	}

	slug, err := s.uniqueSlug(ctx, product.Slugify(in.Name))
	if err != nil {
		return nil, err
	}

	return s.store.Create(ctx, in, slug)
}

// uniqueSlug çakışma varsa -2, -3 ... ekler.
func (s *Service) uniqueSlug(ctx context.Context, base string) (string, error) {
	candidate := base
	for i := 2; ; i++ {
		exists, err := s.store.SlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

// Update kısmi günceller. Slug değişmez — kategori URL'leri sabit kalmalı (spec §4.2).
func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (*Category, error) {
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: kategori adı boş olamaz", errorsx.ErrInvalidInput)
		}
		in.Name = &trimmed
	}
	return s.store.Update(ctx, id, in)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.store.Delete(ctx, id)
}

func (s *Service) ListPublic(ctx context.Context, axis *Axis) ([]Category, error) {
	if axis != nil && !axis.Valid() {
		return nil, fmt.Errorf("%w: geçersiz eksen %q", errorsx.ErrInvalidInput, *axis)
	}
	return s.store.ListPublic(ctx, axis)
}

func (s *Service) ListFeatured(ctx context.Context) ([]Category, error) {
	return s.store.ListFeatured(ctx)
}

func (s *Service) ListAdmin(ctx context.Context) ([]Category, error) {
	return s.store.ListAdmin(ctx)
}

func (s *Service) GetPublicBySlug(ctx context.Context, slug string) (*Category, error) {
	return s.store.GetPublicBySlug(ctx, slug)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Category, error) {
	return s.store.GetByID(ctx, id)
}

// ProductCount silme öncesi uyarı için — "Bu kategoride N ürün var" (spec §4.1).
func (s *Service) ProductCount(ctx context.Context, id int64) (int, error) {
	return s.store.ProductCount(ctx, id)
}
```

- [ ] **Step 6: Testi çalıştır, geçtiğini gör**

Run: `go test ./internal/category/ -v`
Expected: PASS — 15 test

- [ ] **Step 7: Commit**

```bash
git add internal/category/
git commit -m "feat: kategori CRUD — iki eksen, is_active/is_featured, slug"
```

---

## Task 8: Ürün — model, store (slug geçmişi dahil)

Bu planın en kritik parçası: slug geçmişi ve iki eksenli filtre.

**Files:**
- Create: `internal/product/model.go`, `internal/product/store.go`
- Test: `internal/product/store_test.go`

**Interfaces:**
- Consumes: `database.NewTestDB`, `errorsx`, `product.Slugify`
- Produces:
  - `product.Product{ID int64, Name string, Slug string, Description string, Price decimal.Decimal, IsActive bool, CategoryIDs []int64, CreatedAt time.Time, UpdatedAt time.Time}`
  - `product.Filter{OccasionSlug *string, TypeSlug *string, Limit int, Offset int}`
  - `product.NewStore(pool) *Store`
  - `(*Store).Create(ctx, in CreateInput, slug string) (*Product, error)`
  - `(*Store).Update(ctx, id int64, in UpdateInput) (*Product, error)`
  - `(*Store).Delete(ctx, id int64) error`
  - `(*Store).GetByID(ctx, id int64) (*Product, error)`
  - `(*Store).FindSlug(ctx, slug string) (productID int64, isCurrent bool, err error)`
  - `(*Store).GetPublicByID(ctx, id int64) (*Product, error)`
  - `(*Store).AddSlug(ctx, productID int64, slug string) error`
  - `(*Store).SlugExists(ctx, slug string) (bool, error)`
  - `(*Store).ListPublic(ctx, f Filter) ([]Product, error)`
  - `(*Store).ListAdmin(ctx, limit, offset int) ([]Product, error)`
  - `(*Store).SetCategories(ctx, productID int64, categoryIDs []int64) error`
  - `product.CreateInput{Name, Description string, Price decimal.Decimal, IsActive bool, CategoryIDs []int64}`
  - `product.UpdateInput{Name *string, Description *string, Price *decimal.Decimal, IsActive *bool, CategoryIDs []int64}`

- [ ] **Step 1: model.go yaz**

`internal/product/model.go`:
```go
package product

import (
	"time"

	"github.com/shopspring/decimal"
)

type Product struct {
	ID          int64
	Name        string
	Slug        string
	Description string
	Price       decimal.Decimal
	IsActive    bool
	CategoryIDs []int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateInput struct {
	Name        string
	Description string
	Price       decimal.Decimal
	IsActive    bool
	CategoryIDs []int64
}

// UpdateInput pointer alanlar PATCH semantiği — nil değişmez.
// CategoryIDs nil ise kategoriler değişmez; boş slice ise hepsi kaldırılır.
type UpdateInput struct {
	Name        *string
	Description *string
	Price       *decimal.Decimal
	IsActive    *bool
	CategoryIDs []int64
}

// Filter iki eksenli filtreleme. İkisi de doluysa AND — her iki koşula da
// uyan ürünler (spec §5.6).
type Filter struct {
	OccasionSlug *string
	TypeSlug     *string
	Limit        int
	Offset       int
}
```

- [ ] **Step 2: Testi yaz — slug geçmişi**

`internal/product/store_test.go`:
```go
package product

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) (*Store, *pgxpool.Pool, context.Context) {
	t.Helper()
	pool := database.NewTestDB(t)
	return NewStore(pool), pool, context.Background()
}

// insertCategory test için doğrudan kategori ekler — category paketine
// bağımlılık yaratmamak için (import cycle olurdu).
func insertCategory(t *testing.T, pool *pgxpool.Pool, name, slug, axis string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO categories (name, slug, axis, is_active)
		 VALUES ($1, $2, $3, true) RETURNING id`,
		name, slug, axis,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func price(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	require.NoError(t, err)
	return d
}

func TestStore_Create(t *testing.T) {
	store, _, ctx := newTestStore(t)

	p, err := store.Create(ctx, CreateInput{
		Name:        "51 Gül Buket",
		Description: "Kırmızı güller",
		Price:       price(t, "1850.00"),
		IsActive:    true,
	}, "51-gul-buket")

	require.NoError(t, err)
	assert.Equal(t, "51 Gül Buket", p.Name)
	assert.Equal(t, "51-gul-buket", p.Slug)
	assert.True(t, p.Price.Equal(price(t, "1850.00")))
	assert.True(t, p.IsActive)
}

func TestStore_Create_PriceKeepsPrecision(t *testing.T) {
	store, _, ctx := newTestStore(t)

	p, err := store.Create(ctx, CreateInput{
		Name: "Test", Price: price(t, "1234.56"), IsActive: true,
	}, "test")

	require.NoError(t, err)
	assert.Equal(t, "1234.56", p.Price.StringFixed(2))
}

func TestStore_FindSlug_Current(t *testing.T) {
	store, _, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "100"), IsActive: true,
	}, "buket")
	require.NoError(t, err)

	id, isCurrent, err := store.FindSlug(ctx, "buket")

	require.NoError(t, err)
	assert.Equal(t, p.ID, id)
	assert.True(t, isCurrent)
}

func TestStore_FindSlug_NotFound(t *testing.T) {
	store, _, ctx := newTestStore(t)

	_, _, err := store.FindSlug(ctx, "yok-boyle-bir-slug")

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

// Spec §4.2: isim değişince eski slug is_current=false olur, yeni slug eklenir.
// Eski slug'a gelen istek 301 ile yönlendirilir.
func TestStore_AddSlug_OldSlugStillResolves(t *testing.T) {
	store, _, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "51 Gül Buket", Price: price(t, "1850"), IsActive: true,
	}, "51-gul-buket")
	require.NoError(t, err)

	require.NoError(t, store.AddSlug(ctx, p.ID, "51-kirmizi-gul-buketi"))

	oldID, oldIsCurrent, err := store.FindSlug(ctx, "51-gul-buket")
	require.NoError(t, err)
	assert.Equal(t, p.ID, oldID)
	assert.False(t, oldIsCurrent, "eski slug is_current=false olmalı")

	newID, newIsCurrent, err := store.FindSlug(ctx, "51-kirmizi-gul-buketi")
	require.NoError(t, err)
	assert.Equal(t, p.ID, newID)
	assert.True(t, newIsCurrent, "yeni slug güncel olmalı")
}

func TestStore_AddSlug_ProductSlugFieldReflectsCurrent(t *testing.T) {
	store, _, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "100"), IsActive: true,
	}, "buket")
	require.NoError(t, err)

	require.NoError(t, store.AddSlug(ctx, p.ID, "gul-buketi"))

	fetched, err := store.GetByID(ctx, p.ID)

	require.NoError(t, err)
	assert.Equal(t, "gul-buketi", fetched.Slug, "GetByID güncel slug'ı dönmeli")
}

func TestStore_SlugExists(t *testing.T) {
	store, _, ctx := newTestStore(t)
	_, err := store.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "100"), IsActive: true,
	}, "buket")
	require.NoError(t, err)

	exists, err := store.SlugExists(ctx, "buket")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = store.SlugExists(ctx, "yok")
	require.NoError(t, err)
	assert.False(t, exists)
}

// Eski slug da çakışma sayılır — aynı slug iki ürüne verilemez.
func TestStore_SlugExists_IncludesOldSlugs(t *testing.T) {
	store, _, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "100"), IsActive: true,
	}, "buket")
	require.NoError(t, err)
	require.NoError(t, store.AddSlug(ctx, p.ID, "gul-buketi"))

	exists, err := store.SlugExists(ctx, "buket")

	require.NoError(t, err)
	assert.True(t, exists, "eski slug hâlâ rezerve olmalı")
}

func TestStore_GetPublicByID_HidesInactive(t *testing.T) {
	store, _, ctx := newTestStore(t)
	p, err := store.Create(ctx, CreateInput{
		Name: "Pasif", Price: price(t, "100"), IsActive: false,
	}, "pasif")
	require.NoError(t, err)

	_, err = store.GetPublicByID(ctx, p.ID)

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestStore_SetCategories(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	dogumGunu := insertCategory(t, pool, "Doğum Günü", "dogum-gunu", "occasion")
	buket := insertCategory(t, pool, "Buket", "buket", "type")
	p, err := store.Create(ctx, CreateInput{
		Name: "Test", Price: price(t, "100"), IsActive: true,
	}, "test")
	require.NoError(t, err)

	require.NoError(t, store.SetCategories(ctx, p.ID, []int64{dogumGunu, buket}))

	fetched, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int64{dogumGunu, buket}, fetched.CategoryIDs)
}

func TestStore_SetCategories_ReplacesExisting(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	a := insertCategory(t, pool, "A", "a", "occasion")
	b := insertCategory(t, pool, "B", "b", "occasion")
	p, err := store.Create(ctx, CreateInput{
		Name: "Test", Price: price(t, "100"), IsActive: true,
	}, "test")
	require.NoError(t, err)
	require.NoError(t, store.SetCategories(ctx, p.ID, []int64{a}))

	require.NoError(t, store.SetCategories(ctx, p.ID, []int64{b}))

	fetched, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, []int64{b}, fetched.CategoryIDs)
}

func TestStore_SetCategories_Empty(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	a := insertCategory(t, pool, "A", "a", "occasion")
	p, err := store.Create(ctx, CreateInput{
		Name: "Test", Price: price(t, "100"), IsActive: true,
	}, "test")
	require.NoError(t, err)
	require.NoError(t, store.SetCategories(ctx, p.ID, []int64{a}))

	require.NoError(t, store.SetCategories(ctx, p.ID, []int64{}))

	fetched, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Empty(t, fetched.CategoryIDs)
}

func TestStore_Delete_CascadesSlugsAndCategories(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	cat := insertCategory(t, pool, "A", "a", "occasion")
	p, err := store.Create(ctx, CreateInput{
		Name: "Test", Price: price(t, "100"), IsActive: true,
	}, "test")
	require.NoError(t, err)
	require.NoError(t, store.SetCategories(ctx, p.ID, []int64{cat}))
	require.NoError(t, store.AddSlug(ctx, p.ID, "test-2"))

	require.NoError(t, store.Delete(ctx, p.ID))

	_, _, err = store.FindSlug(ctx, "test")
	assert.ErrorIs(t, err, errorsx.ErrNotFound, "slug geçmişi de silinmeli")

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM product_categories WHERE product_id = $1`, p.ID).Scan(&count))
	assert.Equal(t, 0, count, "junction kayıtları silinmeli")
}

// Kategori silinince ürün silinmez, sadece bağ kopar (spec §4.1).
func TestStore_CategoryDelete_KeepsProduct(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	cat := insertCategory(t, pool, "Silinecek", "silinecek", "occasion")
	p, err := store.Create(ctx, CreateInput{
		Name: "Ürün", Price: price(t, "100"), IsActive: true,
	}, "urun")
	require.NoError(t, err)
	require.NoError(t, store.SetCategories(ctx, p.ID, []int64{cat}))

	_, err = pool.Exec(ctx, `DELETE FROM categories WHERE id = $1`, cat)
	require.NoError(t, err)

	fetched, err := store.GetByID(ctx, p.ID)
	require.NoError(t, err, "ürün silinmemeli")
	assert.Empty(t, fetched.CategoryIDs, "sadece bağ kopmalı")
}
```

- [ ] **Step 3: Testi çalıştır, başarısız olduğunu gör**

Run: `go test ./internal/product/ -run TestStore -v`
Expected: FAIL — `undefined: NewStore`

- [ ] **Step 4: store.go yaz — temel CRUD ve slug**

`internal/product/store.go`:
```go
package product

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

// productSelect ürünü güncel slug'ı ve kategori id'leriyle birlikte çeker.
const productSelect = `
	SELECT p.id, p.name,
	       COALESCE(ps.slug, ''),
	       p.description, p.price, p.is_active,
	       COALESCE(
	         (SELECT array_agg(pc.category_id ORDER BY pc.category_id)
	          FROM product_categories pc WHERE pc.product_id = p.id),
	         '{}'
	       ),
	       p.created_at, p.updated_at
	FROM products p
	LEFT JOIN product_slugs ps ON ps.product_id = p.id AND ps.is_current
`

func scanProduct(row pgx.Row) (*Product, error) {
	var p Product
	err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price,
		&p.IsActive, &p.CategoryIDs, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("ürün scan: %w", err)
	}
	return &p, nil
}

// Create ürünü ve ilk slug'ını tek transaction'da yazar.
func (s *Store) Create(ctx context.Context, in CreateInput, slug string) (*Product, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO products (name, description, price, is_active)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		in.Name, in.Description, in.Price, in.IsActive,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("ürün ekle: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO product_slugs (slug, product_id, is_current) VALUES ($1, $2, true)`,
		slug, id)
	if err != nil {
		return nil, fmt.Errorf("slug ekle: %w", err)
	}

	if len(in.CategoryIDs) > 0 {
		if err := insertCategories(ctx, tx, id, in.CategoryIDs); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx commit: %w", err)
	}

	return s.GetByID(ctx, id)
}

func insertCategories(ctx context.Context, tx pgx.Tx, productID int64, categoryIDs []int64) error {
	for _, cid := range categoryIDs {
		_, err := tx.Exec(ctx,
			`INSERT INTO product_categories (product_id, category_id) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`,
			productID, cid)
		if err != nil {
			return fmt.Errorf("kategori bağla: %w", err)
		}
	}
	return nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Product, error) {
	return scanProduct(s.pool.QueryRow(ctx, productSelect+` WHERE p.id = $1`, id))
}

// GetPublicByID sadece aktif ürünü döner — is_active filtresi store'da (spec §4.6).
func (s *Store) GetPublicByID(ctx context.Context, id int64) (*Product, error) {
	return scanProduct(s.pool.QueryRow(ctx,
		productSelect+` WHERE p.id = $1 AND p.is_active`, id))
}

// FindSlug slug'ın hangi ürüne ait olduğunu ve güncel olup olmadığını döner.
// isCurrent=false ise handler 301 redirect yapmalı (spec §4.2).
func (s *Store) FindSlug(ctx context.Context, slug string) (int64, bool, error) {
	var productID int64
	var isCurrent bool
	err := s.pool.QueryRow(ctx,
		`SELECT product_id, is_current FROM product_slugs WHERE slug = $1`, slug,
	).Scan(&productID, &isCurrent)

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, errorsx.ErrNotFound
	}
	if err != nil {
		return 0, false, fmt.Errorf("slug ara: %w", err)
	}
	return productID, isCurrent, nil
}

// AddSlug yeni slug ekler ve eskisini is_current=false yapar.
// Partial unique index bir üründe tek güncel slug garantiler.
func (s *Store) AddSlug(ctx context.Context, productID int64, slug string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE product_slugs SET is_current = false
		 WHERE product_id = $1 AND is_current`, productID)
	if err != nil {
		return fmt.Errorf("eski slug pasifle: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO product_slugs (slug, product_id, is_current) VALUES ($1, $2, true)`,
		slug, productID)
	if err != nil {
		return fmt.Errorf("yeni slug ekle: %w", err)
	}

	return tx.Commit(ctx)
}

// SlugExists eski slug'ları da kapsar — bir slug bir kez kullanıldıysa
// başka ürüne verilemez.
func (s *Store) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM product_slugs WHERE slug = $1)`, slug,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("slug kontrol: %w", err)
	}
	return exists, nil
}

func (s *Store) Update(ctx context.Context, id int64, in UpdateInput) (*Product, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx,
		`UPDATE products SET
		   name        = COALESCE($2, name),
		   description = COALESCE($3, description),
		   price       = COALESCE($4, price),
		   is_active   = COALESCE($5, is_active),
		   updated_at  = now()
		 WHERE id = $1`,
		id, in.Name, in.Description, in.Price, in.IsActive)
	if err != nil {
		return nil, fmt.Errorf("ürün güncelle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, errorsx.ErrNotFound
	}

	// CategoryIDs nil ise dokunma; boş slice ise hepsini kaldır.
	if in.CategoryIDs != nil {
		if _, err := tx.Exec(ctx,
			`DELETE FROM product_categories WHERE product_id = $1`, id); err != nil {
			return nil, fmt.Errorf("kategori temizle: %w", err)
		}
		if err := insertCategories(ctx, tx, id, in.CategoryIDs); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx commit: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("ürün sil: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

// SetCategories ürünün kategorilerini tamamen değiştirir.
func (s *Store) SetCategories(ctx context.Context, productID int64, categoryIDs []int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM product_categories WHERE product_id = $1`, productID); err != nil {
		return fmt.Errorf("kategori temizle: %w", err)
	}

	if err := insertCategories(ctx, tx, productID, categoryIDs); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
```

- [ ] **Step 5: Testi çalıştır**

Run: `go test ./internal/product/ -run TestStore -v`
Expected: PASS — `ListPublic`/`ListAdmin` testleri henüz yok, onlar Step 6'da

- [ ] **Step 6: Filtre testini yaz**

`internal/product/store_filter_test.go`:
```go
package product

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Spec §5.6: iki eksen de seçiliyse AND — her iki koşula da uyan ürünler.
// AND/OR karışması bu tür sorgularda klasik bir hatadır (spec §6).
func TestStore_ListPublic_TwoAxisFilterIsAND(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	dogumGunu := insertCategory(t, pool, "Doğum Günü", "dogum-gunu", "occasion")
	taziye := insertCategory(t, pool, "Taziye", "taziye", "occasion")
	buket := insertCategory(t, pool, "Buket", "buket", "type")
	orkide := insertCategory(t, pool, "Orkide", "orkide", "type")

	// Doğum Günü + Buket — eşleşmeli
	match, err := store.Create(ctx, CreateInput{
		Name: "Doğum Günü Buketi", Price: price(t, "500"), IsActive: true,
		CategoryIDs: []int64{dogumGunu, buket},
	}, "dogum-gunu-buketi")
	require.NoError(t, err)

	// Doğum Günü + Orkide — tip uymuyor
	_, err = store.Create(ctx, CreateInput{
		Name: "Doğum Günü Orkidesi", Price: price(t, "800"), IsActive: true,
		CategoryIDs: []int64{dogumGunu, orkide},
	}, "dogum-gunu-orkidesi")
	require.NoError(t, err)

	// Taziye + Buket — amaç uymuyor
	_, err = store.Create(ctx, CreateInput{
		Name: "Taziye Buketi", Price: price(t, "600"), IsActive: true,
		CategoryIDs: []int64{taziye, buket},
	}, "taziye-buketi")
	require.NoError(t, err)

	occasion, typ := "dogum-gunu", "buket"
	list, err := store.ListPublic(ctx, Filter{
		OccasionSlug: &occasion, TypeSlug: &typ, Limit: 20,
	})

	require.NoError(t, err)
	require.Len(t, list, 1, "sadece iki koşula da uyan ürün gelmeli (AND, OR değil)")
	assert.Equal(t, match.ID, list[0].ID)
}

func TestStore_ListPublic_SingleAxisFilter(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	dogumGunu := insertCategory(t, pool, "Doğum Günü", "dogum-gunu", "occasion")
	buket := insertCategory(t, pool, "Buket", "buket", "type")
	orkide := insertCategory(t, pool, "Orkide", "orkide", "type")

	_, err := store.Create(ctx, CreateInput{
		Name: "A", Price: price(t, "100"), IsActive: true,
		CategoryIDs: []int64{dogumGunu, buket},
	}, "a")
	require.NoError(t, err)
	_, err = store.Create(ctx, CreateInput{
		Name: "B", Price: price(t, "100"), IsActive: true,
		CategoryIDs: []int64{dogumGunu, orkide},
	}, "b")
	require.NoError(t, err)

	occasion := "dogum-gunu"
	list, err := store.ListPublic(ctx, Filter{OccasionSlug: &occasion, Limit: 20})

	require.NoError(t, err)
	assert.Len(t, list, 2, "tek eksen filtresi ikisini de getirmeli")
}

func TestStore_ListPublic_NoFilter(t *testing.T) {
	store, _, ctx := newTestStore(t)
	_, err := store.Create(ctx, CreateInput{
		Name: "A", Price: price(t, "100"), IsActive: true,
	}, "a")
	require.NoError(t, err)
	_, err = store.Create(ctx, CreateInput{
		Name: "B", Price: price(t, "100"), IsActive: true,
	}, "b")
	require.NoError(t, err)

	list, err := store.ListPublic(ctx, Filter{Limit: 20})

	require.NoError(t, err)
	assert.Len(t, list, 2)
}

// Spec §6: is_active sızıntısı regresyon testi.
func TestStore_ListPublic_HidesInactive(t *testing.T) {
	store, _, ctx := newTestStore(t)
	_, err := store.Create(ctx, CreateInput{
		Name: "Aktif", Price: price(t, "100"), IsActive: true,
	}, "aktif")
	require.NoError(t, err)
	_, err = store.Create(ctx, CreateInput{
		Name: "Pasif", Price: price(t, "100"), IsActive: false,
	}, "pasif")
	require.NoError(t, err)

	list, err := store.ListPublic(ctx, Filter{Limit: 20})

	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Aktif", list[0].Name)
}

// Pasif kategoriye bağlı ürün, o kategori filtresiyle gelmemeli.
func TestStore_ListPublic_InactiveCategoryNotFilterable(t *testing.T) {
	store, pool, ctx := newTestStore(t)
	var catID int64
	err := pool.QueryRow(ctx,
		`INSERT INTO categories (name, slug, axis, is_active)
		 VALUES ('Anneler Günü', 'anneler-gunu', 'occasion', false) RETURNING id`,
	).Scan(&catID)
	require.NoError(t, err)

	_, err = store.Create(ctx, CreateInput{
		Name: "Anneler Günü Buketi", Price: price(t, "500"), IsActive: true,
		CategoryIDs: []int64{catID},
	}, "anneler-gunu-buketi")
	require.NoError(t, err)

	slug := "anneler-gunu"
	list, err := store.ListPublic(ctx, Filter{OccasionSlug: &slug, Limit: 20})

	require.NoError(t, err)
	assert.Empty(t, list, "pasif kategori filtresi sonuç döndürmemeli")
}

func TestStore_ListPublic_Pagination(t *testing.T) {
	store, _, ctx := newTestStore(t)
	for _, name := range []string{"A", "B", "C"} {
		_, err := store.Create(ctx, CreateInput{
			Name: name, Price: price(t, "100"), IsActive: true,
		}, Slugify(name))
		require.NoError(t, err)
	}

	first, err := store.ListPublic(ctx, Filter{Limit: 2, Offset: 0})
	require.NoError(t, err)
	second, err := store.ListPublic(ctx, Filter{Limit: 2, Offset: 2})
	require.NoError(t, err)

	assert.Len(t, first, 2)
	assert.Len(t, second, 1)
}

func TestStore_ListAdmin_ShowsInactive(t *testing.T) {
	store, _, ctx := newTestStore(t)
	_, err := store.Create(ctx, CreateInput{
		Name: "Aktif", Price: price(t, "100"), IsActive: true,
	}, "aktif")
	require.NoError(t, err)
	_, err = store.Create(ctx, CreateInput{
		Name: "Pasif", Price: price(t, "100"), IsActive: false,
	}, "pasif")
	require.NoError(t, err)

	list, err := store.ListAdmin(ctx, 20, 0)

	require.NoError(t, err)
	assert.Len(t, list, 2)
}
```

- [ ] **Step 7: Testi çalıştır, başarısız olduğunu gör**

Run: `go test ./internal/product/ -run TestStore_List -v`
Expected: FAIL — `store.ListPublic undefined`

- [ ] **Step 8: store.go'ya filtre metodlarını ekle**

`internal/product/store.go` dosyasının sonuna:
```go
// ListPublic aktif ürünleri filtreyle listeler.
// İki eksen de doluysa AND — her iki koşula da uyan ürünler (spec §5.6).
// Pasif kategoriler filtrede eşleşmez.
func (s *Store) ListPublic(ctx context.Context, f Filter) ([]Product, error) {
	query := productSelect + ` WHERE p.is_active`
	args := []any{}
	argN := 1

	if f.OccasionSlug != nil {
		query += fmt.Sprintf(`
			AND EXISTS (
				SELECT 1 FROM product_categories pc
				JOIN categories c ON c.id = pc.category_id
				WHERE pc.product_id = p.id
				  AND c.slug = $%d AND c.axis = 'occasion' AND c.is_active
			)`, argN)
		args = append(args, *f.OccasionSlug)
		argN++
	}

	if f.TypeSlug != nil {
		query += fmt.Sprintf(`
			AND EXISTS (
				SELECT 1 FROM product_categories pc
				JOIN categories c ON c.id = pc.category_id
				WHERE pc.product_id = p.id
				  AND c.slug = $%d AND c.axis = 'type' AND c.is_active
			)`, argN)
		args = append(args, *f.TypeSlug)
		argN++
	}

	query += fmt.Sprintf(` ORDER BY p.created_at DESC, p.id DESC LIMIT $%d OFFSET $%d`,
		argN, argN+1)
	args = append(args, f.Limit, f.Offset)

	return s.queryProducts(ctx, query, args...)
}

func (s *Store) ListAdmin(ctx context.Context, limit, offset int) ([]Product, error) {
	return s.queryProducts(ctx,
		productSelect+` ORDER BY p.created_at DESC, p.id DESC LIMIT $1 OFFSET $2`,
		limit, offset)
}

func (s *Store) queryProducts(ctx context.Context, query string, args ...any) ([]Product, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ürün listele: %w", err)
	}
	defer rows.Close()

	out := make([]Product, 0)
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price,
			&p.IsActive, &p.CategoryIDs, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ürün scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
```

İki ayrı `EXISTS` alt sorgusu, `AND` ile bağlı — bu AND semantiğinin kaynağı. Tek bir `JOIN ... WHERE c.slug IN (...)` yazılsaydı OR olurdu ve test bunu yakalar.

- [ ] **Step 9: Testi çalıştır**

Run: `go test ./internal/product/ -v`
Expected: PASS — tüm store testleri

- [ ] **Step 10: Commit**

```bash
git add internal/product/
git commit -m "feat: ürün store — slug geçmişi, iki eksenli AND filtre"
```

---

## Task 9: Ürün service

**Files:**
- Create: `internal/product/service.go`
- Test: `internal/product/service_test.go`

**Interfaces:**
- Consumes: `product.Store`, `product.Slugify`
- Produces:
  - `product.NewService(store *Store) *Service`
  - `(*Service).Create(ctx, CreateInput) (*Product, error)`
  - `(*Service).Update(ctx, id int64, UpdateInput) (*Product, error)` — isim değişince slug geçmişi yönetir
  - `(*Service).Delete(ctx, id int64) error`
  - `(*Service).GetPublicBySlug(ctx, slug string) (p *Product, redirectTo string, err error)` — `redirectTo != ""` ise 301
  - `(*Service).GetByID(ctx, id int64) (*Product, error)`
  - `(*Service).ListPublic(ctx, Filter) ([]Product, error)`
  - `(*Service).ListAdmin(ctx, limit, offset int) ([]Product, error)`

- [ ] **Step 1: Testi yaz**

`internal/product/service_test.go`:
```go
package product

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestProductService(t *testing.T) (*Service, *pgxpool.Pool, context.Context) {
	t.Helper()
	pool := database.NewTestDB(t)
	return NewService(NewStore(pool)), pool, context.Background()
}

func TestProductService_Create_GeneratesSlug(t *testing.T) {
	svc, _, ctx := newTestProductService(t)

	p, err := svc.Create(ctx, CreateInput{
		Name: "51 Gül Buket", Price: price(t, "1850"), IsActive: true,
	})

	require.NoError(t, err)
	assert.Equal(t, "51-gul-buket", p.Slug)
}

func TestProductService_Create_DuplicateNameGetsSuffix(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	_, err := svc.Create(ctx, CreateInput{
		Name: "Gül Buketi", Price: price(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	second, err := svc.Create(ctx, CreateInput{
		Name: "Gül Buketi", Price: price(t, "700"), IsActive: true,
	})

	require.NoError(t, err)
	assert.Equal(t, "gul-buketi-2", second.Slug)
}

func TestProductService_Create_EmptyName(t *testing.T) {
	svc, _, ctx := newTestProductService(t)

	_, err := svc.Create(ctx, CreateInput{Name: "   ", Price: price(t, "100")})

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestProductService_Create_NegativePrice(t *testing.T) {
	svc, _, ctx := newTestProductService(t)

	_, err := svc.Create(ctx, CreateInput{Name: "Test", Price: price(t, "-5")})

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

// Spec §4.2: isim değişince yeni slug üretilir, eskisi 301 için saklanır.
func TestProductService_Update_NameCreatesNewSlugAndKeepsOld(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	p, err := svc.Create(ctx, CreateInput{
		Name: "51 Gül Buket", Price: price(t, "1850"), IsActive: true,
	})
	require.NoError(t, err)

	newName := "51 Kırmızı Gül Buketi"
	updated, err := svc.Update(ctx, p.ID, UpdateInput{Name: &newName})

	require.NoError(t, err)
	assert.Equal(t, "51-kirmizi-gul-buketi", updated.Slug)

	// Eski slug 301 ile yönlendirmeli
	fetched, redirectTo, err := svc.GetPublicBySlug(ctx, "51-gul-buket")
	require.NoError(t, err)
	assert.Equal(t, "51-kirmizi-gul-buketi", redirectTo, "eski slug redirect vermeli")
	assert.Equal(t, p.ID, fetched.ID)
}

func TestProductService_Update_SameNameDoesNotCreateNewSlug(t *testing.T) {
	svc, pool, ctx := newTestProductService(t)
	p, err := svc.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	sameName := "Buket"
	_, err = svc.Update(ctx, p.ID, UpdateInput{Name: &sameName})
	require.NoError(t, err)

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM product_slugs WHERE product_id = $1`, p.ID).Scan(&count))
	assert.Equal(t, 1, count, "aynı isim yeni slug üretmemeli")
}

func TestProductService_Update_PriceOnlyDoesNotTouchSlug(t *testing.T) {
	svc, pool, ctx := newTestProductService(t)
	p, err := svc.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	newPrice := price(t, "600")
	updated, err := svc.Update(ctx, p.ID, UpdateInput{Price: &newPrice})

	require.NoError(t, err)
	assert.Equal(t, "buket", updated.Slug)
	assert.Equal(t, "600.00", updated.Price.StringFixed(2))

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM product_slugs WHERE product_id = $1`, p.ID).Scan(&count))
	assert.Equal(t, 1, count)
}

// Eski isme geri dönülürse: eski slug rezerve olduğu için -2 eki alır.
func TestProductService_Update_RevertToOldNameGetsSuffix(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	p, err := svc.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	changed := "Gül Buketi"
	_, err = svc.Update(ctx, p.ID, UpdateInput{Name: &changed})
	require.NoError(t, err)

	reverted := "Buket"
	updated, err := svc.Update(ctx, p.ID, UpdateInput{Name: &reverted})

	require.NoError(t, err)
	assert.Equal(t, "buket-2", updated.Slug, "eski slug rezerve, -2 almalı")
}

func TestProductService_GetPublicBySlug_CurrentNoRedirect(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	_, err := svc.Create(ctx, CreateInput{
		Name: "Buket", Price: price(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	p, redirectTo, err := svc.GetPublicBySlug(ctx, "buket")

	require.NoError(t, err)
	assert.Empty(t, redirectTo, "güncel slug redirect vermemeli")
	assert.Equal(t, "Buket", p.Name)
}

func TestProductService_GetPublicBySlug_NotFound(t *testing.T) {
	svc, _, ctx := newTestProductService(t)

	_, _, err := svc.GetPublicBySlug(ctx, "olmayan-urun")

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

// Pasif ürünün slug'ı çözülse bile public'te 404.
func TestProductService_GetPublicBySlug_InactiveIsNotFound(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	_, err := svc.Create(ctx, CreateInput{
		Name: "Pasif Ürün", Price: price(t, "500"), IsActive: false,
	})
	require.NoError(t, err)

	_, _, err = svc.GetPublicBySlug(ctx, "pasif-urun")

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestProductService_Delete(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	p, err := svc.Create(ctx, CreateInput{
		Name: "Silinecek", Price: price(t, "100"), IsActive: true,
	})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(ctx, p.ID))

	_, _, err = svc.GetPublicBySlug(ctx, "silinecek")
	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestProductService_ListPublic_DefaultLimit(t *testing.T) {
	svc, _, ctx := newTestProductService(t)
	_, err := svc.Create(ctx, CreateInput{
		Name: "A", Price: price(t, "100"), IsActive: true,
	})
	require.NoError(t, err)

	list, err := svc.ListPublic(ctx, Filter{})

	require.NoError(t, err)
	assert.Len(t, list, 1, "limit 0 verilse bile varsayılan uygulanmalı")
}
```

- [ ] **Step 2: Testi çalıştır, başarısız olduğunu gör**

Run: `go test ./internal/product/ -run TestProductService -v`
Expected: FAIL — `undefined: NewService`

- [ ] **Step 3: service.go yaz**

`internal/product/service.go`:
```go
package product

import (
	"context"
	"fmt"
	"strings"

	"github.com/omerkoc/cicekci/pkg/errorsx"
)

const (
	defaultLimit = 24
	maxLimit     = 100
)

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*Product, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, fmt.Errorf("%w: ürün adı boş olamaz", errorsx.ErrInvalidInput)
	}
	if in.Price.IsNegative() {
		return nil, fmt.Errorf("%w: fiyat negatif olamaz", errorsx.ErrInvalidInput)
	}

	slug, err := s.uniqueSlug(ctx, Slugify(in.Name))
	if err != nil {
		return nil, err
	}

	return s.store.Create(ctx, in, slug)
}

// uniqueSlug çakışma varsa -2, -3 ... ekler. Eski slug'lar da çakışma sayılır.
func (s *Service) uniqueSlug(ctx context.Context, base string) (string, error) {
	candidate := base
	for i := 2; ; i++ {
		exists, err := s.store.SlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

// Update ürünü günceller. İsim değişirse yeni slug üretilir ve eskisi
// is_current=false olarak saklanır — eski linkler 301 ile yaşamaya devam
// eder (spec §4.2).
func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (*Product, error) {
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: ürün adı boş olamaz", errorsx.ErrInvalidInput)
		}
		in.Name = &trimmed
	}
	if in.Price != nil && in.Price.IsNegative() {
		return nil, fmt.Errorf("%w: fiyat negatif olamaz", errorsx.ErrInvalidInput)
	}

	current, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updated, err := s.store.Update(ctx, id, in)
	if err != nil {
		return nil, err
	}

	// İsim değiştiyse ve yeni slug farklıysa slug geçmişini güncelle.
	if in.Name != nil && *in.Name != current.Name {
		newSlugBase := Slugify(*in.Name)
		if newSlugBase != current.Slug {
			newSlug, err := s.uniqueSlug(ctx, newSlugBase)
			if err != nil {
				return nil, err
			}
			if err := s.store.AddSlug(ctx, id, newSlug); err != nil {
				return nil, err
			}
			updated, err = s.store.GetByID(ctx, id)
			if err != nil {
				return nil, err
			}
		}
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.store.Delete(ctx, id)
}

// GetPublicBySlug ürünü slug'dan bulur. Slug eskiyse redirectTo dolu döner
// ve handler 301 yapmalıdır. Pasif ürün ErrNotFound döner.
func (s *Service) GetPublicBySlug(ctx context.Context, slug string) (*Product, string, error) {
	productID, isCurrent, err := s.store.FindSlug(ctx, slug)
	if err != nil {
		return nil, "", err
	}

	p, err := s.store.GetPublicByID(ctx, productID)
	if err != nil {
		return nil, "", err
	}

	if !isCurrent {
		return p, p.Slug, nil
	}
	return p, "", nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Product, error) {
	return s.store.GetByID(ctx, id)
}

func (s *Service) ListPublic(ctx context.Context, f Filter) ([]Product, error) {
	f.Limit = normalizeLimit(f.Limit)
	if f.Offset < 0 {
		f.Offset = 0
	}
	return s.store.ListPublic(ctx, f)
}

func (s *Service) ListAdmin(ctx context.Context, limit, offset int) ([]Product, error) {
	limit = normalizeLimit(limit)
	if offset < 0 {
		offset = 0
	}
	return s.store.ListAdmin(ctx, limit, offset)
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}
```

- [ ] **Step 4: Testi çalıştır**

Run: `go test ./internal/product/ -v`
Expected: PASS — tüm ürün testleri

- [ ] **Step 5: Commit**

```bash
git add internal/product/service.go internal/product/service_test.go
git commit -m "feat: ürün service — slug geçmişi yönetimi, doğrulama"
```

---

## Task 10: JWT middleware ve hata çevirici

**Files:**
- Create: `internal/auth/middleware.go`
- Create: `internal/api/httperr.go`
- Test: `internal/auth/middleware_test.go`

**Interfaces:**
- Consumes: `auth.ParseToken`, `errorsx`
- Produces:
  - `auth.CookieName = "cicekci_token"`
  - `auth.Middleware(secret string) fiber.Handler`
  - `api.WriteError(c *fiber.Ctx, err error) error` — domain hatasını HTTP'ye çevirir

- [ ] **Step 1: httperr.go yaz**

`internal/api/httperr.go`:
```go
package api

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// WriteError domain hatasını HTTP yanıtına çevirir.
// Bilinmeyen hatalar 500 olur ve detayı sızmaz — sadece log'a düşer.
func WriteError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, errorsx.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error: ErrorBody{Code: "not_found", Message: "Kayıt bulunamadı"},
		})
	case errors.Is(err, errorsx.ErrInvalidInput):
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: ErrorBody{Code: "invalid_input", Message: err.Error()},
		})
	case errors.Is(err, errorsx.ErrUnauthorized):
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error: ErrorBody{Code: "unauthorized", Message: "Yetkisiz"},
		})
	case errors.Is(err, errorsx.ErrConflict):
		return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
			Error: ErrorBody{Code: "conflict", Message: "Bu kayıt zaten var"},
		})
	default:
		log.Printf("beklenmeyen hata: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: ErrorBody{Code: "internal", Message: "Sunucu hatası"},
		})
	}
}
```

- [ ] **Step 2: Testi yaz**

`internal/auth/middleware_test.go`:
```go
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Get("/korumali", Middleware(testSecret), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"username": c.Locals("username")})
	})
	return app
}

func TestMiddleware_NoCookie(t *testing.T) {
	app := newTestApp(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/korumali", nil))

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMiddleware_ValidToken(t *testing.T) {
	app := newTestApp(t)
	token, err := GenerateToken(1, "cicekci", testSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/korumali", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestMiddleware_InvalidToken(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/korumali", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "bu-token-degil"})
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMiddleware_TokenSignedWithOtherSecret(t *testing.T) {
	app := newTestApp(t)
	token, err := GenerateToken(1, "saldirgan", "baska-secret-yeterince-uzun-32ch")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/korumali", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
```

- [ ] **Step 3: Testi çalıştır, başarısız olduğunu gör**

Run: `go test ./internal/auth/ -run TestMiddleware -v`
Expected: FAIL — `undefined: Middleware`

- [ ] **Step 4: middleware.go yaz**

`internal/auth/middleware.go`:
```go
package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// CookieName — JWT HttpOnly cookie'de tutulur, localStorage'da değil (spec §4.5).
const CookieName = "cicekci_token"

// Middleware JWT cookie'sini doğrular. Geçerliyse Locals'a user bilgisi koyar.
func Middleware(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies(CookieName)
		if token == "" {
			return api.WriteError(c, errorsx.ErrUnauthorized)
		}

		claims, err := ParseToken(token, secret)
		if err != nil {
			return api.WriteError(c, errorsx.ErrUnauthorized)
		}

		c.Locals("userID", claims.UserID)
		c.Locals("username", claims.Username)
		return c.Next()
	}
}
```

- [ ] **Step 5: Testi çalıştır**

Run: `go test ./internal/auth/ -v`
Expected: PASS — 13 test

- [ ] **Step 6: Commit**

```bash
git add internal/auth/middleware.go internal/auth/middleware_test.go internal/api/httperr.go
git commit -m "feat: JWT middleware ve HTTP hata çevirici"
```

---

## Task 11: Public API — kategori ve ürün handler'ları

**Files:**
- Create: `internal/api/app/category_view.go`, `internal/api/app/category_handler.go`
- Create: `internal/api/app/product_view.go`, `internal/api/app/product_handler.go`
- Create: `internal/api/app/router.go`
- Test: `internal/api/app/product_handler_test.go`

**Interfaces:**
- Consumes: `category.Service`, `product.Service`, `api.WriteError`
- Produces:
  - `app.CategoryView{ID int64, Name string, Slug string, Axis string}`
  - `app.ProductView{ID int64, Name string, Slug string, Description string, Price string, CategoryIDs []int64}`
  - `app.Register(router fiber.Router, catSvc *category.Service, prodSvc *product.Service)`

- [ ] **Step 1: category_view.go yaz**

`internal/api/app/category_view.go`:
```go
package app

import "github.com/omerkoc/cicekci/internal/category"

// CategoryView public kategori gösterimi.
// is_active ve is_featured alanları KASITLI olarak yok — public'e sızmaz (spec §4.6).
type CategoryView struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Axis string `json:"axis"`
}

func toCategoryView(c category.Category) CategoryView {
	return CategoryView{
		ID:   c.ID,
		Name: c.Name,
		Slug: c.Slug,
		Axis: string(c.Axis),
	}
}

func toCategoryViews(list []category.Category) []CategoryView {
	out := make([]CategoryView, 0, len(list))
	for _, c := range list {
		out = append(out, toCategoryView(c))
	}
	return out
}
```

- [ ] **Step 2: product_view.go yaz**

`internal/api/app/product_view.go`:
```go
package app

import "github.com/omerkoc/cicekci/internal/product"

// ProductView public ürün gösterimi.
// is_active alanı KASITLI olarak yok — public'e sızmaz (spec §4.6).
// Price string olarak gider: JSON float precision sorununu önler.
type ProductView struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	Price       string  `json:"price"`
	CategoryIDs []int64 `json:"category_ids"`
}

func toProductView(p product.Product) ProductView {
	return ProductView{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Price:       p.Price.StringFixed(2),
		CategoryIDs: p.CategoryIDs,
	}
}

func toProductViews(list []product.Product) []ProductView {
	out := make([]ProductView, 0, len(list))
	for _, p := range list {
		out = append(out, toProductView(p))
	}
	return out
}
```

- [ ] **Step 3: category_handler.go yaz**

`internal/api/app/category_handler.go`:
```go
package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/category"
)

type categoryHandler struct {
	svc *category.Service
}

// list GET /api/categories?axis=occasion|type
func (h *categoryHandler) list(c *fiber.Ctx) error {
	var axis *category.Axis
	if raw := c.Query("axis"); raw != "" {
		a := category.Axis(raw)
		axis = &a
	}

	list, err := h.svc.ListPublic(c.Context(), axis)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryViews(list))
}

// listFeatured GET /api/categories/featured
func (h *categoryHandler) listFeatured(c *fiber.Ctx) error {
	list, err := h.svc.ListFeatured(c.Context())
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryViews(list))
}

// getBySlug GET /api/categories/:slug
func (h *categoryHandler) getBySlug(c *fiber.Ctx) error {
	cat, err := h.svc.GetPublicBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryView(*cat))
}
```

- [ ] **Step 4: product_handler.go yaz**

`internal/api/app/product_handler.go`:
```go
package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/product"
)

type productHandler struct {
	svc *product.Service
}

// list GET /api/products?amac=&tip=&page=
func (h *productHandler) list(c *fiber.Ctx) error {
	f := product.Filter{
		Limit:  c.QueryInt("limit", 24),
		Offset: 0,
	}

	if page := c.QueryInt("page", 1); page > 1 {
		f.Offset = (page - 1) * f.Limit
	}
	if amac := c.Query("amac"); amac != "" {
		f.OccasionSlug = &amac
	}
	if tip := c.Query("tip"); tip != "" {
		f.TypeSlug = &tip
	}

	list, err := h.svc.ListPublic(c.Context(), f)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toProductViews(list))
}

// getBySlug GET /api/products/:slug
// Slug eskiyse 301 ile güncel URL'e yönlendirir (spec §4.2).
func (h *productHandler) getBySlug(c *fiber.Ctx) error {
	p, redirectTo, err := h.svc.GetPublicBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		return api.WriteError(c, err)
	}

	if redirectTo != "" {
		return c.Redirect("/api/products/"+redirectTo, fiber.StatusMovedPermanently)
	}

	return c.JSON(toProductView(*p))
}
```

- [ ] **Step 5: router.go yaz**

`internal/api/app/router.go`:
```go
package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/product"
)

// Register public rotaları bağlar. Auth yok — herkes erişebilir.
func Register(router fiber.Router, catSvc *category.Service, prodSvc *product.Service) {
	ch := &categoryHandler{svc: catSvc}
	ph := &productHandler{svc: prodSvc}

	router.Get("/products", ph.list)
	router.Get("/products/:slug", ph.getBySlug)

	// /categories/featured, /categories/:slug'dan ÖNCE tanımlanmalı —
	// yoksa "featured" slug olarak yakalanır.
	router.Get("/categories/featured", ch.listFeatured)
	router.Get("/categories", ch.list)
	router.Get("/categories/:slug", ch.getBySlug)
}
```

- [ ] **Step 6: Handler testini yaz**

`internal/api/app/product_handler_test.go`:
```go
package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAPI(t *testing.T) (*fiber.App, *product.Service, *category.Service) {
	t.Helper()
	pool := database.NewTestDB(t)
	prodSvc := product.NewService(product.NewStore(pool))
	catSvc := category.NewService(category.NewStore(pool))

	app := fiber.New()
	Register(app.Group("/api"), catSvc, prodSvc)
	return app, prodSvc, catSvc
}

func mustPrice(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	require.NoError(t, err)
	return d
}

func TestProductHandler_List(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	_, err := svc.Create(context.Background(), product.CreateInput{
		Name: "51 Gül Buket", Price: mustPrice(t, "1850"), IsActive: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products", nil))
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var views []ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	require.Len(t, views, 1)
	assert.Equal(t, "51 Gül Buket", views[0].Name)
	assert.Equal(t, "1850.00", views[0].Price)
}

// Spec §4.6: public uçlar pasif ürünü hiç görmez.
func TestProductHandler_List_HidesInactive(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	_, err := svc.Create(context.Background(), product.CreateInput{
		Name: "Pasif", Price: mustPrice(t, "100"), IsActive: false,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products", nil))
	require.NoError(t, err)

	var views []ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	assert.Empty(t, views)
}

// Public viewmodel'de is_active alanı hiç olmamalı.
func TestProductHandler_List_ViewHasNoIsActiveField(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	_, err := svc.Create(context.Background(), product.CreateInput{
		Name: "Aktif", Price: mustPrice(t, "100"), IsActive: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products", nil))
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.NotContains(t, string(body), "is_active")
}

func TestProductHandler_GetBySlug(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	_, err := svc.Create(context.Background(), product.CreateInput{
		Name: "Buket", Price: mustPrice(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products/buket", nil))
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	assert.Equal(t, "Buket", view.Name)
}

// Spec §4.2: eski slug 301 ile güncel URL'e yönlendirir.
func TestProductHandler_GetBySlug_OldSlugRedirects(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	ctx := context.Background()
	p, err := svc.Create(ctx, product.CreateInput{
		Name: "51 Gül Buket", Price: mustPrice(t, "1850"), IsActive: true,
	})
	require.NoError(t, err)
	newName := "51 Kırmızı Gül Buketi"
	_, err = svc.Update(ctx, p.ID, product.UpdateInput{Name: &newName})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products/51-gul-buket", nil))
	require.NoError(t, err)

	assert.Equal(t, http.StatusMovedPermanently, resp.StatusCode)
	assert.Equal(t, "/api/products/51-kirmizi-gul-buketi", resp.Header.Get("Location"))
}

func TestProductHandler_GetBySlug_NotFound(t *testing.T) {
	app, _, _ := newTestAPI(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products/yok", nil))
	require.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCategoryHandler_Featured_RouteNotShadowed(t *testing.T) {
	app, _, catSvc := newTestAPI(t)
	_, err := catSvc.Create(context.Background(), category.CreateInput{
		Name: "Doğum Günü", Axis: category.AxisOccasion, IsActive: true, IsFeatured: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/categories/featured", nil))
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var views []CategoryView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	require.Len(t, views, 1)
	assert.Equal(t, "Doğum Günü", views[0].Name)
}
```

- [ ] **Step 7: Testi çalıştır**

Run: `go test ./internal/api/app/ -v`
Expected: PASS — 7 test

- [ ] **Step 8: Commit**

```bash
git add internal/api/app/
git commit -m "feat: public API — ürün ve kategori uçları, slug 301"
```

---

## Task 12: Admin API — handler'lar

**Files:**
- Create: `internal/api/idare/auth_handler.go`
- Create: `internal/api/idare/category_view.go`, `internal/api/idare/category_handler.go`
- Create: `internal/api/idare/product_view.go`, `internal/api/idare/product_handler.go`
- Create: `internal/api/idare/router.go`
- Test: `internal/api/idare/product_handler_test.go`

**Interfaces:**
- Consumes: `auth.Service`, `auth.Middleware`, `category.Service`, `product.Service`, `api.WriteError`
- Produces:
  - `idare.CategoryView{..., IsActive bool, IsFeatured bool, SortOrder int, ProductCount int}`
  - `idare.ProductView{..., IsActive bool}`
  - `idare.Register(router fiber.Router, cfg Deps)`
  - `idare.Deps{AuthSvc *auth.Service, CatSvc *category.Service, ProdSvc *product.Service, JWTSecret string, SecureCookie bool}`

- [ ] **Step 1: auth_handler.go yaz**

`internal/api/idare/auth_handler.go`:
```go
package idare

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/auth"
)

type authHandler struct {
	svc          *auth.Service
	secureCookie bool
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// login POST /api/admin/login
// Token HttpOnly cookie'ye yazılır, response body'de dönmez (spec §4.5).
func (h *authHandler) login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz istek"},
		})
	}

	token, err := h.svc.Login(c.Context(), req.Username, req.Password)
	if err != nil {
		return api.WriteError(c, err)
	}

	c.Cookie(&fiber.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		HTTPOnly: true,
		Secure:   h.secureCookie,
		SameSite: "Strict",
		Path:     "/",
		Expires:  time.Now().Add(auth.TokenTTL),
	})

	return c.JSON(fiber.Map{"ok": true})
}

// logout POST /api/admin/logout
func (h *authHandler) logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     auth.CookieName,
		Value:    "",
		HTTPOnly: true,
		Secure:   h.secureCookie,
		SameSite: "Strict",
		Path:     "/",
		Expires:  time.Now().Add(-time.Hour),
	})
	return c.JSON(fiber.Map{"ok": true})
}

// me GET /api/admin/me — frontend'in oturum kontrolü için
func (h *authHandler) me(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"username": c.Locals("username")})
}
```

- [ ] **Step 2: category_view.go yaz**

`internal/api/idare/category_view.go`:
```go
package idare

import "github.com/omerkoc/cicekci/internal/category"

// CategoryView admin kategori gösterimi — is_active ve is_featured DAHİL.
type CategoryView struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Axis       string `json:"axis"`
	IsActive   bool   `json:"is_active"`
	IsFeatured bool   `json:"is_featured"`
	SortOrder  int    `json:"sort_order"`
}

func toCategoryView(c category.Category) CategoryView {
	return CategoryView{
		ID:         c.ID,
		Name:       c.Name,
		Slug:       c.Slug,
		Axis:       string(c.Axis),
		IsActive:   c.IsActive,
		IsFeatured: c.IsFeatured,
		SortOrder:  c.SortOrder,
	}
}

func toCategoryViews(list []category.Category) []CategoryView {
	out := make([]CategoryView, 0, len(list))
	for _, c := range list {
		out = append(out, toCategoryView(c))
	}
	return out
}
```

- [ ] **Step 3: category_handler.go yaz**

`internal/api/idare/category_handler.go`:
```go
package idare

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/category"
)

type categoryHandler struct {
	svc *category.Service
}

type createCategoryRequest struct {
	Name       string `json:"name"`
	Axis       string `json:"axis"`
	IsActive   *bool  `json:"is_active"`
	IsFeatured *bool  `json:"is_featured"`
	SortOrder  *int   `json:"sort_order"`
}

type updateCategoryRequest struct {
	Name       *string `json:"name"`
	IsActive   *bool   `json:"is_active"`
	IsFeatured *bool   `json:"is_featured"`
	SortOrder  *int    `json:"sort_order"`
}

// list GET /api/admin/categories — pasifler dahil
func (h *categoryHandler) list(c *fiber.Ctx) error {
	list, err := h.svc.ListAdmin(c.Context())
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryViews(list))
}

// create POST /api/admin/categories
func (h *categoryHandler) create(c *fiber.Ctx) error {
	var req createCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz istek"},
		})
	}

	in := category.CreateInput{
		Name:     req.Name,
		Axis:     category.Axis(req.Axis),
		IsActive: true, // varsayılan aktif
	}
	if req.IsActive != nil {
		in.IsActive = *req.IsActive
	}
	if req.IsFeatured != nil {
		in.IsFeatured = *req.IsFeatured
	}
	if req.SortOrder != nil {
		in.SortOrder = *req.SortOrder
	}

	cat, err := h.svc.Create(c.Context(), in)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toCategoryView(*cat))
}

// update PATCH /api/admin/categories/:id
func (h *categoryHandler) update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz id"},
		})
	}

	var req updateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz istek"},
		})
	}

	cat, err := h.svc.Update(c.Context(), int64(id), category.UpdateInput{
		Name:       req.Name,
		IsActive:   req.IsActive,
		IsFeatured: req.IsFeatured,
		SortOrder:  req.SortOrder,
	})
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCategoryView(*cat))
}

// productCount GET /api/admin/categories/:id/product-count
// Silme öncesi uyarı için: "Bu kategoride N ürün var" (spec §4.1).
func (h *categoryHandler) productCount(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz id"},
		})
	}

	count, err := h.svc.ProductCount(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(fiber.Map{"product_count": count})
}

// delete DELETE /api/admin/categories/:id
// Junction kayıtları CASCADE ile gider, ürünler silinmez (spec §4.1).
func (h *categoryHandler) delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz id"},
		})
	}

	if err := h.svc.Delete(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
```

- [ ] **Step 4: product_view.go yaz**

`internal/api/idare/product_view.go`:
```go
package idare

import "github.com/omerkoc/cicekci/internal/product"

// ProductView admin ürün gösterimi — is_active DAHİL.
type ProductView struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	Price       string  `json:"price"`
	IsActive    bool    `json:"is_active"`
	CategoryIDs []int64 `json:"category_ids"`
}

func toProductView(p product.Product) ProductView {
	return ProductView{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Price:       p.Price.StringFixed(2),
		IsActive:    p.IsActive,
		CategoryIDs: p.CategoryIDs,
	}
}

func toProductViews(list []product.Product) []ProductView {
	out := make([]ProductView, 0, len(list))
	for _, p := range list {
		out = append(out, toProductView(p))
	}
	return out
}
```

- [ ] **Step 5: product_handler.go yaz**

`internal/api/idare/product_handler.go`:
```go
package idare

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/shopspring/decimal"
)

type productHandler struct {
	svc *product.Service
}

type createProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       string  `json:"price"`
	IsActive    *bool   `json:"is_active"`
	CategoryIDs []int64 `json:"category_ids"`
}

type updateProductRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Price       *string  `json:"price"`
	IsActive    *bool    `json:"is_active"`
	CategoryIDs []int64  `json:"category_ids"`
}

// list GET /api/admin/products — pasifler dahil
func (h *productHandler) list(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 24)
	offset := 0
	if page := c.QueryInt("page", 1); page > 1 {
		offset = (page - 1) * limit
	}

	list, err := h.svc.ListAdmin(c.Context(), limit, offset)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toProductViews(list))
}

// get GET /api/admin/products/:id — pasif olsa da döner
func (h *productHandler) get(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz id"},
		})
	}

	p, err := h.svc.GetByID(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toProductView(*p))
}

// create POST /api/admin/products
func (h *productHandler) create(c *fiber.Ctx) error {
	var req createProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz istek"},
		})
	}

	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz fiyat"},
		})
	}

	in := product.CreateInput{
		Name:        req.Name,
		Description: req.Description,
		Price:       price,
		IsActive:    true, // varsayılan aktif (spec §3.2)
		CategoryIDs: req.CategoryIDs,
	}
	if req.IsActive != nil {
		in.IsActive = *req.IsActive
	}

	p, err := h.svc.Create(c.Context(), in)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toProductView(*p))
}

// update PATCH /api/admin/products/:id
func (h *productHandler) update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz id"},
		})
	}

	var req updateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz istek"},
		})
	}

	in := product.UpdateInput{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
		CategoryIDs: req.CategoryIDs,
	}

	if req.Price != nil {
		price, err := decimal.NewFromString(*req.Price)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
				Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz fiyat"},
			})
		}
		in.Price = &price
	}

	p, err := h.svc.Update(c.Context(), int64(id), in)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toProductView(*p))
}

// delete DELETE /api/admin/products/:id
func (h *productHandler) delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz id"},
		})
	}

	if err := h.svc.Delete(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
```

- [ ] **Step 6: router.go yaz**

`internal/api/idare/router.go`:
```go
package idare

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/product"
)

type Deps struct {
	AuthSvc      *auth.Service
	CatSvc       *category.Service
	ProdSvc      *product.Service
	JWTSecret    string
	SecureCookie bool
}

// Register admin rotalarını bağlar. /login hariç hepsi JWT korumalı.
func Register(router fiber.Router, d Deps) {
	ah := &authHandler{svc: d.AuthSvc, secureCookie: d.SecureCookie}
	ch := &categoryHandler{svc: d.CatSvc}
	ph := &productHandler{svc: d.ProdSvc}

	router.Post("/login", ah.login)

	protected := router.Group("", auth.Middleware(d.JWTSecret))

	protected.Post("/logout", ah.logout)
	protected.Get("/me", ah.me)

	protected.Get("/products", ph.list)
	protected.Post("/products", ph.create)
	protected.Get("/products/:id", ph.get)
	protected.Patch("/products/:id", ph.update)
	protected.Delete("/products/:id", ph.delete)

	protected.Get("/categories", ch.list)
	protected.Post("/categories", ch.create)
	protected.Patch("/categories/:id", ch.update)
	protected.Get("/categories/:id/product-count", ch.productCount)
	protected.Delete("/categories/:id", ch.delete)
}
```

- [ ] **Step 7: Testi yaz**

`internal/api/idare/product_handler_test.go`:
```go
package idare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-that-is-long-enough-32"

func newTestAdminAPI(t *testing.T) (*fiber.App, string) {
	t.Helper()
	pool := database.NewTestDB(t)

	authSvc := auth.NewService(auth.NewStore(pool), testSecret)
	require.NoError(t, authSvc.CreateAdmin(context.Background(), "cicekci", "test-sifre-123"))

	app := fiber.New()
	Register(app.Group("/api/admin"), Deps{
		AuthSvc:      authSvc,
		CatSvc:       category.NewService(category.NewStore(pool)),
		ProdSvc:      product.NewService(product.NewStore(pool)),
		JWTSecret:    testSecret,
		SecureCookie: false,
	})

	token, err := auth.GenerateToken(1, "cicekci", testSecret)
	require.NoError(t, err)
	return app, token
}

func authedRequest(method, path, body, token string) *http.Request {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	return req
}

func TestAdmin_ProductsRequireAuth(t *testing.T) {
	app, _ := newTestAdminAPI(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/admin/products", nil))

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAdmin_Login_SetsHttpOnlyCookie(t *testing.T) {
	app, _ := newTestAdminAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/login",
		strings.NewReader(`{"username":"cicekci","password":"test-sifre-123"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	cookies := resp.Cookies()
	require.NotEmpty(t, cookies)
	var tokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.CookieName {
			tokenCookie = c
		}
	}
	require.NotNil(t, tokenCookie, "token cookie'si set edilmeli")
	assert.True(t, tokenCookie.HttpOnly, "cookie HttpOnly olmalı (spec §4.5)")
	assert.NotEmpty(t, tokenCookie.Value)
}

func TestAdmin_Login_WrongPassword(t *testing.T) {
	app, _ := newTestAdminAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/login",
		strings.NewReader(`{"username":"cicekci","password":"yanlis"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAdmin_CreateProduct(t *testing.T) {
	app, token := newTestAdminAPI(t)

	resp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"51 Gül Buket","description":"Kırmızı","price":"1850.00"}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	assert.Equal(t, "51-gul-buket", view.Slug)
	assert.Equal(t, "1850.00", view.Price)
	assert.True(t, view.IsActive, "varsayılan aktif olmalı")
}

func TestAdmin_CreateProduct_InvalidPrice(t *testing.T) {
	app, token := newTestAdminAPI(t)

	resp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"Test","price":"bes-yuz-lira"}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// Admin listesi pasif ürünleri de görür (spec §4.6).
func TestAdmin_ListProducts_ShowsInactive(t *testing.T) {
	app, token := newTestAdminAPI(t)
	_, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"Pasif","price":"100","is_active":false}`, token))
	require.NoError(t, err)

	resp, err := app.Test(authedRequest(http.MethodGet, "/api/admin/products", "", token))

	require.NoError(t, err)
	var views []ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	require.Len(t, views, 1)
	assert.False(t, views[0].IsActive)
}

func TestAdmin_CategoryProductCount(t *testing.T) {
	app, token := newTestAdminAPI(t)

	catResp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/categories",
		`{"name":"Buket","axis":"type"}`, token))
	require.NoError(t, err)
	var cat CategoryView
	require.NoError(t, json.NewDecoder(catResp.Body).Decode(&cat))

	_, err = app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"Test","price":"100","category_ids":[`+itoa(cat.ID)+`]}`, token))
	require.NoError(t, err)

	resp, err := app.Test(authedRequest(http.MethodGet,
		"/api/admin/categories/"+itoa(cat.ID)+"/product-count", "", token))

	require.NoError(t, err)
	var body struct {
		ProductCount int `json:"product_count"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, 1, body.ProductCount)
}

func itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}
```

- [ ] **Step 8: Testi çalıştır**

Run: `go test ./internal/api/idare/ -v`
Expected: PASS — 7 test

- [ ] **Step 9: Commit**

```bash
git add internal/api/idare/
git commit -m "feat: admin API — auth, ürün ve kategori CRUD"
```

---

## Task 13: Server main — her şeyi bağla

**Files:**
- Create: `cmd/server/main.go`

**Interfaces:**
- Consumes: hepsi

- [ ] **Step 1: main.go yaz**

`cmd/server/main.go`:
```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/omerkoc/cicekci/internal/api/app"
	"github.com/omerkoc/cicekci/internal/api/idare"
	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/pkg/config"
	"github.com/omerkoc/cicekci/pkg/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("veritabanı: %v", err)
	}
	defer pool.Close()

	authSvc := auth.NewService(auth.NewStore(pool), cfg.JWTSecret)
	catSvc := category.NewService(category.NewStore(pool))
	prodSvc := product.NewService(product.NewStore(pool))

	isProduction := strings.HasPrefix(cfg.SiteURL, "https://")

	f := fiber.New(fiber.Config{
		AppName:               "cicekci",
		DisableStartupMessage: false,
		BodyLimit:             10 * 1024 * 1024, // Plan 2'de görsel yükleme için
	})

	f.Use(recover.New())
	f.Use(logger.New())
	f.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.SiteURL,
		AllowCredentials: true, // cookie gönderimi için zorunlu
		AllowMethods:     "GET,POST,PATCH,DELETE,OPTIONS",
	}))

	f.Get("/health", func(c *fiber.Ctx) error {
		if err := pool.Ping(c.Context()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "db down"})
		}
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// apiGroup — "api" adı internal/api paketiyle çakışırdı.
	apiGroup := f.Group("/api")
	app.Register(apiGroup, catSvc, prodSvc)
	idare.Register(apiGroup.Group("/admin"), idare.Deps{
		AuthSvc:      authSvc,
		CatSvc:       catSvc,
		ProdSvc:      prodSvc,
		JWTSecret:    cfg.JWTSecret,
		SecureCookie: isProduction,
	})

	go func() {
		if err := f.Listen(":" + cfg.Port); err != nil {
			log.Fatalf("sunucu: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("kapatılıyor...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := f.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("kapatma hatası: %v", err)
	}
}
```

`AllowCredentials: true` ve `AllowOrigins: cfg.SiteURL` birlikte zorunlu — cookie'nin Nuxt'tan gönderilebilmesi için. `AllowOrigins: "*"` ile `AllowCredentials: true` tarayıcı tarafından reddedilir.

- [ ] **Step 2: Sunucuyu başlat**

```bash
export DATABASE_URL="postgres://cicekci:cicekci@localhost:5433/cicekci?sslmode=disable"
export JWT_SECRET="local-development-secret-32-chars!"
export WHATSAPP_NUMBER="905551234567"
export SITE_URL="http://localhost:3000"
export PORT=8080
make run
```
Expected: Fiber başlangıç banner'ı, `:8080` dinleniyor

- [ ] **Step 3: Health check**

```bash
curl -s localhost:8080/health
```
Expected: `{"status":"ok"}`

- [ ] **Step 4: Login ve cookie al**

```bash
curl -s -c /tmp/cicekci-cookies.txt -X POST localhost:8080/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"cicekci","password":"test-sifre-123"}'
```
Expected: `{"ok":true}` — ve `/tmp/cicekci-cookies.txt` içinde `cicekci_token`

- [ ] **Step 5: Kategori oluştur**

```bash
curl -s -b /tmp/cicekci-cookies.txt -X POST localhost:8080/api/admin/categories \
  -H "Content-Type: application/json" \
  -d '{"name":"Doğum Günü","axis":"occasion","is_featured":true}'

curl -s -b /tmp/cicekci-cookies.txt -X POST localhost:8080/api/admin/categories \
  -H "Content-Type: application/json" \
  -d '{"name":"Buket","axis":"type"}'
```
Expected: iki `201`, slug'lar `dogum-gunu` ve `buket`

- [ ] **Step 6: Ürün oluştur ve filtreyi test et**

```bash
curl -s -b /tmp/cicekci-cookies.txt -X POST localhost:8080/api/admin/products \
  -H "Content-Type: application/json" \
  -d '{"name":"51 Gül Buket","description":"Kırmızı güller","price":"1850.00","category_ids":[1,2]}'

curl -s "localhost:8080/api/products?amac=dogum-gunu&tip=buket"
```
Expected: 51 Gül Buket listede, `"price":"1850.00"`, `is_active` alanı **yok**

- [ ] **Step 7: Slug 301'ini elle doğrula**

```bash
curl -s -b /tmp/cicekci-cookies.txt -X PATCH localhost:8080/api/admin/products/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"51 Kırmızı Gül Buketi"}'

curl -s -i localhost:8080/api/products/51-gul-buket | head -3
```
Expected: `HTTP/1.1 301 Moved Permanently`, `Location: /api/products/51-kirmizi-gul-buketi`

- [ ] **Step 8: Yetkisiz erişimi doğrula**

```bash
curl -s -i localhost:8080/api/admin/products | head -1
```
Expected: `HTTP/1.1 401 Unauthorized`

- [ ] **Step 9: Tüm testleri çalıştır**

```bash
export TEST_DATABASE_URL="postgres://cicekci:cicekci@localhost:5434/cicekci_test?sslmode=disable"
make test
```
Expected: `ok` — tüm paketler geçiyor

- [ ] **Step 10: Commit**

```bash
git add cmd/server/
git commit -m "feat: sunucu — route bağlama, CORS, graceful shutdown"
```

---

## Plan 1 Bitiş Kriterleri

- [ ] `make test` tüm paketlerde geçiyor
- [ ] `make run` ile sunucu ayağa kalkıyor, `/health` yeşil
- [ ] Login → cookie → admin CRUD çalışıyor
- [ ] `?amac=X&tip=Y` filtresi AND semantiğiyle çalışıyor
- [ ] Ürün adı değişince eski slug 301 veriyor
- [ ] Public uçlarda `is_active` alanı görünmüyor, pasif kayıt dönmüyor
- [ ] Yetkisiz admin erişimi 401

**Sonraki:** Plan 2 — görsel hattı (ImageStore, R2, resize/WebP, upload uçları).
