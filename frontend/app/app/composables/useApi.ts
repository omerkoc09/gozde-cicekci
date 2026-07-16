import type { Axis, Category, Product, Slide } from '~/types/api'

/**
 * Public API istemcisi.
 *
 * Auth YOK — public site hiçbir korumalı uca dokunmaz (/api/admin/* çağrılmaz).
 * Çağrılar same-origin Nitro proxy'sine (/api/go) gider, CORS'a hiç takılmaz.
 * (Backend'in CORS ayarı yalnızca admin panelini ilgilendirir: dev'de :5173
 * ayrı origin'den istek atar, prod'da aynı origin'dedir.)
 *
 * useFetch kullanılıyor, $fetch değil: useFetch SSR'da çektiği veriyi
 * hydration'a taşır, aynı veri tarayıcıda ikinci kez çekilmez.
 */
function apiBase(): string {
  return useRuntimeConfig().public.apiBase
}

/**
 * Ürün listesi. İki eksen birlikte verilirse backend AND uyguluyor —
 * ikisine de uyan ürünler döner (spec §5.6).
 *
 * Filtreler reactive: query değişince useFetch otomatik yeniden çeker
 * (url bir fonksiyon olduğu için Nuxt bağımlılıkları izliyor).
 */
export function useProductList(query: {
  amac?: MaybeRefOrGetter<string | undefined>
  tip?: MaybeRefOrGetter<string | undefined>
  /** true → yalnızca panelden öne çıkarılmış ürünler (ana sayfa vitrini). */
  oneCikan?: boolean
  limit?: number
}) {
  const url = computed(() => {
    const params = new URLSearchParams()
    const amac = toValue(query.amac)
    const tip = toValue(query.tip)

    if (amac)
      params.set('amac', amac)

    if (tip)
      params.set('tip', tip)

    if (query.oneCikan)
      params.set('one_cikan', 'true')

    params.set('limit', String(query.limit ?? 24))

    return `${apiBase()}/products?${params}`
  })

  // url reactive; değişince useFetch yeniden çeker. watch açıkça url'i
  // izliyor ki client tarafında filtre değişince liste yenilensin.
  return useFetch<Product[]>(url, {
    key: () => `products-${url.value}`,
    watch: [url],
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
  return useFetch<Category[]>(() => `${apiBase()}/categories`, {
    key: 'categories',
    default: () => [],
  })
}

/**
 * Öne çıkan kategoriler. Ana sayfa iki bölümü ayrı çekiyor:
 * "Özel Günler" (occasion) ve "Çiçek Türlerine Göre" (type).
 *
 * key eksene göre ayrılıyor — sabit olsaydı iki bölüm aynı önbelleği
 * paylaşır, ikincisi birincinin verisini gösterirdi.
 */
export function useFeaturedCategories(axis?: Axis) {
  return useFetch<Category[]>(
    () => `${apiBase()}/categories/featured${axis ? `?axis=${axis}` : ''}`,
    {
      key: `categories-featured-${axis ?? 'all'}`,
      default: () => [],
    },
  )
}

export function useCategory(slug: string) {
  return useFetch<Category>(`${apiBase()}/categories/${slug}`, {
    key: `category-${slug}`,
  })
}

/**
 * Ana sayfa slider'ı — yalnızca aktif slaytlar, sıraya dizili gelir.
 * Boş dönerse ana sayfa statik hero'ya düşer (bkz. pages/index.vue).
 */
export function useSlides() {
  return useFetch<Slide[]>(() => `${apiBase()}/slides`, {
    key: 'slides',
    default: () => [],
  })
}
