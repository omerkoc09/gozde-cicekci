/** Public API tipleri — backend'in app viewmodel'lerine birebir karşılık. */

export interface ProductImage {
  url_400: string
  url_800: string
  url_1200: string
}

export interface Product {
  id: number
  name: string
  slug: string
  description: string
  /** "1850.00" — float precision için string (spec §4.1) */
  price: string
  category_ids: number[]
  images: ProductImage[]
  option_groups?: ProductOptionGroup[]
}

/** Ana sayfa slider'ında bir slayt. Pasif slaytlar bu uçtan hiç gelmez. */
export interface Slide {
  id: number
  title: string
  subtitle: string
  url_400: string
  url_1200: string
  url_1920: string
}

export type Axis = 'occasion' | 'type'

export interface Category {
  id: number
  name: string
  slug: string
  axis: Axis
  /** Kart görseli. Panelden yüklenmemişse boş — yedek görsele düşülür. */
  url_400: string
  url_900: string
}

export const AXIS_LABELS: Record<Axis, string> = {
  occasion: 'Gönderim Amacına Göre',
  type: 'Ürün Tipine Göre',
}

/** Sepet kalemindeki tek seçim. value_id sunucuya gider; isim ve renk
 *  yalnızca GÖSTERİM için — sipariş anında sunucu DB'den okur. */
export interface CartItemOption {
  value_id: number
  group_name: string
  value_name: string
  swatch_hex: string
}

/** Sepet kalemi — localStorage'da yaşar. Fiyat GÖSTERİM için; sipariş
 *  anında sunucu DB'den okur, buradaki fiyata güvenilmez (spec §2.2). */
export interface CartItem {
  product_id: number
  name: string
  slug: string
  price: string
  image: string
  quantity: number
  /** Seçim yoksa boş dizi. Bu alandan ÖNCE kurulmuş sepetlerde
   *  undefined gelir — okuyan taraf boş dizi kabul eder. */
  options?: CartItemOption[]
}

export interface ProductOptionValue {
  id: number
  name: string
  swatch_hex: string
}

export interface ProductOptionGroup {
  id: number
  name: string
  kind: 'color' | 'text'
  is_required: boolean
  values: ProductOptionValue[]
}

export interface DeliveryConfig {
  fee: string
  slots: string[]
  same_day_cutoff: string
  max_days: number
  districts: string[]
  /** İlçeye özel ücret — olmayan ilçe fee'ye (genel ücret) düşer. */
  district_fees: Record<string, string>
}

export interface CreateOrderInput {
  items: { product_id: number, quantity: number, option_value_ids: number[] }[]
  buyer: { name: string, phone: string }
  recipient: { name: string, phone: string }
  delivery: { address: string, district: string, date: string, slot: string }
  card_message?: string
}

export interface CreateOrderResult {
  order_no: string
  total: string
  paytr_token: string
}

/** Giriş yapmış müşteri — /customer/me, /customer/register, /customer/me (PATCH) yanıtı. */
export interface Customer {
  id: number
  email: string
  name: string
  phone: string
}

/** /customer/orders listesindeki bir kalem — sadeleştirilmiş, minimal görünüm. */
export interface CustomerOrderItem {
  product_name: string
  quantity: number
}

export type CustomerOrderStatus = 'awaiting_payment' | 'paid' | 'delivered' | 'refunded'

/** /customer/orders — her zaman dizi, boşsa [] (asla null). */
export interface CustomerOrder {
  order_no: string
  status: CustomerOrderStatus
  total: string
  delivery_date: string
  items: CustomerOrderItem[]
}

/**
 * /customer/addresses — geçmiş siparişlerden türetilen teslimat adresleri.
 * Adres defteri tablosu YOK; bu veri orders'tan geliyor.
 */
export interface RecentAddress {
  recipient_name: string
  recipient_phone: string
  delivery_address: string
  delivery_district: string
}
