import type { Customer, CustomerOrder, RecentAddress } from '~/types/api'

/**
 * Müşteri oturumu — spec 2026-08-13 üyelik/müşteri hesabı.
 *
 * Oturum HttpOnly `customer_token` cookie'sinde yaşar; JS token'ı okuyamaz.
 * Giriş durumu bu yüzden `me()` çağrısının 200 (giriş var) / 401 (yok)
 * dönüşüyle belirlenir — durum sunucudan gelir, client tahmin yürütmez.
 *
 * me() sonucu useState ile istek başına ÖNBELLEĞE alınır: hesabım
 * ekranlarında layout + sayfa + header aynı veriyi bağımsız olarak
 * istiyordu (tek gezinmede 3 ağ çağrısı). Önbellek "sunucudan gelir"
 * ilkesini bozmaz — yalnızca aynı render döngüsünde tekrarı önler.
 * login/logout/updateProfile önbelleği tazeler, yoksa çıkış yapan
 * kullanıcı hâlâ giriş yapmış görünürdü.
 *
 * Üyelik OPSİYONEL — misafir checkout'u bu composable hiç görmeyebilir.
 */
export function useCustomer() {
  // undefined = henüz sorulmadı, null = giriş yok, Customer = giriş var.
  const cache = useState<Customer | null | undefined>('musteri', () => undefined)

  async function fetchMe(): Promise<Customer | null> {
    try {
      return await $fetch<Customer>(`${apiBase()}/customer/me`)
    }
    catch {
      return null // 401 → giriş yok
    }
  }

  async function register(input: { email: string, password: string, name: string, phone: string }) {
    const musteri = await $fetch<Customer>(`${apiBase()}/customer/register`, { method: 'POST', body: input })
    cache.value = musteri
    return musteri
  }

  async function login(input: { email: string, password: string }) {
    const sonuc = await $fetch<{ ok: boolean }>(`${apiBase()}/customer/login`, { method: 'POST', body: input })
    // Giriş yeni bir oturum açtı — profili taze çek.
    cache.value = await fetchMe()
    return sonuc
  }

  async function logout() {
    const sonuc = await $fetch(`${apiBase()}/customer/logout`, { method: 'POST' })
    cache.value = null
    return sonuc
  }

  /**
   * Giriş yoksa (401) null döner — hata fırlatmaz, çağıran try/catch
   * yazmasın diye. Aynı istek içinde tekrar çağrılırsa ağa çıkmaz.
   * force=true önbelleği atlar.
   */
  async function me(force = false): Promise<Customer | null> {
    if (!force && cache.value !== undefined)
      return cache.value
    cache.value = await fetchMe()
    return cache.value
  }

  async function updateProfile(input: { name: string, phone: string, current_password?: string, new_password?: string }) {
    const musteri = await $fetch<Customer>(`${apiBase()}/customer/me`, { method: 'PATCH', body: input })
    cache.value = musteri // profil değişti — önbellek tazelenmeli
    return musteri
  }

  async function myOrders() {
    return await $fetch<CustomerOrder[]>(`${apiBase()}/customer/orders`)
  }

  /**
   * Geçmiş siparişlerden türetilen teslimat adresleri (adres defteri
   * tablosu yok). Giriş yoksa boş dizi — sipariş formu misafirde
   * bozulmamalı.
   */
  async function myAddresses(): Promise<RecentAddress[]> {
    try {
      return await $fetch<RecentAddress[]>(`${apiBase()}/customer/addresses`)
    }
    catch {
      return []
    }
  }

  return { register, login, logout, me, updateProfile, myOrders, myAddresses }
}
