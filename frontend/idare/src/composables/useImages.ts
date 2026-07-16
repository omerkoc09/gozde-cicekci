import ApiService from '@/services/ApiService'
import type { ProductImage } from '@/model/product'

export function useImages() {
  const list = (productId: number) =>
    ApiService.get<ProductImage[]>(`admin/products/${productId}/images`)

  /**
   * multipart/form-data, alan adı "image". Backend JPEG/PNG/WebP kabul eder;
   * hepsini JPEG'e çevirip saklar.
   */
  const upload = (productId: number, file: File) => {
    const fd = new FormData()

    fd.append('image', file)

    return ApiService.post<ProductImage>(
      `admin/products/${productId}/images`,
      fd,
      { headers: { 'Content-Type': 'multipart/form-data' } },
    )
  }

  /** imageIds ürünün TÜM görsellerini içermeli — ilki kapak olur. */
  const reorder = (productId: number, imageIds: number[]) =>
    ApiService.patch<ProductImage[]>(
      `admin/products/${productId}/images/order`,
      { image_ids: imageIds },
    )

  const remove = (imageId: number) =>
    ApiService.delete<void>(`admin/images/${imageId}`)

  return { list, upload, reorder, remove }
}
