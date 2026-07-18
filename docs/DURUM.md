# Proje Durumu

Son güncelleme: 2026-07-18

## Nerede kaldık

| Aşama | Durum |
|---|---|
| Tasarım (spec) | ✅ Bitti — `docs/superpowers/specs/2026-07-15-cicekci-mvp-design.md` |
| Plan 1 — Backend temeli | ✅ **Uygulandı** — 13/13 task, 100 test |
| Plan 2 — Görsel hattı | ✅ **Uygulandı** — 8/8 iş, 179 test toplam |
| Template — admin panel iskeleti | ✅ Eklendi ve Node 22'ye uyarlandı |
| Plan 4 — Admin panel | ✅ **Uygulandı** — 8/8 task, tarayıcıda doğrulandı |
| Plan 3 — Nuxt public site | ✅ **Uygulandı** — 8/8 task, production SSR doğrulandı |
| Sipariş yönetimi (Faz 2) | ✅ **Uygulandı** — sepet, teslimat ilçe/ücret, admin sipariş listesi/detayı |
| Deployment altyapısı (Faz A) | ✅ **Hazır** — Docker Compose + Caddy, lokalde prod moduyla test edildi |
| Final whole-branch review | ✅ **Yapıldı** (2026-07-18) — 8 açı, 7 bulgu, hepsi düzeltildi (aşağıya bkz) |
| Deployment (Faz B — VPS) | ⬜ Sen çalıştıracaksın — `DEPLOYMENT.md` ← SIRADA |

Branch: `feat/backend-temeli` (her şey burada)

Faz 3 (ödeme entegrasyonu/ETBİS) ve Faz 4 (raporlama, SEO, kampanya) bilinçli
olarak ertelendi — site üzerinden doğrudan satış/ödeme yok (WhatsApp'a
yönlendirme), bu yüzden ETBİS kaydı gerekmiyor. `docs/PROJECT_BRIEF.md` §Sonraki
Fazlar'a bakınız.

## Final review'da bulunup düzeltilen (2026-07-18)

8 açıdan (correctness ×3, reuse, simplification, efficiency, altitude,
conventions) çok-ajanlı review yapıldı, 7 bulgu doğrulanıp düzeltildi:

- **Timezone bug (kritik).** Prod container'larda `TZ` set edilmiyordu →
  UTC'de çalışıyordu. Türkiye sabit UTC+3 (DST yok). İstanbul saatiyle
  00:00–02:59 arası: sipariş tarihi/aynı-gün-kesim-saati mantığı, public
  sitedeki "Bugün" çipi ve sipariş numarası günlük sayacı yanlış gün
  hesaplıyordu. Düzeltme: `docker-compose.prod.yml` + `docker-compose.yml`'de
  `postgres`/`backend`/`app` servislerine `TZ: Europe/Istanbul` eklendi
  (Go'nun `time.Now()` container'ın local tz'sini okur, kod değişikliği
  gerekmedi).
- **Admin sipariş listesi offset/limit tutarsızlığı.** `order_handler.go`
  offset'i clamp edilmemiş ham `limit` ile hesaplıyordu. Şu anki admin
  arayüzü bu parametreleri hiç göndermediği için canlıda tetiklenmiyordu ama
  API sözleşmesinde gerçek bir hataydı. Düzeltildi: limit artık offset
  hesabından önce clamp ediliyor.
- **Sipariş listesi N+1 sorgu.** `Store.List` her sipariş için ayrı bir
  `itemsOf` sorgusu çalıştırıyordu. `itemsOfMany` ile tek batch sorguya
  indirildi (`WHERE order_id = ANY($1)`), regresyon testi eklendi
  (`TestStore_List_HerSiparisKendiKalemleriniAlir`).
- **Kod tekrarları.** `apiBase()` (`useOrders.ts`/`useApi.ts`) ve Türkçe para
  formatlayıcı (`idare`'de 3 sayfa) ortak yerlere taşındı
  (`frontend/idare/src/utils/Currency.ts`).

## Çalışan ne var

Tam bir Go backend'i. Gerçek sunucuda uçtan uca doğrulandı:
- Admin girişi (bcrypt + HttpOnly JWT cookie)
- Kategori CRUD — iki eksen (occasion/type), `is_active` + `is_featured`
- Ürün CRUD — slug geçmişi, eski linkler 301 ile yönleniyor
- İki eksenli AND filtresi
- Görsel hattı — yükleme, 2 boyut üretimi, sıralama, silme
- 179 test

**Ölçülen kazanç:** 446KB/2000x1500 fotoğraf → 21KB/400px + 108KB/1200px (20x)

Çalışan bir admin paneli (`frontend/idare/`). Gerçek tarayıcıda uçtan uca
doğrulandı (Playwright/Chromium):
- Giriş — HttpOnly cookie, localStorage'da token yok (`document.cookie` boş)
- Ürün listesi — kapak görseli, kategori chip'leri, pasif satır soluk
- Ürün formu — iki eksenli kategori seçimi, fiyat string olarak gidiyor
- Görsel yönetimi — çoklu yükleme, sola/sağa sıralama, kapak rozeti taşınıyor
- Kategori yönetimi — iki eksen ayrı tablo, silme öncesi ürün sayısı uyarısı
- `/siparisler` — Faz 2 placeholder'ı

**Ana bundle:** 406.66 kB → 334.46 kB (gzip 139.78 → 120.37) — dört ekran
eklenmesine rağmen ~%18 küçüldü (casl, apexcharts, tiptap, mapbox, swiper,
moment vb. + 16 kullanılmayan demo dialog silindi).

Çalışan bir public site (`frontend/app/`, Nuxt 4 SSR). Production build'e
(`node .output/server`) ve gerçek tarayıcıya karşı doğrulandı:
- Ana sayfa — featured kategoriler + öne çıkan ürünler
- Ürün listesi — iki eksenli AND filtresi, state URL'de (paylaşılabilir,
  geri tuşu çalışıyor), kombinasyonlar noindex
- Ürün detayı — galeri, WhatsApp sipariş butonu, **og:image SSR'da**
  (WhatsApp önizlemesi çalışıyor)
- Slug 301 — eski slug tarayıcı adresini kanonik slug'a taşıyor
- Kategori sayfaları (index), hakkımızda/iletişim, sitemap.xml
- Pasif ürün public'te yok (liste yok, detay 404); görselsiz ürün patlamıyor

Hazır deployment altyapısı (`DEPLOYMENT.md`, `docker-compose.prod.yml`,
`Dockerfile`'lar, `deploy/Caddyfile`). Tek VPS, tek domain path'e göre
dağıtım (Caddy: `/` → Nuxt SSR, `/idare` → admin statik, `/api` → backend).
Lokalde prod moduyla (self-signed TLS, APP_ENV=production) uçtan uca test
edildi: HTTPS routing, migration akışı, SSR→backend iç ağ, Secure cookie,
etkileşimsiz admin oluşturma, günlük yedek — hepsi çalışıyor.

## Sırada: Final whole-branch review + Deployment (Faz B)

Dört planın (1+2+3+4) tamamı `feat/backend-temeli`'nde. Whole-branch review
ertelenmişti — artık backend + admin panel + public site birlikte
gözden geçirilebilir. Deployment altyapısı hazır (Faz A); Faz B (gerçek VPS,
DNS, prod secret'lar, ilk deploy) kullanıcının sunucuda çalıştıracağı
adımlar — `DEPLOYMENT.md`.

## Uygulama sırasında verilen önemli kararlar

Bunlar spec'te yazılı olandan SAPMA veya spec'i tamamlayan kararlar.
Gerekçeleriyle birlikte, çünkü sonradan "neden böyle?" diye sorulacak.

**WebP → JPEG (spec §4.4 güncellendi).** `gen2brain/webp` (saf Go, WASM)
derlenirken Go derleyicisi bellek yetmezliğinden öldürüldü (`signal: killed`).
PaaS build container'ında da patlardı. cgo'lu alternatifler Docker imajına
`libwebp-dev` bağımlılığı getiriyordu. Gerekçe: asıl kazanç formatta değil
boyutta (20x ölçüldü); WebP'nin ek katkısı ~8KB/görsel, marjinal.

**Slug atomikliği (plan dışı düzeltme).** `store.Update` ve `store.AddSlug`
ayrı transaction'lardı. İlki başarılı + ikincisi başarısız = isim yeni, slug
eski VE bu kalıcılaşıyor (sonraki Update'te `current.Name` zaten eşleştiği
için düzeltme dalı hiç çalışmıyor). Probe testiyle kanıtlandı, tek
transaction'a alındı, atomiklik yine testle kanıtlandı.

**APP_ENV eklendi (plan dışı düzeltme).** Production tespiti
`strings.HasPrefix(SiteURL, "https://")` ile yapılıyordu. PaaS TLS'i proxy'de
sonlandırdığı için production'da `Secure` bayrağı olmayan auth cookie
üretilebilirdi — sessizce. Artık açık `APP_ENV` var ve `APP_ENV=production` +
`http://` kombinasyonunda sunucu açılmayı reddediyor.

**Flaky test kök nedeni.** `go test ./...` 4'te 3 patlıyordu, suçlu auth kodu
sanılıyordu. Gerçek sebep: tüm test paketleri aynı DB'yi paylaşıyor ve
`NewTestDB` TRUNCATE çalıştırıyor — paralel paketler birbirini siliyordu.
`make test` artık `-p 1` kullanıyor. **`go test ./...` KULLANMA.**

**Deployment: tek domain + path, VPS (kullanıcı kararı).** Subdomain yerine
`cicekci.com/idare` çünkü backend CORS'u tek origin alıyor — tek origin =
CORS derdi yok. Admin panel Vite `base=/idare/` ile derleniyor (router
`import.meta.env.BASE_URL`'i takip ediyor, dev'de '/' kalıyor). Public site
zaten kendi Nitro proxy'siyle same-origin. Caddy otomatik TLS.

**Prod compose'un dev postgres'i sildiği bug (yakalandı, düzeltildi).** İlk
`docker-compose.prod.yml` dev compose ile aynı proje adını (`cicekci`) ve
volume adını (`pgdata`) paylaşıyordu. Lokal prod testinin `down -v`'si dev'in
`cicekci_pgdata` volume'unu sildi — dev DB verisi gitti (şema + admin migrate
+ seed ile geri geldi). Düzeltme: prod compose `name: cicekci-prod`, volume'lar
ayrı isimli (`cicekci-prod_pgdata`). Artık tam izole; ikinci testte dev
postgres'e dokunulmadığı doğrulandı.

**cmd/seed etkileşimsiz mod (plan dışı, prod zorunluluğu).** `term.ReadPassword`
TTY istiyor; prod'da container içinde ilk admin oluşturmak için `-username`/
`-password` bayrakları eklendi. Bayraksız çağrı eski etkileşimli akışı koruyor.
Plan 4 ve public site testlerinde geçici Go programı yazmak zorunda kalmıştım —
bu artık kalıcı çözüm.

**Şeffaf PNG bug'ı.** `imaging.Paste` alfa kanalını yok sayıp kopyalıyordu,
şeffaf alanlar JPEG'de siyah çıkıyordu. `Overlay` ile harmanlanıyor.

**Public site API proxy'si (Plan 3, plan dışı ZORUNLU).** Plan "SSR olduğu
için CORS'a takılmaz" varsayıyordu. Yanlıştı: filtre chip'ine tıklanınca
Nuxt client-side gezinme yapıyor ve `useFetch` çağrıyı TARAYICIDAN yapıyor.
Backend CORS'u sadece admin origin'ini (`5173`) taşıdığı için public site
(`3000`) bloke oluyordu — filtre sessizce "0 ürün" gösteriyordu. Çözüm:
`frontend/app/server/api/go/[...path].ts` Nitro proxy'si. Composable'lar
same-origin `/api/go/*` çağırıyor, proxy sunucu-sunucu Go'ya iletiyor.
Admin uçlarına proxy yok (404). Gerçek Go adresi `goApiBase` **private**
runtimeConfig'te (`NUXT_API_BASE`), tarayıcıya sızmıyor. og:image ve ürün
görselleri proxy'den geçmiyor — onlar `<img>`, tarayıcı doğrudan 8080'den
yüklüyor (CORS `<img>`'e uygulanmaz). **İki SITE_URL:** admin panel
`SITE_URL=5173`, public site proxy sayesinde origin'den bağımsız çalışıyor.

**Görsel bölümü kaydettikten sonra açılmıyordu (Plan 4, plan dışı düzeltme).**
Ürün oluşturulunca `router.replace` ile `/urunler/yeni` → `/urunler/:id`
oluyor ama bileşen yeniden kurulmadığı için `onMounted` tekrar çalışmıyor,
`product` null kalıyor ve görsel bölümü boş görünüyordu. Esnaf ürünü
kaydedip fotoğraf ekleyemiyordu — sayfayı yenilemesi gerekiyordu. Planın
"esnafın yaşayacağı akış" dediği adımın tam ortası. `watch(rawId, loadProduct)`
ile çözüldü. **Sadece tarayıcıda göründü; curl ile görünmezdi.**

**`casl.ts` silinmedi, stub'a indirildi (Plan 4).** Plan `@casl` paketini
kaldırmayı ve `@core`/`@layouts`'a dokunmamayı birlikte söylüyordu ama
`@layouts`'taki beş navigasyon bileşeni `can`/`canViewNavMenuGroup`'u bu
dosyadan alıyor. Dosya her şeye izin veren stub'a çevrildi: paket gitti,
o beş dosyaya dokunulmadı. Rol sistemi gelirse burası tek değişecek yer.

**`make seed` script'ten çalıştırılamıyor.** `term.ReadPassword` gerçek TTY
istiyor. Plan 4'te admin oluşturmak için aynı `auth.CreateAdmin`'i çağıran
geçici program yazıldı, sonra silindi. Otomasyon gerekiyorsa cmd/seed'e
`--username`/`--password` bayrakları eklenebilir (Faz 2).

**Vuetify locale `tr` (Plan 4, plan dışı).** Tablo altbilgisi "Items per
page" diyordu. Arayüz dili Türkçe kararı (spec) hazır bileşen metinlerini
de kapsıyor.

## Reddedilen review bulguları (gerekçesiyle)

- **"`go mod tidy` çalıştırılmalı" (Task 1):** REDDEDİLDİ. O aşamada henüz
  import edilmemiş paketleri (fiber, pgx, jwt) go.mod'dan silerdi.
- **"Slug uzunluk sınırı yok" (Task 4):** REDDEDİLDİ. Şema kontrol edildi —
  slug kolonları `TEXT`, `VARCHAR(n)` değil. Postgres TEXT sınırı 1GB.

## Ertelenen kararlar (spec §8)

`products.group_id`, `settings` tablosu, müşteri üyeliği, VPS'e taşınma,
yetim R2 dosyası temizliği, Faz 3 stok/ödeme senkronu.

## Bilinen Minor bulgular

Final review'da triyaj edilecek — `.superpowers/sdd/plan1/progress.md` ve
`.superpowers/sdd/progress-plan2.md` içinde tam liste var. Öne çıkanlar:

- `WriteError`'da `ErrInvalidInput` mesajı `err.Error()` ile doğrudan
  kullanıcıya dönüyor. Şu an güvenli (sadece Türkçe doğrulama mesajları
  sarmalanıyor) ama biri DB hatasını `ErrInvalidInput` ile sarmalarsa iç
  detay sızar.
- `uniqueSlug`'da race — iki eşzamanlı Create aynı slug'ı görebilir. DB
  UNIQUE constraint engelliyor ama hata kaba düşüyor. Tek admin panelinde
  pratik risk yok.
- Goroutine içindeki `f.Listen` hatası `log.Fatalf` çağırıyor →
  `defer pool.Close()` atlanıyor. Başlangıç hatalarında olur, pratik etkisi yok.
- Template'te 17 tip hatası (`@core`/`@layouts`) — kullanıcı kararıyla bırakıldı.
  Plan 4'ten sonra da 17; yazılan panel kodunun tamamı tipli.
- 9 lint hatası `@core/utils/validators.ts`'te (regexp capturing group) —
  `@core`'a dokunulmadığı için duruyor. Panel kodunda lint temiz.
- Ürün listesinde sayfalama yok: `limit=100` ile tek sayfa. Backend toplam
  sayı dönmüyor, esnafın 40-100 ürünü olacak. 100'ü geçerse eklenecek.
- `uploads/products/` altında yetim klasör kalabiliyor (görsel kaydı silinip
  dosya kalması). Test sırasında bir örnek görüldü; spec §8'de zaten
  "yetim R2 dosyası temizliği" ertelenmiş kararlar arasında.

## Frontend yapısı

- `frontend/idare/` — Vuetify 3 + Vite SPA, admin paneli (Plan 4 ile çalışır durumda)
- `frontend/app/` — Nuxt 4 SSR public site (Plan 3 ile çalışır durumda)

**Neden ayrı:** Public site SSR olmak ZORUNDA — WhatsApp'ın önizleme botu
JavaScript çalıştırmıyor, SPA'da paylaşılan linkte ürün fotoğrafı çıkmaz.
Bu, tüm tasarımın dayandığı nokta (spec §5.1). Admin panelde SSR gereksiz.
