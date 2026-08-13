import { describe, expect, it } from 'vitest'
import { telefonHatasi, telefonNormalize } from './telefon'

// Bu kurallar backend'deki phone_test.go ile AYNI olmalı — ikisi ayrışırsa
// kullanıcı formda yeşil ışık alıp sunucudan hata yer.
describe('telefonNormalize', () => {
  it('farklı yazımları tek biçime indirger', () => {
    const beklenen = '5551112233'
    for (const girdi of [
      '5551112233',
      '05551112233',
      '905551112233',
      '+905551112233',
      '0555 111 22 33',
      '+90 555 111 22 33',
      '(555) 111-2233',
      '555.111.2233',
      '  5551112233  ',
    ])
      expect(telefonNormalize(girdi), girdi).toBe(beklenen)
  })

  it('geçersiz numaraları reddeder', () => {
    for (const girdi of [
      'asdasd',
      '555111223a',
      'abc5551112233',
      '',
      '   ',
      '555111223', // 9 hane
      '55511122334', // 11 hane
      '2121112233', // sabit hat
      '4441112233', // kurumsal
      '5551112233@',
    ])
      expect(telefonNormalize(girdi), girdi).toBeNull()
  })
})

describe('telefonHatasi', () => {
  it('geçerli numarada boş mesaj döner', () => {
    expect(telefonHatasi('0555 111 22 33')).toBe('')
  })

  it('boş girdide "gerekli" der', () => {
    expect(telefonHatasi('')).toBe('Telefon gerekli.')
    expect(telefonHatasi('   ')).toBe('Telefon gerekli.')
  })

  it('geçersiz girdide örnekli mesaj döner', () => {
    expect(telefonHatasi('asdasd')).toContain('Geçerli bir cep telefonu')
  })
})
