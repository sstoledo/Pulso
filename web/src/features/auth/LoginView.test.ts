import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { authApi } from './api'
import { identityFixture } from '../../test/fixtures'
import { useAuthStore } from '../../app/auth'
import { router } from '../../app/router'
import LoginView from './LoginView.vue'

vi.mock('./api', () => ({
  authApi: {
    register: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    me: vi.fn(),
  },
}))

describe('LoginView', () => {
  beforeEach(async () => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    const auth = useAuthStore()
    auth.status = 'anonymous'
    await router.push('/login?redirect=/monitors')
  })

  it('validates credentials without submitting invalid input', async () => {
    const wrapper = mount(LoginView, { global: { plugins: [router] } })
    await wrapper.get('form').trigger('submit')

    expect(wrapper.text()).toContain('Enter a valid email address.')
    expect(authApi.login).not.toHaveBeenCalled()
  })

  it('logs in and returns to the protected destination', async () => {
    vi.mocked(authApi.login).mockResolvedValue(identityFixture)
    const wrapper = mount(LoginView, { global: { plugins: [router] } })
    await wrapper.get('#email').setValue('ada@example.com')
    await wrapper.get('#password').setValue('password-12345')

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(authApi.login).toHaveBeenCalledWith({
      email: 'ada@example.com',
      password: 'password-12345',
    })
    expect(router.currentRoute.value.name).toBe('monitors')
  })

  it('does not follow an external return URL', async () => {
    await router.push('/login?redirect=https://attacker.example')
    vi.mocked(authApi.login).mockResolvedValue(identityFixture)
    const wrapper = mount(LoginView, { global: { plugins: [router] } })
    await wrapper.get('#email').setValue('ada@example.com')
    await wrapper.get('#password').setValue('password-12345')

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(router.currentRoute.value.name).toBe('monitors')
  })
})
