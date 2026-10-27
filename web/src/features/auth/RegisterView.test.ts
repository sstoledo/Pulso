import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { authApi } from './api'
import { identityFixture } from '../../test/fixtures'
import { useAuthStore } from '../../app/auth'
import { router } from '../../app/router'
import RegisterView from './RegisterView.vue'

vi.mock('./api', () => ({
  authApi: {
    register: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    me: vi.fn(),
  },
}))

describe('RegisterView', () => {
  beforeEach(async () => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    useAuthStore().status = 'anonymous'
    await router.push('/register')
  })

  it('rejects a short password locally', async () => {
    const wrapper = mount(RegisterView, { global: { plugins: [router] } })
    await wrapper.get('#name').setValue('Ada')
    await wrapper.get('#email').setValue('ada@example.com')
    await wrapper.get('#password').setValue('short')

    await wrapper.get('form').trigger('submit')

    expect(wrapper.text()).toContain('Enter between 12 and 128 characters.')
    expect(authApi.register).not.toHaveBeenCalled()
  })

  it('accepts exactly 128 Unicode characters', async () => {
    vi.mocked(authApi.register).mockResolvedValue(identityFixture)
    const wrapper = mount(RegisterView, { global: { plugins: [router] } })
    const password = '😀'.repeat(128)
    await wrapper.get('#name').setValue('Ada Lovelace')
    await wrapper.get('#email').setValue('ada@example.com')
    await wrapper.get('#password').setValue(password)

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(authApi.register).toHaveBeenCalledWith({
      name: 'Ada Lovelace',
      email: 'ada@example.com',
      password,
    })
    expect(router.currentRoute.value.name).toBe('monitors')
  })

  it('registers and opens the protected dashboard', async () => {
    vi.mocked(authApi.register).mockResolvedValue(identityFixture)
    const wrapper = mount(RegisterView, { global: { plugins: [router] } })
    await wrapper.get('#name').setValue('Ada Lovelace')
    await wrapper.get('#email').setValue('ada@example.com')
    await wrapper.get('#password').setValue('password-12345')

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(authApi.register).toHaveBeenCalledWith({
      name: 'Ada Lovelace',
      email: 'ada@example.com',
      password: 'password-12345',
    })
    expect(router.currentRoute.value.name).toBe('monitors')
  })
})
