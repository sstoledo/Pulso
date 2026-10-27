import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { useAuthStore } from './auth'
import { router } from './router'
import { identityFixture } from '../test/fixtures'

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

  it('redirects unauthenticated navigation to login with a return path', async () => {
    const auth = useAuthStore()
    auth.status = 'anonymous'

    await router.push('/protected-probe')

    expect(router.currentRoute.value.name).toBe('login')
    expect(router.currentRoute.value.query.redirect).toBe('/protected-probe')
  })

  it('allows authenticated navigation', async () => {
    const auth = useAuthStore()
    auth.user = identityFixture.user
    auth.organizations = identityFixture.organizations
    auth.status = 'authenticated'

    await router.push('/protected-probe')

    expect(router.currentRoute.value.name).toBe('protected-probe')
  })
})
