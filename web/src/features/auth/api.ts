import { apiFetch } from '../../app/api'
import type { Identity, LoginInput, RegisterInput } from './types'

export const authApi = {
  register(input: RegisterInput): Promise<Identity> {
    return apiFetch<Identity>('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify(input),
    })
  },

  login(input: LoginInput): Promise<Identity> {
    return apiFetch<Identity>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify(input),
    })
  },

  logout(): Promise<void> {
    return apiFetch<void>('/api/v1/auth/logout', { method: 'POST' })
  },

  me(): Promise<Identity> {
    return apiFetch<Identity>('/api/v1/auth/me')
  },
}
