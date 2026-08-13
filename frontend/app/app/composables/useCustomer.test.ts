import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// useCustomer.ts, Nuxt'ın auto-import ettiği $fetch ve apiBase()'i kullanır.
// Bu dosya düz node ortamında (vitest.config.ts) çalıştığından ikisi de
// global olarak stub'lanıyor — useCart.test.ts pure-fonksiyon testi yazdığı
// için bu ihtiyacı hiç görmedi, burada composable ağ çağrısı yaptığı için
// $fetch'i taklit etmek gerekiyor.
const fetchMock = vi.fn()

beforeEach(() => {
  vi.stubGlobal('$fetch', fetchMock)
  vi.stubGlobal('apiBase', () => '/api/go')
})

afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

describe('useCustomer', () => {
  it('me() girişliyse müşteri döner', async () => {
    const { useCustomer } = await import('./useCustomer')
    const musteri = { id: 1, email: 'a@b.com', name: 'Ali Veli', phone: '05551112233' }
    fetchMock.mockResolvedValueOnce(musteri)

    const sonuc = await useCustomer().me()

    expect(sonuc).toEqual(musteri)
    expect(fetchMock).toHaveBeenCalledWith('/api/go/customer/me')
  })

  it('me() 401 atarsa null döner (hata fırlatmaz)', async () => {
    const { useCustomer } = await import('./useCustomer')
    fetchMock.mockRejectedValueOnce({ status: 401, data: { error: { code: 'unauthorized' } } })

    const sonuc = await useCustomer().me()

    expect(sonuc).toBeNull()
  })

  it('login() body ve yol doğru gönderir', async () => {
    const { useCustomer } = await import('./useCustomer')
    fetchMock.mockResolvedValueOnce({ ok: true })

    await useCustomer().login({ email: 'a@b.com', password: 'sifre123' })

    expect(fetchMock).toHaveBeenCalledWith('/api/go/customer/login', {
      method: 'POST',
      body: { email: 'a@b.com', password: 'sifre123' },
    })
  })

  it('register() body ve yol doğru gönderir', async () => {
    const { useCustomer } = await import('./useCustomer')
    const musteri = { id: 2, email: 'a@b.com', name: 'Ayşe', phone: '05551112233' }
    fetchMock.mockResolvedValueOnce(musteri)

    const sonuc = await useCustomer().register({ email: 'a@b.com', password: 'sifre123', name: 'Ayşe', phone: '05551112233' })

    expect(sonuc).toEqual(musteri)
    expect(fetchMock).toHaveBeenCalledWith('/api/go/customer/register', {
      method: 'POST',
      body: { email: 'a@b.com', password: 'sifre123', name: 'Ayşe', phone: '05551112233' },
    })
  })

  it('myOrders() boş liste [] döner', async () => {
    const { useCustomer } = await import('./useCustomer')
    fetchMock.mockResolvedValueOnce([])

    const sonuc = await useCustomer().myOrders()

    expect(sonuc).toEqual([])
  })

  it('updateProfile() PATCH ile gönderir', async () => {
    const { useCustomer } = await import('./useCustomer')
    const musteri = { id: 1, email: 'a@b.com', name: 'Yeni Ad', phone: '05551112233' }
    fetchMock.mockResolvedValueOnce(musteri)

    const sonuc = await useCustomer().updateProfile({ name: 'Yeni Ad', phone: '05551112233' })

    expect(sonuc).toEqual(musteri)
    expect(fetchMock).toHaveBeenCalledWith('/api/go/customer/me', {
      method: 'PATCH',
      body: { name: 'Yeni Ad', phone: '05551112233' },
    })
  })
})
