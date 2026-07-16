# Frontend Yeniden Tasarım — "Luxe Floral Heritage"

**Tarih:** 2026-07-16
**Kapsam:** `frontend/app` (public site). Backend'e ve `frontend/idare`'ye dokunulmaz.
**Referans:** `frontend/app/referans_tasarım/` — 21 HTML mockup + PNG, `luxe_floral_heritage/DESIGN.md`

## 1. Amaç

Mevcut public site fonksiyonel ama görsel olarak jenerik: system-ui fontu, pembe
(`#c2185b`) vurgu, kutu gölgeli kartlar. Referans tasarım editorial minimalizm ve
high-end butik estetiği getiriyor: krem zemin, Libre Caslon serif başlıklar,
altın vurgular, geniş beyaz alan.

Hedef: referans tasarımı olabildiğince birebir uygulamak ve premium hissettiren
bir site çıkarmak.

## 2. Kapsam kararları

### 2.1 Ölü UI — bilinçli karar

Referans tasarımlar tam e-ticaret sitesi: sepet, favoriler, üyelik, 4 hesap
sayfası. Mevcut backend'de bunların hiçbiri yok (`users`, `cart`, `orders`,
`favorites` tabloları mevcut değil) ve MVP spec'i (§3.3) üyeliği ve favorileri
gerekçeli olarak reddediyor, sepeti Faz 2'ye erteliyor.

**Karar (kullanıcı onayı ile):** Bu UI yine de yapılacak, ancak **inert** —
görünür ve tasarıma sadık, fakat backend olmadığı için işlevsiz. Gerekçe: Faz 2
önizlemesi / demo değeri.

**Sınır — "inert ama yalancı değil":**

| Kontrol | Davranış |
|---|---|
| Sepet ikonu | Görünür, **badge yok** (referanstaki "3" rozeti kaldırılır — var olmayan sepet içeriğini iddia etmez) |
| Sepet tıklama | Boş drawer açılır: "Sepetiniz boş" |
| Favori kalbi | Görünür, `/hesabim/favoriler`'e gider |
| Hesap ikonu | Görünür, `/hesabim`'a gider |
| Hesap sayfaları | Statik mock veri ile render (Elif Yılmaz — referanstaki isim) |
| "Sepete Ekle" | Tasarımdaki yerinde ve tonunda durur, tıklanınca boş drawer açar |

Bu ayrım önemlidir: rozetli sepet ("3 ürün") sitenin **sahip olmadığı bir durumu
iddia eder** ve kırık görünür. Rozetsiz sepet sadece henüz boştur.

**Faz 2'de bu UI'ın gerçek hale gelmesi:** Bu ekranlar atılacak değil, backend'e
bağlanacak şekilde yazılır — mock veri tek bir yerden (`~/utils/mockAccount.ts`)
gelir, composable arayüzü gerçek API çağrısına benzer tutulur.

### 2.2 Backend değişmez

Bu iş **yalnızca frontend**. `internal/`, `migrations/`, `cmd/` dosyalarına
dokunulmaz. `utils/price.ts`, `utils/whatsapp.ts` ve mevcut vitest testleri
korunur.

### 2.3 WhatsApp korunur

Sitenin tek gerçek dönüşüm yolu WhatsApp siparişi. Ürün detayda "Sepete Ekle"
(inert) **birincil**, "WhatsApp'tan Sipariş Ver" (gerçek) **ikincil** CTA olarak
durur. WhatsApp tamamen kaldırılırsa site demo değil kırık olur.

Mevcut `WhatsAppButton.vue` (sticky FAB) korunur, referans dile uydurulur.

## 3. Tasarım sistemi

### 3.1 Renk

`DESIGN.md` frontmatter token'ları kullanılır. **Frontmatter kazanır** — prose'daki
değerler (`#FDFBF7` background, `#2D2926` primary) frontmatter ile çelişiyor
(`#fbf9f5`, `#181512`) ve referans HTML'ler frontmatter değerlerini derliyor.

**Altın eklenir:** Prose'un "Accent Gold `#C5A059`" dediği renk frontmatter'da
**yok**, ama referans HTML'lerde 17 kez hardcoded arbitrary value olarak geçiyor
(`text-[#C5A059]` ×12, `bg-[#C5A059]` ×5). Token'a çevrilir:

```
accent-gold:      #C5A059   // ince vurgular, ayraçlar, link hover
accent-gold-dim:  #b08e4f   // hover state (referansta geçiyor)
secondary:        #775a19   // dolu CTA butonları (referans HTML'de kullanılan ton)
```

Ayrıca referansta `#8f6d21` (×3) geçiyor — `secondary` hover'ı olarak token'lanır.

**Dark mode yok.** DESIGN.md açıkça reddediyor ("conflicts with the botanical
sun-drenched brand essence"). Referans config'deki `darkMode: "class"` kaldırılır.

### 3.2 Tipografi

- **Başlık:** Libre Caslon Text
- **Gövde/UI:** Work Sans
- Ölçek DESIGN.md'deki `display-lg` … `nav-link` token'larından

**Self-host:** Referans Google Fonts CDN kullanıyor; `@nuxt/fonts` ile
self-host'a geçilir. CDN ek DNS+TLS round-trip demek, LCP'yi geciktirir —
premium his performansla başlar.

**İkonlar:** Referans, Material Symbols ikon fontunu CDN'den çekiyor (~2MB) ve
**32 farklı ikon** kullanıyor. Font CDN'i self-host mantığıyla çelişir; 32 ikonu
elle SVG yazmak da gereksiz iş. Çözüm: `@nuxt/icon` + `material-symbols`
koleksiyonu — yalnızca kullanılan ikonlar bundle'a girer, tek tek SVG yazılmaz.

### 3.3 Şekil, gölge, boşluk

DESIGN.md'ye sadık: gölge yok (yalnızca drawer/modal'da %2 opacity), hiyerarşi
ince outline ve tonal katmanlarla. Radius 0.25rem (buton/input), 0.5rem (kart).
`container-max` 1280px, `section-gap` 120px.

### 3.4 CSS yöntemi

Tailwind kurulur (`@nuxtjs/tailwindcss`), DESIGN.md token'ları `tailwind.config`'e.
Mevcut Türkçe class isimleri (`.kapsayici`, `.buton`, `.urun-izgara`) kaldırılır.
`main.css` yalnızca token tanımı + base reset katmanı olarak kalır.

## 4. Marka

**Ad:** Gözde Tasarım Çiçekçilik (gerçek müşteri).
**İletişim:** Telefon `0553 614 36 86`, adres Nişantaşı/Teşvikiye — env var'dan
okunmaya devam eder, yalnızca default değerler güncellenir.

**Logo:** Kullanıcı gerçek asset'i (SVG/PNG) verecek. Gelene kadar geçici
placeholder: Libre Caslon serif marka adı + oval "G" ikonu (SVG). Header logo'yu
tek bir `TheLogo.vue` bileşeninden alır — asset gelince tek dosya değişir.

> **Not:** Logo asset'i kalın condensed sans kullanıyor, header mockup'ları ise
> Libre Caslon serif. İki farklı marka dili. Asset geldiğinde hangisinin
> kazanacağı kullanıcıya sorulacak.

## 4.1 Görseller — açık konu

Referans mockup'lardaki **30 görselin tamamı** Google'ın AI CDN'inden hotlink
(`lh3.googleusercontent.com/aida-public/...`). Bunlar AI üretimi mockup
asset'leri: (a) Gözde Tasarım'ın gerçek ürünleri değil, (b) lisanslı değil,
(c) o URL'ler kalıcı değil — er ya da geç ölür.

**Ürün görselleri sorun değil:** backend'den geliyor (`url_400` / `url_1200`),
esnaf admin panelden yüklüyor.

**Sorun, ürün olmayan dekoratif görseller:**

| Nerede | Ne gerekiyor |
|---|---|
| Ana sayfa hero | Tam genişlik şakayık/çiçek görseli |
| "Gönderim Türüne Göre" | 4 kategori kartı görseli |
| Hesap sayfaları hero banner | Çiçek görseli (mavi gökyüzü + pembe kozmos) |
| Hakkımızda | Atölye görseli + harita |

**Karar:** Bu görseller hotlink edilmez. Uygulama sırasında `public/img/`
altına yerel asset olarak konur ve `<NuxtImg>` ile servis edilir. Kaynak
kullanıcıya sorulacak — üç seçenek: (a) Gözde Tasarım'ın gerçek fotoğrafları
(tercih edilen — premium his gerçek üründen gelir), (b) lisanslı stok
fotoğraf, (c) referans görselleri geçici olarak indirilip kullanılır.

Kategori görselleri ileride admin panelden yönetilmek istenirse
`categories` tablosuna `image_url` kolonu gerekir — **bu iş Faz 2**, şimdilik
kod içinde sabit eşleme.

## 5. Sayfalar

### 5.1 Yapılacaklar

| Sayfa | Rota | Referans |
|---|---|---|
| Ana sayfa | `/` | `ana_sayfa_yeni_logo_yerle_imi` |
| Koleksiyon | `/urunler` | `i_ek_koleksiyonu_yeni_logo_yerle_imi` |
| Kategori | `/kategori/[slug]` | koleksiyon ile aynı şablon |
| Ürün detay | `/urun/[slug]` | `r_n_detay_yeni_logo_yerle_imi` |
| Hakkımızda | `/hakkimizda` | `hakk_m_zda_yeni_logo_yerle_imi` |
| İletişim | `/iletisim` | referansta ayrı sayfa yok — hakkımızda'nın alt bölümü uyarlanır |
| Hesabım — Pano | `/hesabim` | `hesab_m_pano_g_rsel_arka_planl` |
| Hesabım — Adresler | `/hesabim/adresler` | `hesab_m_adresler_g_rsel_arka_planl` |
| Hesabım — Hesap Detayları | `/hesabim/hesap-detaylari` | `hesab_m_hesap_detaylar_g_rsel_arka_planl` |
| Hesabım — Favoriler | `/hesabim/favoriler` | `hesab_m_favoriler_g_rsel_arka_planl` |

Hesap sayfaları **"görsel arka planlı"** varyantı kullanır (çiçek görselli hero
banner) — düz varyanttan belirgin şekilde daha premium.

### 5.2 Referansta olmayan, eklenecek

Referans mockup'ları hep dolu veri gösteriyor. Gerçek sitede bunlar olmadan
premium değil yarım hisseder:

- **404 sayfası** — referans dilinde
- **Boş durumlar** — kategori boş, favoriler boş, arama sonucu yok
- **Yükleniyor durumları** — skeleton, referans tonunda
- **Görsel yer tutucu** — ürünün fotoğrafı yoksa (mevcut `.gorsel-yok` yerine)

### 5.3 Ana sayfa bölümleri

Hero (tam genişlik çiçek görseli + serif başlık + altın CTA) → "Gönderim Türüne
Göre" (4 kategori kartı, görsel üstü etiket) → altın ayraç ("G" ikonu ortada) →
"En Çok Tercih Edilenler" (4 ürün + "Tümünü Gör") → teslimat şeridi → footer.

Kategori kartları `useFeaturedCategories()`'den gelir (mevcut davranış korunur).

## 6. Referanstan sapmalar

Referans set kendi içinde tutarsız; normalize edilir.

### 6.1 Dil — hepsi Türkçe

Referans ekranlar karışık: nav "Flowers / Special Days / Collections / About Us /
Contact", sidebar kimi ekranda İngilizce ("Profile/Orders/Favorites/Addresses")
kimi ekranda Türkçe ("Pano/Siparişler/Adresler"). Site Türkçe (`lang="tr"`),
hepsi Türkçeye çevrilir:

Nav: **Çiçekler / Özel Günler / Koleksiyonlar / Hakkımızda / İletişim**

### 6.2 Hesap sidebar — tek yapıya sabitlenir

Referansta ekranlar arası değişiyor (kimi 4 kimi 5 item, "Wishlist" vs
"Favoriler", "Account Details" vs "Profile"). Sabit yapı:

**Pano / Adresler / Hesap Detayları / Favoriler / Çıkış Yap**

**"Siparişler" çıkarılır** (kullanıcı onayı ile): sipariş diye bir kavram yok,
sayfası da yok — tıklanınca gidecek yer olmayan bir link, ölü UI'ın bile kabul
etmeyeceği kadar kırık. Pano'daki "Siparişler" kartı da kaldırılır, kalan 3 kart
(Adresler, Hesap Detayları, Favoriler) grid'e yayılır.

### 6.3 Diğer

- **Favoriler görselleri:** Referansta ürün fotoğrafı yerine yanlışlıkla
  *uygulama ekran görüntüleri* konmuş. Gerçek ürün görselleri kullanılır.
- **Fiyat formatı:** Referansta hem `₺2.749,00` hem `2.749,00 ₺` var. Mevcut
  `formatPrice()` util'i tek kaynak — `2.749,00 ₺` kazanır.
- **"(KDV dahil)" etiketi:** Referansta tutarlı biçimde kullanılıyor (36 yerde);
  fiyatın yanında küçük ve soluk. Aynen korunur.
- **Harita:** Hakkımızda'daki Google Maps. Canlı embed yerine **statik görsel +
  "Yol Tarifi Al" linki** — embed üçüncü taraf çerez taşır (KVKK) ve sayfayı
  yavaşlatır.
- **Sepet rozeti:** Kaldırılır — bkz §2.1.

## 7. Bileşen mimarisi

### 7.1 Değişecek

- `TheHeader.vue` — tam yeniden yazım: serif logo, 5'li nav, eksen dropdown'ları,
  ikon grubu, sticky + backdrop blur (DESIGN.md §Elevation), mobil drawer
- `TheFooter.vue` — 4 sütun (Kurumsal / Yardım / Bizi Takip Edin / Bülten)
- `ProductCard.vue` — gölge yok, hover'da ince border, serif ad, sans fiyat
- `CategoryFilter.vue` — referans chip tasarımı (dolu koyu = aktif, outline = pasif)
- `WhatsAppButton.vue` — referans diline uydurulur

### 7.2 Yeni

- `TheLogo.vue` — tek kaynak, asset gelince burası değişir
- `TheCartDrawer.vue` — boş durum
- `components/icons/*` — inline SVG ikonlar
- `components/account/AccountSidebar.vue`, `AccountHero.vue`
- `layouts/account.vue` — sidebar + içerik grid'i
- `utils/mockAccount.ts` — hesap sayfalarının statik verisi, tek yerde

### 7.3 Korunacak

`composables/useApi.ts` (API sözleşmesi değişmiyor), `utils/price.ts`,
`utils/whatsapp.ts`, `types/api.ts` ve mevcut testler.

## 8. Duyarlılık (responsive)

Referanslar yalnızca desktop. Mobil/tablet davranışı DESIGN.md §Layout'tan
türetilir:

- **Desktop (1280px+):** 12 sütun, container 1280px, margin 64px, section-gap 120px
- **Tablet (768–1279px):** 8 sütun, margin 32px, ürün grid 3 sütun, section-gap 80px
- **Mobil (<768px):** 4 sütun, margin 20px, ürün grid 2 sütun, section-gap 56px,
  nav → drawer, hesap sidebar → üstte yatay scroll sekme

Tipografi mobilde küçülür (`headline-lg` 48px → `headline-lg-mobile` 32px).

## 9. Doğrulama

- Her sayfa referans PNG'siyle yan yana karşılaştırılır
- Mevcut vitest testleri geçmeli (`pnpm test`)
- Üç kırılma noktası kontrol edilir: 375px, 768px, 1440px
- Erişilebilirlik: metin kontrastı AA (krem zemin `#fbf9f5` üstünde
  `on-surface-variant` `#4d4540` kontrolü), ikon butonlarında `aria-label`,
  klavye ile gezilebilirlik
- İnert kontroller tıklanınca hata vermemeli (sessizce boş drawer/sayfa)

## 9.1 Bilinen açık konular

Uygulamayı bloke etmez ama karara bağlanmalı:

1. **Dekoratif görsellerin kaynağı** (§4.1) — hero, kategori kartları, hesap
   banner'ı. Gerçek fotoğraf gelene kadar geçici asset ile ilerlenir.
2. **Logo asset'i** (§4) — kullanıcı verecek. Gelene kadar serif + oval "G"
   placeholder. Asset gelince serif mi condensed sans mı kazanacağı sorulacak.

## 10. Kapsam dışı

- Backend değişikliği (users/cart/orders/favorites) — Faz 2/3
- Gerçek sepet, favori kalıcılığı, üyelik, ödeme
- Arama işlevi (ikon durur, inert)
- Bülten kaydı (form durur, inert)
- Admin panel (`frontend/idare`)
- Dark mode (DESIGN.md reddediyor)
