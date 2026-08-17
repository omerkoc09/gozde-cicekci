import { describe, expect, it } from 'vitest'
import {
  buildOrderMessage,
  buildOutOfStockMessage,
  buildOutOfStockUrl,
  buildWhatsAppUrl,
} from './whatsapp'

const urun = {
  id: 1,
  name: '51 Gül Buket',
  slug: '51-gul-buket',
  description: '',
  price: '1850.00',
  old_price: null,
  in_stock: true,
  stock_quantity: null,
  discount_remaining: null,
  category_ids: [],
  images: [],
}

describe('buildOrderMessage', () => {
  it('sipariş niyetli, fiyatlı ve linkli mesaj kurar (spec §5.3)', () => {
    expect(buildOrderMessage(urun, 'https://cicekci.com')).toBe(
      'Merhaba, bu ürünü sipariş etmek istiyorum:\n'
      + '51 Gül Buket — 1.850 ₺\n'
      + 'https://cicekci.com/urun/51-gul-buket',
    )
  })

  it('mesaj "bilgi almak" değil "sipariş etmek" diyor — niyet müşterinin ağzından', () => {
    expect(buildOrderMessage(urun, 'https://x.com')).toContain('sipariş etmek istiyorum')
  })
})

describe('buildWhatsAppUrl', () => {
  it('wa.me linki üretir, numara ülke kodlu', () => {
    const url = buildWhatsAppUrl('905551234567', urun, 'https://cicekci.com')

    expect(url.startsWith('https://wa.me/905551234567?text=')).toBe(true)
  })

  it('Türkçe karakterler ve satır başları bozulmadan kodlanıyor', () => {
    const url = buildWhatsAppUrl('905551234567', urun, 'https://cicekci.com')
    const text = new URL(url).searchParams.get('text')

    // URL'den geri okunduğunda mesaj birebir aynı olmalı
    expect(text).toBe(buildOrderMessage(urun, 'https://cicekci.com'))
    // Satır başları korunmuş
    expect(text).toContain('\n')
  })

  it('ham Türkçe karakter URL\'de kodlanmış halde durur', () => {
    const url = buildWhatsAppUrl('905551234567', urun, 'https://cicekci.com')

    // Query string'de çiğ "ü" olmamalı — %C3%BC olarak kodlanmalı
    const query = url.split('?text=')[1]!

    expect(query).not.toContain('ü')
    expect(query).toContain('%0A') // satır başı
  })
})

describe('buildOutOfStockMessage', () => {
  it('tükenen ürün için ne zaman geleceğini sorar (spec §6.1)', () => {
    const msg = buildOutOfStockMessage(urun, 'https://cicekci.com')

    expect(msg).toContain('51 Gül Buket')
    expect(msg).toContain('tükenmiş')
    expect(msg).toContain('https://cicekci.com/urun/51-gul-buket')
  })

  // Tükenen üründe fiyat yazmak anlamsız: müşteri satın alamıyor, pazarlığa
  // davet etmiş oluruz. Sipariş mesajından ayrıldığı nokta bu.
  it('fiyat İÇERMEZ', () => {
    const msg = buildOutOfStockMessage(urun, 'https://cicekci.com')

    expect(msg).not.toContain('1.850')
  })

  it('wa.me linki Türkçe karakterleri kodlar', () => {
    const url = buildOutOfStockUrl('905551234567', urun, 'https://cicekci.com')
    const text = new URL(url).searchParams.get('text')

    expect(url).toContain('https://wa.me/905551234567')
    expect(text).toBe(buildOutOfStockMessage(urun, 'https://cicekci.com'))
  })
})
