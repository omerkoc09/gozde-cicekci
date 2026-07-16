export interface ProductImage {
  id: number
  url_400: string
  url_1200: string
  sort_order: number
}

export interface Product {
  id: number
  name: string
  slug: string
  description: string
  price: string // "1850.00" — float precision için string (spec §4.1)
  is_active: boolean
  /** Ana sayfa "En Çok Tercih Edilenler" vitrininde gösterilir. */
  is_featured: boolean
  category_ids: number[]
  images: ProductImage[]
}

export interface ProductCreate {
  name: string
  description: string
  price: string
  is_active?: boolean
  is_featured?: boolean
  category_ids?: number[]
}

// PATCH semantiği: undefined alan değişmez.
// category_ids özel: undefined → dokunma, [] → hepsini kaldır.
export interface ProductUpdate {
  name?: string
  description?: string
  price?: string
  is_active?: boolean
  is_featured?: boolean
  category_ids?: number[]
}
