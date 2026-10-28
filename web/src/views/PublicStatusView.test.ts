import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '../app/api'
import { useAuthStore } from '../app/auth'
import { router } from '../app/router'
import { statusApi } from '../features/status/api'
import PublicStatusView from './PublicStatusView.vue'

vi.mock('../features/status/api', () => ({ statusApi: { get: vi.fn() } }))

function mountPublicStatus() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return mount(PublicStatusView, {
    global: { plugins: [pinia, router, [VueQueryPlugin, { queryClient }]] },
  })
}

describe('PublicStatusView', () => {
  beforeEach(async () => {
    vi.clearAllMocks()
    setActivePinia(createPinia())
    useAuthStore().handleUnauthorized()
    await router.push('/status/acme')
  })

  it('renders monitors and incidents without authentication', async () => {
    vi.mocked(statusApi.get).mockResolvedValue({
      title: 'Acme Status',
      monitors: [
        { name: 'Public API', status: 'UP', availability_24h: 0.995 },
        { name: 'No Data API', status: 'UNKNOWN', availability_24h: null },
      ],
      incidents: [
        {
          monitor_name: 'Public API',
          opened_at: '2026-01-01T00:00:00Z',
          resolved_at: '2026-01-01T00:10:00Z',
          duration_ms: 600000,
          status: 'RESOLVED',
        },
      ],
    })
    const wrapper = mountPublicStatus()
    await flushPromises()

    expect(statusApi.get).toHaveBeenCalledWith('acme')
    expect(router.currentRoute.value.name).toBe('public-status')
    expect(wrapper.text()).toContain('Acme Status')
    expect(wrapper.text()).toContain('Public API')
    expect(wrapper.text()).toContain('99.5%')
    expect(wrapper.text()).toContain('No data')
    expect(wrapper.text()).toContain('10m 0s')
    expect(wrapper.text()).toContain('RESOLVED')
  })

  it('shows empty states without fabricating data', async () => {
    vi.mocked(statusApi.get).mockResolvedValue({
      title: 'Acme Status',
      monitors: [],
      incidents: [],
    })
    const wrapper = mountPublicStatus()
    await flushPromises()

    expect(wrapper.text()).toContain('No public monitors yet.')
    expect(wrapper.text()).toContain('No incidents reported.')
    expect(wrapper.text()).not.toContain('100.0%')
  })

  it('reports an unpublished page as not found', async () => {
    vi.mocked(statusApi.get).mockRejectedValue(
      new ApiError(404, 'status_page_not_found', 'Status page not found.'),
    )
    const wrapper = mountPublicStatus()
    await flushPromises()

    expect(wrapper.text()).toContain('Status page not found')
    expect(router.currentRoute.value.name).toBe('public-status')
  })

  it('shows a generic error for other failures', async () => {
    vi.mocked(statusApi.get).mockRejectedValue(new ApiError(500, 'internal_error', 'Boom.'))
    const wrapper = mountPublicStatus()
    await flushPromises()

    expect(wrapper.text()).toContain('Unable to load this status page.')
  })
})
