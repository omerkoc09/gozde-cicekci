import ApiService from '@/services/ApiService'
import type { Category, CategoryCreate, CategoryUpdate } from '@/model/category'

export function useCategories() {
  const list = () => ApiService.get<Category[]>('admin/categories')

  const create = (data: CategoryCreate) =>
    ApiService.post<Category>('admin/categories', data)

  const update = (id: number, data: CategoryUpdate) =>
    ApiService.patch<Category>(`admin/categories/${id}`, data)

  const remove = (id: number) =>
    ApiService.delete<void>(`admin/categories/${id}`)

  // Silme öncesi uyarı için: "Bu kategoride N ürün var" (spec §4.1)
  const productCount = (id: number) =>
    ApiService.get<{ product_count: number }>(`admin/categories/${id}/product-count`)

  /** Kart görseli. Ayrı uç: eski dosya backend'de siliniyor. */
  const replaceImage = (id: number, file: File) => {
    const fd = new FormData()

    fd.append('image', file)

    return ApiService.put<Category>(`admin/categories/${id}/image`, fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  }

  /** Görseli kaldırır — kategori silinmez, site yedek görsele döner. */
  const removeImage = (id: number) =>
    ApiService.delete<Category>(`admin/categories/${id}/image`)

  return { list, create, update, remove, productCount, replaceImage, removeImage }
}
