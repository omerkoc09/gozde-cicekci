export interface Slide {
  id: number
  title: string
  subtitle: string
  is_active: boolean
  sort_order: number
  url_400: string
  url_1200: string
}

// PATCH semantiği: undefined alan değişmez. Görsel BURADA YOK —
// değişimi ayrı uçta (useSlides().replaceImage), çünkü eski dosyanın
// saklamadan silinmesi gerekiyor.
export interface SlideUpdate {
  title?: string
  subtitle?: string
  is_active?: boolean
  sort_order?: number
}
