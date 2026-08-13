import type { Order } from '@/model/order'

export interface Customer {
  id: number
  email: string
  name: string
  phone: string
  created_at: string
  updated_at: string
}

export interface CustomerDetail extends Customer {
  orders: Order[]
}

export interface CustomerListResponse {
  items: Customer[]
  total: number
}
