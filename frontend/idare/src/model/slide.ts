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
// saklamadan silinmesi gerekiyor. sort_order da yok: sıra reorder
// ucundan (listedeki ok butonları) değişir.
export interface SlideUpdate {
  title?: string
  subtitle?: string
  is_active?: boolean
}
