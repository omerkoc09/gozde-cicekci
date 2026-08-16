export type OptionKind = 'color' | 'text'

export const KIND_LABELS: Record<OptionKind, string> = {
  color: 'Renk',
  text: 'Metin',
}

export interface OptionValue {
  id: number
  name: string

  /** kind='text' grupta boş. */
  swatch_hex: string
  sort_order: number
  is_active: boolean
}

// sort_order YOK: yeni grup sona eklenir, sıra reorder ucundan değişir.
export interface OptionGroupCreate {
  name: string
  kind: OptionKind
}

// kind YOK: tip oluşturulduktan sonra değişmez.
export interface OptionGroupUpdate {
  name?: string
  is_active?: boolean
}

export interface OptionGroup {
  id: number
  name: string
  slug: string
  kind: OptionKind
  sort_order: number
  is_active: boolean
  values: OptionValue[]
}

/** Bir seçenek grubunu kullanan ürün — "bu grup nerede soruluyor" listesi. */
export interface GroupProduct {
  id: number
  name: string
  is_active: boolean
}

export interface OptionValueCreate {
  name: string
  swatch_hex: string
}

export interface OptionValueUpdate {
  name?: string
  swatch_hex?: string
  is_active?: boolean
}
