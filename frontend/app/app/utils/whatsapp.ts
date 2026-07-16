import type { Product } from '~/types/api'
import { whatsappPrice } from './price'

/**
 * WhatsApp sipariş mesajı (spec §5.3).
 *
 * Mesaj "bilgi almak istiyorum" değil "sipariş etmek istiyorum" diyor —
 * müşteri kendi ağzından niyetini kuruyor, konuşma pazarlıkla değil
 * siparişle başlıyor. Fiyat mesajda: esnafı koruyor (müşteri hangi fiyatı
 * gördüğünü belgeliyor) ve esnaf ürünü sitede aramak zorunda kalmıyor.
 *
 * Teslimat/tarih/kart mesajı alanları KASITLI olarak yok — Faz 2'nin işi.
 * Şablona form gibi alanlar koymak müşteriye ödev listesi yaratır.
 *
 * Nuxt'tan bağımsız tutuldu ki test edilebilsin; composable sarmalıyor.
 */
export function buildOrderMessage(product: Product, siteUrl: string): string {
  return [
    'Merhaba, bu ürünü sipariş etmek istiyorum:',
    `${product.name} — ${whatsappPrice(product.price)}`,
    `${siteUrl}/urun/${product.slug}`,
  ].join('\n')
}

/** wa.me linki. encodeURIComponent Türkçe karakterleri ve satır başlarını güvenli kodlar. */
export function buildWhatsAppUrl(
  phoneNumber: string,
  product: Product,
  siteUrl: string,
): string {
  const message = buildOrderMessage(product, siteUrl)

  return `https://wa.me/${phoneNumber}?text=${encodeURIComponent(message)}`
}
