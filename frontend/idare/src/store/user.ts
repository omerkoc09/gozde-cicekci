import { defineStore } from 'pinia'
import ApiService from '@/services/ApiService'

/**
 * Oturum durumu. Token HttpOnly cookie'de olduğu için JavaScript onu
 * okuyamaz — oturumun geçerliliği /me çağrısıyla anlaşılır.
 * Tek admin var, rol sistemi yok.
 */
export const useUserStore = defineStore('UserStore', {
  state: () => ({
    username: '',
    isAuthenticated: false,
  }),
  actions: {
    async login(username: string, password: string): Promise<string | null> {
      const [err] = await ApiService.post('admin/login', { username, password })
      if (err)
        return err.message

      await this.checkSession()

      return null
    },

    async logout() {
      await ApiService.post('admin/logout')
      this.username = ''
      this.isAuthenticated = false
    },

    /** Cookie geçerli mi — sayfa yenilendiğinde oturumu geri kazanmak için. */
    async checkSession(): Promise<boolean> {
      const [err, data] = await ApiService.get<{ username: string }>('admin/me')
      if (err) {
        this.isAuthenticated = false
        this.username = ''

        return false
      }
      this.username = data.username
      this.isAuthenticated = true

      return true
    },
  },
})
