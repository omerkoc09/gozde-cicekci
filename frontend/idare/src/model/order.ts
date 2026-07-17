export type OrderStatus = 'pending' | 'confirmed' | 'delivered' | 'cancelled'

export interface OrderItem {
  product_id: number | null
  product_name: string
  price_at_order: string
  quantity: number
}

export interface Order {
  id: number
  order_no: string
  status: OrderStatus

  buyer_name: string
  buyer_phone: string
  buyer_email: string

  recipient_name: string
  recipient_phone: string
  delivery_address: string
  delivery_district: string
  delivery_date: string
  delivery_slot: string
  card_message: string

  items_total: string
  delivery_fee: string
  total: string

  note: string
  items: OrderItem[]
  created_at: string
}

export interface OrderUpdate {
  status?: OrderStatus
  note?: string
}

export const STATUS_LABELS: Record<OrderStatus, string> = {
  pending: 'Yeni',
  confirmed: 'Onaylandı',
  delivered: 'Teslim Edildi',
  cancelled: 'İptal',
}

export const STATUS_COLORS: Record<OrderStatus, string> = {
  pending: 'warning',
  confirmed: 'info',
  delivered: 'success',
  cancelled: 'error',
}
