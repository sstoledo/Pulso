import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { authApi } from '../features/auth/api'
import { ApiError } from './api'
import { identityFixture } from '../test/fixtures'
import { useAuthStore } from './auth'

vi.mock('../features/auth/api', () => ({
  authApi: {
    register: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    me: vi.fn(),
  },
}))

describe('authentication store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('restores an authenticated session', async () => {
    vi.mocked(authApi.me).mockResolvedValue(identityFixture)
    const auth = useAuthStore()

    await auth.restore()

    expect(auth.isAuthenticated).toBe(true)
    expect(auth.user?.email).toBe('ada@example.com')
    expect(auth.organizations[0]?.role).toBe('owner')
  })

  it('marks an expired session anonymous', async () => {
    vi.mocked(authApi.me).mockRejectedValue(
      new ApiError(401, 'unauthorized', 'Authentication required.'),
    )
    const auth = useAuthStore()

    await auth.restore()

    expect(auth.isAuthenticated).toBe(false)
    expect(auth.status).toBe('anonymous')
  })

  it('clears authentication after logout and API unauthorized responses', async () => {
    vi.mocked(authApi.login).mockResolvedValue(identityFixture)
    vi.mocked(authApi.logout).mockResolvedValue(undefined)
    const auth = useAuthStore()
    await auth.login({ email: 'ada@example.com', password: 'password-12345' })
    expect(auth.isAuthenticated).toBe(true)

    await auth.logout()
    expect(auth.isAuthenticated).toBe(false)

    await auth.login({ email: 'ada@example.com', password: 'password-12345' })
    auth.handleUnauthorized()
    expect(auth.status).toBe('anonymous')
    expect(auth.user).toBeNull()
  })
})
