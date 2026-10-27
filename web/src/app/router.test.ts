import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { useAuthStore } from './auth'
import { router } from './router'

describe('protected navigation', () => {
  let removeProbeRoute: () => void

  beforeEach(() => {
    setActivePinia(createPinia())
    removeProbeRoute = router.addRoute({
      path: '/protected-probe',
      name: 'protected-probe',
      component: { template: '<p>Protected</p>' },
      meta: { requiresAuth: true },
    })
  })

  afterEach(() => removeProbeRoute())

  it('redirects unauthenticated navigation home', async () => {
    useAuthStore().isAuthenticated = false

    await router.push('/protected-probe')

    expect(router.currentRoute.value.name).toBe('home')
  })

  it('allows authenticated navigation', async () => {
    useAuthStore().isAuthenticated = true

    await router.push('/protected-probe')

    expect(router.currentRoute.value.name).toBe('protected-probe')
  })
})
