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
  is_active: boolean
  values: OptionValue[]
}

/**
 * Ürün formundan giden bağ. Zorunluluk YOK: müşteri sayfasında her grubun
 * ilk değeri otomatik seçili geliyor, "seçmeden geçilemez" kuralına
 * gerek kalmadı.
 */
export interface ProductOptionGroupLink {
  group_id: number
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

  /** Stok takibi ürün başına isteğe bağlı; false ise ürün sınırsız satılır. */
  track_stock: boolean
  stock_quantity: number

  /** Ödeme bekleyen adet — satılabilir = stock_quantity - stock_reserved. */
  stock_reserved: number

  /** null ise indirim yok. Kota dolunca indirim kendiliğinden söner. */
  discount_price: string | null
  discount_quota: number | null
  discount_sold: number
}

/** Stok hareketi sebebi — DB'deki CHECK ile birebir aynı. */
export type StokSebep
  = 'siparis' | 'whatsapp_satisi' | 'sayim_duzeltme'
  | 'yeni_parti' | 'iptal_iade' | 'rezervasyon_iptal'

/** Panelde gösterilen stok hareketi. */
export interface StokHareket {
  id: number
  delta: number
  reason: StokSebep
  order_id: number | null
  was_discounted: boolean
  note: string
  created_at: string
}

/** Elle stok düzeltmesi gövdesi. Delta negatif → düşüş, pozitif → giriş. */
export interface StokDuzeltme {
  delta: number
  reason: StokSebep
  was_discounted?: boolean
  note?: string
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

  track_stock?: boolean
  stock_quantity?: number

  /** discount_price ve discount_quota BİRLİKTE gönderilir. */
  discount_price?: string
  discount_quota?: number

  /** true ise indirim kaldırılır ve satılan sayacı sıfırlanır. */
  clear_discount?: boolean
}
