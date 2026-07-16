import type { Category, Product } from '~/types/api'

/**
 * Public API istemcisi.
 *
 * Auth YOK — public site hiçbir korumalı uca dokunmaz (/api/admin/* çağrılmaz).
 * SSR'da bu çağrılar sunucudan yapılır, CORS'a takılmaz; backend'in CORS ayarı
 * admin panelin origin'ini (5173) taşıyor.
 *
 * useFetch kullanılıyor, $fetch değil: useFetch SSR'da çektiği veriyi
 * hydration'a taşır, aynı veri tarayıcıda ikinci kez çekilmez.
 */
function apiBase(): string {
  return useRuntimeConfig().public.apiBase
}

/** Backend hata formatı: {"error": {"code": "...", "message": "..."}} */
export function apiErrorMessage(err: unknown): string {
  const e = err as { data?: { error?: { message?: string } }, statusCode?: number }

  if (e?.data?.error?.message)
    return e.data.error.message

  if (e?.statusCode === 404)
    return 'Ürün bulunamadı'

  return 'Bir şeyler ters gitti, lütfen tekrar deneyin'
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

export function useFeaturedCategories() {
  return useFetch<Category[]>(() => `${apiBase()}/categories/featured`, {
    key: 'categories-featured',
    default: () => [],
  })
}

export function useCategory(slug: string) {
  return useFetch<Category>(`${apiBase()}/categories/${slug}`, {
    key: `category-${slug}`,
  })
}
