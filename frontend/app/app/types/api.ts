/** Public API tipleri — backend'in app viewmodel'lerine birebir karşılık. */

export interface ProductImage {
  url_400: string
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

/** Sepet kalemi — localStorage'da yaşar. Fiyat GÖSTERİM için; sipariş
 *  anında sunucu DB'den okur, buradaki fiyata güvenilmez (spec §2.2). */
export interface CartItem {
  product_id: number
  name: string
  slug: string
  price: string
  image: string
  quantity: number
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
  items: { product_id: number, quantity: number }[]
  buyer: { name: string, phone: string }
  recipient: { name: string, phone: string }
  delivery: { address: string, district: string, date: string, slot: string }
  card_message?: string
}

export interface CreateOrderResult {
  order_no: string
  total: string
}
