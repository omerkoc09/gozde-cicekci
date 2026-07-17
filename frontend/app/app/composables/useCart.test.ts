import { beforeEach, describe, expect, it } from 'vitest'
import { cartTotal, addItem, removeItem, setItemQuantity } from './cartLogic'
import type { CartItem } from '~/types/api'

function urun(id: number, price: string, qty = 1): CartItem {
  return {
    product_id: id,
    name: `Ürün ${id}`,
    slug: `urun-${id}`,
    price,
    image: '',
    quantity: qty,
  }
}

describe('cartTotal', () => {
  it('kalem fiyatlarını adetle çarpıp toplar', () => {
    const items = [urun(1, '1850.00', 2), urun(2, '500.50', 1)]
    expect(cartTotal(items)).toBe('4200.50')
  })

  it('boş sepette sıfır', () => {
    expect(cartTotal([])).toBe('0.00')
  })

  it('kuruşlu fiyatlarda yuvarlama hatası yapmaz', () => {
    const items = [urun(1, '0.10', 3)]
    expect(cartTotal(items)).toBe('0.30')
  })
})

describe('addItem', () => {
  it('yeni ürünü ekler', () => {
    const out = addItem([], urun(1, '100.00'))
    expect(out).toHaveLength(1)
    expect(out[0]!.quantity).toBe(1)
  })

  it('var olan ürünün adedini artırır, kopya oluşturmaz', () => {
    const out = addItem([urun(1, '100.00', 2)], urun(1, '100.00', 3))
    expect(out).toHaveLength(1)
    expect(out[0]!.quantity).toBe(5)
  })
})

describe('setItemQuantity', () => {
  it('adedi değiştirir', () => {
    const out = setItemQuantity([urun(1, '100.00', 1)], 1, 4)
    expect(out[0]!.quantity).toBe(4)
  })

  it('adet 0 veya altına inerse kalemi siler', () => {
    expect(setItemQuantity([urun(1, '100.00', 1)], 1, 0)).toHaveLength(0)
  })
})

describe('removeItem', () => {
  it('kalemi siler', () => {
    expect(removeItem([urun(1, '100.00'), urun(2, '200.00')], 1)).toHaveLength(1)
  })
})
