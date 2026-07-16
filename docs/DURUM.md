# Proje Durumu

Son güncelleme: 2026-07-16

## Nerede kaldık

| Aşama | Durum |
|---|---|
| Tasarım (spec) | ✅ Bitti — `docs/superpowers/specs/2026-07-15-cicekci-mvp-design.md` |
| Plan 1 — Backend temeli | ✅ **Uygulandı** — 13/13 task, 100 test |
| Plan 2 — Görsel hattı | ✅ **Uygulandı** — 8/8 iş, 179 test toplam |
| Template — admin panel iskeleti | ✅ Eklendi ve Node 22'ye uyarlandı |
| Plan 4 — Admin panel | 📋 **Plan hazır, uygulanmadı** ← SIRADA |
| Plan 3 — Nuxt public site | ⬜ Plan yazılmadı |
| Final whole-branch review | ⬜ Ertelendi (Plan 1+2 birlikte yapılacak) |

Branch: `feat/backend-temeli` (her şey burada)

## Çalışan ne var

Tam bir Go backend'i. Gerçek sunucuda uçtan uca doğrulandı:
- Admin girişi (bcrypt + HttpOnly JWT cookie)
- Kategori CRUD — iki eksen (occasion/type), `is_active` + `is_featured`
- Ürün CRUD — slug geçmişi, eski linkler 301 ile yönleniyor
- İki eksenli AND filtresi
- Görsel hattı — yükleme, 2 boyut üretimi, sıralama, silme
- 179 test

**Ölçülen kazanç:** 446KB/2000x1500 fotoğraf → 21KB/400px + 108KB/1200px (20x)

## Sırada: Plan 4 — Admin Panel

Plan: `docs/superpowers/plans/2026-07-16-plan-4-admin-panel.md`

Planın başındaki "Başlangıç Durumu" bölümünü oku — ortam bilgileri,
backend'i ayağa kaldırma komutları ve template'te zaten yapılanlar orada.

8 task: temizlik → ApiService → auth → tipler/composable'lar → kategori
ekranı → ürün listesi → ürün formu + görseller → placeholder + test.

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

**Şeffaf PNG bug'ı.** `imaging.Paste` alfa kanalını yok sayıp kopyalıyordu,
şeffaf alanlar JPEG'de siyah çıkıyordu. `Overlay` ile harmanlanıyor.

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

## Frontend yapısı

- `frontend/idare/` — Vuetify 3 + Vite SPA, admin paneli (template eklendi)
- `frontend/app/` — Nuxt 3 public site (henüz boş, Plan 3'te)

**Neden ayrı:** Public site SSR olmak ZORUNDA — WhatsApp'ın önizleme botu
JavaScript çalıştırmıyor, SPA'da paylaşılan linkte ürün fotoğrafı çıkmaz.
Bu, tüm tasarımın dayandığı nokta (spec §5.1). Admin panelde SSR gereksiz.
