import ApiService from '@/services/ApiService'
import type { CustomerDetail, CustomerListResponse } from '@/model/customer'

export interface CustomerListParams {
  q?: string
  page?: number
  limit?: number
}

export function useCustomers() {
  const list = (params: CustomerListParams = {}) => {
    const query = new URLSearchParams()

    if (params.q)
      query.set('q', params.q)
    if (params.page)
      query.set('page', String(params.page))
    if (params.limit)
      query.set('limit', String(params.limit))

    const qs = query.toString()

    return ApiService.get<CustomerListResponse>(`admin/customers${qs ? `?${qs}` : ''}`)
  }

  const get = (id: number) => ApiService.get<CustomerDetail>(`admin/customers/${id}`)

  return { list, get }
}
