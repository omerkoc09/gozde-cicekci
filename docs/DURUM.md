# Proje Durumu

Son güncelleme: 2026-07-19

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
| Deployment (Faz B — VPS) | ✅ **CANLI** (2026-07-18) — https://gozdetasarimcicekcilik.com, Hetzner CPX12 |
| Faz 3 — Ödeme (PayTR) | 🟡 **CANLIDA — TEST MODUNDA** (2026-08-14) — gerçek PayTR provider aktif, `test_mode: true`. Müşteri ödeme YAPAMAZ; ETBİS tamamlanınca `PAYTR_TEST_MODE=0` yapılacak |
| Üyelik / müşteri hesabı | ✅ **CANLI** (2026-08-14) — 11 task + review/fix turları; migration 9 uygulandı, tarayıcıda doğrulandı |
| Yedekleme | ✅ **Doğrulandı + off-site** (2026-08-14) — günlük db+uploads, R2'ye kopya, 30 gün saklama, restore testi geçti |
| Panel erişimi + sıralama | ✅ **Uygulandı** (2026-08-16) — footer'da "Yönetim" linki, sıra sayısı yerine ok butonları |
| Ürün özelleştirme (buket tasarla) | ✅ **Uygulandı** (2026-08-16) — migration 10, seçenek grupları panelden yönetiliyor |

### Sıradaki deploy (2026-08-16 çalışması)

Branch `panel-erisim-ve-siralama` main'e merge edilip deploy edilecek.
**Migration 10** (`option_groups`, `option_values`, `product_option_groups`,
`order_item_options` + üç grup × 10 renk seed) compose'daki `migrate`
servisiyle OTOMATİK uygulanır — elle komut gerekmez. Migration patlarsa
backend hiç başlamaz (`service_completed_successfully` bağımlılığı).

Deploy sonrası doğrulanacak: `/idare/secenekler` açılıyor mu, bir üründe
grup aktifleştirilip public sayfada renkler görünüyor mu, sipariş
detayında seçimler renk noktasıyla listeleniyor mu.

Branch: `main` (deploy edilen). **2026-08-13:** `uyelik` main'e merge edildi;
`uyelik` `odeme-sistemi` üstüne kurulduğu için PayTR de aynı merge'le main'e
geldi (49 commit). Yani main artık HEM üyelik HEM ödeme içeriyor.

### Deploy (2026-08-14 gecesi) — TAMAMLANDI ✅

Doğrulananlar:
- `migrate-1 | 9/u customers` — migration 9 uygulandı (`customers` +
  `orders.customer_id`). Prod'da migration compose'daki `migrate` servisiyle
  OTOMATİK çalışıyor; `backend` onun başarıyla bitmesini bekliyor
  (`service_completed_successfully`), yani migration patlarsa backend hiç
  başlamaz. Elle migrate komutu ÇALIŞTIRMAYA GEREK YOK.
- `ödeme: PayTR provider aktif (test_mode: true )` — doğru mod.
- Tarayıcıda uçtan uca doğrulandı (kayıt, giriş, sipariş, hesabım, admin
  müşteriler sayfası).

> ⚠️ **PayTR TEST MODUNDA** — müşteri GERÇEKTEN ödeme yapamaz, sipariş
> `awaiting_payment` statüsünde kalır. Karar: ETBİS beklediği için (PayTR
> zaten ETBİS'siz canlı moda izin vermiyor) ve henüz gerçek satış yok.
>
> **Gerçek satışa başlarken:** `.env.prod`'da `PAYTR_TEST_MODE=0` yapıp
> servisi yeniden başlatın, sonra logdan `test_mode: false` olduğunu
> doğrulayın. Bu adım atlanırsa siparişler sessizce yarım kalır.
>
> Kod varsayılanı güvenli tarafta: `os.Getenv("PAYTR_TEST_MODE") != "0"` —
> değişken yoksa/boşsa test modu. Canlıya geçmek açıkça `=0` yazmayı ister.

## Üyelik / Müşteri Hesabı, 2026-08-13

Opsiyonel e-posta+şifre müşteri üyeliği. Misafir siparişi aynen korunuyor —
üyelik hiçbir yerde zorunlu değil.

**Mimari:** `internal/customer/` paketi, mevcut `internal/auth/` (admin)
deseninin ayrı ikizi. Ayrı tablo (`customers`), ayrı cookie (`customer_token`,
admin'inki `cicekci_token`), ayrı middleware. JWT claim'inde `typ:"customer"` —
admin token'ı müşteri ucuna geçemiyor.

**Lokal E2E doğrulandı (2026-08-13):**

| Senaryo | Sonuç |
|---|---|
| Kayıt → HttpOnly cookie, yanıtta şifre yok | ✅ 201, `SameSite=Strict`, 7 gün |
| Giriş / yanlış şifre / cookie'siz `/me` | ✅ 200 / 401 / 401 |
| Giriş yapmışken sipariş → `customer_id` bağlanır | ✅ `1308-0001 → customer_id=1` |
| Misafir sipariş → `customer_id` NULL | ✅ `1308-0002 → NULL` (akış bozulmadı) |
| `/customer/orders` yalnız kendi siparişleri | ✅ misafir siparişi görünmüyor |
| Çıkış sonrası `/me` | ✅ 401 |
| **Admin token'ı müşteri ucunda** | ✅ **401** — aynı token `/api/admin/orders`'da 200, `/api/customer/me`'de 401 (tip ayrımı kanıtlandı) |

**Bilinen (üyelikle ilgisiz):** `pkg/config` PayTR testleri (
`TestLoad_PayTRDefaultsUnconfigured`, `TestLoad_PayTRUnconfiguredWhenPartiallySet`)
kırmızı — `setBaseEnv` PAYTR_* değişkenlerini temizlemiyor, `config.Load()` kök
`.env`'deki gerçek anahtarlara düşüyor. Bu branch'ten önce de kırıktı.

**Dev ortam notu:** cicekci postgres portu 5433 → **5435** taşındı; 5433'ü başka
bir projenin container'ı (`backend-db-1`) tutuyordu ve `localhost:5433` yanlış
veritabanına gidiyordu. Test DB 5434'te değişmedi.

## Ürün Özelleştirme ("Buket Tasarla"), 2026-08-16

Spec: `docs/superpowers/specs/2026-08-16-urun-ozellestirme-design.md`
Plan: `docs/superpowers/plans/2026-08-16-urun-ozellestirme.md`

Müşteri sipariş ederken ambalaj/kurdele/kutu rengi seçebiliyor. Seçenek
grupları **panelden** yönetiliyor (`/idare/secenekler`) — yeni bir grup
("Çiçek Rengi" vb.) eklemek kod değişikliği veya migration GEREKTİRMİYOR.
Migration 10'daki üç grup yalnızca başlangıç verisi; silinebilir,
yeniden adlandırılabilir.

**Kararlar:** merkezi seçenek havuzu + ürün başına aç/kapa, grup başına
tip (`color`/`text`), **fiyat farkı YOK**, farklı seçim = ayrı sepet kalemi.

**Zorunluluk kaldırıldı, yerine otomatik ilk seçim (2026-08-16).** Önce
"ürün başına zorunlu grup" vardı; müşteri seçmeden sepete ekleyemiyordu.
Bunun yerine ürün sayfası açılınca **her grubun ilk değeri otomatik
seçili** geliyor. Müşteri hiçbir şeye dokunmasa bile sipariş
"Ambalaj: Pembe" ile gidiyor — esnaf her siparişte ne hazırlayacağını
görüyor, müşteriye de fazladan adım çıkmıyor. Sunucu artık seçimsiz
siparişi de reddetmiyor.

> `product_option_groups.is_required` sütunu DB'de DURUYOR ama hiçbir kod
> okumuyor/yazmıyor (INSERT varsayılanı `false`). Migration yazılmadı:
> karar geri alınırsa sütun yerinde. Şemada ölü bir sütun olduğunu bilin.

### Kritik tasarım noktaları (sonradan "neden böyle?" diye sorulacak)

**Seçim siparişe KOPYALANIR.** `order_item_options` gruba/değere referans
tutmaz — `group_name`, `value_name`, `swatch_hex` kopyalanır. Esnaf
sonradan "Pembe"yi silerse eski siparişin ne olduğu bilgisi bozulmaz.
`order_items.product_name`/`price_at_order` deseninin aynısı. Şema
seviyesinde garanti: o tabloda `option_values`/`option_groups` FK'si YOK.
Regresyon testi var (`TestCreate_DegerSilininceEskiSiparisBozulmaz`).

**Sunucu tarayıcıya güvenmez.** İstemci YALNIZCA `option_value_ids`
gönderir; isim ve renk DB'den okunur. Reddedilenler: değer yok/pasif,
değerin grubu bu ürüne kapalı, aynı gruptan birden fazla değer, zorunlu
grup eksik. Fiyattaki *"FİYAT YOK — sunucu DB'den okur"* kuralının aynısı.

**Sepet birleştirme anahtarı** `product_id` → `product_id + sıralı seçim
id'leri` oldu (`cartLineKey`). `addItem`/`removeItem`/`setItemQuantity`
ÜÇÜ BİRDEN değişti — yoksa "pembeyi sil" beyazı da silerdi. Seçim
id'leri sıralanıyor: müşteri renkleri hangi sırayla seçerse seçsin aynı
satır olmalı.

**Eski sepetler korundu.** Bu alandan önce kurulmuş sepetlerde `options`
alanı yok; okuma tarafı `undefined`'ı boş dizi kabul ediyor, kimsenin
sepeti sıfırlanmadı.

**Ödemeye etkisi yok.** Fiyat farkı olmadığı için `itemsTotal`, PayTR
sepeti ve callback tutar doğrulaması değişmedi.

**Grup bağlama kısmi başarısı önlendi.** `SetProductGroups` ürün
create/update transaction'ının içine ALINAMIYOR (`productoption` zaten
`product`'ı import ediyor, tersi döngü olur). Bunun yerine
`ValidateGroupLinks` ile gruplar ürün kaydedilmeden ÖNCE doğrulanıyor —
geçersiz `group_id` artık ürünü hiç oluşturmuyor. Kalan risk: doğrulama
ile yazma arasındaki çok kısa pencerede grup silinirse (DB arızası
sınıfı, pratikte yok denecek kadar nadir).

**Renk seçici görünümü.** Nokta 22px, seçilide noktanın DIŞINDAN geçen
altın halka — halka dışarıda olduğu için seçim değişince satır zıplamıyor.
Başlık `text-label-caps` (uppercase, secondary rengi), yanında seçilen
rengin adı: renk tek başına bilgi taşımamalı (erişilebilirlik).

### Sıralama artık elle girilmiyor (aynı turda yapıldı)

Kategori ve slider formlarındaki "Sıra" sayı kutusu kaldırıldı; yerine
listede ▲▼ ok butonları. Yeni kayıt sunucuda sona ekleniyor
(`MAX(sort_order)+1`), sıra yalnızca `PUT .../reorder` ucundan değişiyor.
Uçlar listenin TAMAMINI ister — kısmi liste, listede olmayan kayıtları
sessizce 0'a düşürüp sırayı bozardı.

> **Tuzak (üç sayfada düzeltildi):** ok butonları satır index'ine
> dayanıyor. `VDataTable` kolonları varsayılan olarak sıralanabilir
> olduğu için başlığa tıklandığında ekrandaki sıra değişiyor ve bir
> sonraki ok tıklaması GÖRÜNMEYEN bir çifti takas ediyordu.
> `secenekler.vue`, `kategoriler.vue` ve `slider.vue`'da tüm kolonlar
> artık açıkça `sortable: false` — gerekçesi kodda yorumla yazılı.

Ayrıca public site footer'ına `/idare` linki eklendi ("Yönetim",
`rel="nofollow"`) — esnafın panele girmek için URL ezberlemesi gerekmiyor.

## Yedekleme, 2026-08-14

`docker-compose.prod.yml`'deki `backup` servisi günde bir kez (`sleep 86400`
döngüsü) çalışır: `pg_dump -Fc` + `uploads` arşivi → `/backups` (host'ta
`~/cicekci/backups`), 7 günden eskiler silinir. Ayrıca **R2'ye off-site
kopyalanır**.

**Rutin kontrol (ayda bir yeter):**
```bash
cat ~/cicekci/backups/SON_BASARILI    # son başarılı yedeğin damgası
cat ~/cicekci/backups/SON_HATA        # yoksa sorun yok
```
`SON_BASARILI` güncel değilse veya `SON_HATA` varsa yedekleme bozulmuş.
Mantık: başarılı turda damga güncellenir + `SON_HATA` silinir; başarısız
turda damga DOKUNULMAZ (böylece "en son gerçekten ne zaman yedek alındı"
bilgisi kaybolmaz).

**Off-site:** R2 bucket `cicekci-yedek` — görsellerin bucket'ından AYRI ve
**private**. Dump'lar müşteri e-postası/telefonu ve bcrypt hash'i içeriyor.
Lifecycle rule: 30 gün sonra otomatik silinir (~450 MB'da sabitlenir,
ücretsiz 10 GB kotanın %4.5'i). `R2_BACKUP_BUCKET` tanımsızsa kopyalama
sessizce atlanır, yerel yedek yine alınır.

> Prod `.env.prod`'da `STORAGE_DRIVER=local` — R2 değişkenleri YALNIZCA
> yedek servisi için tanımlı, görseller diskte duruyor.

**Restore testi (2026-08-14 yapıldı, geçti — yılda bir tekrarlanmalı):**
```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod exec postgres createdb -U cicekci restore_testi
docker compose -f docker-compose.prod.yml --env-file .env.prod exec backup pg_restore -d restore_testi -U cicekci -h postgres /backups/<dosya>.dump
docker compose -f docker-compose.prod.yml --env-file .env.prod exec postgres psql -U cicekci -d restore_testi -c "\dt"
docker compose -f docker-compose.prod.yml --env-file .env.prod exec postgres dropdb -U cicekci restore_testi
```
**Ders:** Yedeğin var olması yetmez — geri yüklenebildiği doğrulanmadıkça
yedek sayılmaz. Hata görünürlüğü eklendiği ilk turda gerçek bir hatayı
(boş `R2_ACCOUNT_ID` → rclone DNS hatası) yakaladı; öncesinde bu sessizce
kaybolurdu.

## Faz 3 — Ödeme Entegrasyonu (PayTR), 2026-07-19

Spec: `docs/superpowers/specs/2026-07-19-faz3-odeme-entegrasyonu-design.md`
Plan: `docs/superpowers/plans/2026-07-19-faz3-odeme-entegrasyonu.md`

**Kararlar:** PayTR iFrame API, direkt çekim + iade (provizyon değil), önce
sipariş kaydet (`awaiting_payment`) → sonra öde, tek adım onay (ödendi=onaylı),
panelden tek tıkla PayTR iadesi, ödenmemiş sipariş esnafa görünmez, bildirim yok
(PayTR maili yeter). Anahtarlar gelene kadar `MockProvider` ile geliştirildi.

**Statü seti değişti:** `awaiting_payment` → `paid` → `delivered`, `refunded`.
Eski `pending`/`confirmed`/`cancelled` kaldırıldı (canlıda gerçek sipariş yoktu).

**Mock ile uçtan uca doğrulandı (2026-07-19):** sipariş oluştur→`awaiting_payment`
+`payment_ref` ✅, callback success→`paid`+`paid_at`+`callback_ok` izi ✅,
idempotency (aynı callback iki kez→ikisi de `OK`, tek `callback_ok`) ✅, admin
listesi `awaiting_payment` gizliyor ✅, iade→`refunded`+`refunded_at`+`refund` izi ✅.

**E2E'de bulunup düzeltilen KRİTİK bug (BUG B):** `payment_events.raw_payload`
JSONB; handler callback denetim izine PayTR'nin **form-encoded** ham gövdesini
(`c.Body()`) yazıyordu → geçerli JSON olmadığı için JSONB reddediyor, hata
`AddPaymentEvent`'te (`_ =`) yutuluyor → `callback_ok` hiç yazılmıyor →
idempotency `HasPaymentEvent`'e dayandığı için ikinci callback'te bozuluyor,
handler PayTR'ye `FAIL` dönüyordu (sonsuz retry). Fix: parse edilmiş
`CallbackInput` JSON'a marshal edilip yazılıyor; yazım hataları artık loglanıyor.
Regresyon testi eklendi (`fix(payment)` commit'i). **Ders: unit testler payload
olarak geçerli JSON geçiriyordu; gerçek form-encoded yolu test edilmemişti.**

**Bilinen kırılganlık (BUG A, prod'u BUGÜN etkilemiyor):** `order_no` günlük
öneki Go `time.Now()` (Go TZ) ile, günlük sayaç `Postgres CURRENT_DATE` (Postgres
TZ) ile üretiliyor. İki TZ ayrışırsa (ör. UTC Postgres) `order_no` çakışıp sipariş
oluşturma 500 verir. Prod'da compose her iki servise de `TZ=Europe/Istanbul`
verdiği için tutarlı. Faz 2 kodu; Faz 3 doğrulaması (lokal UTC Postgres) ortaya
çıkardı. Sayaç ve önek aynı TZ kaynağından beslenmeli — ayrı iş olarak ele alınacak.

**Sırada (Faz 3 kapanışı):** (1) whole-branch review, (2) gerçek PayTR sandbox
anahtarlarıyla uçtan uca test — Task 12 (callback localhost'a ulaşmaz, tünel/deploy
gerekir), (3) ETBİS kaydı (kullanıcının işi, canlıya almadan önce zorunlu).

Faz 4 (raporlama, SEO, kampanya) ertelendi. `docs/PROJECT_BRIEF.md` §Sonraki
Fazlar'a bakınız.

## Canlı ortam (2026-07-18)

- **Domain:** gozdetasarimcicekcilik.com (Cloudflare'de kayıtlı, DNS-only —
  proxy KAPALI, çünkü Caddy kendi Let's Encrypt TLS'ini alıyor)
- **Sunucu:** Hetzner CPX12 (2 vCPU / **2GB RAM**), Falkenstein, Ubuntu 24.04,
  IP 167.233.225.38
- **Deploy edilen branch:** `main`
- **Admin kullanıcı:** askinaktas
- **Güncelleme:** sunucuda `cd /root/cicekci && git pull origin main &&
  docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build`

### Deploy sırasında çıkan sorunlar ve çözümleri (sonra lazım olur)

**2GB RAM build'de yetmiyor (OOM).** İlk `docker compose build` sırasında
Nuxt/Node build'i "JavaScript heap out of memory" ve sistem OOM killer ile
düşüyordu. İki katmanlı çözüm: (1) sunucuya kalıcı **4GB swap** eklendi
(`/swapfile`, fstab'da), (2) `frontend/app/Dockerfile`'da build adımına
`ENV NODE_OPTIONS="--max-old-space-size=2048"`. Runtime'da 2GB fazlasıyla
yeter — sorun sadece build anındaki geçici tepe kullanımdı.

**Public site hiç veri göstermiyordu (KRİTİK, iki ayrı bug).** Ana sayfa,
kategoriler, slider hep boştu — ama veriler backend'de KAYITLIYDI. İki neden
üst üste bindi:
1. **Caddy routing:** genel `handle /api/*` kuralı `/api/go/*` isteklerini de
   yakalayıp doğrudan backend'e gönderiyordu; backend'de o route yok → 404.
   `handle /api/go/* { reverse_proxy app:3000 }` bloğu genel kuraldan ÖNCE
   eklendi (`deploy/Caddyfile`).
2. **Env adı uyuşmazlığı:** Nuxt `runtimeConfig.goApiBase`'i runtime'da yalnızca
   `NUXT_GO_API_BASE` env'iyle override eder, ama compose `NUXT_API_BASE`
   gönderiyordu. Override çalışmayınca proxy build'e gömülü `localhost:8080`
   default'una düşüp ECONNREFUSED alıyordu. `nuxt.config.ts`'te
   `process.env.NUXT_API_BASE` okuması da build zamanında değerlenip default'u
   imaja gömdüğü için kaldırıldı; compose `NUXT_GO_API_BASE`'e çevrildi.
   **Ders:** `/api/go/*` proxy'sini kullanan HER yol (tüm public composable'lar)
   buna bağlı — bu iki nokta doğru değilse public site komple boş görünür ama
   sessizce (hata sayfası değil, boş liste).

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
