import type { Category, Product } from '~/types/api'

/**
 * Sitemap için ürün ve kategori URL'leri — sadece aktif olanlar (backend zaten
 * pasifleri göstermiyor). Filtre kombinasyonları KASITLI olarak yok: onlar
 * noindex (spec §4.2), sitemap'e girmemeli.
 *
 * Sunucuda çalışır — Go API'yi doğrudan çağırır (proxy'ye gerek yok).
 */
export default defineSitemapEventHandler(async () => {
  const cfg = useRuntimeConfig()
  const base = cfg.goApiBase

  const [products, categories] = await Promise.all([
    $fetch<Product[]>(`${base}/products`, { query: { limit: 100 } }),
    $fetch<Category[]>(`${base}/categories`),
  ])

  return [
    ...products.map(p => ({ loc: `/urun/${p.slug}`, changefreq: 'weekly' as const })),
    ...categories.map(c => ({ loc: `/kategori/${c.slug}`, changefreq: 'weekly' as const })),
  ]
})
