import ApiService from '@/services/ApiService'
import type {
  Product,
  ProductCreate,
  ProductUpdate,
  StokDuzeltme,
  StokHareket,
} from '@/model/product'

export function useProducts() {
  const list = (page = 1, limit = 24) =>
    ApiService.get<Product[]>(`admin/products?page=${page}&limit=${limit}`)

  const get = (id: number) => ApiService.get<Product>(`admin/products/${id}`)

  const create = (data: ProductCreate) =>
    ApiService.post<Product>('admin/products', data)

  const update = (id: number, data: ProductUpdate) =>
    ApiService.patch<Product>(`admin/products/${id}`, data)

  const remove = (id: number) => ApiService.delete<void>(`admin/products/${id}`)

  /** Elle stok hareketi — WhatsApp satışı, sayım düzeltmesi, yeni parti. */
  const adjustStock = (id: number, data: StokDuzeltme) =>
    ApiService.post<Product>(`admin/products/${id}/stock`, data)

  const movements = (id: number, limit = 50) =>
    ApiService.get<StokHareket[]>(`admin/products/${id}/movements?limit=${limit}`)

  return { list, get, create, update, remove, adjustStock, movements }
}
