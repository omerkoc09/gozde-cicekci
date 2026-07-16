import type { Product } from '~/types/api'
import { buildWhatsAppUrl } from '~/utils/whatsapp'

/**
 * Ürün için WhatsApp sipariş linki. Mesaj kurma mantığı utils/whatsapp.ts'te —
 * Nuxt'tan bağımsız olduğu için test edilebiliyor.
 */
export function useWhatsAppLink(product: MaybeRefOrGetter<Product>) {
  const { public: cfg } = useRuntimeConfig()

  return computed(() =>
    buildWhatsAppUrl(cfg.whatsappNumber, toValue(product), cfg.siteUrl))
}

/** Ürünsüz genel iletişim linki (iletişim sayfası için). */
export function useGeneralWhatsAppLink(message = 'Merhaba, bilgi almak istiyorum.') {
  const { public: cfg } = useRuntimeConfig()

  return `https://wa.me/${cfg.whatsappNumber}?text=${encodeURIComponent(message)}`
}
