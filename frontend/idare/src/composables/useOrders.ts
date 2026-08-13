import ApiService from '@/services/ApiService'
import type { Order, OrderUpdate } from '@/model/order'

export function useOrders() {
  const list = (status?: string) => {
    const q = status ? `?status=${status}` : ''

    return ApiService.get<Order[]>(`admin/orders${q}`)
  }

  const get = (id: number) => ApiService.get<Order>(`admin/orders/${id}`)

  const update = (id: number, data: OrderUpdate) =>
    ApiService.patch<Order>(`admin/orders/${id}`, data)

  const refund = (id: number) =>
    ApiService.post<Order>(`admin/orders/${id}/refund`)

  return { list, get, update, refund }
}
