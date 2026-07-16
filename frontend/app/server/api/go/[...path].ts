/**
 * Go API'ye proxy: /api/go/* → Go backend /api/*.
 *
 * Tarayıcı bu Nuxt sunucusuna (same-origin) çağrı yapar, Nitro isteği Go
 * backend'ine sunucu-sunucu iletir — CORS'a takılmaz.
 *
 * Neden gerekli: public site SSR-öncelikli ama filtre değişince Nuxt
 * client-side gezinme yapıyor ve useFetch tarayıcıdan çağırıyor. Backend'in
 * CORS'u sadece admin panelin origin'ini (5173) taşıyor. Proxy ile hem SSR
 * hem client çağrıları same-origin gidiyor.
 *
 * Sadece public uçlar — admin uçlarına asla proxy yok.
 *
 * 301 takibi: $fetch varsayılan olarak backend'in eski-slug 301'ini takip
 * eder ve kanonik ürünü (yeni slug'la) döndürür. Sayfa katmanı yanıttaki
 * slug'ı istenen slug'la karşılaştırıp kendi 301'ini yapıyor (urun/[slug].vue).
 */
export default defineEventHandler(async event => {
  const cfg = useRuntimeConfig()
  const path = getRouterParam(event, 'path') || ''

  if (path.startsWith('admin'))
    throw createError({ statusCode: 404, statusMessage: 'Not found' })

  const query = getQuery(event)

  try {
    return await $fetch(`${cfg.goApiBase}/${path}`, { query })
  }
  catch (err: unknown) {
    const e = err as { status?: number, statusCode?: number, data?: unknown }

    throw createError({
      statusCode: e?.status ?? e?.statusCode ?? 502,
      data: e?.data,
    })
  }
})
