# Plan 3 — Nuxt Public Site Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Müşterinin gördüğü site. Ürünleri gezip WhatsApp'tan sipariş verdiği yer. Projenin tek dönüşüm noktası.

**Architecture:** Nuxt 3 SSR (`frontend/app/`). Go API'yi (`localhost:8080`) sunucu tarafında tüketir. Auth YOK — tamamen public.

**Tech Stack:** Nuxt 3, Vue 3, TypeScript, `@nuxt/image`, `@nuxtjs/sitemap`.

**Spec:** `docs/superpowers/specs/2026-07-15-cicekci-mvp-design.md` — §5.1 (SSR zorunluluğu), §5.2 (sayfalar), §5.3 (WhatsApp), §5.4 (görseller), §5.5 (SEO), §5.6 (filtreleme).

**Önkoşul:** Plan 1, 2, 4 tamamlandı. Backend çalışıyor (179 test), admin panel çalışıyor.

---

## Başlangıç Durumu — yeni bir oturumda uygulayacaksan ÖNCE OKU

**Repo:** `/Users/omerkoc/GolandProjects/cicekci`, branch `feat/backend-temeli`
(her şey bu branch'te — ayrı branch açma).

**Ortam (doğrulandı):** Go 1.25.4, Node 22.22.3, pnpm 10.24, Docker çalışıyor.

**`frontend/app/` şu an boş** (sadece `.gitkeep`). Nuxt projesini sen kuracaksın.

**Backend'i ayağa kaldırmak:**
```bash
cd /Users/omerkoc/GolandProjects/cicekci
export DATABASE_URL="postgres://cicekci:cicekci@localhost:5433/cicekci?sslmode=disable"
export JWT_SECRET="local-development-secret-32-chars!"
export WHATSAPP_NUMBER="905551234567"
export SITE_URL="http://localhost:3000"   # Nuxt'ın portu — CORS için
export APP_ENV=development STORAGE_DRIVER=local
make db-up && make migrate-up && make run
```

**Not:** Admin panel `SITE_URL=http://localhost:5173` istiyor, public site
`http://localhost:3000`. İkisini aynı anda çalıştırmak gerekirse backend'in
CORS ayarı tek origin alıyor — public site SSR olduğu için tarayıcıdan değil
sunucudan çağırır, CORS'a takılmaz. Yani `SITE_URL=http://localhost:5173`
bırakıp public site'ı SSR'da çalıştırmak sorunsuz. **Client-side fetch
yaparsan CORS'a takılırsın** — bu zaten yapılmamalı (bkz. Task 3).

**Testler:** Backend için `make test` (`go test ./...` DEĞİL — paketler aynı
test DB'sini paylaşıyor, `-p 1` gerekli).

**Paket yöneticisi: pnpm.**

**Referans olarak bak:** `frontend/idare/src/services/ApiService.ts` — admin
panelin API deseni (`[error, data]` dönüşü, hata formatı çıkarma). Public
site'ta auth yok, o yüzden `withCredentials` gerekmeyecek, ama hata formatı
aynı.

---

## Global Constraints

- **SSR ZORUNLU.** Bu projenin en kritik kısıtı. WhatsApp'ın link önizleme
  botu JavaScript çalıştırmıyor — SPA'da paylaşılan linkte ürün fotoğrafı
  çıkmaz. Bu sitede her satış bir WhatsApp mesajından geçtiği için fotoğrafın
  ilk göründüğü yer çoğu zaman site değil, sohbet ekranı. `og:image`
  sunucudan gelmek zorunda (spec §5.1).
- **Auth YOK.** Public site hiçbir korumalı uca dokunmaz. `/api/admin/*`
  çağrısı yapma.
- **Fiyat string.** API `"1850.00"` döner. Gösterirken formatla (`1.850 ₺`)
  ama hesaplama yapma.
- **Arayüz dili Türkçe.**
- **Yorumlar Türkçe.**
- **Filtre state'i URL'de.** Vue state'inde değil — filtrelenmiş liste
  paylaşılabilmeli, tarayıcı geri tuşu çalışmalı (spec §5.6).

---

## Public API Sözleşmesi (canlı sunucuda doğrulandı)

```
GET /api/products?amac=<slug>&tip=<slug>&page=1&limit=24
GET /api/products/:slug          → eski slug ise 301 + Location
GET /api/categories?axis=occasion|type
GET /api/categories/featured
GET /api/categories/:slug
```

**Ürün yanıtı** (`is_active` YOK — public'e sızmıyor):
```json
{"id":3, "name":"gül", "slug":"gul", "description":"gül",
 "price":"1000.00", "category_ids":[1,2],
 "images":[{"url_400":"http://localhost:8080/uploads/.../400.jpg",
            "url_1200":"http://localhost:8080/uploads/.../1200.jpg"}]}
```

**Kategori yanıtı** (`is_active`/`is_featured`/`sort_order` YOK):
```json
{"id":1, "name":"Doğum Günü", "slug":"dogum-gunu", "axis":"occasion"}
```

**Doğrulanmış davranışlar:**
- Boş liste `[]` döner, `null` değil — `v-for` güvenli
- 404 gövdesi: `{"error":{"code":"not_found","message":"Kayıt bulunamadı"}}`
- Pasif ürün/kategori public uçlarda hiç görünmez (store katmanında filtreli)
- İki eksen birlikte verilirse **AND** — ikisine de uyan ürünler (spec §5.6)

**⚠️ ÖNEMLİ — ürün yanıtında kategori İSİMLERİ yok, sadece `category_ids`.**
Ürün detayında "Doğum Günü" yazmak için `GET /api/categories` ile tüm
kategorileri çekip id→isim eşlemesi yapman gerekiyor. Kategori sayısı ~16,
tek çağrı, SSR'da cache'lenebilir. Backend'i değiştirmeye gerek yok.

**⚠️ KRİTİK — 301 yönlendirmesi API yolunu döner, sayfa yolunu değil.**
`GET /api/products/51-gul-buket` eski slug ise:
```
HTTP/1.1 301 Moved Permanently
Location: /api/products/51-kirmizi-gul-buketi
```
`Location` **`/api/products/...`** diyor — `/urun/...` değil. `$fetch` bu
301'i şeffafça takip edip ürünü getirir, ama **tarayıcının adresi eski
slug'da kalır.** Sonuç: Google iki ayrı URL'de aynı içeriği görür ve slug
geçmişinin amacı boşa gider. Task 5 bunu ele alıyor — çözüm: yanıttaki
`slug` alanı istenen slug'dan farklıysa Nuxt'ta 301 yap.

---

## Dosya Yapısı

```
frontend/app/
  nuxt.config.ts
  app.vue
  .env                    → NUXT_PUBLIC_API_BASE, NUXT_PUBLIC_WHATSAPP_NUMBER
  types/
    api.ts                → Product, Category, ProductImage
  composables/
    useApi.ts             → $fetch sarmalayıcı, hata formatı
    useWhatsApp.ts        → wa.me link üretimi
  utils/
    price.ts              → "1850.00" → "1.850 ₺"
  components/
    ProductCard.vue       → liste kartı
    CategoryFilter.vue    → iki eksenli filtre
    ProductGallery.vue    → detay galerisi
    WhatsAppButton.vue    → sipariş butonu
    TheHeader.vue / TheFooter.vue
  layouts/
    default.vue
  pages/
    index.vue             → ana sayfa (featured kategoriler + öne çıkanlar)
    urunler/index.vue     → liste + filtre
    urun/[slug].vue       → detay + galeri + WhatsApp
    kategori/[slug].vue   → kategori sayfası (SEO)
    hakkimizda.vue
    iletisim.vue
```

---

## Task 1: Nuxt kurulumu ve yapılandırma

**Files:**
- Create: `frontend/app/` (Nuxt projesi)
- Create: `frontend/app/.env`, `.env.example`

- [ ] **Step 1: Nuxt projesini kur**

```bash
cd /Users/omerkoc/GolandProjects/cicekci/frontend
rm -f app/.gitkeep
pnpm dlx nuxi@latest init app --packageManager pnpm --gitInit false
cd app
```
Kurulum sırasında soru sorarsa: TypeScript **evet**, ESLint **evet**.

- [ ] **Step 2: Gerekli modülleri ekle**

```bash
pnpm add -D @nuxt/image @nuxtjs/sitemap
```

Sadece bu ikisi. Tailwind, UI kütüphanesi vb. EKLEME — public site beş
sayfadan ibaret, kendi CSS'imizi yazacağız. Admin panelde Vuetify var ama
public site'ta ağır bir bileşen kütüphanesi taşımak gereksiz; sayfa hızı
bu sitenin satış aracı.

- [ ] **Step 3: nuxt.config.ts**

```ts
export default defineNuxtConfig({
  compatibilityDate: '2026-07-16',
  devtools: { enabled: true },

  modules: ['@nuxt/image', '@nuxtjs/sitemap'],

  runtimeConfig: {
    public: {
      // SSR'da sunucu-sunucu çağrısı; tarayıcıya sızmayacak
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://localhost:8080/api',
      whatsappNumber: process.env.NUXT_PUBLIC_WHATSAPP_NUMBER || '905551234567',
      siteUrl: process.env.NUXT_PUBLIC_SITE_URL || 'http://localhost:3000',
    },
  },

  // Görseller backend'den geliyor (local'de :8080, prod'da R2 CDN)
  image: {
    domains: ['localhost'],
  },

  site: {
    url: process.env.NUXT_PUBLIC_SITE_URL || 'http://localhost:3000',
  },

  app: {
    head: {
      htmlAttrs: { lang: 'tr' },
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      ],
    },
  },
})
```

- [ ] **Step 4: .env ve .env.example**

`.env`:
```
NUXT_PUBLIC_API_BASE=http://localhost:8080/api
NUXT_PUBLIC_WHATSAPP_NUMBER=905551234567
NUXT_PUBLIC_SITE_URL=http://localhost:3000
```
`.env.example` aynısı (numara örnek değerle).

`.gitignore`'a `.env` ekli olduğunu doğrula (Nuxt varsayılan ekler).

- [ ] **Step 5: Çalıştığını doğrula**

```bash
pnpm dev
```
`http://localhost:3000` açılmalı — Nuxt karşılama sayfası.
`Ctrl+C` ile durdur.

- [ ] **Step 6: Commit**

```bash
cd /Users/omerkoc/GolandProjects/cicekci
git add frontend/app
git commit -m "feat: Nuxt 3 public site iskeleti

SSR zorunlu (spec §5.1) — WhatsApp'ın önizleme botu JavaScript
çalıştırmıyor, SPA'da paylaşılan linkte ürün fotoğrafı çıkmaz.

Ağır UI kütüphanesi eklenmedi: beş sayfa var ve sayfa hızı bu sitenin
satış aracı."
```

---

## Task 2: Tipler ve fiyat formatı

**Files:**
- Create: `types/api.ts`, `utils/price.ts`
- Test: `utils/price.test.ts` (Vitest)

- [ ] **Step 1: types/api.ts**

```ts
/** Public API tipleri — backend'in app viewmodel'lerine birebir karşılık. */

export interface ProductImage {
  url_400: string
  url_1200: string
}

export interface Product {
  id: number
  name: string
  slug: string
  description: string
  /** "1850.00" — float precision için string (spec §4.1) */
  price: string
  category_ids: number[]
  images: ProductImage[]
}

export type Axis = 'occasion' | 'type'

export interface Category {
  id: number
  name: string
  slug: string
  axis: Axis
}

export const AXIS_LABELS: Record<Axis, string> = {
  occasion: 'Gönderim Amacına Göre',
  type: 'Ürün Tipine Göre',
}
```

- [ ] **Step 2: Vitest kur**

```bash
pnpm add -D vitest @nuxt/test-utils
```

`vitest.config.ts`:
```ts
import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: { environment: 'node' },
})
```

`package.json` scripts'e ekle: `"test": "vitest run"`

- [ ] **Step 3: Fiyat formatı testini yaz**

`utils/price.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import { formatPrice, whatsappPrice } from './price'

describe('formatPrice', () => {
  it('binlik ayracı koyar ve ₺ ekler', () => {
    expect(formatPrice('1850.00')).toBe('1.850 ₺')
  })

  it('kuruş varsa gösterir', () => {
    expect(formatPrice('1850.50')).toBe('1.850,50 ₺')
  })

  it('kuruş sıfırsa göstermez — esnaf 1.850 ₺ yazar, 1.850,00 ₺ değil', () => {
    expect(formatPrice('500.00')).toBe('500 ₺')
  })

  it('binlik altı', () => {
    expect(formatPrice('99.00')).toBe('99 ₺')
  })

  it('geçersiz girdide boş döner, patlamaz', () => {
    expect(formatPrice('abc')).toBe('')
    expect(formatPrice('')).toBe('')
  })
})

describe('whatsappPrice', () => {
  it('mesajda kullanılan sade format', () => {
    expect(whatsappPrice('1850.00')).toBe('1.850 ₺')
  })
})
```

- [ ] **Step 4: Testi çalıştır, BAŞARISIZ olduğunu gör**

Run: `pnpm test`
Expected: FAIL — `formatPrice` yok

- [ ] **Step 5: utils/price.ts yaz**

```ts
/**
 * API fiyatı string döner ("1850.00") — float precision sorunu olmasın diye.
 * Burada sadece GÖSTERİM formatlanıyor; hesaplama yapılmıyor.
 */
export function formatPrice(price: string): string {
  const n = Number(price)
  if (!price || Number.isNaN(n))
    return ''

  const hasKurus = n % 1 !== 0

  return `${n.toLocaleString('tr-TR', {
    minimumFractionDigits: hasKurus ? 2 : 0,
    maximumFractionDigits: 2,
  })} ₺`
}

/** WhatsApp mesajında kullanılan format — şimdilik aynı. */
export function whatsappPrice(price: string): string {
  return formatPrice(price)
}
```

- [ ] **Step 6: Testi çalıştır, GEÇTİĞİNİ gör**

Run: `pnpm test`
Expected: PASS — 6 test

- [ ] **Step 7: Commit**

```bash
git add frontend/app/types frontend/app/utils frontend/app/vitest.config.ts frontend/app/package.json
git commit -m "feat: public site tipleri ve fiyat formatı

Fiyat API'den string geliyor (1850.00), gösterimde 1.850 ₺ oluyor.
Kuruş sıfırsa gösterilmiyor — esnaf öyle yazar."
```

---

## Task 3: API katmanı

**Files:**
- Create: `composables/useApi.ts`

**Interfaces:**
- Produces: `useProducts()`, `useCategories()` — SSR uyumlu veri çekme

- [ ] **Step 1: composables/useApi.ts**

```ts
import type { Category, Product } from '~/types/api'

/**
 * Public API istemcisi.
 *
 * Auth YOK — public site hiçbir korumalı uca dokunmaz.
 * SSR'da bu çağrılar sunucudan yapılır (CORS'a takılmaz); client tarafında
 * hydration sonrası yapılan çağrılar tarayıcıdan gider.
 */
function apiBase(): string {
  return useRuntimeConfig().public.apiBase
}

/** Backend hata formatı: {"error": {"code": "...", "message": "..."}} */
export function apiErrorMessage(err: unknown): string {
  const e = err as { data?: { error?: { message?: string } }; statusCode?: number }
  if (e?.data?.error?.message)
    return e.data.error.message
  if (e?.statusCode === 404)
    return 'Ürün bulunamadı'

  return 'Bir şeyler ters gitti, lütfen tekrar deneyin'
}

export function useProductList(query: {
  amac?: string
  tip?: string
  limit?: number
}) {
  const params = new URLSearchParams()
  if (query.amac)
    params.set('amac', query.amac)
  if (query.tip)
    params.set('tip', query.tip)
  params.set('limit', String(query.limit ?? 24))

  return useFetch<Product[]>(`${apiBase()}/products?${params}`, {
    key: `products-${params}`,
    default: () => [],
  })
}

export function useProduct(slug: string) {
  return useFetch<Product>(`${apiBase()}/products/${slug}`, {
    key: `product-${slug}`,
  })
}

/**
 * Tüm kategoriler. Ürün yanıtında sadece category_ids var, isimler yok —
 * id→isim eşlemesi için bu liste gerekiyor. ~16 kayıt, tek çağrı.
 */
export function useCategoryList() {
  return useFetch<Category[]>(`${apiBase()}/categories`, {
    key: 'categories',
    default: () => [],
  })
}

export function useFeaturedCategories() {
  return useFetch<Category[]>(`${apiBase()}/categories/featured`, {
    key: 'categories-featured',
    default: () => [],
  })
}
```

**Not:** `useFetch` SSR'da sunucudan çağırır ve sonucu hydration'a taşır —
aynı veri iki kez çekilmez. `$fetch` kullanma (SSR'da tekrar çeker).

- [ ] **Step 2: Backend'i başlat ve API'yi doğrula**

Backend çalışırken:
```bash
curl -s localhost:8080/api/products | head -c 200
curl -s localhost:8080/api/categories
```
Yanıtların yukarıdaki tiplere uyduğunu gör.

- [ ] **Step 3: Commit**

```bash
git add frontend/app/composables
git commit -m "feat: public API katmanı — useFetch ile SSR uyumlu"
```

---

## Task 4: Ana sayfa ve layout

**Files:**
- Create: `layouts/default.vue`, `components/TheHeader.vue`, `components/TheFooter.vue`
- Create: `components/ProductCard.vue`
- Modify: `app.vue`, `pages/index.vue`

- [ ] **Step 1: app.vue**

```vue
<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>
```

- [ ] **Step 2: layouts/default.vue + TheHeader + TheFooter**

Header: logo/isim, navigasyon (Ürünler, Hakkımızda, İletişim), mobilde
hamburger menü. Footer: iletişim bilgisi, telif.

**Mobile-first** (spec §5.1, brief): önce dar ekran, sonra genişlet.
Medya sorgusu `min-width` ile yazılsın.

- [ ] **Step 3: components/ProductCard.vue**

Props: `product: Product`

- Kapak görseli: `product.images[0]?.url_400`, yoksa yer tutucu
- **`<NuxtImg loading="lazy">`** — ekran dışı fotoğraflar ertelenir, sayfa
  daha hızlı açılır (spec §5.4)
- **Sabit `aspect-ratio`** — fotoğraf inerken kart yüksekliği değişmesin,
  sayfa zıplamasın (spec §5.4). Zıplayan sayfa yavaş sayfadan sinir bozucu.
- Ürün adı, fiyat (`formatPrice`)
- Tıklanınca `/urun/[slug]`

- [ ] **Step 4: pages/index.vue — ana sayfa**

Spec §5.2: featured kategoriler + öne çıkan ürünler.

- `useFeaturedCategories()` → kategori kartları. **Ana sayfanın işi vitrin,
  katalog değil** — 4-6 kart. Esnaf `is_featured` ile seçiyor (spec §4.1).
- `useProductList({ limit: 8 })` → öne çıkan ürünler
- Kısa tanıtım metni
- `useSeoMeta()`: title, description, og:title, og:description

- [ ] **Step 5: Tarayıcıda doğrula**

Backend + `pnpm dev` çalışırken `http://localhost:3000`:
- Featured kategoriler görünüyor mu (admin panelden `is_featured` açık
  kategori olmalı)
- Öne çıkan ürünler ve fotoğrafları görünüyor mu
- Mobil genişlikte (DevTools) düzen bozulmuyor mu

**SSR kanıtı — bu kritik:**
```bash
curl -s localhost:3000 | grep -o "<title>[^<]*</title>"
curl -s localhost:3000 | grep -c "gül"   # ürün adı HTML'de olmalı
```
Ürün adı **curl çıktısında görünmeli** — JavaScript çalışmadan. Görünmüyorsa
SSR çalışmıyor demektir ve bu sitenin varlık sebebi kaybolur.

- [ ] **Step 6: Commit**

```bash
git add frontend/app
git commit -m "feat: ana sayfa, layout ve ürün kartı"
```

---

## Task 5: Ürün detayı, galeri ve WhatsApp — projenin kalbi

**Files:**
- Create: `pages/urun/[slug].vue`, `components/ProductGallery.vue`,
  `components/WhatsAppButton.vue`, `composables/useWhatsApp.ts`

Bu planın en kritik task'ı. Sitenin tek dönüşüm noktası burada.

- [ ] **Step 1: composables/useWhatsApp.ts**

Spec §5.3 — mesaj formatı tartışıldı ve seçildi:

```ts
import type { Product } from '~/types/api'
import { whatsappPrice } from '~/utils/price'

/**
 * WhatsApp sipariş linki (spec §5.3).
 *
 * Mesaj "bilgi almak istiyorum" değil "sipariş etmek istiyorum" diyor —
 * müşteri kendi ağzından niyetini kuruyor, konuşma pazarlıkla değil
 * siparişle başlıyor. Fiyat mesajda: esnafı koruyor (müşteri hangi fiyatı
 * gördüğünü belgeliyor) ve esnaf ürünü sitede aramak zorunda kalmıyor.
 *
 * Teslimat/tarih/kart mesajı alanları KASITLI olarak yok — Faz 2'nin işi.
 * Şablona form gibi alanlar koymak müşteriye ödev listesi yaratır.
 */
export function useWhatsAppLink(product: Product) {
  const config = useRuntimeConfig()
  const url = `${config.public.siteUrl}/urun/${product.slug}`

  const message = [
    'Merhaba, bu ürünü sipariş etmek istiyorum:',
    `${product.name} — ${whatsappPrice(product.price)}`,
    url,
  ].join('\n')

  // encodeURIComponent Türkçe karakterleri ve satır başlarını güvenli kodlar
  return `https://wa.me/${config.public.whatsappNumber}?text=${encodeURIComponent(message)}`
}
```

- [ ] **Step 2: Mesaj formatını test et**

`composables/useWhatsApp.test.ts` — Türkçe karakter kodlaması kritik:

```ts
import { describe, expect, it } from 'vitest'

describe('WhatsApp mesajı', () => {
  it('Türkçe karakterler ve satır başları doğru kodlanıyor', () => {
    const msg = 'Merhaba, bu ürünü sipariş etmek istiyorum:\n51 Gül Buket — 1.850 ₺'
    const encoded = encodeURIComponent(msg)

    // Kod çözülünce aynısına dönmeli — bozulma olmamalı
    expect(decodeURIComponent(encoded)).toBe(msg)
    // Satır başı kodlanmış olmalı
    expect(encoded).toContain('%0A')
    // Ham Türkçe karakter kalmamalı
    expect(encoded).not.toContain('ü')
  })
})
```

Run: `pnpm test` → PASS

- [ ] **Step 3: components/ProductGallery.vue**

Props: `images: ProductImage[]`, `alt: string`

- Büyük görsel: `url_1200`
- Birden fazla görsel varsa altta küçük thumbnail'lar (`url_400`), tıklayınca
  büyük değişir
- Tek görsel varsa thumbnail şeridi gösterme
- **Görsel yoksa yer tutucu** — çiçekçi henüz fotoğraf yüklememiş olabilir
- İlk görsel `loading="eager"` (ekranda hemen görünüyor), diğerleri lazy

- [ ] **Step 4: components/WhatsAppButton.vue**

Props: `product: Product`

- Büyük, belirgin, WhatsApp yeşili
- Metin: "WhatsApp'tan Sipariş Ver"
- `target="_blank"`, `rel="noopener"`
- Mobilde tam genişlik, sticky olabilir (sayfa uzunsa hep görünsün)

- [ ] **Step 5: pages/urun/[slug].vue**

```vue
<script setup lang="ts">
const route = useRoute()
const slug = computed(() => String(route.params.slug))

const { data: product, error } = await useProduct(slug.value)

// 404
if (error.value || !product.value) {
  throw createError({
    statusCode: 404,
    statusMessage: 'Ürün bulunamadı',
    fatal: true,
  })
}

// ⚠️ KRİTİK — slug geçmişi 301'i (spec §4.2)
// Backend eski slug'a 301 döner ama Location "/api/products/..." — yani API
// yolu. $fetch bunu şeffafça takip edip ürünü getirir, ama tarayıcının
// adresi eski slug'da kalır. Sonuç: Google iki URL'de aynı içeriği görür ve
// slug geçmişinin amacı boşa gider.
// Çözüm: yanıttaki slug istenen slug'dan farklıysa SAYFA yolunda 301 yap.
if (product.value.slug !== slug.value) {
  await navigateTo(`/urun/${product.value.slug}`, {
    redirectCode: 301,
    replace: true,
  })
}
</script>
```

Ayrıca:
- `ProductGallery` + ürün adı + fiyat + açıklama
- Kategoriler: `useCategoryList()` ile id→isim eşle, chip olarak göster,
  tıklanınca `/kategori/[slug]`
- `WhatsAppButton`
- **`useSeoMeta()` — WhatsApp önizlemesinin çalıştığı yer:**
```ts
useSeoMeta({
  title: () => `${product.value?.name} | Çiçekçi`,
  description: () => product.value?.description || product.value?.name,
  ogTitle: () => product.value?.name,
  ogDescription: () => product.value?.description || product.value?.name,
  ogImage: () => product.value?.images[0]?.url_1200,
  ogType: 'website',
})
```

- [ ] **Step 6: SSR ve WhatsApp önizlemesini DOĞRULA — bu task'ın asıl testi**

```bash
# Ürün sayfası SSR'da tam mı geliyor?
curl -s localhost:3000/urun/gul > /tmp/urun.html

# og:image VAR MI? — WhatsApp önizlemesinin çalışması buna bağlı
grep -o '<meta property="og:image"[^>]*>' /tmp/urun.html

# og:title, og:description
grep -o '<meta property="og:[^"]*" content="[^"]*"' /tmp/urun.html

# Ürün adı ve fiyat HTML'de mi? (JavaScript olmadan)
grep -c "gül" /tmp/urun.html

# WhatsApp linki HTML'de mi ve doğru mu kodlanmış?
grep -o 'href="https://wa.me/[^"]*"' /tmp/urun.html | head -1
```

**Hepsi dolu gelmeli.** `og:image` boşsa veya ürün adı HTML'de yoksa SSR
çalışmıyor demektir — bu sitenin tüm tasarımı bunun üstüne kurulu.

- [ ] **Step 7: 301'i canlı test et**

Admin panelden (veya API'den) ürünün adını değiştir, sonra:
```bash
curl -s -i localhost:3000/urun/gul | head -3
```
Beklenen: `HTTP/1.1 301` + `Location: /urun/<yeni-slug>`
**Adres çubuğu yeni slug'a geçmeli** — tarayıcıda da doğrula.

- [ ] **Step 8: WhatsApp butonunu GERÇEKTEN tıkla**

Tarayıcıda ürün sayfasını aç, butona tıkla. WhatsApp Web/uygulama açılmalı
ve mesaj kutusunda şu görünmeli:
```
Merhaba, bu ürünü sipariş etmek istiyorum:
gül — 1.000 ₺
http://localhost:3000/urun/gul
```
Türkçe karakterler bozuk olmamalı, satır başları korunmalı.

- [ ] **Step 9: Commit**

```bash
git add frontend/app
git commit -m "feat: ürün detayı, galeri ve WhatsApp sipariş butonu

Sitenin tek dönüşüm noktası. og:image SSR'da geliyor — WhatsApp
önizlemesinin çalışması buna bağlı (spec §5.1).

Slug 301: backend Location'ı /api/products/... döndüğü için tarayıcı
adresi eski slug'da kalıyordu. Yanıttaki slug farklıysa sayfa yolunda
301 yapılıyor — eski linkler ölmüyor, Google tek URL görüyor."
```

---

## Task 6: Ürün listesi ve iki eksenli filtre

**Files:**
- Create: `pages/urunler/index.vue`, `components/CategoryFilter.vue`

- [ ] **Step 1: components/CategoryFilter.vue**

Spec §5.6: **iki ayrı filtre grubu** — "Gönderim Amacına Göre" ve "Ürün
Tipine Göre". Kullanıcı ikisinden de seçebilir; sonuç **her iki koşula da
uyan** ürünler (AND).

- `useCategoryList()` → `axis`'e göre iki gruba ayır
- Her grup: seçilebilir chip'ler veya radio listesi
- Seçim → URL query'sini güncelle (`?amac=dogum-gunu&tip=buket`)
- "Temizle" butonu
- Mobilde: filtreler açılır panelde (yer kaplamasın)

- [ ] **Step 2: pages/urunler/index.vue**

**Filtre state'i URL'de** (spec §5.6) — Vue state'inde değil:
```ts
const route = useRoute()
const amac = computed(() => route.query.amac as string | undefined)
const tip = computed(() => route.query.tip as string | undefined)

const { data: products } = await useProductList({
  amac: amac.value,
  tip: tip.value,
  limit: 100,
})
```

Query değişince veri yenilenmeli — `useFetch`'in `watch` seçeneği veya
`key`'i query'ye bağlayarak.

- Ürün grid'i (`ProductCard`)
- Sonuç yoksa: "Bu filtreye uyan ürün bulunamadı" + filtreyi temizleme linki
- `useSeoMeta()`: title "Ürünler"

**Filtre kombinasyonları `noindex`** (spec §4.2):
```ts
useSeoMeta({
  robots: () => (amac.value || tip.value) ? 'noindex, follow' : 'index, follow',
})
```
Sebep: 10 occasion × 6 type = 60 kombinasyon, Google'da ince içerikli sayfa
üretir. Tekil kategoriler kendi path'lerinde indexlenir (`/kategori/[slug]`).

- [ ] **Step 3: AND filtresini DOĞRULA — spec §5.6'nın kalbi**

Admin panelden iki kategori ve şu ürünleri hazırla:
- Ürün A: Doğum Günü + Buket
- Ürün B: Doğum Günü + Orkide
- Ürün C: Taziye + Buket

Sonra:
```bash
curl -s "localhost:3000/urunler?amac=dogum-gunu&tip=buket" | grep -c "Ürün A"  # 1
curl -s "localhost:3000/urunler?amac=dogum-gunu&tip=buket" | grep -c "Ürün B"  # 0
curl -s "localhost:3000/urunler?amac=dogum-gunu&tip=buket" | grep -c "Ürün C"  # 0
```
**Sadece A gelmeli.** B veya C gelirse filtre OR çalışıyor demektir ve
"Doğum Günü + Buket" seçen müşteri taziye çelengi görür.

Tarayıcıda da: filtre seç → URL değişmeli → geri tuşu çalışmalı →
URL'i kopyalayıp yeni sekmede aç → aynı filtre gelmeli.

- [ ] **Step 4: Commit**

```bash
git add frontend/app
git commit -m "feat: ürün listesi ve iki eksenli AND filtresi

Filtre state'i URL'de — liste paylaşılabiliyor, geri tuşu çalışıyor.
Kombinasyonlar noindex: 60 kombinasyon Google'da ince içerik üretir."
```

---

## Task 7: Kategori sayfası, statik sayfalar, sitemap

**Files:**
- Create: `pages/kategori/[slug].vue`, `pages/hakkimizda.vue`, `pages/iletisim.vue`
- Modify: `nuxt.config.ts` (sitemap)

- [ ] **Step 1: pages/kategori/[slug].vue**

SEO'nun asıl hedefi — "geçmiş olsun çiçeği" arayan müşteri buraya düşer.

- `useFetch<Category>` ile kategoriyi çek (`/api/categories/:slug`), 404 ele al
- `useProductList()` ile o kategorinin ürünleri:
  - `axis === 'occasion'` → `?amac=<slug>`
  - `axis === 'type'` → `?tip=<slug>`
- `useSeoMeta()`: title `"{Kategori} | Çiçekçi"`, description
- **Bu sayfa indexlenir** (filtre kombinasyonlarının aksine)

- [ ] **Step 2: hakkimizda.vue ve iletisim.vue**

Basit statik sayfalar. İletişim bilgileri `runtimeConfig`'den gelsin
(spec §5.3 — `settings` tablosu ertelendi, `.env`'den okunuyor).

`nuxt.config.ts` runtimeConfig.public'e ekle:
```ts
contactPhone: process.env.NUXT_PUBLIC_CONTACT_PHONE || '',
contactAddress: process.env.NUXT_PUBLIC_CONTACT_ADDRESS || '',
contactHours: process.env.NUXT_PUBLIC_CONTACT_HOURS || '',
```
`.env` ve `.env.example`'a karşılıklarını ekle.

İletişim sayfasında WhatsApp linki de olsun (ürünsüz, genel mesaj).

- [ ] **Step 3: Sitemap**

`@nuxtjs/sitemap` kurulu. Dinamik URL'ler için `server/api/_sitemap-urls.ts`:

```ts
import type { Category, Product } from '~/types/api'

/** Sitemap için ürün ve kategori URL'leri — sadece aktif olanlar
 *  (backend zaten pasifleri göstermiyor). */
export default defineSitemapEventHandler(async () => {
  const config = useRuntimeConfig()
  const base = config.public.apiBase

  const [products, categories] = await Promise.all([
    $fetch<Product[]>(`${base}/products?limit=100`),
    $fetch<Category[]>(`${base}/categories`),
  ])

  return [
    ...products.map(p => ({ loc: `/urun/${p.slug}`, changefreq: 'weekly' as const })),
    ...categories.map(c => ({ loc: `/kategori/${c.slug}`, changefreq: 'weekly' as const })),
  ]
})
```

`nuxt.config.ts`:
```ts
sitemap: {
  sources: ['/api/_sitemap-urls'],
},
```

- [ ] **Step 4: Sitemap'i doğrula**

```bash
curl -s localhost:3000/sitemap.xml | head -30
```
Ürün ve kategori URL'leri görünmeli. **Filtre kombinasyonları GÖRÜNMEMELİ.**

- [ ] **Step 5: Commit**

```bash
git add frontend/app
git commit -m "feat: kategori sayfası, statik sayfalar, sitemap"
```

---

## Task 8: Son doğrulama — esnafın ve müşterinin akışı

- [ ] **Step 1: Build**

```bash
cd frontend/app
pnpm build
```
Hatasız olmalı.

- [ ] **Step 2: Production build'i çalıştır ve SSR'ı doğrula**

```bash
node .output/server/index.mjs
```
`pnpm dev` değil — production'da SSR gerçekten çalışıyor mu görmek için.

- [ ] **Step 3: MÜŞTERİNİN AKIŞI — uçtan uca**

Temiz veriyle (admin panelden hazırla):
1. Ana sayfa → featured kategoriler ve öne çıkan ürünler görünüyor
2. "Doğum Günü" kartına tıkla → kategori sayfası, o kategorinin ürünleri
3. Ürünler sayfasına git → iki eksenden filtrele → AND çalışıyor
4. Bir ürüne tıkla → galeri, fiyat, açıklama, kategoriler
5. Fotoğrafları gez (birden fazla varsa)
6. "WhatsApp'tan Sipariş Ver" → mesaj doğru, Türkçe karakterler bozulmamış
7. Mobil genişlikte (DevTools) hepsini tekrar gez — düzen bozulmuyor

- [ ] **Step 4: SSR KANITI — bu projenin varlık sebebi**

```bash
# Ürün sayfası JavaScript olmadan tam mı?
curl -s localhost:3000/urun/<slug> | grep -o '<meta property="og:image" content="[^"]*"'
curl -s localhost:3000/urun/<slug> | grep -o '<title>[^<]*</title>'
```

`og:image` dolu olmalı. **Bu satır, müşterinin WhatsApp'ta ürün fotoğrafını
görüp görmeyeceğini belirliyor.** Boşsa tüm SSR kararı boşa gitmiş demektir.

- [ ] **Step 5: Pasif ürün sızıyor mu**

Admin panelden bir ürünü pasif yap:
```bash
curl -s localhost:3000/urunler | grep -c "<pasif ürün adı>"   # 0 olmalı
curl -s -i localhost:3000/urun/<pasif-slug> | head -1          # 404 olmalı
```

- [ ] **Step 6: Görselsiz ürün patlatıyor mu**

Görseli olmayan bir ürün oluştur, sayfasını aç. Yer tutucu görünmeli,
sayfa patlamamalı. `og:image` boş olacak — bu kabul edilebilir (fotoğrafı
yok), ama sayfa çalışmalı.

- [ ] **Step 7: Commit**

```bash
git add -A frontend/app
git commit -m "feat: public site tamamlandı — uçtan uca doğrulandı"
```

---

## Plan 3 Bitiş Kriterleri

- [ ] `pnpm build` hatasız, production build çalışıyor
- [ ] `pnpm test` geçiyor (fiyat formatı + WhatsApp kodlaması)
- [ ] **`og:image` SSR'da geliyor** — curl ile doğrulandı
- [ ] Ürün adı ve fiyatı JavaScript olmadan HTML'de
- [ ] WhatsApp mesajı doğru: sipariş niyetli, fiyatlı, Türkçe karakterler sağlam
- [ ] Eski slug → 301 → **tarayıcı adresi yeni slug'a geçiyor**
- [ ] İki eksenli filtre AND çalışıyor (OR değil)
- [ ] Filtre URL'de, geri tuşu çalışıyor, liste paylaşılabiliyor
- [ ] Filtre kombinasyonları `noindex`, kategori sayfaları indexleniyor
- [ ] `sitemap.xml` ürün ve kategorileri içeriyor
- [ ] Pasif ürün public'te görünmüyor (liste yok, detay 404)
- [ ] Görselsiz ürün sayfası patlamıyor
- [ ] Mobile-first: dar ekranda düzen bozulmuyor
- [ ] Görseller lazy, sabit aspect-ratio ile — sayfa zıplamıyor

**Sonraki:** Final whole-branch review (Plan 1+2+3+4), sonra deployment.
