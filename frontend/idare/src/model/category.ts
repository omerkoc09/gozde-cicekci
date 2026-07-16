export type Axis = 'occasion' | 'type'

export interface Category {
  id: number
  name: string
  slug: string
  axis: Axis
  is_active: boolean
  is_featured: boolean
  sort_order: number
  /** Kart görseli. Yüklenmemişse boş string — site yedek görsel gösterir. */
  url_400: string
  url_900: string
}

export interface CategoryCreate {
  name: string
  axis: Axis
  is_active?: boolean
  is_featured?: boolean
  sort_order?: number
}

// PATCH semantiği: undefined alan değişmez. axis DEĞİŞTİRİLEMEZ —
// eksen değişimi ürün ilişkilerini anlamsız kılar (spec §4.1).
export interface CategoryUpdate {
  name?: string
  is_active?: boolean
  is_featured?: boolean
  sort_order?: number
}

export const AXIS_LABELS: Record<Axis, string> = {
  occasion: 'Gönderim Amacına Göre',
  type: 'Ürün Tipine Göre',
}
