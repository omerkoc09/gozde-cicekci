import type { AxiosInstance } from 'axios'
import axios from 'axios'

// Backend hata formatı (spec §4.6): {"error": {"code": "...", "message": "..."}}
export interface ApiError {
  code: string
  message: string
}

/**
 * ApiService Go backend'ini tüketir.
 *
 * Auth HttpOnly cookie ile (spec §4.5) — Authorization header YOK,
 * localStorage'da token YOK. withCredentials cookie'nin gönderilmesini sağlar.
 * Refresh token yok: backend tek token veriyor (7 gün).
 */
class ApiService {
  private static instance: AxiosInstance

  private static init() {
    ApiService.instance = axios.create({
      baseURL: import.meta.env.VITE_API_BASE_URL,
      withCredentials: true, // HttpOnly cookie için zorunlu
      headers: { Accept: 'application/json' },
    })
  }

  private static get client(): AxiosInstance {
    if (!ApiService.instance)
      ApiService.init()

    return ApiService.instance
  }

  /** Hata mesajını backend formatından çıkarır, yoksa genel mesaj döner. */
  private static toError(err: any): ApiError {
    const body = err?.response?.data?.error
    if (body?.message)
      return { code: body.code ?? 'unknown', message: body.message }

    if (err?.response?.status === 401)
      return { code: 'unauthorized', message: 'Oturumunuz sona ermiş' }

    return { code: 'network', message: 'Sunucuya ulaşılamadı' }
  }

  /** [error, data] döner — Go'daki hata deseninin TypeScript karşılığı. */
  static async request<T>(
    method: 'get' | 'post' | 'put' | 'patch' | 'delete',
    url: string,
    data?: unknown,
    config?: object,
  ): Promise<[ApiError | null, T]> {
    try {
      const resp = await ApiService.client.request<T>({ method, url, data, ...config })

      return [null, resp.data]
    }
    catch (err) {
      return [ApiService.toError(err), undefined as T]
    }
  }

  static get<T>(url: string, config?: object) {
    return ApiService.request<T>('get', url, undefined, config)
  }

  static post<T>(url: string, data?: unknown, config?: object) {
    return ApiService.request<T>('post', url, data, config)
  }

  static put<T>(url: string, data?: unknown, config?: object) {
    return ApiService.request<T>('put', url, data, config)
  }

  static patch<T>(url: string, data?: unknown, config?: object) {
    return ApiService.request<T>('patch', url, data, config)
  }

  static delete<T>(url: string) {
    return ApiService.request<T>('delete', url)
  }
}

export default ApiService
