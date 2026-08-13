import { describe, expect, it } from 'vitest'
import { apiErrorMessage, apiErrorStatus } from './useOrders'

// Hata gövdesi kullanıcıya gösterilecek mesajın TEK kaynağı. Nitro proxy'si
// hatayı createError({ data }) ile yeniden fırlattığı için gövde bir seviye
// derine iniyor; yalnızca düz şekle bakan bir çözümleyici HER hatayı
// "Bir şeyler ters gitti"ye düşürür ve kullanıcı yanlış şifre mi girdiğini
// asla öğrenemez. Bu testler o regresyonu kilitliyor.
describe('apiErrorMessage', () => {
  it('düz gövdeden mesajı çıkarır (Go\'ya doğrudan istek)', () => {
    const e = { data: { error: { code: 'unauthorized', message: 'Yetkisiz' } } }
    expect(apiErrorMessage(e)).toBe('Yetkisiz')
  })

  it('proxy\'nin sardığı gövdeden mesajı çıkarır (data.data.error)', () => {
    const e = {
      statusCode: 409,
      data: { data: { error: { code: 'conflict', message: 'bu e-posta ile hesap var, giriş yapın' } } },
    }
    expect(apiErrorMessage(e)).toBe('bu e-posta ile hesap var, giriş yapın')
  })

  it('mesaj yoksa genel mesaja düşer (ağ hatası vb.)', () => {
    expect(apiErrorMessage(new Error('network'))).toBe('Bir şeyler ters gitti. Lütfen tekrar deneyin.')
    expect(apiErrorMessage(undefined)).toBe('Bir şeyler ters gitti. Lütfen tekrar deneyin.')
  })

  it('"geçersiz girdi:" önekini kırpar (kullanıcıya gürültü)', () => {
    const e = { data: { error: { message: 'geçersiz girdi: şifre en az 8 karakter olmalı' } } }
    expect(apiErrorMessage(e)).toBe('şifre en az 8 karakter olmalı')
  })

  it('düz şekil, sarılmış şekle göre önceliklidir', () => {
    const e = {
      data: {
        error: { message: 'dıştaki' },
        data: { error: { message: 'içteki' } },
      },
    }
    expect(apiErrorMessage(e)).toBe('dıştaki')
  })
})

describe('apiErrorStatus', () => {
  it('statusCode alanını okur', () => {
    expect(apiErrorStatus({ statusCode: 401 })).toBe(401)
  })

  it('status alanına düşer', () => {
    expect(apiErrorStatus({ status: 409 })).toBe(409)
  })

  it('durum kodu yoksa undefined döner', () => {
    expect(apiErrorStatus(new Error('network'))).toBeUndefined()
  })
})
