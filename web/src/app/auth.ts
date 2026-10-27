import { defineStore } from 'pinia'

import { authApi } from '../features/auth/api'
import type {
  Identity,
  LoginInput,
  Organization,
  RegisterInput,
  User,
} from '../features/auth/types'

export type AuthStatus = 'unknown' | 'restoring' | 'authenticated' | 'anonymous'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    organizations: [] as Organization[],
    status: 'unknown' as AuthStatus,
  }),

  getters: {
    isAuthenticated: (state) => state.user !== null,
  },

  actions: {
    applyIdentity(identity: Identity) {
      this.user = identity.user
      this.organizations = identity.organizations
      this.status = 'authenticated'
    },

    async restore() {
      if (this.status !== 'unknown') {
        return
      }
      this.status = 'restoring'
      try {
        this.applyIdentity(await authApi.me())
      } catch {
        this.handleUnauthorized()
      }
    },

    async register(input: RegisterInput) {
      this.applyIdentity(await authApi.register(input))
    },

    async login(input: LoginInput) {
      this.applyIdentity(await authApi.login(input))
    },

    async logout() {
      try {
        await authApi.logout()
      } finally {
        this.handleUnauthorized()
      }
    },

    handleUnauthorized() {
      this.user = null
      this.organizations = []
      this.status = 'anonymous'
    },
  },
})
