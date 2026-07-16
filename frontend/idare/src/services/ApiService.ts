import type { AxiosInstance, AxiosRequestConfig, AxiosRequestHeaders } from 'axios'
import axios from 'axios'
import JwtService from '@/services/JwtService'
import { useUserStore } from '@/store/user'
import type { IApiPagination, IApiQuery, IApiQueryPagination } from '@/model/api'

interface ApiResponse<T> {
  data: T
  data_count: number
  error_code: number
  error_message: string
}
interface RefreshQueue {
  resolve: (value: unknown) => void
  reject: (reason?: any) => void
}
class ApiService {
  private static baseUrl = import.meta.env.VITE_API_BASE_URL
  private static instance: AxiosInstance
  private static isRefreshing = false
  private static failedQueue: RefreshQueue[] = []
  private static init() {
    ApiService.instance = axios.create({
      baseURL: ApiService.baseUrl,
    })
    ApiService.instance.defaults.headers.common.Accept = 'application/json'
    ApiService.instance.interceptors.request.use(
      req => {
        const token = JwtService.getAccessToken()
        if (token && req.headers)
          req.headers.Authorization = `Bearer ${token}`

        return req
      },
    )
  }

  private static async refresh(error: any): Promise<any> {
    const originalRequest = error.config
    if (originalRequest._retry)
      throw error
    originalRequest._retry = true

    if (originalRequest.url.includes('login'))
      throw error

    const refreshToken = JwtService.getRefreshToken()
    if (!refreshToken) {
      await useUserStore().logout()

      throw error
    }

    if (this.isRefreshing) {
      return new Promise<any>((resolve, reject) => {
        ApiService.failedQueue.push({ resolve, reject })
      })
        .then(token => {
          originalRequest.headers.Authorization = `Bearer ${token}`

          return axios.request(originalRequest)
        })
        .catch(err => {
          throw err
        })
    }
    this.isRefreshing = true

    return new Promise<any>((resolve, reject) => {
      // refresh durumuna göre promise resolve veya reject edilecek
      axios
        .post(`${ApiService.baseUrl}/auth/refresh`, { refresh_token: JwtService.getRefreshToken() })
        .then(({ data }) => {
          JwtService.saveTokens(data.data.access_token, data.data.refresh_token)
          originalRequest.headers.Authorization = `Bearer ${JwtService.getAccessToken()}`
          ApiService.processQueue(null, data.data.access_token)
          resolve(axios.request(originalRequest)) // başarılı oldu ise orjinal requesti tekrar gönder ve resolve et
        })
        .catch(err => {
          ApiService.processQueue(err, null) // başarısız oldu ise hepsini reject et ve çıkış yap
          useUserStore().logout()
          reject(err)
        })
        .finally(() => {
          ApiService.isRefreshing = false
        })
    })
  }

  private static processQueue(error: any, token = null) {
    this.failedQueue.forEach(prom => {
      if (error)
        prom.reject(error)

      else
        prom.resolve(token)
    })

    console.log('failedQueue', this.failedQueue)
    this.failedQueue = []
  }

  private static async request<T>(
    config: AxiosRequestConfig,
  ): Promise<[null, ApiResponse<T>] | [string, ApiResponse<T>]> {
    if (!ApiService.instance)
      this.init()

    try {
      const { data } = await ApiService.instance.request<ApiResponse<T>>(config)

      return [null, data]
    }
    catch (error: any) {
      const t: ApiResponse<T> = <ApiResponse<T>>{}
      if (error?.response?.status === 401) {
        try {
          const data = await this.refresh(error)

          return [null, data.data as ApiResponse<T>]
        }
        catch (e) {
          // eslint-disable-next-line no-ex-assign
          error = e
        }
      }

      if (!error.response)
        return ['Bağlantı Hatası. Lütfen İnternet Bağlantınızı Kontrol Ediniz.', t]
      const resp = error.response
      if (resp.status === 403)
        return ['Yetki Hatası', t]

      if (resp.status === 500)
        return ['Sunucu içi hata', t]

      const msg = resp.data?.error_message || 'Bilinmeyen Hata'

      return [msg, t]
    }
  }

  public static async get<T>(route: string, query?: IApiQuery | IApiPagination | IApiQueryPagination, urlParams?: URLSearchParams) {
    const searchParams = new URLSearchParams()
    let q: IApiQueryPagination
    if (query) {
      q = query as IApiQueryPagination
      searchParams.set('page', q.page?.toString() || '1')
      searchParams.set('per_page', q.perPage?.toString() || '1000')
      for (let i = 0; i < q.columns?.length; i++) {
        searchParams.append('columns', q.columns[i])
        searchParams.append('column_types', q.columnTypes[i].toString())
        searchParams.append('query', q.query[i])
      }
      for (let i = 0; i < q.sortColumns?.length; i++) {
        searchParams.append('sort_columns', q.sortColumns[i])
        searchParams.append('sort_column_types', q.sortColumnTypes[i].toString())
        searchParams.append('sort_orders', q.sortOrders[i])
      }
    }
    if (urlParams)
      urlParams.forEach((v, k) => searchParams.append(k, v))

    return this.request<T>({
      method: 'get',
      url: `/${route}`,
      params: searchParams,
    })
  }

  public static async post<T>(route: string, data?: unknown, urlParams?: URLSearchParams, headers?: AxiosRequestHeaders) {
    return this.request<T>({
      method: 'post',
      url: `/${route}`,
      data,
      params: urlParams,
      headers,
    })
  }

  public static put<T>(route: string, data?: unknown, urlParams?: URLSearchParams) {
    return this.request<T>({
      method: 'put',
      url: `/${route}`,
      data,
      params: urlParams,
    })
  }

  public static delete<T>(route: string, urlParams?: URLSearchParams) {
    return this.request<T>({
      method: 'delete',
      url: `/${route}`,
      params: urlParams,
    })
  }
}

export default ApiService
