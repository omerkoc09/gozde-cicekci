import type { CreateOrderInput, CreateOrderResult, DeliveryConfig } from '~/types/api'

/**
 * Sipariş API'si. Çağrılar same-origin Nitro proxy'sinden geçer —
 * CORS'a takılmasın. apiBase() useApi.ts'den auto-import edilir.
 */
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

/** Backend hata gövdesi: { error: { code, message } } */
interface ApiErrorBody {
  error?: { code?: string, message?: string }
}

/**
 * $fetch/useFetch hatasından kullanıcıya gösterilecek Türkçe mesajı çıkarır.
 *
 * Gövde İKİ şekilde gelebilir ve ikisi de desteklenmeli:
 *   1. e.data.error.message        — Go'ya doğrudan gidildiğinde
 *   2. e.data.data.error.message   — Nitro proxy'si üzerinden geldiğinde
 *
 * Proxy hatayı createError({ data }) ile yeniden fırlatıyor; h3 bunu
 * { data: <backend gövdesi> } olarak sarıyor, $fetch de kendi .data'sına
 * koyuyor — yani mesaj bir seviye derine iniyor. Yalnızca (1)'e bakılırsa
 * HER hata "Bir şeyler ters gitti"ye düşer ve kullanıcı yanlış şifre mi
 * girdiğini, e-postanın kayıtlı mı olduğunu asla öğrenemez.
 */
export function apiErrorMessage(e: unknown): string {
  const data = (e as { data?: ApiErrorBody & { data?: ApiErrorBody } })?.data
  const mesaj = data?.error?.message ?? data?.data?.error?.message
  if (!mesaj)
    return 'Bir şeyler ters gitti. Lütfen tekrar deneyin.'

  // Backend doğrulama hatalarını "geçersiz girdi: şifre en az 8 karakter
  // olmalı" gibi sentinel önekiyle sarıyor. Önek Go tarafında hata
  // sınıflandırması için anlamlı, kullanıcı için gürültü — kırpıyoruz.
  return mesaj.replace(/^geçersiz girdi:\s*/i, '')
}

/**
 * Hatanın HTTP durum kodunu çıkarır (401, 409, ...). Sayfalar duruma özel
 * mesaj verebilsin diye — backend bazı kodlarda sabit mesaj döndüğü için
 * ("Yetkisiz") mesajın kendisi yeterince açıklayıcı olmuyor.
 */
export function apiErrorStatus(e: unknown): number | undefined {
  const err = e as { statusCode?: number, status?: number }
  return err?.statusCode ?? err?.status
}
