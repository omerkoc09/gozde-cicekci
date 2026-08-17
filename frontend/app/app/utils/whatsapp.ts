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

/**
 * Tükenen ürün için WhatsApp mesajı (spec §6.1).
 *
 * Ürün sitede görünür kalıyor — müşteri "ne zaman gelir" diye sorabilsin,
 * satış fırsatı kaybolmasın. Sipariş mesajının aksine FİYAT YOK: müşteri
 * satın alamadığı bir ürünün fiyatını konuşmaya davet edilmemeli.
 */
export function buildOutOfStockMessage(product: Product, siteUrl: string): string {
  return [
    'Merhaba, bu ürün tükenmiş görünüyor:',
    product.name,
    `${siteUrl}/urun/${product.slug}`,
    'Ne zaman tekrar gelir?',
  ].join('\n')
}

/** Tükenen ürün için wa.me linki. */
export function buildOutOfStockUrl(
  phoneNumber: string,
  product: Product,
  siteUrl: string,
): string {
  const message = buildOutOfStockMessage(product, siteUrl)

  return `https://wa.me/${phoneNumber}?text=${encodeURIComponent(message)}`
}
