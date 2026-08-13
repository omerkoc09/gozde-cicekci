import type { Customer, CustomerOrder } from '~/types/api'

/**
 * Müşteri oturumu — spec 2026-08-13 üyelik/müşteri hesabı.
 *
 * Oturum HttpOnly `customer_token` cookie'sinde yaşar; JS token'ı okuyamaz.
 * Giriş durumu bu yüzden `me()` çağrısının 200 (giriş var) / 401 (yok)
 * dönüşüyle belirlenir — ayrı bir "isLoggedIn" state'i tutulmuyor, her
 * sayfa kendi onMounted'ında me() çağırır (useApi.ts'teki composable'larla
 * aynı idiom: durum sunucudan gelir, client tahmin yürütmez).
 *
 * Üyelik OPSİYONEL — misafir checkout'u bu composable hiç görmeyebilir.
 */
export function useCustomer() {
  async function register(input: { email: string, password: string, name: string, phone: string }) {
    return await $fetch<Customer>(`${apiBase()}/customer/register`, { method: 'POST', body: input })
  }

  async function login(input: { email: string, password: string }) {
    return await $fetch<{ ok: boolean }>(`${apiBase()}/customer/login`, { method: 'POST', body: input })
  }

  async function logout() {
    return await $fetch(`${apiBase()}/customer/logout`, { method: 'POST' })
  }

  /** Giriş yoksa (401) null döner — hata fırlatmaz, çağıran try/catch yazmasın diye. */
  async function me(): Promise<Customer | null> {
    try {
      return await $fetch<Customer>(`${apiBase()}/customer/me`)
    }
    catch {
      return null
    }
  }

  async function updateProfile(input: { name: string, phone: string, current_password?: string, new_password?: string }) {
    return await $fetch<Customer>(`${apiBase()}/customer/me`, { method: 'PATCH', body: input })
  }

  async function myOrders() {
    return await $fetch<CustomerOrder[]>(`${apiBase()}/customer/orders`)
  }

  return { register, login, logout, me, updateProfile, myOrders }
}
