import { describe, expect, it } from 'vitest'
import { donusYolunuCoz } from './authRedirect'

describe('donusYolunuCoz', () => {
  it('site içi yolu olduğu gibi döner', () => {
    expect(donusYolunuCoz('/siparis')).toBe('/siparis')
    expect(donusYolunuCoz('/hesabim/hesap-detaylari')).toBe('/hesabim/hesap-detaylari')
  })

  it('sorgu parametresi taşıyan yolu korur', () => {
    expect(donusYolunuCoz('/urunler?kategori=gul')).toBe('/urunler?kategori=gul')
  })

  // Açık yönlendirme (open redirect) koruması: bunlar giriş sonrası
  // kullanıcıyı site DIŞINA atabilirdi.
  it('protokol-rölatif URL\'i reddeder', () => {
    expect(donusYolunuCoz('//kotusite.com')).toBe('/hesabim')
  })

  it('mutlak URL\'i reddeder', () => {
    expect(donusYolunuCoz('https://kotusite.com')).toBe('/hesabim')
    expect(donusYolunuCoz('http://kotusite.com')).toBe('/hesabim')
  })

  it('"/" ile başlamayan değeri reddeder', () => {
    expect(donusYolunuCoz('siparis')).toBe('/hesabim')
  })

  it('eksik veya string olmayan değerde varsayılana düşer', () => {
    expect(donusYolunuCoz(undefined)).toBe('/hesabim')
    expect(donusYolunuCoz(null)).toBe('/hesabim')
    // Nuxt aynı parametre iki kez verilirse dizi döndürür.
    expect(donusYolunuCoz(['/a', '/b'])).toBe('/hesabim')
  })

  it('varsayılan değer değiştirilebilir', () => {
    expect(donusYolunuCoz(undefined, '/')).toBe('/')
  })
})
