import type { OptionKind, OptionValue } from '@/model/option'

export interface ProductImage {
  id: number
  url_400: string
  url_1200: string
  sort_order: number
}

/** Ürüne açık seçenek grubu — panel görünümü, pasifler dahil. */
export interface ProductOptionGroup {
  id: number
  name: string
  kind: OptionKind
  is_required: boolean
  is_active: boolean
  values: OptionValue[]
}

/** Ürün formundan giden bağ. */
export interface ProductOptionGroupLink {
  group_id: number
  is_required: boolean
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
  option_groups: ProductOptionGroup[]
}

export interface ProductCreate {
  name: string
  description: string
  price: string
  is_active?: boolean
  is_featured?: boolean
  category_ids?: number[]
  option_groups?: ProductOptionGroupLink[]
}

// PATCH semantiği: undefined alan değişmez.
// category_ids/option_groups özel: undefined → dokunma, [] → hepsini kaldır.
export interface ProductUpdate {
  name?: string
  description?: string
  price?: string
  is_active?: boolean
  is_featured?: boolean
  category_ids?: number[]
  option_groups?: ProductOptionGroupLink[]
}
