# Plan 4 — Admin Panel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Çiçekçi esnafının ürün, kategori ve görsel yönettiği panel. Mevcut Vuetify template'i temel alınıyor.

**Architecture:** Vite + Vue 3 + Vuetify 3 SPA (`frontend/idare/`). SSR yok — admin paneli bot görmez, SEO gereksiz. Go API'yi (`localhost:8080`) HttpOnly cookie ile tüketiyor.

**Tech Stack:** Vue 3.4, Vuetify 3.6, Vite 5, TypeScript, Pinia, vue-router (dosya tabanlı, `unplugin-vue-router`), axios.

**Spec:** `docs/superpowers/specs/2026-07-15-cicekci-mvp-design.md` — §4.5 (auth), §4.6 (admin uçları), §5.2 (sayfa yapısı).

**Önkoşul:** Plan 1 ve 2 tamamlandı. Backend çalışıyor: 179 test, admin API'nin tamamı hazır.

---

## Başlangıç Durumu — bu planı yeni bir oturumda uygulayacaksan ÖNCE OKU

**Repo:** `/Users/omerkoc/GolandProjects/cicekci`, branch `feat/backend-temeli`
(Plan 1, 2 ve template hepsi bu branch'te — ayrı branch açma).

**Ortam (doğrulandı):** Go 1.25.4, Node 22.22.3, pnpm 10.24, Docker çalışıyor,
golang-migrate kurulu.

**Backend'i ayağa kaldırmak:**
```bash
cd /Users/omerkoc/GolandProjects/cicekci
export DATABASE_URL="postgres://cicekci:cicekci@localhost:5433/cicekci?sslmode=disable"
export JWT_SECRET="local-development-secret-32-chars!"
export WHATSAPP_NUMBER="905551234567"
export SITE_URL="http://localhost:5173"   # Vite portu — CORS için KRİTİK
export APP_ENV=development STORAGE_DRIVER=local
make db-up && make migrate-up && make run
```
Admin kullanıcısı yoksa: `make seed` (interaktif TTY ister; script'ten
çalıştırmak için `expect` gerekir).

**Testler:** `make test` kullan, `go test ./...` DEĞİL. Tüm test paketleri aynı
test veritabanını paylaşıyor ve `NewTestDB` TRUNCATE çalıştırıyor — paralel
çalışırlarsa birbirlerinin verisini siler. Makefile'da `-p 1` var.

**Template'te zaten yapılanlar (commit 6885c6b) — tekrarlama:**
- `build-icons.ts` Node 22 uyumu: `createRequire` + `fileURLToPath` eklendi
  (`package.json`'da `"type": "module"` var, CommonJS globalleri tanımsızdı)
- `NavSearchBar.vue` silindi — ölü kod (mock veri katmanı `@db` yok, bileşen
  hiçbir yerde kullanılmıyordu)
- `Popup.ts`'de 4 `querySelector` `HTMLElement`'e tiplendi, router'da
  `'index'` → `'root'` düzeltildi
- `yarn.lock` silindi — pnpm-lock ile ikisi bir aradaydı

**Bilinen durum:** `pnpm typecheck` 17 hata veriyor, hepsi `@core`/`@layouts`
çekirdeğinde (Vue/TS sürüm katılaşması). **Bu beklenen** — kullanıcı kararıyla
bırakıldı. `pnpm build` sorunsuz çalışıyor (12sn). Bu dosyalara dokunma;
kendi yazacağın kod tipli olsun.

**Paket yöneticisi: pnpm.** npm veya yarn kullanma.

## Global Constraints

- **Auth: HttpOnly cookie.** `Authorization` header YOK, `localStorage`'da token YOK. Bu spec §4.5'in bilinçli kararı — localStorage'daki token XSS ile çalınabilir. Axios'ta `withCredentials: true` zorunlu.
- **Refresh token YOK.** Backend tek token veriyor (7 gün TTL). Template'in refresh kuyruğu silinecek.
- **Rol sistemi YOK.** Tek admin var. Template'in `UserRole` ('admin' | 'teacher' | 'parent') ve `@casl` yetkilendirmesi silinecek.
- **API yanıt formatı düz JSON.** Template `{data, data_count, error_code, error_message}` sarmalayıcısı bekliyor — bizim API'de böyle bir şey yok. Hatalar: `{"error": {"code": "...", "message": "..."}}`.
- **Fiyat string.** JSON'da `"1850.00"` olarak gidip geliyor (float precision). `decimal.NewFromString` parse ediyor, geçersizse 400.
- **Arayüz dili Türkçe.** Template zaten Türkçe (navigasyon, login).
- **Yorumlar Türkçe.**

---

## Backend API Sözleşmesi (Plan 1+2'de uygulandı ve doğrulandı)

Bu plan bu uçlara karşı yazılıyor. Hepsi gerçek sunucuda test edildi.

```
POST   /api/admin/login          {username, password} → {ok:true} + Set-Cookie
POST   /api/admin/logout         → {ok:true}
GET    /api/admin/me             → {username}

GET    /api/admin/products?page=1&limit=24
POST   /api/admin/products       {name, description, price:"1850.00", is_active?, category_ids?}
GET    /api/admin/products/:id
PATCH  /api/admin/products/:id   (kısmi — gönderilmeyen alan değişmez)
DELETE /api/admin/products/:id

GET    /api/admin/products/:id/images
POST   /api/admin/products/:id/images        multipart, alan adı: "image"
PATCH  /api/admin/products/:id/images/order  {image_ids:[3,1,2]} — TÜM görseller
DELETE /api/admin/images/:id

GET    /api/admin/categories
POST   /api/admin/categories     {name, axis:"occasion"|"type", is_active?, is_featured?, sort_order?}
PATCH  /api/admin/categories/:id (kısmi; axis DEĞİŞTİRİLEMEZ)
GET    /api/admin/categories/:id/product-count
DELETE /api/admin/categories/:id
```

**Ürün yanıtı:**
```json
{"id":1, "name":"51 Gül Buket", "slug":"51-gul-buket", "description":"",
 "price":"1850.00", "is_active":true, "category_ids":[1,2],
 "images":[{"id":1,"url_400":"...","url_1200":"...","sort_order":0}]}
```

**PATCH semantiği (kritik):** Alan gönderilmezse değişmez. `category_ids` özel: gönderilmezse dokunulmaz, `[]` gönderilirse hepsi kaldırılır.

---

## Template'ten ne alınıyor, ne atılıyor

**Alınan:**
- Vuetify + tema + `@core`/`@layouts` (panelin görünümü)
- `src/pages/auth/login.vue` — form ve tasarım hazır, sadece auth çağrısı değişecek
- `src/utils/Popup.ts` — `ErrorPopup`, `SuccessToast`, `WarningPopup`, `ConfirmPopup`
- Dosya tabanlı router, TypeScript kurulumu, ESLint
- `src/navigation/vertical/index.ts` — menü tanımı (içeriği değişecek)

**Atılan (bizim API'mize uymuyor veya ölü):**
- `src/services/JwtService.ts` — localStorage; biz HttpOnly cookie'deyiz
- `ApiService`'in refresh-token kuyruğu — bizde refresh token yok
- `src/components/extable.vue` (594 satır) — kendi sorgu protokolüne bağlı
  (`QUERY_COLUMN_TYPES`, `SORT_COLUMN_TYPES.JSONB`, `ASSOCIATION_STRING`).
  Bizim backend bu protokolü sunmuyor ve beş ekranlık panel için sunmamalı.
  Yerine Vuetify `VDataTable` doğrudan kullanılacak.
- `src/model/api.ts` — extable'ın sorgu modelleri
- `src/pages/users.vue`, `second-page.vue`, `version.vue`, `profile.vue` — demo
- `src/store/user.ts`'in rol sistemi (`teacher`, `parent`, `@casl`)
- `@casl/ability`, `@casl/vue`, `apexcharts`, `chart.js`, `vue-chartjs`,
  `vue3-apexcharts`, `mapbox-gl`, `@tiptap/*`, `swiper`, `prismjs`,
  `vue-prism-component`, `shepherd.js`, `moment`, `@formkit/drag-and-drop`,
  `jwt-decode` — hiçbiri kullanılmayacak

**Not:** 17 tip hatası `@core`/`@layouts` çekirdeğinde duruyor (Vue/TS sürüm
katılaşması). Uygulama derleniyor (`pnpm build` 12sn). Kullanıcı kararıyla
şimdilik bırakıldı — biz o dosyalara dokunmayacağız, kendi kodumuz tipli olacak.

---

## Dosya Yapısı

```
frontend/idare/src/
  services/
    ApiService.ts        → SADELEŞTİR: cookie auth, düz JSON, refresh yok
    JwtService.ts        → SİL
  store/
    user.ts              → SADELEŞTİR: rol yok, cookie tabanlı oturum
  model/
    api.ts               → SİL (extable'ın sorgu modelleri)
    product.ts           → YENİ: Product, ProductImage tipleri
    category.ts          → YENİ: Category, Axis tipleri
  composables/
    useProducts.ts       → YENİ: ürün API çağrıları
    useCategories.ts     → YENİ: kategori API çağrıları
    useImages.ts         → YENİ: görsel yükleme/sıralama/silme
  pages/
    auth/login.vue       → auth çağrısını değiştir
    index.vue            → ürün listesi (ana ekran)
    urunler/[id].vue     → ürün formu (yeni + düzenle)
    kategoriler.vue      → kategori yönetimi
    siparisler.vue       → YENİ: Faz 2 placeholder
    users.vue            → SİL
    second-page.vue      → SİL
    version.vue          → SİL
    profile.vue          → SİL
  components/
    extable.vue          → SİL
    ProductImageManager.vue → YENİ: yükleme + sıralama + silme
  navigation/vertical/index.ts → menüyü değiştir
```

---

## Task 1: Temizlik — kullanılmayan bağımlılıklar ve demo sayfalar

**Files:**
- Delete: `src/services/JwtService.ts`, `src/components/extable.vue`, `src/model/api.ts`
- Delete: `src/pages/users.vue`, `src/pages/second-page.vue`, `src/pages/version.vue`, `src/pages/profile.vue`
- Modify: `package.json`

- [x] **Step 1: Kullanılmayan sayfaları ve bileşenleri sil**

```bash
cd frontend/idare
rm -f src/pages/users.vue src/pages/second-page.vue src/pages/version.vue src/pages/profile.vue
rm -f src/components/extable.vue src/services/JwtService.ts src/model/api.ts
```

- [x] **Step 2: Bu dosyalara referans veren yerleri bul**

```bash
grep -rn "extable\|JwtService\|model/api\|second-page\|'users'\|version.vue" src/ --include="*.vue" --include="*.ts" | grep -v node_modules
```
Çıkan her referans düzeltilecek (sonraki task'larda ele alınıyor: `ApiService`, `store/user.ts`, `navigation`).

- [x] **Step 3: Kullanılmayan bağımlılıkları kaldır**

```bash
pnpm remove @casl/ability @casl/vue apexcharts chart.js vue-chartjs vue3-apexcharts \
  mapbox-gl @tiptap/extension-highlight @tiptap/extension-image @tiptap/extension-link \
  @tiptap/extension-text-align @tiptap/pm @tiptap/starter-kit @tiptap/vue-3 \
  swiper prismjs vue-prism-component shepherd.js moment jwt-decode \
  @formkit/drag-and-drop
```

- [x] **Step 4: Kırılan importları temizle**

```bash
grep -rn "casl\|apexchart\|chart.js\|mapbox\|tiptap\|swiper\|prism\|shepherd\|moment\|jwt-decode\|drag-and-drop" src/ --include="*.ts" --include="*.vue" | grep -v node_modules
```
Bulunan her importu ve kullanan kodu sil. `src/plugins/` altında kayıt dosyaları olabilir.

- [x] **Step 5: Build hâlâ çalışıyor mu**

Run: `pnpm build`
Expected: `✓ built in ...` — hata yok. Bundle boyutu öncekinden küçük olmalı.

- [x] **Step 6: Commit**

```bash
git add -A frontend/idare
git commit -m "chore: kullanılmayan template parçalarını temizle

extable.vue kendi sorgu protokolüne bağlıydı (QUERY_COLUMN_TYPES,
JSONB sıralama) — bizim backend bunu sunmuyor, beş ekranlık panel için
sunmamalı. VDataTable doğrudan kullanılacak.

JwtService localStorage kullanıyordu — biz HttpOnly cookie'deyiz.

Silinen bağımlılıklar: casl, apexcharts, chart.js, mapbox, tiptap,
swiper, prism, shepherd, moment, jwt-decode, drag-and-drop."
```

---

## Task 2: ApiService — cookie auth, düz JSON

**Files:**
- Modify: `src/services/ApiService.ts` (199 satır → ~60 satır)
- Modify: `.env.development`, `.env.production`

**Interfaces:**
- Produces: `ApiService.get/post/patch/delete<T>(url, data?) → Promise<[error, response]>`

- [x] **Step 1: .env dosyalarını backend'e göre ayarla**

`.env.development`:
```
VITE_API_BASE_URL=http://localhost:8080/api
```

`.env.production`:
```
VITE_API_BASE_URL=/api
```

Not: Backend'in CORS'u `AllowOrigins: cfg.SiteURL` + `AllowCredentials: true`
kullanıyor. Geliştirmede `SITE_URL=http://localhost:5173` (Vite'ın portu)
olmalı, yoksa tarayıcı cookie'yi göndermez.

- [x] **Step 2: ApiService'i yeniden yaz**

`src/services/ApiService.ts`:
```ts
import type { AxiosInstance } from 'axios'
import axios from 'axios'

// Backend hata formatı (spec §4.6): {"error": {"code": "...", "message": "..."}}
export interface ApiError {
  code: string
  message: string
}

/**
 * ApiService Go backend'ini tüketir.
 *
 * Auth HttpOnly cookie ile (spec §4.5) — Authorization header YOK,
 * localStorage'da token YOK. withCredentials cookie'nin gönderilmesini sağlar.
 * Refresh token yok: backend tek token veriyor (7 gün).
 */
class ApiService {
  private static instance: AxiosInstance

  private static init() {
    ApiService.instance = axios.create({
      baseURL: import.meta.env.VITE_API_BASE_URL,
      withCredentials: true, // HttpOnly cookie için zorunlu
      headers: { Accept: 'application/json' },
    })
  }

  private static get client(): AxiosInstance {
    if (!ApiService.instance)
      ApiService.init()

    return ApiService.instance
  }

  /** Hata mesajını backend formatından çıkarır, yoksa genel mesaj döner. */
  private static toError(err: any): ApiError {
    const body = err?.response?.data?.error
    if (body?.message)
      return { code: body.code ?? 'unknown', message: body.message }

    if (err?.response?.status === 401)
      return { code: 'unauthorized', message: 'Oturumunuz sona ermiş' }

    return { code: 'network', message: 'Sunucuya ulaşılamadı' }
  }

  /** [error, data] döner — Go'daki hata deseninin TypeScript karşılığı. */
  static async request<T>(
    method: 'get' | 'post' | 'patch' | 'delete',
    url: string,
    data?: unknown,
    config?: object,
  ): Promise<[ApiError | null, T]> {
    try {
      const resp = await ApiService.client.request<T>({ method, url, data, ...config })

      return [null, resp.data]
    }
    catch (err) {
      return [ApiService.toError(err), undefined as T]
    }
  }

  static get<T>(url: string, config?: object) {
    return ApiService.request<T>('get', url, undefined, config)
  }

  static post<T>(url: string, data?: unknown, config?: object) {
    return ApiService.request<T>('post', url, data, config)
  }

  static patch<T>(url: string, data?: unknown) {
    return ApiService.request<T>('patch', url, data)
  }

  static delete<T>(url: string) {
    return ApiService.request<T>('delete', url)
  }
}

export default ApiService
```

- [x] **Step 3: Build kontrolü**

Run: `pnpm build`
Expected: ApiService'i kullanan yerler kırılmış olabilir (login.vue, store/user.ts) — sonraki task'larda düzeltiliyor. Şimdilik hata varsa not al.

- [x] **Step 4: Commit**

```bash
git add src/services/ApiService.ts .env.development .env.production
git commit -m "feat: ApiService'i cookie auth ve düz JSON'a uyarla

Template Bearer header + localStorage + refresh kuyruğu kullanıyordu.
Bizim backend HttpOnly cookie veriyor ve refresh token yok — 199 satır
60 satıra indi."
```

---

## Task 3: Auth — store ve login

**Files:**
- Modify: `src/store/user.ts`
- Modify: `src/pages/auth/login.vue`
- Modify: `src/plugins/1.router/index.ts` (guard)
- Delete: `src/pages/auth/forgot-password.vue` (backend'de karşılığı yok)

**Interfaces:**
- Produces: `useUserStore()` → `{ username, isAuthenticated, login(), logout(), checkSession() }`

- [x] **Step 1: store/user.ts'i yeniden yaz**

`src/store/user.ts`:
```ts
import { defineStore } from 'pinia'
import ApiService from '@/services/ApiService'

/**
 * Oturum durumu. Token HttpOnly cookie'de olduğu için JavaScript onu
 * okuyamaz — oturumun geçerliliği /me çağrısıyla anlaşılır.
 * Tek admin var, rol sistemi yok.
 */
export const useUserStore = defineStore('UserStore', {
  state: () => ({
    username: '',
    isAuthenticated: false,
  }),
  actions: {
    async login(username: string, password: string): Promise<string | null> {
      const [err] = await ApiService.post('admin/login', { username, password })
      if (err)
        return err.message

      await this.checkSession()

      return null
    },

    async logout() {
      await ApiService.post('admin/logout')
      this.username = ''
      this.isAuthenticated = false
    },

    /** Cookie geçerli mi — sayfa yenilendiğinde oturumu geri kazanmak için. */
    async checkSession(): Promise<boolean> {
      const [err, data] = await ApiService.get<{ username: string }>('admin/me')
      if (err) {
        this.isAuthenticated = false
        this.username = ''

        return false
      }
      this.username = data.username
      this.isAuthenticated = true

      return true
    },
  },
})
```

- [x] **Step 2: forgot-password sayfasını sil**

```bash
rm -f src/pages/auth/forgot-password.vue
grep -rn "forgot-password" src/ --include="*.vue" --include="*.ts" | grep -v node_modules
```
Bulunan linkleri kaldır (login.vue'da olabilir).

- [x] **Step 3: login.vue'nun script kısmını değiştir**

Mevcut `onSubmit` şuna benzer:
```ts
const [error, resp] = await ApiService.post<any>('auth/login', form.value)
if (error) return ErrorPopup(error)
await useUserStore().login(resp.data.access_token, resp.data.refresh_token)
```

Yenisi:
```ts
const onSubmit = async () => {
  const { valid } = await formRef.value!.validate()
  if (!valid)
    return

  loading.value = true
  const errMsg = await useUserStore().login(form.value.username, form.value.password)
  loading.value = false

  if (errMsg)
    return ErrorPopup({ message: errMsg })

  await router.push('/')
}
```

Formun alan adları backend'e uymalı: `username` ve `password`. Template'te
`email` olabilir — `username`'e çevir, etiket "Kullanıcı Adı" olsun.

`console.log(import.meta.env.VITE_API_BASE_URL)` satırını sil.

- [x] **Step 4: Router guard'ını sadeleştir**

`src/plugins/1.router/index.ts` içindeki guard rol kontrolü yapıyorsa kaldır.
Sadece şu kalsın: giriş yapılmamışsa ve sayfa korumalıysa `/auth/login`'e gönder.

Oturum kontrolü `isAuthenticated` state'ine bakmalı; uygulama açılışında
bir kez `checkSession()` çağrılmalı (`main.ts` veya guard'da).

- [x] **Step 5: Backend'i başlat ve login'i ELLE test et**

```bash
# Terminal 1 — backend
cd /Users/omerkoc/GolandProjects/cicekci
export DATABASE_URL="postgres://cicekci:cicekci@localhost:5433/cicekci?sslmode=disable"
export JWT_SECRET="local-development-secret-32-chars!"
export WHATSAPP_NUMBER="905551234567"
export SITE_URL="http://localhost:5173"   # Vite portu — CORS için kritik
export APP_ENV=development STORAGE_DRIVER=local
make db-up && make migrate-up && make run
```

```bash
# Terminal 2 — admin kullanıcısı (yoksa)
make seed   # kullanıcı: cicekci, şifre: test-sifre-123
```

```bash
# Terminal 3 — frontend
cd frontend/idare && pnpm dev
```

Tarayıcıda `http://localhost:5173/auth/login`:
- Yanlış şifre → hata popup'ı
- Doğru şifre → ana sayfaya yönlenme
- **DevTools → Application → Cookies:** `cicekci_token` görünmeli,
  `HttpOnly` işaretli olmalı
- **DevTools → Application → Local Storage:** token OLMAMALI
- Sayfayı yenile → oturum korunmalı (checkSession çalışıyor)
- Çıkış yap → cookie silinmeli

- [x] **Step 6: Commit**

```bash
git add -A src/store src/pages/auth src/plugins/1.router
git commit -m "feat: cookie tabanlı auth — rol sistemi kaldırıldı

Template localStorage + refresh token + 3 rol (admin/teacher/parent)
kullanıyordu; okul projesinden kalmış. Bizde tek admin ve HttpOnly
cookie var. Oturum /me çağrısıyla doğrulanıyor — token JavaScript'ten
okunamaz."
```

---

## Task 4: Tipler ve API composable'ları

**Files:**
- Create: `src/model/product.ts`, `src/model/category.ts`
- Create: `src/composables/useProducts.ts`, `src/composables/useCategories.ts`, `src/composables/useImages.ts`

**Interfaces:**
- Produces: `Product`, `ProductImage`, `Category`, `Axis` tipleri ve API fonksiyonları

- [x] **Step 1: model/category.ts**

```ts
export type Axis = 'occasion' | 'type'

export interface Category {
  id: number
  name: string
  slug: string
  axis: Axis
  is_active: boolean
  is_featured: boolean
  sort_order: number
}

export interface CategoryCreate {
  name: string
  axis: Axis
  is_active?: boolean
  is_featured?: boolean
  sort_order?: number
}

// PATCH semantiği: undefined alan değişmez. axis DEĞİŞTİRİLEMEZ —
// eksen değişimi ürün ilişkilerini anlamsız kılar (spec §4.1).
export interface CategoryUpdate {
  name?: string
  is_active?: boolean
  is_featured?: boolean
  sort_order?: number
}

export const AXIS_LABELS: Record<Axis, string> = {
  occasion: 'Gönderim Amacına Göre',
  type: 'Ürün Tipine Göre',
}
```

- [x] **Step 2: model/product.ts**

```ts
export interface ProductImage {
  id: number
  url_400: string
  url_1200: string
  sort_order: number
}

export interface Product {
  id: number
  name: string
  slug: string
  description: string
  price: string // "1850.00" — float precision için string (spec §4.1)
  is_active: boolean
  category_ids: number[]
  images: ProductImage[]
}

export interface ProductCreate {
  name: string
  description: string
  price: string
  is_active?: boolean
  category_ids?: number[]
}

// PATCH semantiği: undefined alan değişmez.
// category_ids özel: undefined → dokunma, [] → hepsini kaldır.
export interface ProductUpdate {
  name?: string
  description?: string
  price?: string
  is_active?: boolean
  category_ids?: number[]
}
```

- [x] **Step 3: composables/useCategories.ts**

```ts
import ApiService from '@/services/ApiService'
import type { Category, CategoryCreate, CategoryUpdate } from '@/model/category'

export function useCategories() {
  const list = () => ApiService.get<Category[]>('admin/categories')

  const create = (data: CategoryCreate) =>
    ApiService.post<Category>('admin/categories', data)

  const update = (id: number, data: CategoryUpdate) =>
    ApiService.patch<Category>(`admin/categories/${id}`, data)

  const remove = (id: number) =>
    ApiService.delete<void>(`admin/categories/${id}`)

  // Silme öncesi uyarı için: "Bu kategoride N ürün var" (spec §4.1)
  const productCount = (id: number) =>
    ApiService.get<{ product_count: number }>(`admin/categories/${id}/product-count`)

  return { list, create, update, remove, productCount }
}
```

- [x] **Step 4: composables/useProducts.ts**

```ts
import ApiService from '@/services/ApiService'
import type { Product, ProductCreate, ProductUpdate } from '@/model/product'

export function useProducts() {
  const list = (page = 1, limit = 24) =>
    ApiService.get<Product[]>(`admin/products?page=${page}&limit=${limit}`)

  const get = (id: number) => ApiService.get<Product>(`admin/products/${id}`)

  const create = (data: ProductCreate) =>
    ApiService.post<Product>('admin/products', data)

  const update = (id: number, data: ProductUpdate) =>
    ApiService.patch<Product>(`admin/products/${id}`, data)

  const remove = (id: number) => ApiService.delete<void>(`admin/products/${id}`)

  return { list, get, create, update, remove }
}
```

- [x] **Step 5: composables/useImages.ts**

```ts
import ApiService from '@/services/ApiService'
import type { ProductImage } from '@/model/product'

export function useImages() {
  const list = (productId: number) =>
    ApiService.get<ProductImage[]>(`admin/products/${productId}/images`)

  /** multipart/form-data, alan adı "image". Backend JPEG/PNG kabul eder. */
  const upload = (productId: number, file: File) => {
    const fd = new FormData()

    fd.append('image', file)

    return ApiService.post<ProductImage>(
      `admin/products/${productId}/images`,
      fd,
      { headers: { 'Content-Type': 'multipart/form-data' } },
    )
  }

  /** imageIds ürünün TÜM görsellerini içermeli — ilki kapak olur. */
  const reorder = (productId: number, imageIds: number[]) =>
    ApiService.patch<ProductImage[]>(
      `admin/products/${productId}/images/order`,
      { image_ids: imageIds },
    )

  const remove = (imageId: number) =>
    ApiService.delete<void>(`admin/images/${imageId}`)

  return { list, upload, reorder, remove }
}
```

- [x] **Step 6: Build kontrolü ve commit**

```bash
pnpm build
git add src/model src/composables
git commit -m "feat: ürün/kategori/görsel tipleri ve API composable'ları"
```

---

## Task 5: Kategori yönetimi ekranı

**Files:**
- Create: `src/pages/kategoriler.vue`
- Modify: `src/navigation/vertical/index.ts`

- [x] **Step 1: Menüyü çiçekçiye göre değiştir**

`src/navigation/vertical/index.ts`:
```ts
export default [
  {
    title: 'Ürünler',
    to: { name: 'root' },
    icon: { icon: 'tabler-flower' },
  },
  {
    title: 'Kategoriler',
    to: { name: 'kategoriler' },
    icon: { icon: 'tabler-tags' },
  },
  {
    title: 'Siparişler',
    to: { name: 'siparisler' },
    icon: { icon: 'tabler-shopping-cart' },
  },
]
```

`src/navigation/horizontal/index.ts` varsa aynı içeriği koy veya dosyayı sil
(dikey nav kullanılıyorsa).

- [x] **Step 2: kategoriler.vue yaz**

İki eksen ayrı tablolarda gösterilmeli (spec §5.6: "Gönderim Amacına Göre" ve
"Ürün Tipine Göre" iki ayrı grup).

Gereksinimler:
- İki `VDataTable`: occasion ve type eksenleri ayrı
- Her satırda: ad, slug (salt okunur), `is_active` switch, `is_featured` switch,
  sıra numarası, düzenle/sil butonları
- "Yeni Kategori" butonu → dialog: ad + eksen seçimi (radio: Gönderim Amacı /
  Ürün Tipi) + aktif/öne çıkan switch'leri
- Düzenle dialog'unda **eksen değiştirilemez** — salt okunur göster
- **Silme öncesi:** `productCount(id)` çağır, sonra `ConfirmPopup`:
  "Bu kategoride N ürün var. Silerseniz bu ürünler kategoriden çıkacak
  (ürünler silinmez). Devam?"
- `is_active` ve `is_featured` switch'leri anında `update()` çağırsın
  (optimistic değil — yanıt gelince state güncellensin, hata olursa geri al)

**İki kritik iş kuralını arayüzde göster:**
- `is_active=false` ise satır soluk görünsün ve `is_featured` switch'i devre
  dışı olsun — pasif kategori featured olsa bile görünmez (spec §4.1).
  Yanına ipucu: "Pasif kategori ana sayfada görünmez."
- Slug salt okunur ve değişmez — kategori URL'leri sabit (spec §4.2).

- [x] **Step 3: ELLE test**

Backend ve frontend çalışırken:
- İki eksende kategori oluştur: "Doğum Günü" (occasion), "Buket" (type)
- `is_featured` aç/kapa → sayfayı yenile, korunmalı
- `is_active` kapat → satır solmalı, featured switch'i kilitlenmeli
- Kategoriyi sil → "0 ürün var" uyarısı çıkmalı
- Ürün bağlı bir kategoriyi sil → doğru sayı görünmeli, silince ürün
  silinmemeli (ürünler ekranından doğrula)

- [x] **Step 4: Commit**

```bash
git add src/pages/kategoriler.vue src/navigation
git commit -m "feat: kategori yönetimi ekranı — iki eksen, silme uyarısı"
```

---

## Task 6: Ürün listesi (ana ekran)

**Files:**
- Modify: `src/pages/index.vue`

- [x] **Step 1: index.vue'yu ürün listesine çevir**

Gereksinimler:
- `VDataTable` — kolonlar: kapak görseli (küçük thumbnail, `url_400`), ad,
  fiyat, kategoriler (chip'ler), durum (aktif/pasif chip), işlemler
- Kapak görseli yoksa yer tutucu ikon göster
- "Yeni Ürün" butonu → `/urunler/yeni`
- Satır tıklama veya düzenle butonu → `/urunler/:id`
- Sil butonu → `ConfirmPopup`: "Bu ürün ve tüm görselleri silinecek. Devam?"
- `is_active` chip'i: aktif=yeşil, pasif=gri. Pasif satır soluk.
- Sayfalama: backend `?page=&limit=` alıyor, limit max 100

**Not:** Backend `data_count` dönmüyor — toplam sayı yok. Sayfalama için
"sonraki sayfa boş mu" mantığı kullan veya limit'i yüksek tut (esnafın
40-100 ürünü olacak, tek sayfada gösterilebilir). Basit tut: `limit=100`
ile tek sayfa, sayfalama YAPMA. Ürün sayısı büyürse Faz 2'de eklenir.

- [x] **Step 2: ELLE test**

- Ürün listesi yükleniyor mu
- Kapak görseli görünüyor mu (görsel yüklenmiş bir üründe)
- Görseli olmayan üründe yer tutucu çıkıyor mu
- Pasif ürün soluk mu
- Silme onayı çıkıyor mu, silince liste güncelleniyor mu

- [x] **Step 3: Commit**

```bash
git add src/pages/index.vue
git commit -m "feat: ürün listesi ekranı"
```

---

## Task 7: Ürün formu + görsel yönetimi

**Files:**
- Create: `src/pages/urunler/[id].vue`
- Create: `src/components/ProductImageManager.vue`

Bu planın en büyük parçası — görsel yönetimi burada.

- [x] **Step 1: ProductImageManager.vue yaz**

Props: `productId: number`, `images: ProductImage[]`
Emits: `update` (görseller değişince liste yenilensin)

Gereksinimler:
- Görselleri grid'de göster (`url_400` thumbnail)
- İlk görselde "Kapak" rozeti — `sort_order=0` olan kapak (spec §4.4)
- Her görselde: sil butonu, sola/sağa taşı butonları
- **Sürükle-bırak YOK** — `@formkit/drag-and-drop` kaldırıldı. Basit
  yukarı/aşağı (veya sola/sağa) butonları yeterli (spec §4.4:
  "sürükle-bırak değil, basit yukarı/aşağı butonları yeter")
- Taşıma → `reorder(productId, yeniSıralamaIdListesi)` — TÜM id'ler gönderilmeli
- `VFileInput` veya sürükle-bırak alanı ile yükleme
- Yükleme sırasında spinner; backend işleme birkaç yüz ms sürüyor
- Hata durumunda `ErrorPopup` — backend "Sadece JPEG veya PNG" gibi net
  mesajlar dönüyor
- Silme → `ConfirmPopup`

**Önemli:** Ürün oluşturulmadan görsel yüklenemez (backend ürünün varlığını
kontrol ediyor, yoksa 404). Yeni ürün formunda görsel bölümü ürün
kaydedilene kadar devre dışı olsun ve bunu kullanıcıya söyle:
"Görsel eklemek için önce ürünü kaydedin."

- [x] **Step 2: urunler/[id].vue yaz**

Route: `/urunler/yeni` (oluşturma) ve `/urunler/:id` (düzenleme).
`id === 'yeni'` ise oluşturma modu.

Gereksinimler:
- Alanlar: ad (zorunlu), açıklama (textarea), fiyat (zorunlu, sayı),
  aktif switch, kategori seçimi
- **Kategori seçimi iki grupta:** "Gönderim Amacına Göre" ve "Ürün Tipine
  Göre" — iki ayrı `VSelect` (multiple) veya checkbox grubu. Kategoriler
  `useCategories().list()` ile yüklenip `axis`'e göre ayrılır.
  Pasif kategoriler seçilebilir olsun ama "(pasif)" etiketiyle.
- **Fiyat girişi:** kullanıcı `1850` veya `1850.50` yazar, API'ye
  `"1850.00"` string'i gider. `parseFloat(x).toFixed(2)` ile normalize et.
  Negatif ve boş reddedilsin (backend de reddediyor ama kullanıcıya
  anında geri bildirim ver).
- Slug salt okunur gösterilsin (düzenleme modunda) + bilgi notu:
  "Ürün adını değiştirirseniz yeni bir link oluşur, eski link yeni linke
  yönlendirilir." — bu, spec §4.2'nin kullanıcıya yansıması
- Kaydet → oluşturmada `create()`, düzenlemede `update()`
- Oluşturma başarılıysa `/urunler/:yeniId`'ye yönlen (görsel eklenebilsin)
- `ProductImageManager` sadece düzenleme modunda aktif

- [x] **Step 3: ELLE test — tam akış**

Esnafın yaşayacağı akış:
1. "Yeni Ürün" → ad "51 Gül Buket", fiyat 1850, kategori seç → Kaydet
2. Düzenleme sayfasına yönlenmeli, görsel bölümü açılmalı
3. Fotoğraf yükle → thumbnail görünmeli, "Kapak" rozeti olmalı
4. İkinci fotoğraf yükle → yan yana görünmeli
5. İkinciyi öne taşı → "Kapak" rozeti ona geçmeli
6. Sayfayı yenile → sıra korunmalı
7. Ürün adını "51 Kırmızı Gül Buketi" yap → kaydet → slug değişmeli
8. Bir görseli sil → gitmeli
9. PDF yüklemeyi dene → net hata mesajı çıkmalı
10. Ürünü sil → liste ekranına dön, ürün gitmiş olmalı

- [x] **Step 4: Commit**

```bash
git add src/pages/urunler src/components/ProductImageManager.vue
git commit -m "feat: ürün formu ve görsel yönetimi"
```

---

## Task 8: Sipariş placeholder + son temizlik

**Files:**
- Create: `src/pages/siparisler.vue`
- Modify: `src/pages/[...error].vue` veya `[...all].vue` (gerekiyorsa)

- [x] **Step 1: siparisler.vue — Faz 2 placeholder**

Spec §5.2: *"admin routing'i, ileride sipariş yönetimi sayfası eklenecekmiş
gibi genişletilebilir tut"*.

Basit bir sayfa:
```vue
<template>
  <VCard>
    <VCardText class="text-center py-12">
      <VIcon icon="tabler-shopping-cart" size="64" class="mb-4 text-disabled" />
      <h5 class="text-h5 mb-2">
        Sipariş Yönetimi
      </h5>
      <p class="text-body-1 text-medium-emphasis">
        Bu bölüm Faz 2'de gelecek. Şu an siparişler WhatsApp üzerinden
        alınıyor.
      </p>
    </VCardText>
  </VCard>
</template>
```

- [x] **Step 2: Kalan demo izlerini temizle**

```bash
grep -rn "second-page\|users\|version\|profile\|teacher\|parent\|casl" src/ \
  --include="*.vue" --include="*.ts" | grep -v node_modules
```
Kalan referansları temizle.

- [x] **Step 3: Build + bundle boyutu**

Run: `pnpm build`
Expected: hatasız. Bundle, Task 1 öncesine göre belirgin küçük olmalı
(apexcharts, mapbox, tiptap gittikten sonra).

- [x] **Step 4: Tam akış ELLE testi**

Temiz bir veritabanıyla baştan sona:
```bash
docker compose exec -T postgres psql -U cicekci -d cicekci \
  -c "TRUNCATE products, product_slugs, product_images, categories, product_categories RESTART IDENTITY CASCADE;"
```
1. Giriş yap
2. İki kategori oluştur (biri occasion, biri type)
3. Ürün oluştur, ikisini de seç
4. İki fotoğraf yükle, sırala
5. Public API'den doğrula:
   `curl localhost:8080/api/products/51-gul-buket | python3 -m json.tool`
   → görseller görünmeli, `is_active` GÖRÜNMEMELİ
6. Ürünü pasif yap → public listede kaybolmalı:
   `curl localhost:8080/api/products` → boş dizi
7. Çıkış yap → login'e dönmeli

- [x] **Step 5: Commit**

```bash
git add -A frontend/idare
git commit -m "feat: sipariş placeholder (Faz 2) ve son temizlik"
```

---

## Plan 4 Bitiş Kriterleri

- [x] `pnpm build` hatasız
- [x] Giriş HttpOnly cookie ile çalışıyor; localStorage'da token YOK
- [x] Sayfa yenilenince oturum korunuyor
- [x] Kategori: iki eksen ayrı, silme uyarısı ürün sayısını gösteriyor
- [x] Pasif kategoride `is_featured` kilitli
- [x] Ürün CRUD çalışıyor, fiyat string olarak gidiyor
- [x] Kategori seçimi iki grupta
- [x] Görsel yükleme çalışıyor, JPEG/PNG dışı reddediliyor
- [x] Görsel sıralama çalışıyor, kapak değişiyor
- [x] Ürün silinince görselleri de gidiyor
- [x] `/siparisler` placeholder'ı var
- [x] Kullanılmayan bağımlılıklar kaldırıldı

**Sonraki:** Plan 3 — Nuxt public site. Admin panel bittikten sonra yazılacak;
o zaman API'nin frontend'den nasıl tüketildiği de görülmüş olur.

---

## Uygulama Notu (2026-07-16)

Tüm kriterler gerçek sunucu + gerçek tarayıcıya (Playwright/Chromium)
karşı doğrulandı. Backend testleri: 179 geçti, 0 başarısız.

**Plandan sapmalar ve nedenleri:**

1. **Task 1-3 tek commit.** Plan üçüne ayrı commit istiyordu ama
   `JwtService` silinince `ApiService`, `store/user` ve router guard'ın
   üçü birden kırılıyor — build ancak hepsi yazılınca yeşile dönüyor.
   Yeşil olmayan ara commit atmak yerine birleştirildi.

2. **`ErrorPopup({ message })` → `ErrorPopup(message)`.** Plandaki
   çağrı yanlıştı: `Popup.ts`'de imza `ErrorPopup(text: string)`.
   Nesne geçilse kullanıcı `[object Object]` görecekti.

3. **`make seed` kullanılamadı.** `term.ReadPassword` gerçek TTY istiyor,
   script'ten beslenemiyor. Aynı `auth.CreateAdmin` fonksiyonunu çağıran
   geçici bir program yazıldı, sonra silindi. (Kalıcı bir
   `--username/--password` bayrağı cmd/seed'e eklenebilir — Faz 2.)

4. **`casl.ts` silinmedi, stub'a indirildi.** `@layouts`'taki beş
   navigasyon bileşeni `can`/`canViewNavMenuGroup` import ediyor. Plan
   `@layouts`'a dokunmamayı söylüyordu; dosyayı her şeye izin veren
   stub'a çevirmek hem paketi kaldırdı hem o beş dosyaya dokunmadı.

5. **Plan dışı ek temizlik:** 16 demo dialog + AppPricing +
   AppSearchHeader (kapalı ada, dışarıdan referans yok), `model/table.ts`
   ve `utils/ExDate.ts` (öksüz), chartjs/tiptap/apex bileşenleri ve
   orphan SCSS'ler. Ana bundle 406.66 kB → 334.46 kB (gzip 139.78 →
   120.37), dört ekran eklenmesine rağmen ~%18 küçüldü.

6. **Plan dışı düzeltmeler:** Vuetify `locale: 'tr'` (tablo altbilgisi
   İngilizceydi), uygulama başlığı `go-template2` → `çiçekçi`.

**Yakalanan gerçek hata:** Ürün oluşturulduktan sonra `router.replace`
ile düzenleme moduna geçilirken bileşen yeniden kurulmadığı için
`onMounted` tekrar çalışmıyor, `product` null kalıyor ve görsel bölümü
boş görünüyordu — esnaf ürünü kaydedip fotoğraf ekleyemiyor, sayfayı
yenilemek zorunda kalıyordu. `watch(rawId, loadProduct)` ile çözüldü.
Bu, planın "en önemli akış" dediği adımdı ve ancak tarayıcıda ortaya
çıktı; curl ile görünmezdi.

**Elle (gözle) bakılmadı, otomasyonla doğrulandı:** cookie `HttpOnly`
bayrağı Set-Cookie başlığından ve `document.cookie === ''` ile;
localStorage'ın token içermediği `Object.entries(localStorage)` ile.
Ekran görüntüleri alındı ve incelendi (kapak rozeti, pasif satır,
iki eksenli tablolar).
