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

export type Axis = 'occasion' | 'type'

export interface Category {
  id: number
  name: string
  slug: string
  axis: Axis
}

export const AXIS_LABELS: Record<Axis, string> = {
  occasion: 'Gönderim Amacına Göre',
  type: 'Ürün Tipine Göre',
}
