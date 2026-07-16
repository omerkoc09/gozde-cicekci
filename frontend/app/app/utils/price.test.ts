import { describe, expect, it } from 'vitest'
import { formatPrice, whatsappPrice } from './price'

describe('formatPrice', () => {
  it('binlik ayracı koyar ve ₺ ekler', () => {
    expect(formatPrice('1850.00')).toBe('1.850 ₺')
  })

  it('kuruş varsa gösterir', () => {
    expect(formatPrice('1850.50')).toBe('1.850,50 ₺')
  })

  it('kuruş sıfırsa göstermez — esnaf 1.850 ₺ yazar, 1.850,00 ₺ değil', () => {
    expect(formatPrice('500.00')).toBe('500 ₺')
  })

  it('binlik altı', () => {
    expect(formatPrice('99.00')).toBe('99 ₺')
  })

  it('geçersiz girdide boş döner, patlamaz', () => {
    expect(formatPrice('abc')).toBe('')
    expect(formatPrice('')).toBe('')
  })
})

describe('whatsappPrice', () => {
  it('mesajda kullanılan sade format', () => {
    expect(whatsappPrice('1850.00')).toBe('1.850 ₺')
  })
})
