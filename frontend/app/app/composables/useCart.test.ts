import { beforeEach, describe, expect, it } from 'vitest'
import { cartLineKey, cartTotal, addItem, removeItem, setItemQuantity } from './cartLogic'
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
    const out = setItemQuantity([urun(1, '100.00', 1)], cartLineKey(urun(1, '100.00', 1)), 4)
    expect(out[0]!.quantity).toBe(4)
  })

  it('adet 0 veya altına inerse kalemi siler', () => {
    expect(setItemQuantity([urun(1, '100.00', 1)], cartLineKey(urun(1, '100.00', 1)), 0)).toHaveLength(0)
  })
})

describe('removeItem', () => {
  it('kalemi siler', () => {
    const u1 = urun(1, '100.00')
    const u2 = urun(2, '200.00')
    expect(removeItem([u1, u2], cartLineKey(u1))).toHaveLength(1)
  })
})

// --- Task 9: cartLineKey testleri ---

const secim = (id: number, ad: string) => ({
  value_id: id, group_name: 'Ambalaj', value_name: ad, swatch_hex: '#FFFFFF',
})

describe('cartLineKey', () => {
  it('seçim sırası anahtarı değiştirmez', () => {
    const a = { ...urun(1, '100.00'), options: [secim(3, 'Pembe'), secim(7, 'Beyaz')] }
    const b = { ...urun(1, '100.00'), options: [secim(7, 'Beyaz'), secim(3, 'Pembe')] }

    expect(cartLineKey(a)).toBe(cartLineKey(b))
  })

  it('seçimsiz kalem (eski sepet) patlamaz', () => {
    const eski = urun(1, '100.00') // options alanı YOK

    expect(cartLineKey(eski)).toBe('1:')
  })

  it('farklı seçim farklı anahtar üretir', () => {
    const pembe = { ...urun(1, '100.00'), options: [secim(3, 'Pembe')] }
    const beyaz = { ...urun(1, '100.00'), options: [secim(7, 'Beyaz')] }

    expect(cartLineKey(pembe)).not.toBe(cartLineKey(beyaz))
  })
})

describe('addItem — seçimli', () => {
  it('aynı ürün farklı seçim AYRI kalem olur', () => {
    const pembe = { ...urun(1, '100.00'), options: [secim(3, 'Pembe')] }
    const beyaz = { ...urun(1, '100.00'), options: [secim(7, 'Beyaz')] }

    const out = addItem(addItem([], pembe), beyaz)

    expect(out).toHaveLength(2)
  })

  it('aynı ürün aynı seçim TEK kalemde birleşir', () => {
    const pembe = { ...urun(1, '100.00', 1), options: [secim(3, 'Pembe')] }

    const out = addItem(addItem([], pembe), { ...pembe, quantity: 2 })

    expect(out).toHaveLength(1)
    expect(out[0]!.quantity).toBe(3)
  })

  it('seçimsiz eski kalemler eskisi gibi birleşir', () => {
    const out = addItem([urun(1, '100.00', 2)], urun(1, '100.00', 3))

    expect(out).toHaveLength(1)
    expect(out[0]!.quantity).toBe(5)
  })
})

describe('removeItem / setItemQuantity — seçimli', () => {
  it('yalnızca hedef satırı siler, aynı ürünün diğer rengine dokunmaz', () => {
    const pembe = { ...urun(1, '100.00'), options: [secim(3, 'Pembe')] }
    const beyaz = { ...urun(1, '100.00'), options: [secim(7, 'Beyaz')] }
    const sepet = addItem(addItem([], pembe), beyaz)

    const out = removeItem(sepet, cartLineKey(pembe))

    expect(out).toHaveLength(1)
    expect(out[0]!.options![0]!.value_name).toBe('Beyaz')
  })

  it('adet değişimi yalnızca hedef satıra uygulanır', () => {
    const pembe = { ...urun(1, '100.00', 1), options: [secim(3, 'Pembe')] }
    const beyaz = { ...urun(1, '100.00', 1), options: [secim(7, 'Beyaz')] }
    const sepet = addItem(addItem([], pembe), beyaz)

    const out = setItemQuantity(sepet, cartLineKey(pembe), 5)

    expect(out.find(i => i.options![0]!.value_name === 'Pembe')!.quantity).toBe(5)
    expect(out.find(i => i.options![0]!.value_name === 'Beyaz')!.quantity).toBe(1)
  })
})
