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
