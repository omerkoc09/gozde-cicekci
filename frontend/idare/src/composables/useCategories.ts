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

  return { list, create, update, remove, productCount }
}
