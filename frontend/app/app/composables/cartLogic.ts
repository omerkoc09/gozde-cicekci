import type { CartItem } from '~/types/api'

/**
 * Sepet mantığı — Nuxt'tan bağımsız saf fonksiyonlar, test edilebilir.
 * useCart bunları sarmalayıp localStorage'a bağlar.
 *
 * Fiyat string olarak geliyor ("1850.00") — float precision sorunu olmasın
 * diye kuruş cinsinden integer'a çevrilip hesaplanıyor.
 */

/** "1850.50" → 185050 (kuruş). Float toplamada 0.1+0.2=0.30000000000000004 olur. */
function toKurus(price: string): number {
  return Math.round(Number.parseFloat(price) * 100)
}

export function cartTotal(items: CartItem[]): string {
  const kurus = items.reduce((sum, i) => sum + toKurus(i.price) * i.quantity, 0)

  return (kurus / 100).toFixed(2)
}

/**
 * Sepet satırının kimliği: ürün + seçilen değerler.
 *
 * Seçimler SIRALANIR — müşteri renkleri hangi sırayla seçerse seçsin aynı
 * satır olmalı. Bu alandan önce kurulmuş sepetlerde options undefined
 * gelir; boş dizi kabul edilir, sepet sıfırlanmaz.
 */
export function cartLineKey(item: CartItem): string {
  const ids = (item.options ?? []).map(o => o.value_id).sort((a, b) => a - b)

  return `${item.product_id}:${ids.join(',')}`
}

export function addItem(items: CartItem[], yeni: CartItem): CartItem[] {
  const key = cartLineKey(yeni)
  const mevcut = items.find(i => cartLineKey(i) === key)
  if (mevcut) {
    return items.map(i =>
      cartLineKey(i) === key
        ? { ...i, quantity: i.quantity + yeni.quantity }
        : i)
  }

  return [...items, yeni]
}

export function removeItem(items: CartItem[], lineKey: string): CartItem[] {
  return items.filter(i => cartLineKey(i) !== lineKey)
}

export function setItemQuantity(items: CartItem[], lineKey: string, qty: number): CartItem[] {
  if (qty <= 0)
    return removeItem(items, lineKey)

  return items.map(i => (cartLineKey(i) === lineKey ? { ...i, quantity: qty } : i))
}
