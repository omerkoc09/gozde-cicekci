import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// useCustomer.ts, Nuxt'ın auto-import ettiği $fetch ve apiBase()'i kullanır.
// Bu dosya düz node ortamında (vitest.config.ts) çalıştığından ikisi de
// global olarak stub'lanıyor — useCart.test.ts pure-fonksiyon testi yazdığı
// için bu ihtiyacı hiç görmedi, burada composable ağ çağrısı yaptığı için
// $fetch'i taklit etmek gerekiyor.
const fetchMock = vi.fn()

// useState (Nuxt auto-import) me() sonucunu istek başına önbelleğe alıyor.
// Testte basit bir ref sözlüğüyle taklit ediliyor; her testten önce
// sıfırlanıyor ki testler birbirinin önbelleğini görmesin.
let stateStore: Record<string, { value: unknown }> = {}

beforeEach(() => {
  stateStore = {}
  vi.stubGlobal('$fetch', fetchMock)
  vi.stubGlobal('apiBase', () => '/api/go')
  vi.stubGlobal('useState', (key: string, init: () => unknown) => {
    if (!(key in stateStore))
      stateStore[key] = { value: init() }
    return stateStore[key]
  })
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

  // --- Önbellek davranışı (M5) ---
  // Hesabım ekranlarında layout + sayfa + header aynı veriyi bağımsız
  // istiyordu: tek gezinmede 3 ağ çağrısı.

  it('me() aynı istekte ikinci kez ağa çıkmaz', async () => {
    const { useCustomer } = await import('./useCustomer')
    const musteri = { id: 1, email: 'a@b.com', name: 'Ali', phone: '5551112233' }
    fetchMock.mockResolvedValueOnce(musteri)

    const c = useCustomer()
    expect(await c.me()).toEqual(musteri)
    expect(await c.me()).toEqual(musteri)
    expect(await c.me()).toEqual(musteri)

    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('me() giriş yokken de (null) önbelleğe alır', async () => {
    const { useCustomer } = await import('./useCustomer')
    fetchMock.mockRejectedValue(new Error('401'))

    const c = useCustomer()
    expect(await c.me()).toBeNull()
    expect(await c.me()).toBeNull()

    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('me(true) önbelleği atlar', async () => {
    const { useCustomer } = await import('./useCustomer')
    const musteri = { id: 1, email: 'a@b.com', name: 'Ali', phone: '5551112233' }
    fetchMock.mockResolvedValue(musteri)

    const c = useCustomer()
    await c.me()
    await c.me(true)

    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  // Kritik: çıkış yapan kullanıcı önbellek yüzünden hâlâ girişli görünmemeli.
  it('logout() sonrası me() null döner (önbellek temizlenir)', async () => {
    const { useCustomer } = await import('./useCustomer')
    const musteri = { id: 1, email: 'a@b.com', name: 'Ali', phone: '5551112233' }
    fetchMock.mockResolvedValueOnce(musteri) // ilk me()
    const c = useCustomer()
    expect(await c.me()).toEqual(musteri)

    fetchMock.mockResolvedValueOnce({ ok: true }) // logout
    await c.logout()

    expect(await c.me()).toBeNull()
  })

  it('register() sonrası me() ağa çıkmadan yeni müşteriyi döner', async () => {
    const { useCustomer } = await import('./useCustomer')
    const musteri = { id: 2, email: 'yeni@b.com', name: 'Yeni', phone: '5551112233' }
    fetchMock.mockResolvedValueOnce(musteri)

    const c = useCustomer()
    await c.register({ email: 'yeni@b.com', password: 'sifre1234', name: 'Yeni', phone: '5551112233' })

    expect(await c.me()).toEqual(musteri)
    expect(fetchMock).toHaveBeenCalledTimes(1) // yalnızca register
  })
})
