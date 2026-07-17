import type { CreateOrderInput, CreateOrderResult, DeliveryConfig } from '~/types/api'

/**
 * Sipariş API'si. Çağrılar same-origin Nitro proxy'sinden geçer —
 * CORS'a takılmasın (bkz. useApi.ts).
 */
function apiBase(): string {
  return useRuntimeConfig().public.apiBase
}

export function useDeliveryConfig() {
  return useFetch<DeliveryConfig>(() => `${apiBase()}/delivery-config`, {
    key: 'delivery-config',
  })
}

/** Sipariş oluşturur. Hata mesajı backend'den gelir (Türkçe). */
export async function createOrder(input: CreateOrderInput): Promise<CreateOrderResult> {
  return await $fetch<CreateOrderResult>(`${apiBase()}/orders`, {
    method: 'POST',
    body: input,
  })
}

/**
 * $fetch/useFetch hatasından kullanıcıya gösterilecek Türkçe mesajı çıkarır.
 * Backend hata gövdesi { error: { code, message } } — o mesaj varsa kullanılır,
 * yoksa (ağ hatası vb.) genel bir mesaja düşülür.
 */
export function apiErrorMessage(e: unknown): string {
  const data = (e as { data?: { error?: { message?: string } } })?.data
  return data?.error?.message ?? 'Bir şeyler ters gitti. Lütfen tekrar deneyin.'
}
