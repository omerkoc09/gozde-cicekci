# Çiçekçi Sipariş Sitesi — Faz 1 (MVP) Tasarım Dokümanı

Tarih: 2026-07-15
Durum: Onaylandı, implementasyon planı bekliyor
Kaynak: `docs/PROJECT_BRIEF.md` üzerine yapılan tasarım tartışması

---

## 1. Amaç ve Kapsam

Bir çiçekçi esnafı için ürün vitrini ve WhatsApp üzerinden sipariş yönlendirmesi
yapan site. Faz 1'de site üzerinden **ödeme alınmaz**; müşteri ürünü seçer,
WhatsApp'a yönlendirilir, sipariş orada tamamlanır.

Sitenin tek dönüşüm noktası WhatsApp butonudur. Tasarımın tamamı müşteriyi o
butona sağlıklı bir şekilde götürmek üzerine kuruludur.

### Bu fazda var
- Public site: ana sayfa, ürün listesi (iki eksenli filtre), ürün detayı,
  kategori sayfaları, hakkımızda, iletişim
- Admin panel (`/admin`): JWT login, ürün CRUD, kategori CRUD, görsel yönetimi
- WhatsApp sipariş yönlendirmesi (önceden doldurulmuş mesaj)
- SEO: SSR, meta etiketleri, sitemap.xml, WhatsApp link önizlemesi
- Görsel işleme: yükleme anında yeniden boyutlandırma + WebP

### Bu fazda yok (bilinçli)
- Ödeme entegrasyonu (Faz 3), sepet (Faz 2), teslimat planlama (Faz 2)
- Müşteri hesabı / üyelik, sipariş takibi — bkz. §3.3
- Gerçek stok takibi (Faz 2) — bkz. §3.2
- Ürün varyantları — bkz. §3.1

---

## 2. Teknoloji Stack

| Katman | Seçim | Not |
|---|---|---|
| Backend | Go + Fiber | Kullanıcının deneyimi bu yönde. Fiber fasthttp üzerine kurulu, `net/http` middleware ekosistemi doğrudan çalışmaz; Fiber'in kendi seti kullanılacak. |
| Veritabanı | PostgreSQL | |
| Migration | golang-migrate | Şema ilk günden migration ile yönetilir, elle DDL yok. |
| Frontend | Nuxt 3 (Vue 3, Composition API) | SSR zorunlu — bkz. §5.1 |
| Görsel saklama | Cloudflare R2 (S3-uyumlu) | `ImageStore` interface'i arkasında — bkz. §4.4 |
| Görsel işleme | `disintegration/imaging` + WebP encode | |
| Deployment | PaaS (Railway benzeri) + R2 | Kesin platform ürün hazır olunca seçilecek |
| Auth | JWT, HttpOnly cookie | |

---

## 3. Ürün Modeli Kararları

### 3.1 Fiyatlandırma: tek fiyat, varyant yok

Her ürünün **tek fiyatı** vardır. Boyut/adet farkı olan ürünler **ayrı ürün**
olarak girilir: "11 Gül Buket" ve "51 Gül Buket" iki farklı kayıttır.

**Neden:** Varyant sistemi (küçük/orta/büyük) tartışıldı ve reddedildi. Ayrı
ürün yaklaşımının kazandırdıkları:
- Her boyut kendi fotoğrafına sahip olur — 11 gül ile 51 gül gözle bambaşka
  şeylerdir, tek fotoğrafa sıkıştırmak ürünü yanlış tanıtır
- Her boyut kendi URL'ini ve SEO sayfasını alır ("51 gül buket" araması ayrı
  sayfaya düşer)
- Veri modeli tek tablo, admin formu düz kalır

**Kabul edilen bedel:** Ürün listesi tekrarlı görünür (aynı buket 3 kart),
açıklama 3 yerde tekrar eder, esnaf düzeltmeyi 3 kez yapar.

**Ertelenen karar:** `products.group_id` — aynı ürünün farklı boyutlarını
gruplayıp listede tek kart gösterme. Esnafın ürünlerinin kaçının gerçekten çok
boyutlu olduğu bilinmediği için ertelendi. Liste tekrarı rahatsız ederse
eklenecek; migration ucuz.

### 3.2 Stok yok, görünürlük var: `is_active`

`products.is_active` bir **görünürlük anahtarıdır, stok göstergesi değildir.**
Varsayılan `true`. Esnaf dönemsel olarak dokunur ("lale sezonu bitti, mayısa
kadar görünmesin"). Pasif ürün public tarafta hiç görünmez; admin panelde durur,
fotoğrafı ve açıklaması korunur.

**Neden stok boolean'ı değil:** Günlük "var/yok" güncellemesi gerçekçi değil —
esnaf iki hafta sonra dokunmayı bırakır ve stok bilgisi yalan söylemeye başlar.
Yanlış stok bilgisi, hiç bilgi olmamasından kötüdür. Faz 1'de yanlış bilginin
bedeli zaten sadece esnafın WhatsApp'ta "o bitti, şunu önereyim" demesidir —
ödeme yok, iade yok.

**Neden silme değil:** Silmek fotoğrafı ve açıklamayı da götürür; esnaf gelecek
sezon laleyi sıfırdan yükler.

**Faz 3'e taşınan açık soru:** Ödeme entegrasyonu geldiğinde "müşteri ödedi ama
ürün yok" senaryosu gerçek bir risk haline gelir. Bunu stok boolean'ı çözmez —
esnaf güncellemeyi unutursa boolean yine yalan söyler. Faz 3'te iki seçenek
değerlendirilecek: (a) siparişi "onay bekliyor" durumunda tutup esnaf
onayladıktan sonra çekim yapmak, (b) Faz 2'deki gerçek stok takibiyle satılanı
düşmek. **Bu karar Faz 3 tasarımında verilecek, unutulmamalı.**

### 3.3 Müşteri üyeliği yok, misafir kullanım

Site müşteriden hesap açmasını istemez. Kayıt/giriş ekranı, `customers` tablosu
yoktur. Faz 1'de tek auth admin panelidir (§4.5).

**Neden:** Üyeliğin işe yaraması için üyenin bir şey yapabilmesi gerekir; bu
fazda hiçbirinin karşılığı yok:
- *Sipariş geçmişi* — sipariş sitede oluşmuyor, WhatsApp'ta oluşuyor. Sitenin
  gösterecek geçmişi yok.
- *Kayıtlı kart* — ödeme yok.
- *Kayıtlı adres* — adres WhatsApp'ta konuşuluyor.
- *Favoriler* — çiçek alımı yılda 2-3 kez, plansız ve aceleyle yapılır; kimse
  bunun için hesap açmaz.

Karşılığında kayıt formu, şifre sıfırlama, e-posta doğrulama ve KVKK metni
gelir. En kritiği: sitenin tek dönüşüm noktası WhatsApp butonudur, araya konan
her adım o butona giden yolu uzatır. Çiçek alan insanın acelesi vardır (hastane
ziyareti, unutulan yıldönümü, cenaze) — hesap açmaz, çıkar gider.

**Kayıtlı adres bu sektörde e-ticaretteki kadar değerli değil:** çiçek
siparişinde adres genelde müşterinin değil **alıcının** adresidir ve her
seferinde değişir (anne, hastanedeki arkadaş, iş yeri). Trendyol'da adres
kaydetmek işe yarar çünkü ürün hep sana gelir; çiçekte ürün hep başkasına gider.

**Faz 2 üyelik gerektirmez:** Faz 2'nin sepeti ödeme almayan bir sepettir —
birden fazla ürün seçilir, tek WhatsApp mesajına dönüşür. Adres yine WhatsApp'ta
konuşulur.

**Faz 3'te tekrar değerlendirilecek** (§8). Beklenen yön: misafir sipariş
varsayılan, hesap opsiyonel ("bu bilgileri kaydet"). Zorunlu üyelik dönüşümü
düşürür.

**Bugün hazırlık gerekmiyor:** `customers` tablosu Faz 3'te mevcut şemaya
dokunmadan eklenebilir; `orders.customer_id` nullable olur (misafir siparişi =
NULL). Şimdi hazırlık yapmanın kazancı sıfır.

---

## 4. Backend Tasarımı

### 4.1 Veritabanı Şeması

```sql
products
  id            BIGSERIAL PK
  name          TEXT NOT NULL
  description   TEXT
  price         NUMERIC(10,2) NOT NULL
  is_active     BOOLEAN NOT NULL DEFAULT true
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()

product_slugs
  slug          TEXT PK
  product_id    BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE
  is_current    BOOLEAN NOT NULL DEFAULT true

product_images
  id            BIGSERIAL PK
  product_id    BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE
  image_key     TEXT NOT NULL
  sort_order    INT NOT NULL DEFAULT 0

categories
  id            BIGSERIAL PK
  name          TEXT NOT NULL
  slug          TEXT NOT NULL UNIQUE
  axis          TEXT NOT NULL CHECK (axis IN ('occasion','type'))
  is_active     BOOLEAN NOT NULL DEFAULT true
  is_featured   BOOLEAN NOT NULL DEFAULT false
  sort_order    INT NOT NULL DEFAULT 0

product_categories
  product_id    BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE
  category_id   BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE
  PRIMARY KEY (product_id, category_id)

admin_users
  id            BIGSERIAL PK
  username      TEXT NOT NULL UNIQUE
  password_hash TEXT NOT NULL
```

**Şema kararlarının gerekçeleri:**

- **`price` NUMERIC, float değil.** Para float'ta tutulmaz, yuvarlama hatası
  birikir. Go tarafında decimal tipi veya kuruş cinsinden `int64`.

- **`image_key`, tam URL değil.** Veritabanına domain yazılmaz. URL'i
  `ImageStore` üretir (§4.4). R2'den başka bir yere geçildiğinde tek config
  satırı değişir, veri migration'ı gerekmez. Boyut başına ayrı kolon da yok —
  iki boyut aynı key'den türetilir, üçüncü boyut gerekirse kolon değil sabit
  eklenir.

- **`axis` = kategori ekseni.** Brief'teki iki kategori listesi tek tabloda
  durur; `axis` hangisinin hangi listeye ait olduğunu söyler. "Gönderim amacına
  göre" (`occasion`) ve "ürün tipine göre" (`type`) iki farklı soru sorar, bir
  ürün ikisinden de kategori alır. İki ayrı tablo yerine tek tablo + eksen
  kolonu: junction bölünmez, CRUD ekranı bölünmez, sorgular tek yerde.
  Postgres enum yerine `TEXT` + `CHECK`: üçüncü eksen gerekirse constraint
  değiştirmek enum'a değer eklemekten kolay.

- **`is_active` ve `is_featured` ayrı boolean'lar.** Farklı sorular:
  `is_active` = "bu kategori sitede var mı", `is_featured` = "ana sayfada kart
  olarak çıkıyor mu". Tek boolean yetmez, kanıt: **Taziye** kategorisi sitede
  olmalı (`is_active=true`, insanlar arıyor, `/kategori/taziye` çalışmalı) ama
  ana sayfada doğum günü buketinin yanında görünmemeli (`is_featured=false`).
  `is_active=false` her şeyi ezer — pasif kategori featured olsa bile görünmez,
  böylece esnaf mevsimlik kategoriyi tek anahtarla geri getirir.

  | Kategori | `is_active` | `is_featured` | Sonuç |
  |---|---|---|---|
  | Doğum Günü | ✓ | ✓ | Hep var, ana sayfada kart |
  | Taziye | ✓ | ✗ | Hep var, ana sayfada yok |
  | Anneler Günü (nisan) | ✓ | ✓ | Sezonda, ana sayfada öne çıkıyor |
  | Anneler Günü (aralık) | ✗ | — | Sitede hiç yok |

- **`product_categories` composite PK.** Ayrı `id` yok; aynı ürün aynı
  kategoriye iki kez eklenemez, garanti veritabanında.

- **Kategori silme: CASCADE + admin uyarısı.** Kategori silinince junction
  kayıtları gider, **ürünler silinmez** — sadece o kategori etiketini
  kaybederler. Admin panel silme öncesi uyarır: *"Bu kategoride 15 ürün var.
  Silerseniz bu ürünler kategoriden çıkacak (ürünler silinmez). Devam?"*
  Soft delete (`deleted_at`) reddedildi: bu ölçekte maliyeti faydasından fazla,
  ve "geçici kaldır" ihtiyacı zaten `is_active` ile karşılanıyor. İki mekanizma
  iki ayrı niyet için: `is_active` = geçici gizle, silme = kalıcı kaldır.

- **Faz 2 hazırlığı:** `products.id` normal `BIGSERIAL`; Faz 2'de
  `order_items.product_id` buraya FK verir. **Dikkat:** sipariş anındaki fiyat
  ürünün bugünkü fiyatından farklı olacaktır — `order_items` kendi
  `price_at_order` kolonunu tutmalı, FK üzerinden güncel fiyatı okumamalıdır.

### 4.2 URL ve Slug Stratejisi

Ürün URL'i: `/urun/51-gul-buket` (temiz slug, ID yok).

**`product_slugs` geçmiş tablosu:** İsim değişince yeni slug üretilir, eskisi
`is_current=false` olarak tabloda kalır. İstek gelince slug aranır; slug eski
ise **301 redirect** ile güncel URL'e yönlendirilir.

**Neden:** Bu sitede her link bir WhatsApp mesajının içine gidiyor ve orada
aylarca yaşıyor. Esnaf ürün adını düzelttiğinde ("51 Gül Buket" → "51 Kırmızı
Gül Buketi") o eski linkler ölmemeli. Tek slug kolonuyla bu imkânsız: ya slug
sabit kalır (link yaşar, slug yanlış görünür) ya güncellenir (slug doğru, link
ölür). Geçmiş tablosu ikisini birden verir; 301 sayesinde Google da sıralamayı
yeni URL'e taşır.

Değerlendirilip reddedilenler:
- `/urun/42-51-gul-buket` (ID+slug): link garantisi verir ama URL çirkin ve bu
  linkler WhatsApp'ta insan gözünün önünde duruyor
- Slug'ı isimden ayırıp esnafa elle yönettirmek: çiçekçiden slug kavramını
  bilmesini beklemek gerçekçi değil; unutulan slug isimle tamamen alakasız
  kalır ("Kırmızı Gül Buketi" → `/urun/beyaz-orkide`)

**Slug üretim kuralları:**
- Türkçe karakter dönüşümü: `İ→i`, `ı→i`, `ş→s`, `ğ→g`, `ü→u`, `ö→o`, `ç→c`
- Küçük harf, boşluk → `-`, alfanumerik dışı temizlenir
- Çakışma varsa sonuna `-2`, `-3` eklenir

Kategori URL'i: `/kategori/dogum-gunu` — kategori isimleri neredeyse hiç
değişmez, geçmiş tablosuna gerek yok.

Filtre URL'i: `/urunler?amac=dogum-gunu&tip=buket` — query string, `noindex`.
Kombinasyonlar kombinatoryal patlar (10×6=60 sayfa) ve Google'da ince içerikli
sayfa üretir. Tekil kategoriler kendi path'lerini alır ve indexlenir.

### 4.3 Paket Yapısı

Domain dışta, HTTP ayrımı içeride:

```
cmd/
  server/main.go            → uygulama başlatma
  seed/main.go              → ilk admin kullanıcısı
internal/
  product/                  → model.go, service.go, store.go
  category/
  image/                    → store.go (interface), r2.go, local.go, process.go
  auth/                     → jwt.go, service.go, middleware.go
  api/
    app/                    → public handler + viewmodel + router
    idare/                  → admin handler + viewmodel + router
pkg/
  config/ database/ log/ errorsx/
migrations/
```

**Katmanlar:** `handler` → HTTP'yi bilir, iş mantığını bilmez. `service` → iş
mantığını bilir, HTTP'yi ve SQL'i bilmez. `store` → SQL'i bilir, iş mantığını
bilmez.

**Neden katmanlar:** Bu projede iş mantığı HTTP'ye ait olmayan gerçek işler
içeriyor — slug üretimi + çakışma + geçmiş yönetimi, görsel işleme + yarım
kalan yükleme temizliği, ürün silinince R2 temizliği, `is_active` filtresinin
public'te hep uygulanması. Bunlar handler'a gömülürse HTTP'ye bağlanır ve
test etmek için request kurmak gerekir. Ayrıca Fiber bağımlılığı tek katmanda
hapsolur.

**Neden `app` / `idare` ayrımı:** İki farklı kitle, iki farklı auth rejimi, iki
farklı veri görünümü. `product/service.go` tektir, ikisi de onu çağırır; ama
`api/app` sadece aktif ürünleri döndürür, `api/idare` hepsini döndürür ve JWT
ister. Viewmodel'ler ayrı: public'e `is_active` alanı gönderilmez.

**Neden domain dışta, katman dışta değil:** Ürüne bir alan eklerken beş klasör
dolaşmak yerine `internal/product/` içinde kalınır.

**Neden `internal/`, `pkg/` değil:** `pkg/` başka projelerin import edeceği kod
içindir. Bu tek uygulama; `internal/` derleyici seviyesinde dış importu engeller.
`pkg/` sadece gerçekten jenerik altyapı için (config, log, errorsx).

### 4.4 Görsel Yönetimi

**`ImageStore` interface'i** — mimari kısıt: saklama arkasında durur, bugün R2,
yarın disk, uygulama kodu değişmez.

```go
type ImageStore interface {
    Put(ctx context.Context, key string, size Size, data []byte) error
    Delete(ctx context.Context, key string) error
    URL(key string, size Size) string
}
```

İki implementasyon: `r2.go` (production), `local.go` (geliştirme/test).
Base URL `.env`'den gelir.

**Boyutlar:** `Size400` (liste kartları), `Size1200` (detay + `og:image`).
Orijinal saklanmaz.

**Format: sadece WebP.** JPEG fallback yazılmayacak — 2026'da desteklemeyen
kitle %1'in altında, iki format üretmek iki kat depolama ve `<picture>`
karmaşası demek.

**Yükleme akışı:** handler multipart alır → `imaging` ile 400 ve 1200 üretilir →
WebP encode → her ikisi de R2'ye yazılır → **ikisi de başarılıysa** DB'ye
`image_key` yazılır. Ara adımda hata olursa yazılanlar silinir. Senkron, kuyruk
yok (tek fotoğraf birkaç yüz ms).

**Çoklu fotoğraf:** Ürün başına birden fazla görsel, `sort_order=0` kapak.
Admin formunda yukarı/aşağı butonlarıyla sıralama (sürükle-bırak gerekmez).

**Neden çoklu:** Bu projede fotoğraf ürünün kendisidir. Buket tek açıdan
anlaşılmaz. Ayrıca sonradan `image_url` kolonundan `product_images` tablosuna
geçmek sadece migration değil — admin formu, detay sayfası, liste kartı ve
upload akışının yeniden yazılması demek. Görsel kodu bu projenin en çok
dokunulan yeri; iki kez yazılmamalı.

**Ürün silme:** Önce key'ler okunur → DB'den silinir (CASCADE junction ve
images'ı temizler) → R2'den dosyalar silinir. R2 silme başarısız olursa log'a
düşer; yetim dosya kalır ama site bozulmaz. Bu ölçekte kabul edilebilir.

### 4.5 Auth

`admin_users` tablosu + `cmd/seed` CLI komutu ile ilk kullanıcı oluşturulur.
Public kayıt ekranı yok.

**Neden tablo, `.env`'de hash değil:** Şifre değişimi için deploy gerekmesi
gerçek bir zaaf — güvenlik olayında şifre hızla döndürülebilmeli.

**JWT HttpOnly cookie'de**, localStorage'da değil: localStorage'daki token XSS
ile çalınabilir. `SameSite=Strict` + `Secure`.

Bcrypt ile hash.

### 4.6 API Rotaları

```
Public:
  GET  /api/products              ?amac=&tip=&page=
  GET  /api/products/{slug}       → eski slug ise 301
  GET  /api/categories            ?axis=
  GET  /api/categories/featured

Admin (JWT korumalı):
  POST   /api/admin/login
  GET    /api/admin/products      → is_active=false dahil
  POST   /api/admin/products
  PATCH  /api/admin/products/{id}
  DELETE /api/admin/products/{id}
  POST   /api/admin/products/{id}/images
  DELETE /api/admin/images/{id}
  PATCH  /api/admin/products/{id}/images/order
  GET    /api/admin/categories
  POST   /api/admin/categories
  PATCH  /api/admin/categories/{id}
  DELETE /api/admin/categories/{id}
```

Public uçlarda `is_active=false` filtresi **store katmanındadır** — handler'da
unutulabilecek bir yerde değil.

**Hata formatı:** `{"error": {"code": "...", "message": "..."}}`. Public tarafta
iç detay sızmaz, admin tarafta detaylı.

---

## 5. Frontend Tasarımı

### 5.1 Nuxt 3, SSR zorunlu

**Neden SPA değil:** Bu sitenin iki hedefi (SEO ve WhatsApp) de sunucudan gelen
HTML'e bağlı. **WhatsApp'ın link önizleme botu JavaScript çalıştırmaz** — SPA'da
paylaşılan link çıplak URL olarak görünür, ürün fotoğrafı çıkmaz. Bu sitede her
satış bir WhatsApp mesajından geçtiği için, fotoğrafın ilk göründüğü yer çoğu
zaman site değil, sohbet ekranıdır. `og:image` sunucudan gelmek zorundadır.

İkincil olarak Faz 4'teki "izmir çiçekçi" SEO hedefinin temeli de SSR'dır.

**Rendering stratejisi sayfaya göre:**
- Public sayfalar: SSR
- `/admin/*`: SPA (`ssr: false`) — bot görmez, SEO gereksiz, sunucu yükü boşuna

### 5.2 Sayfa Yapısı

```
pages/
  index.vue                 → featured kategoriler + öne çıkan ürünler
  urunler/index.vue         → liste + iki eksenli filtre
  urun/[slug].vue           → detay + galeri + WhatsApp butonu
  kategori/[slug].vue       → tek kategori sayfası (SEO)
  hakkimizda.vue
  iletisim.vue
  admin/
    login.vue
    index.vue               → ürün listesi
    urunler/[id].vue        → ürün formu
    kategoriler.vue
    siparisler.vue          → Faz 2 placeholder ("Faz 2'de gelecek")
```

Admin panel Nuxt içinde, ayrı proje değil: tek build, tek deploy, ortak tipler.
`ssr: false` zaten ayrımı sağlıyor.

### 5.3 WhatsApp Entegrasyonu

Mesaj formatı (sipariş niyetli + fiyatlı):

```
Merhaba, bu ürünü sipariş etmek istiyorum:
51 Gül Buket — 1.850₺
https://site.com/urun/51-gul-buket
```

```js
const msg = `Merhaba, bu ürünü sipariş etmek istiyorum:\n${product.name} — ${price}₺\n${url}`
const link = `https://wa.me/${phone}?text=${encodeURIComponent(msg)}`
```

**Neden bu format:** "Bilgi almak istiyorum" yerine "sipariş etmek istiyorum" —
müşteri kendi ağzından niyetini kurar, konuşma pazarlıkla değil siparişle başlar.
Fiyatın mesajda olması esnafı korur (müşteri hangi fiyatı gördüğünü belgelemiş
olur) ve esnaf ürünü sitede aramak zorunda kalmaz.

**Reddedilen:** Mesaja teslimat/tarih/kart mesajı alanları koymak. Bunlar Faz
2'nin işi; şablona form gibi alanlar koymak müşteriye ödev listesi yaratır, çoğu
silip "merhaba" yazar.

`encodeURIComponent` Türkçe karakterleri ve satır başlarını güvenli kodlar.

Telefon numarası `.env`'de (`WHATSAPP_NUMBER`). **Ertelenen:** `settings`
tablosu — numara yılda bir değişir, ayarlar ekranı YAGNI.

İletişim sayfası içeriği (adres, telefon, çalışma saatleri, harita) `.env`'den
gelir — ürün/kategori gibi veritabanı varlığı değil, config. Aynı gerekçe:
yılda bir değişir. Değişirse `settings` tablosu tetiklenir (§8).

**Esnafa kurulum tavsiyesi (kod işi değil):** WhatsApp Business'a geçmesi
önerilir — ücretsiz, otomatik karşılama mesajı ve işletme profili verir, beklenti
yönetir. `wa.me` linki normal WhatsApp'ta da aynen çalışır. WhatsApp Business
**API** (ücretli, kurumsal) bu proje için gereksizdir.

### 5.4 Görsel Sunumu

- `<NuxtImg loading="lazy">` — ekranda görünmeyen fotoğraflar ertelenir, sayfa
  daha hızlı açılır
- Liste kartlarında 400px, detayda 1200px
- Sabit `aspect-ratio` ile yer tutucu — fotoğraf inerken sayfa zıplamaz
- `og:image` = 1200px görsel

### 5.5 SEO

- `useSeoMeta()` her sayfada: title, description, `og:title`, `og:description`,
  `og:image`
- `@nuxtjs/sitemap` — aktif ürün ve kategorilerden üretilir
- Filtre kombinasyonları `noindex`
- Semantik HTML

### 5.6 Filtreleme

İki ayrı filtre grubu: "Gönderim Amacına Göre" (`axis=occasion`) ve "Ürün Tipine
Göre" (`axis=type`). Kullanıcı ikisinden de seçebilir; sonuç **her iki koşula da
uyan** ürünlerdir (AND, OR değil).

Filtre state'i URL query string'inde tutulur — filtrelenmiş liste paylaşılabilir,
tarayıcı geri tuşu çalışır.

---

## 6. Test Stratejisi

Test edilmeye değen üç şey, `service` ve `store` katmanında (HTTP kurmadan):

1. **Slug mantığı** — Türkçe karakter dönüşümü, çakışma (`-2` eki), eski slug'ın
   301'i, isim değişince geçmiş kaydının doğru yazılması. En çok bug çıkacak yer,
   saf fonksiyon, test etmesi kolay.
2. **Filtre sorgusu** — "Doğum Günü + Buket" gerçekten ikisine de uyan ürünleri
   mi getiriyor? AND/OR hatası bu tür sorgularda klasiktir. Gerçek DB'ye karşı.
3. **`is_active` sızıntısı** — public uçların pasif ürün/kategori döndürmediği.
   Regresyon testi.

Handler testleri az: çoğunlukla auth kontrolü.

---

## 7. Uygulama Sırası

Her adım kendi başına test edilebilir olmalı.

1. Migration'lar + config + DB bağlantısı
2. `auth` — seed komutu, login, JWT middleware (admin uçları buna bağlı)
3. `category` — CRUD, iki eksen, `is_active`/`is_featured`
4. `product` — CRUD, slug mantığı + geçmiş tablosu, kategori bağlama
5. `image` — ImageStore, R2, işleme, upload uçları
6. Nuxt — public sayfalar (liste, detay, filtre, WhatsApp butonu)
7. Nuxt — admin panel
8. SEO — meta etiketleri, sitemap
9. Deploy

---

## 8. Ertelenen Kararlar

Bilinçli olarak ertelendi; tetikleyicisi geldiğinde ele alınacak.

| Karar | Tetikleyici | Maliyet |
|---|---|---|
| `products.group_id` | Liste tekrarı esnafı rahatsız ederse | Migration + liste/detay değişikliği |
| `settings` tablosu | İletişim bilgileri sık değişirse veya esnaf kendi yönetmek isterse | Küçük key-value tablo + admin ekranı |
| VPS'e taşınma | PaaS maliyeti/kısıtları sorun olursa | `ImageStore` sayesinde config değişikliği |
| Yetim R2 dosyası temizliği | Birikirse | Küçük CLI komutu |
| Müşteri üyeliği (`customers`) | Faz 3 ödeme entegrasyonu — **§3.3** | Mevcut şemaya dokunmaz; `orders.customer_id` nullable |
| Faz 3 stok/ödeme senkronu | Ödeme entegrasyonuna geçilince — **§3.2'deki açık soru** | Faz 3 tasarımı |

---

## 9. Sonraki Fazlar (referans)

**Faz 2 — Sipariş yönetimi ve teslimat planlaması**
- Site içi sepet (ödeme yok, sepeti WhatsApp mesajına dönüştürme)
- Teslimat tarihi/saat aralığı seçimi, kart mesajı alanı
- Admin panelde sipariş listesi (`/admin/siparisler` placeholder'ı hazır)
- Gerçek stok/durum takibi

**Faz 3 — Gerçek ödeme entegrasyonu**
- iyzico/PayTR
- **§3.2'deki açık soru burada çözülecek:** ödeme öncesi esnaf onayı mı, gerçek
  stok düşümü mü
- **§3.3 burada tekrar değerlendirilecek:** müşteri üyeliği — beklenen yön
  misafir sipariş varsayılan, hesap opsiyonel
- İade/iptal akışı
- ETBİS kaydı gerekecek (kendi sitesi üzerinden doğrudan satış yapan işletmeler
  için zorunlu)

**Faz 4 — Büyüme ve optimizasyon**
- Gelir/sipariş raporlama
- SEO derinleştirme (blog, bölge bazlı sayfalar — "izmir çiçekçi")
- Kampanya/indirim kodu
- Müşteri hesabı/sipariş geçmişi
- E-posta/SMS bildirimleri
