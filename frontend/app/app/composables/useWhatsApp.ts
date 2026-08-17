import type { Product } from '~/types/api'
import { buildOutOfStockUrl, buildWhatsAppUrl } from '~/utils/whatsapp'

/**
 * Ürün için WhatsApp linki. Mesaj kurma mantığı utils/whatsapp.ts'te —
 * Nuxt'tan bağımsız olduğu için test edilebiliyor.
 *
 * outOfStock true ise "ne zaman gelir" mesajı kurulur: tükenen ürün sitede
 * görünür kalıyor ve müşteri satın alamadığı için sipariş mesajı anlamsız
 * olurdu (spec §6.1).
 */
export function useWhatsAppLink(
  product: MaybeRefOrGetter<Product>,
  outOfStock: MaybeRefOrGetter<boolean> = false,
) {
  const { public: cfg } = useRuntimeConfig()

  return computed(() => {
    const p = toValue(product)

    return toValue(outOfStock)
      ? buildOutOfStockUrl(cfg.whatsappNumber, p, cfg.siteUrl)
      : buildWhatsAppUrl(cfg.whatsappNumber, p, cfg.siteUrl)
  })
}

/** Ürünsüz genel iletişim linki (iletişim sayfası için). */
export function useGeneralWhatsAppLink(message = 'Merhaba, bilgi almak istiyorum.') {
  const { public: cfg } = useRuntimeConfig()

  return `https://wa.me/${cfg.whatsappNumber}?text=${encodeURIComponent(message)}`
}
