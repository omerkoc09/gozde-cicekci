import { defineStore } from 'pinia'
import ApiService from '@/services/ApiService'
import JwtService from '@/services/JwtService'
import { ErrorPopup } from '@/utils/Popup'

export type UserRole = 'admin' | 'teacher' | 'parent'

export interface User {
  id: number
  name: string
  surname: string
  email: string
  role: UserRole
}

export const useUserStore = defineStore('UserStore', {
  state: () => ({
    user: localStorage.getItem('user') ? JSON.parse(localStorage.getItem('user') as string) : {} as User,
    isAuthenticated: !!JwtService.getAccessToken(),
  }),
  getters: {
    isUserAuthenticated: state => state.isAuthenticated,
    hasRole: state => (roles?: UserRole[]) => !roles || roles.includes(state.user.role), // role paramteresi verilmediyse true döndürür
    getRole: state => () => state.user.role, // todo: SOR
  },
  actions: {
    async login(access_token: string, refresh_token: string) {
      JwtService.saveTokens(access_token, refresh_token)
      this.isAuthenticated = true
      await this.updateUser()
    },

    async logout() {
      this.isAuthenticated = false
      this.user = {} as User
      JwtService.destroyTokens()
      localStorage.removeItem('user')

      const redirect = window.location.pathname + window.location.search
      const urlParams = new URLSearchParams()

      urlParams.set('redirect', redirect)
      document.location.href = `/auth/login?${urlParams.toString()}`
    },

    async updateUser() {
      if (this.isAuthenticated !== true)
        return

      const [error, data] = await ApiService.get<User>('user/me')
      if (error) {
        ErrorPopup(error)

        return
      }
      this.user = data.data
      localStorage.setItem('user', JSON.stringify(this.user))
    },
  },
})
