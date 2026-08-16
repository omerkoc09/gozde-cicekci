import ApiService from '@/services/ApiService'
import type { Slide, SlideUpdate } from '@/model/slide'

export function useSlides() {
  const list = () => ApiService.get<Slide[]>('admin/slides')

  /**
   * Görsel ve metin tek istekte gider (multipart) — görselsiz slayt
   * oluşturulamaz. Sıra gönderilmiyor: backend sona ekler.
   */
  const create = (data: {
    title: string
    subtitle: string
    is_active: boolean
    image: File
  }) => {
    const fd = new FormData()

    fd.append('title', data.title)
    fd.append('subtitle', data.subtitle)
    fd.append('is_active', String(data.is_active))
    fd.append('image', data.image)

    return ApiService.post<Slide>('admin/slides', fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  }

  const update = (id: number, data: SlideUpdate) =>
    ApiService.patch<Slide>(`admin/slides/${id}`, data)

  /** Görsel değişimi ayrı uç: eski dosya backend'de siliniyor. */
  const replaceImage = (id: number, file: File) => {
    const fd = new FormData()

    fd.append('image', file)

    return ApiService.put<Slide>(`admin/slides/${id}/image`, fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  }

  const remove = (id: number) => ApiService.delete<void>(`admin/slides/${id}`)

  /**
   * Sırayı yeniden yazar. ids TÜM slaytları içermeli — backend eksik
   * listeyi reddediyor. Güncel liste döner.
   */
  const reorder = (ids: number[]) =>
    ApiService.put<Slide[]>('admin/slides/reorder', { ids })

  return { list, create, update, replaceImage, remove, reorder }
}
