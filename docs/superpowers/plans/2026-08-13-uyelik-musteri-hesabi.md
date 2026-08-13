# Üyelik / Müşteri Hesabı Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Public sitede opsiyonel e-posta+şifre müşteri üyeliği; giriş yapan müşteri siparişini hesabına bağlar, geçmişini görür, sipariş formu bilgileriyle otomatik dolar.

**Architecture:** Backend'de yeni `internal/customer/` paketi — mevcut `internal/auth/` (admin) deseninin ayrı ikizi (bcrypt + HttpOnly JWT cookie, ayrı tablo `customers`, ayrı cookie `customer_token`, ayrı middleware, JWT claim `type:"customer"`). Sipariş oluşturma cookie varsa `customer_id` yazar, yoksa misafir (NULL). Public frontend'de mevcut `/hesabim/*` mock'ları gerçek yapılır.

**Tech Stack:** Go 1.25 + Fiber v2, PostgreSQL (pgx), `golang-jwt/jwt/v5`, `golang.org/x/crypto/bcrypt`; Nuxt 4 SSR (public site).

## Global Constraints

- **Test komutu:** `make test` (`-p 1`). **`go test ./...` KULLANILMAZ** — paralel test paketleri aynı DB'yi TRUNCATE'ler (DURUM.md). Tek paket testi güvenli: `go test ./internal/customer/ -v`.
- **Şifre:** bcrypt (`bcrypt.DefaultCost`), min 8 karakter. `PasswordHash` struct alanı `json:"-"` — asla JSON'a çıkmaz.
- **Auth ayrımı (güvenlik):** Müşteri JWT claim'inde `Type: "customer"`. Customer middleware yalnızca `type=="customer"` token kabul eder. Admin token'ı (type yok / farklı) customer ucuna erişemez. Cookie adı `customer_token` (admin `auth.CookieName = "cicekci_token"`'dan AYRI).
- **Opsiyonel üyelik:** Misafir siparişi bozulmaz. `orders.customer_id` NULLABLE. Cookie yoksa/geçersizse sipariş misafir olarak devam eder (bloklanmaz).
- **Geçmişe dönük eşleştirme YOK.** Sipariş yalnızca giriş yapılmışken `customer_id` alır. E-posta ile eski sipariş eşleştirme yok.
- **E-posta doğrulama YOK, mail YOK.** Kayıt anında hesap aktif.
- **Cookie:** `HTTPOnly: true`, `Secure: <production>`, `SameSite: "Strict"`, `Path: "/"`, `Expires: now + TokenTTL`. (Admin `auth_handler.go` deseni birebir.)
- **Hata formatı:** `{"error": {"code","message"}}` — `api.WriteError`.
- **Dil:** Kullanıcıya dönen mesajlar Türkçe.
- **`/orders` izolasyonu:** Müşteri yalnızca `WHERE customer_id = <token'daki id>` siparişlerini görür.

---

## File Structure

**Yeni backend paketi `internal/customer/`** (admin auth ikizi):
- `model.go` — `Customer` struct (`PasswordHash json:"-"`).
- `jwt.go` — `Claims` (`CustomerID`, `Type`), `GenerateToken`, `ParseToken`. `Type` alanı admin'den ayıran şey.
- `middleware.go` — `CookieName`, `Middleware`: cookie doğrular + `type=="customer"` kontrol eder, `Locals("customerID")` koyar.
- `store.go` — `Create`, `FindByEmail`, `GetByID`, `UpdateProfile`, `UpdatePassword`.
- `service.go` — `Register`, `Login`, `Me`, `UpdateProfile`, `ChangePassword` (bcrypt + doğrulama).
- `*_test.go` — kayıt/giriş/auth-ayrımı/şifre testleri.

**Değişen backend:**
- `internal/order/model.go` — `NewOrder`'a `CustomerID *int64`; `Order`'a `CustomerID *int64`.
- `internal/order/store.go` — INSERT'e customer_id, scan'e customer_id; `ListByCustomer`.
- `internal/order/service.go` — `Create` imzasına `customerID *int64`; `ListByCustomer`.
- `internal/api/app/customer_handler.go` — YENİ: register/login/logout/me/updateMe/orders.
- `internal/api/app/customer_view.go` — YENİ: DTO'lar.
- `internal/api/app/order_handler.go` — `create`: cookie'den customerID çöz, `Create`'e geç.
- `internal/api/app/router.go` — `/customer/*` rotaları (auth middleware korumalı olanlar).
- `cmd/server/main.go` — customer service kurulumu + wiring.
- `migrations/000009_customers.up.sql` / `.down.sql`.

**Değişen frontend (`frontend/app/`):**
- `app/composables/useCustomer.ts` — YENİ: register/login/logout/me/updateProfile/myOrders.
- `app/composables/useCustomer.test.ts` — YENİ: vitest.
- `app/types/api.ts` — Customer/CustomerOrder tipleri.
- `app/pages/giris.vue`, `app/pages/kayit.vue` — YENİ.
- `app/pages/hesabim/index.vue` — gerçek: profil + sipariş geçmişi.
- `app/pages/hesabim/hesap-detaylari.vue` — gerçek: profil + şifre değiştir.
- `app/pages/hesabim/favoriler.vue`, `adresler.vue` — SİL.
- `app/components/account/AccountSidebar.vue` — favoriler/adresler linkleri kaldır.
- `app/pages/siparis/index.vue` — giriş varsa formu otomatik doldur.
- `app/components/TheHeader.vue` — giriş durumuna göre "Hesabım"/"Giriş Yap".
- `app/utils/mockAccount.ts` — SİL (gerçek veriyle değişiyor).

---

## Task 1: Migration — customers tablosu + orders.customer_id

**Files:**
- Create: `backend/migrations/000009_customers.up.sql`
- Create: `backend/migrations/000009_customers.down.sql`

**Interfaces:**
- Produces: `customers` tablosu; `orders.customer_id BIGINT NULL REFERENCES customers(id) ON DELETE SET NULL`; index `idx_orders_customer`.

- [ ] **Step 1: up migration yaz**

`backend/migrations/000009_customers.up.sql`:

```sql
CREATE TABLE customers (
  id            BIGSERIAL PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  name          TEXT NOT NULL,
  phone         TEXT NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE orders
  ADD COLUMN customer_id BIGINT REFERENCES customers(id) ON DELETE SET NULL;

CREATE INDEX idx_orders_customer ON orders (customer_id, created_at DESC);
```

- [ ] **Step 2: down migration yaz**

`backend/migrations/000009_customers.down.sql`:

```sql
DROP INDEX idx_orders_customer;
ALTER TABLE orders DROP COLUMN customer_id;
DROP TABLE customers;
```

- [ ] **Step 3: Uygula ve doğrula**

Run: `cd backend && make migrate-up` (yoksa `migrate -path migrations -database "$DATABASE_URL" up`). DATABASE_URL repo kökü `.env`'de. Test DB'ye de uygula: `TEST_DATABASE_URL` ile (port 5434) — `make test` bunu ister.
Doğrula: `psql "$DATABASE_URL" -c "\d customers"` → tablo görünmeli; `psql "$DATABASE_URL" -c "\d orders"` → `customer_id` kolonu görünmeli.
Expected: Hata yok, tablo + kolon mevcut.

- [ ] **Step 4: Commit**

```bash
git add backend/migrations/000009_customers.up.sql backend/migrations/000009_customers.down.sql
git commit -m "feat(db): customers tablosu + orders.customer_id (üyelik şeması)"
```

---

## Task 2: Customer model + JWT + middleware

**Files:**
- Create: `backend/internal/customer/model.go`
- Create: `backend/internal/customer/jwt.go`
- Create: `backend/internal/customer/middleware.go`
- Test: `backend/internal/customer/jwt_test.go`

**Interfaces:**
- Produces:
  - `type Customer struct { ID int64; Email string; Name string; Phone string; PasswordHash string \`json:"-"\` }`
  - `const TokenTTL = 7 * 24 * time.Hour`
  - `const CookieName = "customer_token"`
  - `type Claims struct { CustomerID int64 \`json:"cid"\`; Type string \`json:"typ"\`; jwt.RegisteredClaims }`
  - `func GenerateToken(customerID int64, secret string) (string, error)` — claim `Type: "customer"`.
  - `func ParseToken(tokenStr, secret string) (*Claims, error)` — HMAC dışı reddeder + `Type != "customer"` reddeder.
  - `func Middleware(secret string) fiber.Handler` — cookie doğrular, `Locals("customerID", int64)` koyar.

- [ ] **Step 1: model.go yaz**

`backend/internal/customer/model.go`:

```go
package customer

// Customer bir müşteri hesabını temsil eder.
type Customer struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
	// PasswordHash asla JSON'a çıkmaz — kazara serialize edilse bile sızmasın.
	PasswordHash string `json:"-"`
}
```

- [ ] **Step 2: Failing test yaz (jwt — type ayrımı kritik)**

`backend/internal/customer/jwt_test.go`:

```go
package customer

import "testing"

func TestGenerateParseToken_RoundTrip(t *testing.T) {
	secret := "test-secret-en-az-32-karakter-olmali-xx"
	tok, err := GenerateToken(42, secret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	claims, err := ParseToken(tok, secret)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.CustomerID != 42 {
		t.Fatalf("CustomerID = %d, beklenen 42", claims.CustomerID)
	}
	if claims.Type != "customer" {
		t.Fatalf("Type = %q, beklenen customer", claims.Type)
	}
}

func TestParseToken_YanlisSecretRed(t *testing.T) {
	tok, _ := GenerateToken(1, "secret-a-en-az-32-karakter-uzunlugunda-x")
	if _, err := ParseToken(tok, "secret-b-en-az-32-karakter-uzunlugunda-x"); err == nil {
		t.Fatal("yanlış secret reddedilmeliydi")
	}
}
```

- [ ] **Step 3: Testi çalıştır — FAIL**

Run: `cd backend && go test ./internal/customer/ -run TestGenerateParseToken -v`
Expected: FAIL — paket/fonksiyonlar yok (derlenmiyor).

- [ ] **Step 4: jwt.go yaz**

`backend/internal/customer/jwt.go`:

```go
package customer

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// TokenTTL üretilen müşteri JWT'lerinin geçerlilik süresi.
const TokenTTL = 7 * 24 * time.Hour

// claimType müşteri token'ını admin token'ından ayıran değer.
// Middleware yalnızca bu değeri taşıyan token'ları kabul eder.
const claimType = "customer"

// Claims müşteri JWT'sinde taşınan bilgiler. Type alanı, admin token'ının
// yanlışlıkla müşteri ucuna geçmesini (veya tersini) engeller.
type Claims struct {
	CustomerID int64  `json:"cid"`
	Type       string `json:"typ"`
	jwt.RegisteredClaims
}

func GenerateToken(customerID int64, secret string) (string, error) {
	claims := Claims{
		CustomerID: customerID,
		Type:       claimType,
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

// ParseToken JWT'yi doğrular. HMAC dışı imza yöntemini ("alg: none" saldırısı)
// ve Type != "customer" olan token'ı (admin token'ı) reddeder.
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
	if !ok || claims.Type != claimType {
		return nil, errorsx.ErrUnauthorized
	}
	return claims, nil
}
```

- [ ] **Step 5: middleware.go yaz**

`backend/internal/customer/middleware.go`:

```go
package customer

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// CookieName müşteri JWT'sinin HttpOnly cookie adı. Admin cookie'sinden
// (cicekci_token) AYRI — iki oturum karışmaz.
const CookieName = "customer_token"

// Middleware müşteri cookie'sini doğrular. Geçerliyse Locals'a customerID koyar.
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
		c.Locals("customerID", claims.CustomerID)
		return c.Next()
	}
}
```

- [ ] **Step 6: Testi çalıştır — PASS**

Run: `cd backend && go test ./internal/customer/ -run TestGenerateParseToken -v && go test ./internal/customer/ -run TestParseToken -v`
Expected: PASS. `go build ./internal/customer/` temiz.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/customer/model.go backend/internal/customer/jwt.go backend/internal/customer/middleware.go backend/internal/customer/jwt_test.go
git commit -m "feat(customer): model + JWT (type ayrımı) + middleware"
```

---

## Task 3: Customer store

**Files:**
- Create: `backend/internal/customer/store.go`
- Test: `backend/internal/customer/store_test.go`

**Interfaces:**
- Consumes: `Customer` (Task 2).
- Produces:
  - `func NewStore(pool *pgxpool.Pool) *Store`
  - `func (s *Store) Create(ctx, email, passwordHash, name, phone string) (*Customer, error)` — email çakışması → `errorsx.ErrConflict`.
  - `func (s *Store) FindByEmail(ctx, email string) (*Customer, error)` — yoksa `errorsx.ErrNotFound`.
  - `func (s *Store) GetByID(ctx, id int64) (*Customer, error)` — yoksa `errorsx.ErrNotFound`.
  - `func (s *Store) UpdateProfile(ctx, id int64, name, phone string) error`
  - `func (s *Store) UpdatePassword(ctx, id int64, passwordHash string) error`

- [ ] **Step 1: store.go yaz**

`backend/internal/customer/store.go`:

```go
package customer

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

// Create yeni müşteri kaydeder. passwordHash zaten hashlenmiş olmalı.
// Email çakışması (UNIQUE) → ErrConflict.
func (s *Store) Create(ctx context.Context, email, passwordHash, name, phone string) (*Customer, error) {
	var cst Customer
	err := s.pool.QueryRow(ctx,
		`INSERT INTO customers (email, password_hash, name, phone)
		 VALUES ($1,$2,$3,$4)
		 RETURNING id, email, name, phone, password_hash`,
		email, passwordHash, name, phone,
	).Scan(&cst.ID, &cst.Email, &cst.Name, &cst.Phone, &cst.PasswordHash)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, errorsx.ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("müşteri oluştur: %w", err)
	}
	return &cst, nil
}

func (s *Store) FindByEmail(ctx context.Context, email string) (*Customer, error) {
	var cst Customer
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name, phone, password_hash FROM customers WHERE email = $1`,
		email,
	).Scan(&cst.ID, &cst.Email, &cst.Name, &cst.Phone, &cst.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("müşteri ara: %w", err)
	}
	return &cst, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Customer, error) {
	var cst Customer
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name, phone, password_hash FROM customers WHERE id = $1`,
		id,
	).Scan(&cst.ID, &cst.Email, &cst.Name, &cst.Phone, &cst.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("müşteri getir: %w", err)
	}
	return &cst, nil
}

func (s *Store) UpdateProfile(ctx context.Context, id int64, name, phone string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE customers SET name=$2, phone=$3, updated_at=now() WHERE id=$1`,
		id, name, phone)
	if err != nil {
		return fmt.Errorf("profil güncelle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

func (s *Store) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE customers SET password_hash=$2, updated_at=now() WHERE id=$1`,
		id, passwordHash)
	if err != nil {
		return fmt.Errorf("şifre güncelle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}
```

- [ ] **Step 2: Failing test yaz**

`backend/internal/customer/store_test.go`. Mevcut test DB helper'ını KULLAN — `backend/internal/auth/store_test.go`'yu (ya da order/store_test.go'yu) OKU, `database.NewTestDB(t)` benzeri gerçek kurulum neyse onu birebir uygula:

```go
package customer

import (
	"context"
	"testing"

	"github.com/omerkoc/cicekci/pkg/database" // gerçek helper adı store_test.go'dan doğrulanacak
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	pool := database.NewTestDB(t) // auth/store_test.go'daki gerçek desenle DEĞİŞTİR
	return NewStore(pool)
}

func TestStore_Create_FindByEmail(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	c, err := s.Create(ctx, "a@b.com", "hash", "Ali", "555")
	require.NoError(t, err)
	require.NotZero(t, c.ID)

	got, err := s.FindByEmail(ctx, "a@b.com")
	require.NoError(t, err)
	require.Equal(t, "Ali", got.Name)
}

func TestStore_Create_EmailCakismasiConflict(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	_, err := s.Create(ctx, "dup@b.com", "h", "A", "1")
	require.NoError(t, err)
	_, err = s.Create(ctx, "dup@b.com", "h", "B", "2")
	require.ErrorIs(t, err, errorsx.ErrConflict)
}

func TestStore_FindByEmail_YokNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.FindByEmail(context.Background(), "yok@b.com")
	require.ErrorIs(t, err, errorsx.ErrNotFound)
}
```

> `newTestStore` gövdesini `auth/store_test.go`'daki gerçek test-DB kurulum deseniyle değiştir (dosyayı önce oku). Test senaryoları tam.

- [ ] **Step 3: Testi çalıştır (önce FAIL derleme yoksa, sonra PASS)**

Run: `cd backend && go test ./internal/customer/ -run TestStore -v`
Expected: PASS (store.go yazıldı; test DB migration 9'da olmalı — Task 1 uyguladı).

- [ ] **Step 4: Commit**

```bash
git add backend/internal/customer/store.go backend/internal/customer/store_test.go
git commit -m "feat(customer): store — Create/FindByEmail/GetByID/UpdateProfile/UpdatePassword"
```

---

## Task 4: Customer service

**Files:**
- Create: `backend/internal/customer/service.go`
- Test: `backend/internal/customer/service_test.go`

**Interfaces:**
- Consumes: `Store` (Task 3), `GenerateToken` (Task 2).
- Produces:
  - `func NewService(store *Store, jwtSecret string) *Service`
  - `func (s *Service) Register(ctx, email, password, name, phone string) (token string, cst *Customer, err error)` — doğrular, hash'ler, kaydeder, otomatik token üretir.
  - `func (s *Service) Login(ctx, email, password string) (string, error)` — hash doğrula → token. Yok/yanlış aynı hata (`ErrUnauthorized`).
  - `func (s *Service) Get(ctx, id int64) (*Customer, error)`
  - `func (s *Service) UpdateProfile(ctx, id int64, name, phone string) (*Customer, error)`
  - `func (s *Service) ChangePassword(ctx, id int64, currentPassword, newPassword string) error` — mevcut şifre doğrulanmadan değişmez.

- [ ] **Step 1: Failing test yaz**

`backend/internal/customer/service_test.go` (test-DB helper aynı desen):

```go
package customer

import (
	"context"
	"testing"

	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) *Service {
	return NewService(newTestStore(t), "test-jwt-secret-en-az-32-karakter-uzun")
}

func TestService_Register_TokenVeHashUretir(t *testing.T) {
	s := newTestService(t)
	tok, c, err := s.Register(context.Background(), "a@b.com", "sifre1234", "Ali", "555")
	require.NoError(t, err)
	require.NotEmpty(t, tok)
	require.NotZero(t, c.ID)
	// Login ile doğrula (hash gerçekten kaydedilmiş mi)
	_, err = s.Login(context.Background(), "a@b.com", "sifre1234")
	require.NoError(t, err)
}

func TestService_Register_KisaSifreRed(t *testing.T) {
	s := newTestService(t)
	_, _, err := s.Register(context.Background(), "a@b.com", "kisa", "Ali", "555")
	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Login_YanlisSifreUnauthorized(t *testing.T) {
	s := newTestService(t)
	_, _, _ = s.Register(context.Background(), "a@b.com", "sifre1234", "Ali", "555")
	_, err := s.Login(context.Background(), "a@b.com", "yanlissifre")
	require.ErrorIs(t, err, errorsx.ErrUnauthorized)
}

func TestService_Login_OlmayanKullaniciUnauthorized(t *testing.T) {
	s := newTestService(t)
	_, err := s.Login(context.Background(), "yok@b.com", "sifre1234")
	require.ErrorIs(t, err, errorsx.ErrUnauthorized)
}

func TestService_ChangePassword_MevcutSifreYanlisRed(t *testing.T) {
	s := newTestService(t)
	_, c, _ := s.Register(context.Background(), "a@b.com", "sifre1234", "Ali", "555")
	err := s.ChangePassword(context.Background(), c.ID, "yanlis", "yenisifre1")
	require.ErrorIs(t, err, errorsx.ErrUnauthorized)
	// doğru mevcut şifreyle geçer
	require.NoError(t, s.ChangePassword(context.Background(), c.ID, "sifre1234", "yenisifre1"))
}
```

- [ ] **Step 2: Testi çalıştır — FAIL**

Run: `cd backend && go test ./internal/customer/ -run TestService -v`
Expected: FAIL — Service/Register/Login yok.

- [ ] **Step 3: service.go yaz**

`backend/internal/customer/service.go`:

```go
package customer

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// Register yeni müşteri hesabı açar ve otomatik giriş token'ı döner.
func (s *Service) Register(ctx context.Context, email, password, name, phone string) (string, *Customer, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	name = strings.TrimSpace(name)
	phone = strings.TrimSpace(phone)

	if !strings.Contains(email, "@") || len(email) < 3 {
		return "", nil, fmt.Errorf("%w: geçerli bir e-posta girin", errorsx.ErrInvalidInput)
	}
	if len(password) < minPasswordLength {
		return "", nil, fmt.Errorf("%w: şifre en az %d karakter olmalı", errorsx.ErrInvalidInput, minPasswordLength)
	}
	if name == "" {
		return "", nil, fmt.Errorf("%w: ad soyad gerekli", errorsx.ErrInvalidInput)
	}
	if phone == "" {
		return "", nil, fmt.Errorf("%w: telefon gerekli", errorsx.ErrInvalidInput)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, fmt.Errorf("şifre hashle: %w", err)
	}

	cst, err := s.store.Create(ctx, email, string(hash), name, phone)
	if errors.Is(err, errorsx.ErrConflict) {
		return "", nil, fmt.Errorf("%w: bu e-posta ile hesap var, giriş yapın", errorsx.ErrConflict)
	}
	if err != nil {
		return "", nil, err
	}

	token, err := GenerateToken(cst.ID, s.jwtSecret)
	if err != nil {
		return "", nil, err
	}
	return token, cst, nil
}

// Login e-posta+şifre doğrular. Kullanıcı yok ile şifre yanlış aynı hatayı
// döner — bilgi sızdırmamak için (admin auth deseni).
func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	cst, err := s.store.FindByEmail(ctx, email)
	if errors.Is(err, errorsx.ErrNotFound) {
		return "", errorsx.ErrUnauthorized
	}
	if err != nil {
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cst.PasswordHash), []byte(password)); err != nil {
		return "", errorsx.ErrUnauthorized
	}
	return GenerateToken(cst.ID, s.jwtSecret)
}

func (s *Service) Get(ctx context.Context, id int64) (*Customer, error) {
	return s.store.GetByID(ctx, id)
}

func (s *Service) UpdateProfile(ctx context.Context, id int64, name, phone string) (*Customer, error) {
	name = strings.TrimSpace(name)
	phone = strings.TrimSpace(phone)
	if name == "" {
		return nil, fmt.Errorf("%w: ad soyad gerekli", errorsx.ErrInvalidInput)
	}
	if phone == "" {
		return nil, fmt.Errorf("%w: telefon gerekli", errorsx.ErrInvalidInput)
	}
	if err := s.store.UpdateProfile(ctx, id, name, phone); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// ChangePassword mevcut şifreyi doğrulamadan değiştirmez (cookie çalınırsa
// şifre değiştirilemesin).
func (s *Service) ChangePassword(ctx context.Context, id int64, currentPassword, newPassword string) error {
	cst, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cst.PasswordHash), []byte(currentPassword)); err != nil {
		return errorsx.ErrUnauthorized
	}
	if len(newPassword) < minPasswordLength {
		return fmt.Errorf("%w: şifre en az %d karakter olmalı", errorsx.ErrInvalidInput, minPasswordLength)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("şifre hashle: %w", err)
	}
	return s.store.UpdatePassword(ctx, id, string(hash))
}
```

- [ ] **Step 4: Testi çalıştır — PASS**

Run: `cd backend && go test ./internal/customer/ -v`
Expected: PASS (tüm customer paketi).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/customer/service.go backend/internal/customer/service_test.go
git commit -m "feat(customer): service — Register/Login/UpdateProfile/ChangePassword"
```

---

## Task 5: Order model + store — customer_id

**Files:**
- Modify: `backend/internal/order/model.go`
- Modify: `backend/internal/order/store.go`
- Test: `backend/internal/order/store_test.go` (ekleme)

**Interfaces:**
- Consumes: mevcut `NewOrder`, `Order`, `Store`.
- Produces:
  - `Order`'a `CustomerID *int64 \`json:"customer_id,omitempty"\``.
  - `NewOrder`'a `CustomerID *int64`.
  - `Store.Create` INSERT'ine customer_id; `scanOrder`/`orderSelect`'e customer_id.
  - `func (s *Store) ListByCustomer(ctx, customerID int64) ([]Order, error)` — o müşterinin siparişleri, en yeni önce (kalemleriyle).

- [ ] **Step 1: model.go — CustomerID alanları**

`Order` struct'a (PaymentRef civarına) ekle:
```go
	CustomerID *int64 `json:"customer_id,omitempty"`
```
`NewOrder` struct'a ekle:
```go
	CustomerID *int64
```

- [ ] **Step 2: store.go — orderSelect + scanOrder + Create güncelle**

`orderSelect`'e `customer_id` kolonu ekle (payment_ref civarı, sabit sıra). `scanOrder`'da ilgili yere `&o.CustomerID` ekle (SELECT sırasıyla eşleşmeli). `createOnce` INSERT'ine `customer_id` kolonu + `in.CustomerID` değeri ekle (nullable — *int64 pgx tarafından NULL yazılır).

> Uygulayıcı: `orderSelect` ve `scanOrder`'ı OKU, customer_id'yi kolon listesine ve Scan argümanlarına AYNI konuma ekle. Sıra uyuşmazlığı = sessiz veri bozulması.

- [ ] **Step 3: store.go — ListByCustomer ekle**

Mevcut `List`/`ListVisible`'ın kalem-doldurma (itemsOfMany) mantığını KULLAN. Dosya sonuna:

```go
// ListByCustomer bir müşterinin kendi siparişlerini en yeniden eskiye döner.
func (s *Store) ListByCustomer(ctx context.Context, customerID int64) ([]Order, error) {
	return s.listWhere(ctx, "customer_id = $1", []any{customerID}, 200, 0)
}
```

> `listWhere` mevcut yardımcı (Task 5 order fix'inde eklenmişti — ListVisible onu kullanıyor). Yoksa `List`'in gövdesini paylaşan biçimde ekle; itemsOfMany tek yerde kalsın. Uygulayıcı store.go'yu okuyup mevcut `listWhere`/`ListVisible` desenini birebir izlesin.

- [ ] **Step 4: Failing test yaz**

`store_test.go`'a ekle (mevcut test helper'larıyla):

```go
func TestStore_ListByCustomer_YalnizKendiSiparisleri(t *testing.T) {
	store := newTestStore(t) // dosyadaki gerçek helper
	ctx := context.Background()

	cid1 := int64(1001)
	cid2 := int64(1002)
	// NOT: customers FK var — testte önce customer satırı gerekebilir.
	// Eğer FK ihlali olursa, testte doğrudan customers'a iki satır ekle
	// (pool.Exec INSERT) ya da mevcut order test helper'ının ürün seed
	// desenine bakıp customer seed helper'ı ekle.
	o1 := createTestOrderWithCustomer(t, store, &cid1) // helper: NewOrder.CustomerID set eder
	_ = createTestOrderWithCustomer(t, store, &cid2)

	list, err := store.ListByCustomer(ctx, cid1)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, o1.ID, list[0].ID)
}
```

> `createTestOrderWithCustomer` helper'ını dosyadaki mevcut `createTestOrder` desenine göre yaz (NewOrder'a CustomerID ekleyen varyant). customers FK'sı için testte iki customer satırı seed et (`store.pool.Exec` ile INSERT customers, ya da customer.Store kullan). Uygulayıcı FK'yı çözer.

- [ ] **Step 5: Testi çalıştır — FAIL sonra PASS**

Run: `cd backend && go test ./internal/order/ -run 'TestStore_ListByCustomer' -v`
Expected: önce FAIL (ListByCustomer yok), implementasyondan sonra PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/order/model.go backend/internal/order/store.go backend/internal/order/store_test.go
git commit -m "feat(order): customer_id alanı + ListByCustomer"
```

---

## Task 6: Order service — Create customerID + ListByCustomer

**Files:**
- Modify: `backend/internal/order/service.go`
- Test: `backend/internal/order/service_test.go` (ekleme)

**Interfaces:**
- Consumes: `Store.ListByCustomer` (Task 5), `NewOrder.CustomerID`.
- Produces:
  - `Create` imzası: `Create(ctx, in CreateInput, userIP string, customerID *int64) (*Order, string, error)` — customerID nil ise misafir.
  - `func (s *Service) ListByCustomer(ctx, customerID int64) ([]Order, error)`.

> **İmza değişikliği uyarısı:** `Create`'e `customerID *int64` eklenince `app/order_handler.go` ve `service_test.go`'daki mevcut `Create` çağrıları kırılır — Task 7'de handler, bu task'ta testler güncellenir.

- [ ] **Step 1: service.go — Create imzası + NewOrder'a CustomerID**

`Create` fonksiyon imzasını değiştir:
```go
func (s *Service) Create(ctx context.Context, in CreateInput, userIP string, customerID *int64) (*Order, string, error) {
```
`s.store.Create(ctx, NewOrder{...})` çağrısına ekle:
```go
		CustomerID: customerID,
```

- [ ] **Step 2: service.go — ListByCustomer ekle**

```go
// ListByCustomer bir müşterinin kendi siparişlerini döner.
func (s *Service) ListByCustomer(ctx context.Context, customerID int64) ([]Order, error) {
	return s.store.ListByCustomer(ctx, customerID)
}
```

- [ ] **Step 3: Mevcut Create çağrılarını testlerde güncelle**

`service_test.go`'da `Create(...)` çağrısı yapan HER mevcut testin sonuna `nil` (misafir) ekle: `svc.Create(ctx, in, "1.2.3.4")` → `svc.Create(ctx, in, "1.2.3.4", nil)`. (Uygulayıcı dosyadaki tüm çağrıları bulup günceller.)

- [ ] **Step 4: Failing test yaz — customerID bağlama**

```go
func TestService_Create_CustomerIDBaglar(t *testing.T) {
	// svc kur, bir müşteri seed et (customers'a satır — FK), Create'i customerID
	// ile çağır, dönen siparişin CustomerID'sinin dolu olduğunu doğrula.
	// Misafir (nil) çağrıda CustomerID nil kalmalı.
}
```

> Test gövdesini mevcut service_test kurulumuyla doldur (fakePay, test store). İki senaryo: customerID dolu → order.CustomerID dolu; nil → nil.

- [ ] **Step 5: Testi çalıştır — PASS**

Run: `cd backend && go test ./internal/order/ -v`
Expected: PASS. `go build ./internal/order/` temiz (app handler hariç — o Task 7).

- [ ] **Step 6: Commit**

```bash
git add backend/internal/order/service.go backend/internal/order/service_test.go
git commit -m "feat(order): Create customerID parametresi + ListByCustomer"
```

---

## Task 7: Customer API handler + order handler entegrasyonu

**Files:**
- Create: `backend/internal/api/app/customer_handler.go`
- Create: `backend/internal/api/app/customer_view.go`
- Modify: `backend/internal/api/app/order_handler.go`
- Modify: `backend/internal/api/app/router.go`
- Test: `backend/internal/api/app/customer_handler_test.go`

**Interfaces:**
- Consumes: `customer.Service` (register/login/get/updateProfile/changePassword), `customer.Middleware`, `customer.CookieName`, `customer.ParseToken`, `order.Service.ListByCustomer`, `order.Service.Create(...,customerID)`.
- Produces: `/api/customer/*` uçları; order create cookie'den customerID çözer.

- [ ] **Step 1: customer_view.go yaz**

```go
package app

import "github.com/omerkoc/cicekci/internal/customer"

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateProfileRequest struct {
	Name string `json:"name"`
	Phone string `json:"phone"`
	// Şifre değiştirme opsiyonel — ikisi de doluysa şifre de değişir.
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type customerView struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

func toCustomerView(c *customer.Customer) customerView {
	return customerView{ID: c.ID, Email: c.Email, Name: c.Name, Phone: c.Phone}
}
```

- [ ] **Step 2: customer_handler.go yaz**

```go
package app

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/customer"
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// errGecersizIstek gövde parse edilemediğinde dönülür. app paketinde
// badRequest helper'ı YOK (o idare'de) — desen api.WriteError + ErrInvalidInput.
var errGecersizIstek = fmt.Errorf("%w: geçersiz istek", errorsx.ErrInvalidInput)

type customerHandler struct {
	svc          *customer.Service
	orderSvc     *order.Service
	secureCookie bool
}

func (h *customerHandler) setCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     customer.CookieName,
		Value:    token,
		HTTPOnly: true,
		Secure:   h.secureCookie,
		SameSite: "Strict",
		Path:     "/",
		Expires:  time.Now().Add(customer.TokenTTL),
	})
}

func (h *customerHandler) register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return api.WriteError(c, errGecersizIstek)
	}
	token, cst, err := h.svc.Register(c.Context(), req.Email, req.Password, req.Name, req.Phone)
	if err != nil {
		return api.WriteError(c, err)
	}
	h.setCookie(c, token)
	return c.Status(fiber.StatusCreated).JSON(toCustomerView(cst))
}

func (h *customerHandler) login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return api.WriteError(c, errGecersizIstek)
	}
	token, err := h.svc.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return api.WriteError(c, err)
	}
	h.setCookie(c, token)
	return c.JSON(fiber.Map{"ok": true})
}

func (h *customerHandler) logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name: customer.CookieName, Value: "", HTTPOnly: true,
		Secure: h.secureCookie, SameSite: "Strict", Path: "/",
		Expires: time.Now().Add(-time.Hour),
	})
	return c.JSON(fiber.Map{"ok": true})
}

func (h *customerHandler) me(c *fiber.Ctx) error {
	id := c.Locals("customerID").(int64)
	cst, err := h.svc.Get(c.Context(), id)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCustomerView(cst))
}

func (h *customerHandler) updateMe(c *fiber.Ctx) error {
	id := c.Locals("customerID").(int64)
	var req updateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return api.WriteError(c, errGecersizIstek)
	}
	// Şifre değiştirme istendiyse önce onu uygula.
	if req.NewPassword != "" {
		if err := h.svc.ChangePassword(c.Context(), id, req.CurrentPassword, req.NewPassword); err != nil {
			return api.WriteError(c, err)
		}
	}
	cst, err := h.svc.UpdateProfile(c.Context(), id, req.Name, req.Phone)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCustomerView(cst))
}

func (h *customerHandler) orders(c *fiber.Ctx) error {
	id := c.Locals("customerID").(int64)
	list, err := h.orderSvc.ListByCustomer(c.Context(), id)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCreateOrderCustomerViews(list)) // aşağıda tanımlanıyor
}
```

> Hata deseni: app paketinde `badRequest` helper'ı YOK (o idare'de). Geçersiz istek için `api.WriteError(c, errGecersizIstek)` kullanılır (yukarıda tanımlandı). `toCreateOrderCustomerViews` — müşteri sipariş listesi view'ı; app paketinde public sipariş görünümü sınırlı (`createOrderResponse` sadece order_no+total+token). Müşteri geçmişi için `customer_view.go`'ya yeni bir view ekle: order_no, status (string), total (StringFixed(2)), delivery_date (Format "2006-01-02"), items (product_name, quantity). Uygulayıcı `order.Order` alanlarından tutarlı bir liste view'ı kurar (idare `order_view.go`'daki `toOrderView` fiyat/tarih formatlama desenini referans alır ama public'e uygun sadeleştirir — buyer/recipient iç detayları müşteriye gerekmez, kendi siparişi zaten).

- [ ] **Step 3: order_handler.go — cookie'den customerID çöz**

`create` handler'ında, `h.svc.Create` çağrısından önce cookie'yi çöz:

```go
	// Giriş yapmış müşteri varsa siparişi ona bağla (opsiyonel — yoksa misafir).
	var customerID *int64
	if tok := c.Cookies(customer.CookieName); tok != "" {
		if claims, err := customer.ParseToken(tok, h.jwtSecret); err == nil {
			customerID = &claims.CustomerID
		}
	}

	o, token, err := h.svc.Create(c.Context(), in, c.IP(), customerID)
```

> `orderHandler` struct'ına `jwtSecret string` alanı eklenmeli (yoksa). `order_handler.go` import'una `"github.com/omerkoc/cicekci/internal/customer"` ekle. Cookie yok/geçersizse `customerID` nil kalır — misafir siparişi (bozulmaz).

- [ ] **Step 4: router.go — customer rotaları**

`Register` imzasına `custSvc *customer.Service, jwtSecret string, secureCookie bool` parametrelerini ekle (main.go da güncellenecek — Task 8). `Register` gövdesine:

```go
	custH := &customerHandler{svc: custSvc, orderSvc: orderSvc, secureCookie: secureCookie}

	router.Post("/customer/register", custH.register)
	router.Post("/customer/login", custH.login)
	router.Post("/customer/logout", custH.logout)

	// Auth korumalı müşteri uçları
	custProtected := router.Group("/customer", customer.Middleware(jwtSecret))
	custProtected.Get("/me", custH.me)
	custProtected.Patch("/me", custH.updateMe)
	custProtected.Get("/orders", custH.orders)
```

> `oh := &orderHandler{...}` kurulumuna `jwtSecret: jwtSecret` ekle. import'a `customer` paketi.

- [ ] **Step 5: Test yaz — auth ayrımı + kendi siparişleri**

`customer_handler_test.go` (mevcut app test kurulumu deseniyle):

```go
func TestCustomer_RegisterLogin_CookieSet(t *testing.T) {
	// register → 201 + customer_token cookie set. login → ok + cookie.
}
func TestCustomer_Me_AuthGerekli(t *testing.T) {
	// cookie'siz GET /customer/me → 401.
}
func TestCustomer_AdminTokenCustomerUcunaErisemez(t *testing.T) {
	// admin token'ı (auth.GenerateToken) customer_token cookie'sine konsa bile
	// /customer/me → 401 (type != customer). GÜVENLİK testi.
}
func TestCustomer_Orders_YalnizKendisi(t *testing.T) {
	// iki müşteri, her biri sipariş verir; A'nın /orders'ı yalnız A'nınkini döner.
}
```

> Gövdeleri mevcut app test app kurulumuyla doldur (Fiber test app, test DB). Admin token testi: `auth.GenerateToken` ile üretilmiş token'ı `customer_token` cookie'sine koy → reddedilmeli (type ayrımı çalışıyor mu).

- [ ] **Step 6: Testi çalıştır — PASS**

Run: `cd backend && go test ./internal/api/app/ -run TestCustomer -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/api/app/customer_handler.go backend/internal/api/app/customer_view.go backend/internal/api/app/order_handler.go backend/internal/api/app/router.go backend/internal/api/app/customer_handler_test.go
git commit -m "feat(api): müşteri auth uçları + sipariş customerID bağlama"
```

---

## Task 8: Wiring (main.go) — customer service devreye

**Files:**
- Modify: `backend/cmd/server/main.go`
- Test: (derleme + make test)

**Interfaces:**
- Consumes: `customer.NewService`, `customer.NewStore`, `app.Register` yeni imza.

- [ ] **Step 1: main.go — customer service kur + wiring**

`orderSvc` kurulumundan sonra:
```go
	custSvc := customer.NewService(customer.NewStore(pool), cfg.JWTSecret)
```
`app.Register(...)` çağrısına yeni parametreleri ekle:
```go
	app.Register(apiGroup, catSvc, prodSvc, imgSvc, sliderSvc, orderSvc, deliveryCfg,
		custSvc, cfg.JWTSecret, isProduction)
```
import'a `"github.com/omerkoc/cicekci/internal/customer"` ekle.

- [ ] **Step 2: Derleme + tüm testler**

Run: `cd backend && go build ./... && make test`
Expected: Derleme temiz, tüm paketler PASS.

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(server): müşteri servisi wiring"
```

---

## Task 9: Public frontend — auth composable + giriş/kayıt

**Files:**
- Create: `frontend/app/app/composables/useCustomer.ts`
- Create: `frontend/app/app/composables/useCustomer.test.ts`
- Modify: `frontend/app/app/types/api.ts`
- Create: `frontend/app/app/pages/giris.vue`
- Create: `frontend/app/app/pages/kayit.vue`

**Interfaces:**
- Consumes: `/api/customer/*` (proxy üzerinden `apiBase()` = `/api/go`).
- Produces: `useCustomer()` composable; `/giris`, `/kayit` sayfaları.

- [ ] **Step 1: types/api.ts — Customer tipleri**

```ts
export interface Customer {
  id: number
  email: string
  name: string
  phone: string
}
```
(CustomerOrder için mevcut sipariş tipi varsa onu kullan; yoksa order_no/status/total/delivery_date/items içeren bir tip ekle.)

- [ ] **Step 2: useCustomer.ts yaz**

```ts
import type { Customer } from '~/types/api'

// Müşteri oturumu HttpOnly cookie'de — JS token'a bakamaz. Oturum durumu
// /api/customer/me çağrısıyla (cookie geçerliyse profil döner) belirlenir.
export function useCustomer() {
  async function register(input: { email: string, password: string, name: string, phone: string }) {
    return await $fetch<Customer>(`${apiBase()}/customer/register`, { method: 'POST', body: input })
  }
  async function login(input: { email: string, password: string }) {
    return await $fetch<{ ok: boolean }>(`${apiBase()}/customer/login`, { method: 'POST', body: input })
  }
  async function logout() {
    return await $fetch(`${apiBase()}/customer/logout`, { method: 'POST' })
  }
  async function me(): Promise<Customer | null> {
    try {
      return await $fetch<Customer>(`${apiBase()}/customer/me`)
    }
    catch {
      return null // 401 → giriş yok
    }
  }
  async function updateProfile(input: { name: string, phone: string, current_password?: string, new_password?: string }) {
    return await $fetch<Customer>(`${apiBase()}/customer/me`, { method: 'PATCH', body: input })
  }
  async function myOrders() {
    return await $fetch(`${apiBase()}/customer/orders`)
  }
  return { register, login, logout, me, updateProfile, myOrders }
}
```

> `apiBase()` `useApi.ts`'den auto-import. `$fetch` cookie'yi same-origin taşır (proxy üzerinden). Not: proxy `/customer/*`'ı geçirir (yalnızca `/admin/*` bloklu).

- [ ] **Step 3: useCustomer.test.ts — vitest**

`me()` 401'de null döner (try/catch mantığı). Basit birim testi (mevcut useCart.test.ts deseni; `$fetch` mock'lanır).

- [ ] **Step 4: giris.vue + kayit.vue yaz**

Mevcut tasarım dili (site-container, form input class'ları — `siparis/index.vue`'dan al). `giris.vue`: email+şifre → `login()` → başarılıysa `/hesabim`'a git. `kayit.vue`: email+şifre+ad+telefon → `register()` → `/hesabim`. Hata mesajı `apiErrorMessage(e)` (mevcut, useOrders.ts) ile Türkçe gösterilir. Zaten giriş yapılmışsa (`me()` doluysa) `/hesabim`'a yönlendir.

- [ ] **Step 5: build + test**

Run: `cd frontend/app && pnpm test && pnpm build`
Expected: testler PASS, build başarılı.

- [ ] **Step 6: Commit**

```bash
git add frontend/app/app/composables/useCustomer.ts frontend/app/app/composables/useCustomer.test.ts frontend/app/app/types/api.ts frontend/app/app/pages/giris.vue frontend/app/app/pages/kayit.vue
git commit -m "feat(public): useCustomer composable + giriş/kayıt sayfaları"
```

---

## Task 10: Public frontend — hesabım ekranları + form otomatik doldurma + header

**Files:**
- Modify: `frontend/app/app/pages/hesabim/index.vue`
- Modify: `frontend/app/app/pages/hesabim/hesap-detaylari.vue`
- Delete: `frontend/app/app/pages/hesabim/favoriler.vue`, `frontend/app/app/pages/hesabim/adresler.vue`
- Modify: `frontend/app/app/components/account/AccountSidebar.vue`
- Modify: `frontend/app/app/pages/siparis/index.vue`
- Modify: `frontend/app/app/components/TheHeader.vue`
- Delete: `frontend/app/app/utils/mockAccount.ts`

**Interfaces:**
- Consumes: `useCustomer()` (Task 9).

- [ ] **Step 1: favoriler/adresler sil + sidebar temizle**

`favoriler.vue`, `adresler.vue`, `mockAccount.ts` sil. `AccountSidebar.vue`'dan favoriler/adresler linklerini kaldır (kalan: hesap özeti, sipariş geçmişi, hesap detayları). `mockAccount.ts`'i import eden kalan yer olmamalı (`grep -rn mockAccount frontend/app/app` → boş).

- [ ] **Step 2: hesabim/index.vue — gerçek profil + sipariş geçmişi**

`onMounted`: `me()` çağır. null ise `/giris`'e yönlendir. Doluysa profil özeti göster + `myOrders()` ile sipariş listesi (order_no, durum rozeti, tarih, tutar). Mevcut `account` layout + tasarım dili korunur. Boş liste → "Henüz siparişiniz yok."

- [ ] **Step 3: hesap-detaylari.vue — profil + şifre değiştir**

`me()` ile mevcut ad/telefon/e-posta göster (e-posta salt-okunur). Ad/telefon düzenlenebilir → `updateProfile()`. Ayrı şifre değiştir bölümü: mevcut şifre + yeni şifre → `updateProfile({current_password, new_password})`. Başarı/hata mesajı.

- [ ] **Step 4: siparis/index.vue — otomatik doldurma**

`onMounted` (veya setup): `useCustomer().me()` çağır. Doluysa `form.buyerName`, `form.buyerPhone`, `form.buyerEmail` alanlarını doldur (müşteri değiştirebilir). Giriş yoksa boş (mevcut davranış). Sepet akışını BOZMA.

- [ ] **Step 5: TheHeader.vue — giriş durumuna göre link**

`me()` ile oturum kontrol; giriş varsa "Hesabım" (→ /hesabim), yoksa "Giriş Yap" (→ /giris). Mevcut header yapısına uygun (zaten "Hesabım" linki vardı — koşullu hale getir).

- [ ] **Step 6: build + test**

Run: `cd frontend/app && pnpm test && pnpm build`
Expected: testler PASS, build başarılı. `grep -rn "mockAccount\|favoriler\|adresler" frontend/app/app/pages frontend/app/app/components` → sipariş/hesap ekranlarında kalıntı yok.

- [ ] **Step 7: Commit**

```bash
git add -A frontend/app/
git commit -m "feat(public): hesabım ekranları gerçek + sipariş formu otomatik doldurma + header"
```

---

## Task 11: Uçtan uca doğrulama (mock — lokal/mümkünse)

**Files:** (doğrulama — kod yok)

- [ ] **Step 1: Backend başlat, migration 9 uygulanmış mı**

`make run` (veya go run). `psql "$DATABASE_URL" -c "\d customers"` → tablo var.

- [ ] **Step 2: Kayıt → cookie**

```bash
curl -s -i -X POST http://localhost:8080/api/customer/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"sifre1234","name":"Test Müşteri","phone":"5551112233"}'
```
Expected: 201 + `Set-Cookie: customer_token=...`. Yanıtta şifre YOK.

- [ ] **Step 3: Giriş + me (cookie ile)**

Cookie'yi sakla (`-c cookies.txt`), sonra:
```bash
curl -s http://localhost:8080/api/customer/me -b cookies.txt
```
Expected: `{"id":...,"email":"test@test.com","name":"Test Müşteri","phone":"5551112233"}`.

- [ ] **Step 4: Giriş yapmışken sipariş → customer_id bağlanır**

Cookie ile `POST /api/orders` (aktif ürün, yarın tarih). Sonra:
`psql "$DATABASE_URL" -c "SELECT order_no, customer_id FROM orders ORDER BY id DESC LIMIT 1;"`
Expected: `customer_id` DOLU.

- [ ] **Step 5: Misafir sipariş → customer_id NULL**

Cookie'siz `POST /api/orders`. Expected: `customer_id` NULL (misafir akışı bozulmadı).

- [ ] **Step 6: Kendi siparişleri**

`curl -s http://localhost:8080/api/customer/orders -b cookies.txt` → yalnızca bu müşterinin siparişleri.

- [ ] **Step 7: Auth ayrımı (güvenlik)**

Admin token'ı `customer_token` cookie'sine koyup `/api/customer/me` çağır → 401 beklenir (type ayrımı).

- [ ] **Step 8: DURUM.md güncelle + commit**

`docs/DURUM.md`'ye üyelik satırı ekle (mock/lokal uçtan uca doğrulandı). Commit.

---

## Self-Review

**Spec coverage (spec → task):**
- §1 kararlar → tüm plana yayılı (opsiyonel üyelik T7 cookie-çöz, e-posta+şifre T4, geçmiş eşleştirme yok — hiçbir task e-posta eşleştirme yapmıyor ✓, mail yok ✓, otomatik doldurma T10, auth ayrımı T2 type + T7 test).
- §2.1 admin ikizi → T2/3/4 (customer paketi auth deseni).
- §2.2 güvenlik sınırı → T2 (Type claim + Middleware) + T7 (admin-token-reddi testi).
- §2.3 opsiyonel bağlama → T7 (order_handler cookie çöz, yoksa nil).
- §3.1 customers → T1 migration + T2 model.
- §3.2 orders.customer_id → T1 + T5.
- §4.1 uçlar → T7 router.
- §4.2 orders customerID → T6/T7.
- §4.3 doğrulamalar → T4 service (email/şifre/tekillik) + T7 auth guard.
- §4.4 auth akışı → T2/4/7.
- §5 kod organizasyonu → T2-8 dosya yapısı.
- §6 frontend → T9/T10.
- §7 test → hash/giriş(T4), auth ayrımı(T2/T7), bağlama(T5/T6), kendi siparişleri(T7), şifre(T4).
- §8 kabul → T11 uçtan uca.
- §9 devreden → hiçbir task adres defteri/mail/geçmiş eşleştirme YAPMIYOR ✓.

**Placeholder scan:** Test gövdelerinde "mevcut helper deseniyle doldur" notları var — gerçek test-DB/app kurulum adları bu plan yazımında bilinmediğinden (auth/order test helper'ları). Uygulayıcı ilgili `_test.go`'yu okuyup birebir helper kullanır; test senaryoları (ne doğrulanacağı) tam yazılı. Store customer_id sıra ekleme (T5 Step 2) prose — çünkü orderSelect/scanOrder'ın tam güncel hali okunmalı, kör satır vermek sıra hatası riski taşır.

**Type consistency:** `customer.Claims.CustomerID` (T2) → `Middleware` Locals (T2) → handler `c.Locals("customerID").(int64)` (T7) tutarlı. `Create(ctx, in, userIP, customerID *int64)` yeni imza T6'da tanımlı, T7 handler + T8 wiring + T6 testler ona göre. `CookieName="customer_token"` T2'de sabit, T7 handler + order_handler + T11 test aynı. `Store.ListByCustomer(customerID int64)` T5 → service T6 → handler T7 tutarlı.

**T5 Step 3 riski:** `listWhere` yardımcısının varlığı Faz 3 order fix'ine bağlı (ListVisible onu kullanıyordu). Uygulayıcı store.go'da `listWhere`/`ListVisible` mevcut mu doğrular; yoksa `List`'in kalem-doldurma gövdesini paylaşan biçimde ekler (itemsOfMany tek yerde).
