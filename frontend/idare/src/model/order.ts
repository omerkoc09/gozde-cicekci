export type OrderStatus = 'awaiting_payment' | 'paid' | 'delivered' | 'refunded'

export interface OrderItemOption {
  group_name: string
  value_name: string

  /** kind='text' seçimde boş — o zaman nokta gösterilmez. */
  swatch_hex: string
}

export interface OrderItem {
  product_id: number | null
  product_name: string
  price_at_order: string
  quantity: number
  options: OrderItemOption[]
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

  paid_at: string | null
  refunded_at: string | null
  payment_ref: string
}

export interface OrderUpdate {
  status?: OrderStatus
  note?: string
}

export const STATUS_LABELS: Record<OrderStatus, string> = {
  awaiting_payment: 'Ödeme Bekliyor',
  paid: 'Ödendi',
  delivered: 'Teslim Edildi',
  refunded: 'İade Edildi',
}

export const STATUS_COLORS: Record<OrderStatus, string> = {
  awaiting_payment: 'warning',
  paid: 'info',
  delivered: 'success',
  refunded: 'error',
}
