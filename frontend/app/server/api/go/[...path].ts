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

  // Gelen isteğin method'unu ve (varsa) body'sini backend'e ilet. Bu olmadan
  // $fetch her isteği GET olarak gönderirdi: sipariş oluşturma (POST /orders)
  // backend'e GET olarak ulaşıp 405 alır, ödeme akışı hiç başlamazdı.
  const method = event.method

  // GET/HEAD dışındaki metodlarda body oku (POST sipariş gövdesi vb.).
  let body: unknown
  if (method !== 'GET' && method !== 'HEAD')
    body = await readBody(event)

  // Cookie plumbing: tarayıcı ↔ Nitro ↔ Go arasında ayrı iki HTTP bağlantısı
  // var — $fetch bunları otomatik köprülemez. Gelen Cookie header'ı Go'ya
  // ELLE iletilmezse ve Go'nun Set-Cookie'si tarayıcıya ELLE aktarılmazsa
  // customer_token hiç oturmaz: register/login sonrası /customer/me hep
  // 401 döner, hesap sayfaları /giris'e döner, sipariş customer_id'si NULL
  // kalır (misafir davranışı — giriş yapılmış olsa bile).
  const cookie = getRequestHeader(event, 'cookie')

  try {
    const res = await $fetch.raw(`${cfg.goApiBase}/${path}`, {
      query,
      method,
      body,
      headers: cookie ? { cookie } : undefined,
    })

    // Go'nun Set-Cookie'lerini (birden fazla olabilir — her biri AYRI
    // appendResponseHeader çağrısıyla, birleştirilmeden) tarayıcıya aktar.
    for (const setCookie of res.headers.getSetCookie?.() ?? [])
      appendResponseHeader(event, 'set-cookie', setCookie)

    return res._data
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
