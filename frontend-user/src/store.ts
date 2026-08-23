import { defineStore } from 'pinia'
import { AuthAPI, clearToken, setToken, type User } from './api'

export const useAuth = defineStore('auth', {
  state: () => ({ user: null as User | null }),
  actions: {
    async boot() {
      try { this.user = await AuthAPI.me() } catch { this.user = null }
    },
    async login(u: string, p: string) {
      const r = await AuthAPI.login(u, p)
      setToken(r.token)
      this.user = r.user
    },
    async register(u: string, p: string, n: string) {
      const r = await AuthAPI.register(u, p, n)
      setToken(r.token)
      this.user = r.user
    },
    logout() {
      clearToken()
      this.user = null
    },
  },
})
