import { describe, expect, it } from 'vitest'
import { kategoriGorselIndex } from './kategoriGorsel'

describe('kategoriGorselIndex', () => {
  it('kategori adındaki anahtar kelimeye göre eşler', () => {
    expect(kategoriGorselIndex('anneler-gunu', 'Anneler Günü')).toBe(0)
    expect(kategoriGorselIndex('sevgiliye', 'Sevgiliye Çiçek')).toBe(1)
    expect(kategoriGorselIndex('dugun', 'Düğün / Açılış')).toBe(2)
    expect(kategoriGorselIndex('soz', 'Söz / Kız İsteme')).toBe(3)
  })

  it('esnaf farklı isim verse de yakalar', () => {
    // Kategori adları admin panelden serbest giriliyor — sabit slug'a
    // güvenemeyiz (spec §4.2).
    expect(kategoriGorselIndex('x', 'Anne Çiçekleri')).toBe(0)
    expect(kategoriGorselIndex('x', 'Yıldönümü')).toBe(1)
    expect(kategoriGorselIndex('x', 'Nikah Organizasyon')).toBe(2)
    expect(kategoriGorselIndex('x', 'Nişan')).toBe(3)
  })

  it('ad tutmazsa slug üzerinden eşler', () => {
    expect(kategoriGorselIndex('dugun-acilis', 'Kurumsal')).toBe(2)
  })

  it('hiçbir anahtar tutmazsa sıraya göre dağıtır — kart boş kalmaz', () => {
    expect(kategoriGorselIndex('bilinmeyen', 'Bilinmeyen', 0)).toBe(0)
    expect(kategoriGorselIndex('bilinmeyen', 'Bilinmeyen', 1)).toBe(1)
    expect(kategoriGorselIndex('bilinmeyen', 'Bilinmeyen', 2)).toBe(2)
  })

  it('index havuzdan büyükse başa döner, taşmaz', () => {
    expect(kategoriGorselIndex('x', 'Y', 99, 4)).toBe(3)
    expect(kategoriGorselIndex('x', 'Y', 4, 4)).toBe(0)
  })

  it('havuz küçülürse taşan anahtar eşleşmesi güvenli sıraya düşer', () => {
    // "Söz" normalde index 3 ister; havuzda 2 görsel varsa taşmamalı.
    expect(kategoriGorselIndex('soz', 'Söz', 0, 2)).toBeLessThan(2)
  })
})
