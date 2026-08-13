# Üyelik / Müşteri Hesabı Tasarım Dokümanı

Tarih: 2026-08-13
Durum: Onaylandı, implementasyon planı bekliyor
Kaynak: Faz 2 spec §1 "Neden üyelik bu fazda değil" + Faz 3 spec §Bu fazda yok
(Üyelik / customers → "Sonra") üzerine yapılan tasarım tartışması

---

## 1. Amaç ve Kapsam

Public sitede **opsiyonel** müşteri üyeliği. Müşteri e-posta+şifreyle hesap açar,
giriş yapar, giriş yaptıktan sonra verdiği siparişleri hesabından görür, sipariş
formu kayıtlı bilgileriyle otomatik dolu gelir.

**Neden şimdi:** Faz 2 spec'i üyeliği ertelerken net bir gerekçe koymuştu:
*"Üyelik bir kapıdır, arkasında oda olmalı. Bugün hesap açan müşteri hesabında
hiçbir şey göremez — sipariş geçmişi yok."* Faz 3 ile ödeme + sipariş sistemi
canlıda çalışıyor. Artık üyeliğin arkasında gerçek bir oda var: sipariş geçmişi.

### Alınan temel kararlar (tartışmada netleşen)

| Karar | Seçim | Gerekçe |
|---|---|---|
| Giriş yöntemi | **E-posta + şifre** | En basit, maliyetsiz; mevcut admin auth deseni (bcrypt + HttpOnly JWT cookie) neredeyse birebir kullanılır. SMS/OTP ek servis + maliyet getirir. |
| Üyelik | **Opsiyonel** | Zorunlu üyelik dönüşümü düşürür (çiçek alımı yılda 2-3 kez — Faz 2 §1). Misafir siparişi aynen devam eder. |
| Sipariş bağlama | **Sadece giriş sonrası** | Geçmişe dönük e-posta eşleştirme YOK. Giriş yapmış müşterinin siparişi `customer_id` alır. |
| E-posta doğrulama | **YOK** | Mail altyapısı gerektirmez. Eşleştirme olmadığı için "başkasının siparişini görme" riski hiç doğmaz. |
| Kayıtlı bilgi | **Form otomatik doldurma** | Giriş yapan müşterinin ad/telefon/e-postası forma hazır gelir. Ayrı adres defteri YOK. |
| Auth ayrımı | **Tamamen ayrı** | `customers` (müşteri) ile `admin_users` (admin) ayrı tablo, ayrı cookie, ayrı middleware. |
| Ekranlar | **Mevcut /hesabim mock'ları gerçek yapılır** | Faz 2'de bırakılan kabuklar gerçek hale gelir. |

### Bu fazda var
- `customers` tablosu + `orders.customer_id` (nullable)
- Müşteri auth: kayıt, giriş, çıkış, /me (bcrypt + HttpOnly `customer_token` cookie)
- Giriş sonrası verilen siparişler hesaba bağlanır
- Müşteri sipariş geçmişini görür (`/hesabim`)
- Sipariş formu giriş yapan müşteri için otomatik dolar
- Profil bilgileri görüntüleme/güncelleme + şifre değiştirme

### Bu fazda yok (bilinçli — YAGNI)

| Ne | Neden | Nereye |
|---|---|---|
| Geçmişe dönük sipariş eşleştirme | E-posta doğrulama gerektirir; güvenlik riski; kullanıcı kararı | — |
| E-posta doğrulama / şifre sıfırlama maili | Mail altyapısı (SMTP) yok | Sonra (mail altyapısı gelirse) |
| Adres/alıcı defteri | Mock sadece görsel kabuk; gerçek defter yeni tablo+CRUD+KVKK. Otomatik doldurma %80'ini çözüyor | Ayrı mini-faz (istenirse) |
| Favoriler | Kapsam dışı | — |
| Pazarlama / bildirim | KVKK açık rıza + mail altyapısı | Sonra |

---

## 2. Mimari Kararlar

### 2.1 Admin auth deseninin ayrı ikizi

Backend'de yeni `internal/customer/` paketi — mevcut `internal/auth/` (admin) ile
**aynı yapı, ayrı tablo, ayrı cookie:**
- `customers` tablosu (email ile, çok kullanıcı)
- bcrypt şifre hash, HttpOnly JWT cookie
- Müşteri cookie'si admin'den **ayrı ad** (admin `auth.CookieName`, müşteri ör.
  `customer_token`)

**Neden ayrı paket, admin auth'u paylaşmıyoruz:** Admin auth "tek kullanıcı,
username ile" kurulu; müşteri "çok kullanıcı, email ile, opsiyonel". Ortak
soyutlama zorlamak yerine kanıtlanmış deseni kopyalayıp uyarlıyoruz. İkisi
bağımsız evrilir. (MVP'nin "tek admin yeterli, rol sistemi yok" kararı admin
tarafında hâlâ geçerli; müşteri tarafı ondan tamamen bağımsız.)

### 2.2 Güvenlik sınırı: iki auth sistemi hiç kesişmez

- Müşteri yalnızca `/api/customer/*` uçlarına erişir.
- Admin yalnızca `/api/admin/*` uçlarına erişir.
- JWT claim'inde `type` (`"customer"` / `"admin"`) taşınır. Müşteri token'ı admin
  middleware'ine takılırsa reddedilir; admin token'ı customer middleware'ine
  takılırsa reddedilir.
- **Public Nitro proxy** (`frontend/app/server/api/go/[...path].ts`) bugün
  `/admin/*` uçlarını bloklar (güvenlik). `/customer/*` uçlarına **izin verilir**
  — proxy admin'i bloklarken customer'ı geçirir.

### 2.3 Opsiyonel bağlama: cookie varsa bağla, yoksa misafir

Sipariş oluşturulurken (`POST /api/orders`):
- `customer_token` cookie'si **varsa** → çözülür, siparişe `customer_id` yazılır.
- Cookie **yoksa** → misafir siparişi, `customer_id = NULL` (mevcut davranış aynen).

Bu, sipariş oluşturmayı **bozmaz** — sadece opsiyonel olarak müşteriye bağlar.
Cookie süresi dolmuşsa sipariş misafir olarak devam eder, bloklanmaz.

---

## 3. Veri Modeli

### 3.1 `customers` tablosu (migration `000009`)

```sql
CREATE TABLE customers (
  id            BIGSERIAL PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE,      -- giriş kimliği, tekil
  password_hash TEXT NOT NULL,             -- bcrypt, asla JSON'a çıkmaz
  name          TEXT NOT NULL,             -- form otomatik doldurma
  phone         TEXT NOT NULL,             -- form otomatik doldurma
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 3.2 `orders.customer_id` (aynı migration)

```sql
ALTER TABLE orders
  ADD COLUMN customer_id BIGINT REFERENCES customers(id) ON DELETE SET NULL;

CREATE INDEX idx_orders_customer ON orders (customer_id, created_at DESC);
```

Faz 2 spec'inin *"Faz 3'te orders.customer_id nullable eklenir"* dediği kolon —
Faz 3'te eklenmemişti, bu fazda geliyor.

### 3.3 Şema kararlarının gerekçeleri

- **`customer_id` NULLABLE.** Misafir siparişlerinde NULL. Sadece giriş yapmış
  müşterinin siparişinde dolu. "Opsiyonel üyelik" kararının şema karşılığı.

- **`ON DELETE SET NULL`.** Müşteri hesabını silerse siparişleri ölmemeli —
  sipariş finansal/operasyonel kayıt (Faz 2 prensibi). Hesap gider, sipariş
  `customer_id=NULL` misafir siparişine döner, esnaf panelinde kalır.

- **`email UNIQUE`.** Bir e-posta bir hesap; giriş kimliği bu.

- **`name`/`phone` müşteride saklanıyor** (siparişten ayrı) — form otomatik
  doldurma için kalıcı profil. Kayıt sırasında alınır, güncellenebilir.

- **`password_hash` JSON'a çıkmaz** — Go struct'ta `json:"-"` (admin auth deseni).
  Kazara serialize edilse bile hash sızmaz.

- **`customers` yeni kişisel veri değil.** `orders` zaten `buyer_email/phone/name`
  tutuyordu. Ama hesap = kalıcı profil; ileride "hesabımı sil" gerekebilir.
  Şimdilik kapsamda değil (YAGNI), `ON DELETE SET NULL` ile silme şemada hazır.

---

## 4. API

### 4.1 Public müşteri uçları (`/api/customer/*`)

```
POST /api/customer/register   → hesap oluştur (email, password, name, phone) + otomatik giriş
POST /api/customer/login      → giriş → HttpOnly customer_token cookie
POST /api/customer/logout     → çıkış → cookie temizle
GET  /api/customer/me         → giriş yapan müşterinin profili (auth gerekli)
PATCH /api/customer/me        → profil güncelle (name/phone) + şifre değiştir (auth gerekli)
GET  /api/customer/orders     → müşterinin KENDİ siparişleri (auth gerekli)
```

### 4.2 Mevcut `POST /api/orders` değişikliği (küçük)

`customer_token` cookie'si varsa çözülür ve siparişe `customer_id` yazılır; yoksa
NULL. Sipariş oluşturma akışı (fiyat DB'den, PayTR token vb.) aynen korunur —
sadece opsiyonel bağlama eklenir.

### 4.3 Sunucu doğrulamaları

| Kural | Neden |
|---|---|
| Email formatı geçerli | Çöp veri girmesin |
| Email tekil (kayıtta) | Bir email bir hesap. Çakışma → "bu e-posta ile hesap var, giriş yapın" |
| Şifre min 8 karakter | Temel güvenlik |
| `/me`, `/orders`, `PATCH /me` auth gerektirir | Başkasının verisi görünmesin |
| `/orders` sadece `WHERE customer_id = <token'daki id>` | Müşteri yalnızca kendi siparişlerini görür |
| Şifre değiştirmede mevcut şifre doğrulanır | Cookie çalınırsa şifre değiştirilemesin |

**Hata formatı** mevcutla aynı: `{"error": {"code", "message"}}`.

### 4.4 Auth akışı (admin deseninin ikizi)

1. **Kayıt:** email+password+name+phone → email tekil mi → bcrypt hash →
   `customers`'a yaz → otomatik giriş (cookie set).
2. **Giriş:** email+password → hash doğrula → JWT üret (claim: `customer_id`,
   `type: "customer"`) → HttpOnly `customer_token` cookie.
3. **Middleware:** `customer_token` doğrulanır, `type == "customer"` kontrol
   edilir. Yanlış type reddedilir.
4. **Cookie:** production'da `Secure` bayrağı (mevcut `APP_ENV`/`SecureCookie`
   deseni). HttpOnly — JS token'a bakamaz.

---

## 5. Kod Organizasyonu

```
internal/customer/
  model.go       → Customer struct (PasswordHash json:"-"), NewCustomer
  store.go       → Create, GetByEmail, GetByID, Update (pgx)
  service.go     → Register, Login, UpdateProfile, ChangePassword (bcrypt, doğrulama)
  jwt.go / middleware.go → customer_token üretme/doğrulama, type kontrolü
  *_test.go      → admin auth testlerinin ikizi

internal/api/app/
  customer_handler.go → register/login/logout/me/orders handler'ları
  customer_view.go    → istek/yanıt DTO'ları (şifre yanıtta yok)
  router.go           → /customer/* rotaları (public grup, auth middleware /me & /orders'ta)

internal/order/
  service.go     → Create opsiyonel customerID parametresi alır (nil = misafir)

migrations/000009_customers.up.sql / .down.sql
```

`customer` paketi `order`'ı tanımaz; `order` paketi `customer`'ı tanımaz. Sipariş
bağlama handler katmanında yapılır (cookie çöz → customerID → order.Create'e geç).

---

## 6. Frontend (`frontend/app/`, Nuxt public site)

Mevcut `/hesabim/*` mock'ları gerçek yapılır; kapsam dışı olanlar kaldırılır.

| Mevcut mock | Ne olacak |
|---|---|
| `/hesabim/index.vue` | **Gerçek** — giriş varsa profil özeti + sipariş geçmişi; yoksa giriş/kayıt yönlendirme |
| `/hesabim/hesap-detaylari.vue` | **Gerçek** — profil (ad/telefon/e-posta) + şifre değiştirme |
| `/hesabim/favoriler.vue` | **Kaldırılır** (YAGNI) |
| `/hesabim/adresler.vue` | **Kaldırılır** (adres defteri yok; otomatik doldurma yeterli) |
| `/giris`, `/kayit` | **YENİ** — giriş ve kayıt formları |

**Yeni composable `useCustomer()`** (HttpOnly cookie — JS token'a bakmaz):
- `register()`, `login()`, `logout()`, `me()`, `updateProfile()`, `myOrders()`
- Giriş durumu `/api/customer/me` ile belirlenir (cookie geçerliyse profil döner)
- Header: giriş varsa "Hesabım", yoksa "Giriş Yap"

**Sipariş formu otomatik doldurma:** `/siparis` açılınca `me()` çağrılır; giriş
varsa `buyerName`/`buyerPhone`/`buyerEmail` hazır dolu gelir. Müşteri
değiştirebilir (alıcı farklı olabilir — çiçekte sık).

**Kenar durumlar:**
- Giriş yapmadan `/hesabim` → `/giris`'e yönlendir
- Zaten giriş yapılmışken `/giris` → `/hesabim`'a yönlendir
- Kayıtta email zaten var → net hata: "Bu e-posta ile hesap var, giriş yapın"
- Sipariş verirken cookie süresi dolmuş → misafir siparişi olarak devam eder

---

## 7. Test Stratejisi

Mevcut desen (service/store katmanı, admin auth testleri referans). `make test`
(`-p 1`) — `go test ./...` KULLANILMAZ.

1. **Kayıt/giriş (en kritik)** — bcrypt hash doğru, email tekilliği, yanlış şifre
   reddi. *Admin auth testlerinin ikizi.*
2. **Auth ayrımı (güvenlik)** — müşteri token'ı admin ucuna erişemez; admin
   token'ı customer ucuna erişemez.
3. **Sipariş bağlama** — giriş yapmış müşterinin siparişi `customer_id` alır;
   misafir siparişi NULL kalır.
4. **Kendi siparişleri** — `/orders` yalnızca o müşterinin siparişlerini döner.
5. **Şifre değiştirme** — mevcut şifre doğrulanmadan değişmez.

Frontend: `useCustomer` için vitest (mevcut `useCart` deseni).

---

## 8. Kabul Kriteri

Müşteri `/kayit`'tan hesap açar → otomatik giriş → `/siparis` formu bilgileriyle
dolu gelir → sipariş verir → sipariş hesabına bağlanır → `/hesabim`'da geçmiş
siparişini görür. Misafir (girişsiz) sipariş akışı bozulmadan çalışır. Müşteri
token'ı admin paneline erişemez.

---

## 9. Sonraki Fazlara Devreden

- **Geçmişe dönük eşleştirme + e-posta doğrulama** — mail altyapısı gelirse.
- **Adres/alıcı defteri** — kullanıcılar isterse ayrı mini-faz (şema hazır:
  `customers` var, üstüne `addresses` eklenir).
- **Şifre sıfırlama** — mail altyapısına bağlı.
- **Hesap silme (KVKK)** — `ON DELETE SET NULL` şemada hazır; UI + uç eklenir.
- **Pazarlama/bildirim** — KVKK açık rıza + mail/SMS altyapısı.
