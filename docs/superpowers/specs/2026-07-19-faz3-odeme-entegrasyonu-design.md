# Faz 3 — Ödeme Entegrasyonu Tasarım Dokümanı

Tarih: 2026-07-19
Durum: Onaylandı, implementasyon planı bekliyor
Kaynak: `docs/superpowers/specs/2026-07-17-faz2-sepet-siparis-design.md` (§9 Faz 3'e
Devreden) üzerine yapılan tasarım tartışması

---

## 1. Amaç ve Kapsam

Müşterinin site üzerinden gerçek ödeme yaparak sipariş verebilmesi; esnafın ödenmiş
siparişleri panelde görüp işleyebilmesi ve gerektiğinde iade edebilmesi.

**Faz 2 sepeti buna hazır kurulmuştu.** Sipariş, fiyat doğrulaması, `order_items`
kopya alanları hep bu faz düşünülerek yapıldı. Bu faz onların üstüne ödeme adımını
ekler — sepeti veya siparişi baştan yazmaz.

### Alınan temel kararlar (tartışmada netleşen)

| Karar | Seçim | Gerekçe |
|---|---|---|
| Sağlayıcı | **PayTR** (iFrame API) | Kart bilgisi hiç bize gelmez → PCI/KVKK yükü minimum. Kullanıcı kararı. |
| Ödeme modeli | **Direkt çekim + iade** | Tek işletmeci, düşük hacim; provizyonun 2 adımlı karmaşıklığı bu ölçekte gereksiz. İade nadir olacak (çiçekçi çoğu siparişi karşılar). |
| Sipariş-ödeme sırası | **Önce kaydet → sonra öde** | Sipariş `awaiting_payment` olarak hemen kaydedilir; müşteri verisi kaybolmaz, PayTR'nin dar alan limitlerine takılmaz. |
| Onay akışı | **Tek adım (ödendi = onaylı)** | Para çekildi, sipariş kesin; esnafın ayrı "onayla" tıklaması gereksiz. Durum sayısı minimumda. |
| İade | **Panelden tek tıkla PayTR iadesi** | Esnaf için en kolay, hata payı düşük. |
| Ödenmemiş sipariş | **Esnafa görünmez** | `awaiting_payment` admin listesinde varsayılan gizli; esnaf sadece gerçek siparişlerle uğraşır. |
| Bildirim | **Yok (PayTR maili yeter)** | PayTR başarılı ödemede otomatik makbuz maili atar. Ek mail/SMS Faz 4. YAGNI. |

### Bu fazda var
- PayTR iFrame ödeme akışı (token → iframe → callback)
- Callback ile sunucu tarafı ödeme doğrulaması (hash), idempotent
- Ödeme durumları: `awaiting_payment` → `paid` → `delivered`, `refunded`
- Admin panelde ödenmiş sipariş listesi, ödeme bilgisi, tek tıkla iade
- `payment_events` denetim izi
- Mock provider ile geliştirme (gerçek PayTR anahtarları henüz yok)

### Bu fazda yok (bilinçli)

| Ne | Neden | Nereye |
|---|---|---|
| Provizyon (pre-auth) | Direkt çekim seçildi; PayTR standart üründe pre-auth vermiyor | (Sağlayıcı değişirse) |
| Stok takibi | Çiçekte stok kaygan; MVP §3.2 bilinçli kaçınmıştı, iade oranı düşük | Faz 4 (gerekirse) |
| Üyelik / `customers` | Faz 2 §1 ile aynı gerekçe hâlâ geçerli | Sonra |
| Bildirim (mail/SMS) | PayTR maili yeterli sinyal | Faz 4 |
| `awaiting_payment` temizlik job'u | Admin listesi zaten göstermiyor, birikme DB'de zararsız | Faz 4 (gerçekten gerekirse) |
| Mağaza içi ETBİS kaydı | Kod işi değil, kullanıcının işi | Kullanıcı (yayından önce) |

---

## 2. Mimari Kararlar

### 2.1 Uçtan uca akış

```
1. Müşteri sepeti doldurur → /siparis formunu doldurur → "Öde"ye basar
        ↓
2. Backend: siparişi 'awaiting_payment' kaydeder (fiyatları DB'den okuyarak — Faz 2 kuralı)
        ↓
3. Backend: PayTR get-token isteği atar (tutar + merchant_oid + hash) → token alır
        ↓
4. Frontend: PayTR iframe'ini token'la açar → müşteri kartını PayTR'nin ekranında girer
        ↓
5. Ödeme sonucu iki kanaldan gelir:
     (a) Müşteri /siparis/tamam veya /siparis/hata sayfasına yönlenir   [GÖRSEL — karar vermez]
     (b) PayTR → backend'e callback POST atar                           [GERÇEK ONAY — hash doğrulanır]
        ↓
6. Callback OK: sipariş 'paid' olur, paid_at set edilir, esnaf panelde görür
   Callback FAIL: sipariş 'awaiting_payment' kalır (esnafa görünmez)
```

### 2.2 Ödeme kararını sadece callback verir

**Tasarımın en kritik güvenlik kuralı.** Müşterinin yönlendiği sayfaya (5a,
`merchant_ok_url`) **asla güvenilmez** — müşteri o URL'i elle açabilir. Ödemenin
gerçekten olduğuna yalnızca sunucu-sunucu callback (5b) karar verir. Callback'te
PayTR'nin gönderdiği hash, `merchant_key` + `merchant_salt` ile yeniden üretilip
karşılaştırılır; tutmuyorsa reddedilir.

Bu, Faz 2'deki **"fiyata asla güvenilmez"** kuralının ödeme karşılığıdır: istemciden
gelen "ödendi" iddiasına değil, sağlayıcının imzaladığı callback'e güvenilir.

### 2.3 Callback idempotent olmalı

PayTR aynı callback'i birden fazla kez gönderebilir (ağ tekrarı, `OK` yanıtını
görmeme). Sipariş zaten `paid` ise ikinci callback'te **tekrar işlem yapılmaz**, yine
PayTR'ye `OK` döneriz. Yoksa çift işlem / tutarsız durum riski.

Idempotency, `payment_events` tablosuna bakılarak yapılır: bu `merchant_oid` için
`callback_ok` zaten işlenmiş mi?

### 2.4 PayTR'ye özel her şey izole

`order` paketi PayTR'yi **tanımaz** — yalnızca bir `PaymentProvider` arayüzü görür.
Hash formülü, endpoint adresleri, kuruş dönüşümü, iade çağrısı — hepsi
`internal/payment/` altında. İleride iyzico veya başka sağlayıcı gerekirse `order`
değişmez, sadece yeni bir provider implementasyonu yazılır.

---

## 3. Veri Modeli

### 3.1 Durum makinesi

```
awaiting_payment ──ödeme OK──→ paid ──esnaf──→ delivered
       │                        │
   (ödeme yok,               esnaf iade
    esnafa görünmez)            ↓
                            refunded
```

Geçerli durum seti: `awaiting_payment`, `paid`, `delivered`, `refunded`.

**Faz 2'nin eski durumları (`pending`/`confirmed`/`cancelled`) kaldırılıyor.**
Gerekçe: canlıda henüz gerçek sipariş yok (Faz 2+3 birlikte yayınlanacaktı — Faz 2
spec §Yayın stratejisi), dolayısıyla veri kaybı yok ve kodu/CHECK'i temiz tutmak
doğru. Migration eski CHECK'i yeni set'le değiştirir, default'u `awaiting_payment`
yapar.

**Geçersiz geçişler reddedilir** (service katmanında): ör. `refunded` → `delivered`
olmaz, `awaiting_payment` → `refunded` olmaz (çekilmiş para yok).

### 3.2 `orders` tablosu değişiklikleri (migration `000007`)

```sql
-- status CHECK'i yeni set'e güncelle:
--   ('awaiting_payment','paid','delivered','refunded'), default 'awaiting_payment'

ALTER TABLE orders
  ADD COLUMN paid_at        TIMESTAMPTZ,   -- ödeme onay anı (callback OK)
  ADD COLUMN refunded_at    TIMESTAMPTZ,   -- iade anı
  ADD COLUMN payment_ref    TEXT;          -- PayTR merchant_oid (callback eşleşme anahtarı)
```

### 3.3 `payment_events` tablosu (yeni)

```sql
CREATE TABLE payment_events (
  id           BIGSERIAL PRIMARY KEY,
  order_id     BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  event_type   TEXT NOT NULL,   -- 'token_requested','callback_ok','callback_fail','refund'
  raw_payload  JSONB,           -- PayTR'nin gönderdiği ham veri (denetim/debug)
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_payment_events_order ON payment_events (order_id);
```

**Neden var:** Ödeme para demek. "Callback geldi mi, ne dedi, hash tuttu mu"
sorularının cevabı bir yerde durmalı. Bir sipariş `paid` olduysa **hangi callback
yüzünden** olduğu buradan görülür. Idempotency kontrolü de buraya bakar. Faz 2'nin
"sipariş bir finansal kayıttır" prensibinin ödeme tarafı.

### 3.4 Şema kararlarının gerekçeleri

- **`payment_ref` = PayTR `merchant_oid`.** PayTR her ödemeye tekil bir sipariş
  kimliği ister ve callback'te geri gönderir. `order_no`'dan türetilir ama **tekil
  olmalı**: PayTR aynı `merchant_oid` ile ikinci token vermez, başarısız ödeme
  sonrası müşteri tekrar denerse yeni `merchant_oid` gerekir. Bu yüzden
  `order_no` + kısa random son ek şeklinde üretilir ve yeni denemede güncellenir.
  Callback geldiğinde bu alanla sipariş bulunur.

- **Tutarlar zaten `orders`'ta** (`items_total`, `delivery_fee`, `total`). PayTR'ye
  giden tutar `total`'dan **okunur, tekrar hesaplanmaz**. PayTR **kuruş bekler**:
  `total` (NUMERIC ₺) × 100, tam sayı olarak gönderilir. Bu dönüşüm tek yerde
  (`payment` paketi) yapılır; ondalık hatası olmamalı (test edilir).

- **`paid_at`/`refunded_at` ayrı kolonlar** (status'tan türetilmez): "ne zaman
  ödendi/iade edildi" finansal olarak önemli ve `updated_at` başka sebeplerle de
  değişir.

---

## 4. API

### 4.1 Public (auth yok)

`POST /api/orders` **değişiyor** — artık ödeme başlatır:

```
POST /api/orders
  → siparişi 'awaiting_payment' kaydet (fiyatları DB'den oku — Faz 2 kuralı)
  → PayTR get-token çağır
  → yanıt: { "order_no": "2607-0042", "total": "3750.00", "paytr_token": "xxx" }
```

Frontend bu token'la iframe açar. (`total` teşekkür/hata sayfası için dönüyor;
gerçek tahsilat tutarını iframe PayTR'den gösterir.)

`POST /api/payment/callback` **yeni** — PayTR sunucu-sunucu callback:

| Kural | Neden |
|---|---|
| Hash doğrulanır (`merchant_key` + `merchant_salt`) | Sahte callback ile sipariş `paid` yapılamasın — en kritik |
| Yanıt gövdesi düz `OK` (metin) olmalı | PayTR bunu görmezse callback'i tekrar tekrar gönderir |
| Idempotent — bu `merchant_oid` için `callback_ok` varsa no-op, yine `OK` | Çift işlem engeli |
| `status=success` → sipariş `paid`, `paid_at` set, `payment_events`+`callback_ok` | Ödeme onayı |
| `status=failed` → `payment_events`+`callback_fail`, sipariş dokunulmaz | İz kalır, sipariş `awaiting_payment` |
| Her durumda `payment_events`'e yazılır | Denetim izi |

PayTR yönlendirme URL'leri (görsel, karar vermez):
```
merchant_ok_url   → https://site/siparis/tamam
merchant_fail_url → https://site/siparis/hata
```

### 4.2 Admin (JWT korumalı)

```
GET   /api/admin/orders          → varsayılan: awaiting_payment HARİÇ (paid/delivered/refunded)
GET   /api/admin/orders/{id}     → + payment_events özeti
PATCH /api/admin/orders/{id}     → status: paid→delivered (Faz 2 mantığı, yeni set)
POST  /api/admin/orders/{id}/refund   → YENİ: PayTR iade API'sini çağır → 'refunded'
```

İade ucu kuralları:

| Kural | Neden |
|---|---|
| Sadece `paid` veya `delivered` sipariş iade edilebilir | `awaiting_payment`'ta çekilmiş para yok |
| PayTR iade API'si çağrılır; başarılıysa `refunded` + `refunded_at` | Tek tıkla iade kararı |
| PayTR reddederse sipariş durumu değişmez, hata döner | Yalan durum oluşmasın |
| `payment_events`'e `refund` yazılır | İz |

**Hata formatı** mevcutla aynı: `{"error": {"code": "...", "message": "..."}}`.

### 4.3 Config (`.env`)

```
PAYTR_MERCHANT_ID=
PAYTR_MERCHANT_KEY=
PAYTR_MERCHANT_SALT=
PAYTR_TEST_MODE=1                                  # anahtarlar gelene kadar test modu
PAYMENT_CALLBACK_URL=https://site/api/payment/callback
```

Anahtarlar henüz yok (başvuru sürecinde). Kod mock provider ve `PAYTR_TEST_MODE=1`
ile geliştirilip test edilir; gerçek anahtarlar gelince `.env`'e girilir ve PayTR
sandbox'ında gerçek kartla uçtan uca denenir.

---

## 5. Kod Organizasyonu

```
internal/payment/
  provider.go     → PaymentProvider arayüzü (StartPayment, VerifyCallback, Refund)
  paytr.go        → PayTR implementasyonu (hash, get-token, iade — HTTP)
  paytr_test.go   → hash üretimi/doğrulaması, kuruş dönüşümü testleri
  mock.go         → test/geliştirme için sahte provider

internal/order/
  service.go      → PaymentProvider'ı alır; Create artık ödeme başlatır
                  → ApplyCallback(merchantOid, status, hash) — callback işler
                  → Refund(orderID) — iade
  store.go        → paid_at/refunded_at/payment_ref güncelleme; payment_events yazma
```

`order` paketi `PaymentProvider` arayüzünü görür, PayTR'yi görmez (bkz. §2.4).

---

## 6. Frontend

### 6.1 Public site (`frontend/app/`, Nuxt SSR)

| Ne | Değişiklik |
|---|---|
| `/siparis` "Öde" butonu | Sipariş oluşturur → dönen `paytr_token` ile PayTR iframe açılır |
| PayTR iframe | PayTR'nin resmi `iframeResizer.js` ile gömülür; müşteri kartını orada girer |
| `/siparis/tamam` | Mevcut teşekkür sayfası (`merchant_ok_url`). **"Ödendi" DEMEZ** — "siparişiniz alındı, ödeme onaylanıyor" der (gerçek onay callback'te, §2.2) |
| `/siparis/hata` | YENİ — ödeme başarısız/iptal; sepet **korunur**, "tekrar dene" |
| Sepet temizleme | `/siparis/tamam`'a gelince temizlenir (Faz 2 davranışı korunur) |

**iframe seçildi (yönlendirme değil):** müşteri siteden ayrılmaz, tasarım bütünlüğü
korunur. Tek gereksinim PayTR CDN'inden `iframeResizer` scriptini yüklemek.

**CSP/güvenlik notu:** PayTR iframe'i harici script + harici iframe demek. Deploy'da
Caddy/CSP ayarları hassas (bkz. DURUM.md — public site routing geçmişi). PayTR'nin
script ve iframe origin'leri için CSP/`frame-src`/`script-src` izni gerekebilir.
**Plan aşamasında ele alınacak.**

### 6.2 Admin panel (`frontend/idare/`)

| Ne | Değişiklik |
|---|---|
| `/siparisler` liste | Sadece `paid`/`delivered`/`refunded` gösterir (awaiting_payment gizli) |
| Durum rozetleri | Ödendi / Teslim Edildi / İade Edildi |
| Sipariş detayı | Ödeme bilgisi bloğu: durum, `paid_at`, `payment_ref` |
| "İade Et" butonu | `paid`/`delivered` siparişlerde görünür → onay diyaloğu → `POST .../refund` |
| "Teslim Edildi" butonu | `paid` → `delivered` |

---

## 7. Test Stratejisi

Faz 2 ile aynı felsefe: service/store katmanında, HTTP kurmadan. `make test`
(`-p 1`) kullanılır — `go test ./...` KULLANILMAZ (DURUM.md flaky test kök nedeni).

1. **Hash doğrulama (en kritik)** — `payment` paketinde: doğru hash kabul, yanlış
   hash red. Sahte callback ile sipariş `paid` olmamalı. *Kırılırsa: bedava sipariş.*
2. **Idempotency** — aynı callback iki kez → sipariş bir kez `paid`, ikinci sefer
   no-op ama `OK`. *Kırılırsa: çift işlem.*
3. **Kuruş dönüşümü** — `1850.50 ₺` → `185050` kuruş, tam sayı, ondalık hatasız.
   *Kırılırsa: yanlış tutar çekilir.*
4. **Fiyat doğrulama (Faz 2'den devam)** — sipariş oluşurken fiyat DB'den okunur;
   PayTR'ye giden tutar DB toplamı.
5. **İade kuralları** — sadece `paid`/`delivered` iade edilebilir; `awaiting_payment`
   iade reddedilir; PayTR reddederse durum değişmez.
6. **Durum geçişleri** — geçersiz geçiş reddi (`refunded`→`delivered`, vb.).

**Mock provider ile geliştirme:** PayTR anahtarları henüz yok. `payment.MockProvider`
gerçek PayTR'yi taklit eder (token üretir, callback'i simüle eder). Uçtan uca akış
gerçek anahtar olmadan test edilir. Anahtarlar gelince PayTR sandbox'ında gerçek
kartla doğrulanır.

Frontend: mevcut `useCart` vitest korunur; iframe akışı PayTR sandbox gelince
manuel/E2E doğrulanır.

---

## 8. Kabul Kriteri

Müşteri sepete ürün atar → form doldurur → "Öde"ye basar → PayTR iframe açılır →
kartla öder → callback OK gelir → sipariş `paid` olur, esnaf panelde görür →
`delivered` işaretler. Esnaf yetişemezse panelden "İade Et" → PayTR iadesi →
`refunded`. Sahte callback (yanlış hash) reddedilir. Ödenmeyen sipariş esnafa
görünmez.

---

## 9. Sonraki Fazlara / Kullanıcıya Devreden

- **ETBİS kaydı** — kendi sitesinden satış yapan işletmeler için zorunlu (MVP §9).
  Kod işi değil, kullanıcının işi. **Yayından önce tamamlanmalı.**
- **`awaiting_payment` temizlik job'u** — gerçekten birikirse Faz 4.
- **Stok takibi** — iade oranı yüksek çıkarsa Faz 4'te değerlendirilir.
- **Üyelik / sipariş geçmişi** — Faz 2 §1 ile aynı, sonra.
- **Bildirim (mail/SMS)** — PayTR maili yetmezse Faz 4.
