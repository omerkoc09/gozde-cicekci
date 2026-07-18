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
  const rawPath = getRouterParam(event, 'path') || ''

  // Güvenlik: yalnızca public uçlar. admin kontrolü büyük/küçük harfe
  // duyarsız ve baştaki eğik çizgi/segment normalize edilerek yapılıyor —
  // "Admin/me" veya "/admin/..." gibi varyantlar da bloke olsun.
  const path = rawPath.replace(/^\/+/, '')

  if (/^admin(\/|$)/i.test(path))
    throw createError({ statusCode: 404, statusMessage: 'Not found' })

  const query = getQuery(event)

  try {
    return await $fetch(`${cfg.goApiBase}/${path}`, { query })
  }
  catch (err: unknown) {
    const e = err as { status?: number, statusCode?: number, data?: unknown, message?: string }

    // Backend'e ulaşılamazsa (yanlış goApiBase, ağ vb.) hata sessizce boş
    // veriye dönüşmesin — sunucu logunda görünür olsun.
    console.error(`[api/go proxy] ${path} → ${cfg.goApiBase} başarısız:`, e?.message)

    throw createError({
      statusCode: e?.status ?? e?.statusCode ?? 502,
      data: e?.data,
    })
  }
})
