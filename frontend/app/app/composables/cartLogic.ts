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

export function addItem(items: CartItem[], yeni: CartItem): CartItem[] {
  const mevcut = items.find(i => i.product_id === yeni.product_id)
  if (mevcut) {
    return items.map(i =>
      i.product_id === yeni.product_id
        ? { ...i, quantity: i.quantity + yeni.quantity }
        : i)
  }

  return [...items, yeni]
}

export function removeItem(items: CartItem[], productId: number): CartItem[] {
  return items.filter(i => i.product_id !== productId)
}

export function setItemQuantity(items: CartItem[], productId: number, qty: number): CartItem[] {
  if (qty <= 0)
    return removeItem(items, productId)

  return items.map(i => (i.product_id === productId ? { ...i, quantity: qty } : i))
}
