import ApiService from '@/services/ApiService'
import type {
  OptionGroup, OptionGroupCreate, OptionGroupUpdate,
  OptionValue, OptionValueCreate, OptionValueUpdate,
} from '@/model/option'

export function useOptions() {
  const list = () => ApiService.get<OptionGroup[]>('admin/option-groups')

  const createGroup = (data: OptionGroupCreate) =>
    ApiService.post<OptionGroup>('admin/option-groups', data)

  const updateGroup = (id: number, data: OptionGroupUpdate) =>
    ApiService.patch<OptionGroup>(`admin/option-groups/${id}`, data)

  const removeGroup = (id: number) =>
    ApiService.delete<void>(`admin/option-groups/${id}`)

  /** Silme öncesi uyarı için: "Bu grup N üründe kullanılıyor". */
  const groupProductCount = (id: number) =>
    ApiService.get<{ product_count: number }>(`admin/option-groups/${id}/product-count`)

  /** ids TÜM grupları içermeli — backend eksik listeyi reddediyor. */
  const reorderGroups = (ids: number[]) =>
    ApiService.put<OptionGroup[]>('admin/option-groups/reorder', { ids })

  const createValue = (groupId: number, data: OptionValueCreate) =>
    ApiService.post<OptionValue>(`admin/option-groups/${groupId}/values`, data)

  const updateValue = (id: number, data: OptionValueUpdate) =>
    ApiService.patch<OptionValue>(`admin/option-values/${id}`, data)

  const removeValue = (id: number) =>
    ApiService.delete<void>(`admin/option-values/${id}`)

  /** ids O GRUBUN tüm değerlerini içermeli. */
  const reorderValues = (groupId: number, ids: number[]) =>
    ApiService.put<OptionGroup[]>(`admin/option-groups/${groupId}/values/reorder`, { ids })

  return {
    list,
    createGroup,
    updateGroup,
    removeGroup,
    groupProductCount,
    reorderGroups,
    createValue,
    updateValue,
    removeValue,
    reorderValues,
  }
}
