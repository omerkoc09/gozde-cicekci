import ApiService from '@/services/ApiService'
import type { AdminUser, AdminUserCreate } from '@/model/adminUser'

export function useAdminUsers() {
  const list = () => ApiService.get<AdminUser[]>('admin/users')

  const create = (data: AdminUserCreate) =>
    ApiService.post<void>('admin/users', data)

  const remove = (id: number) =>
    ApiService.delete<void>(`admin/users/${id}`)

  const resetPassword = (id: number, password: string) =>
    ApiService.patch<void>(`admin/users/${id}/password`, { password })

  return { list, create, remove, resetPassword }
}
