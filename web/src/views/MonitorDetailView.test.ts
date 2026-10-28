import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '../app/api'
import { useAuthStore } from '../app/auth'
import { identityFixture } from '../test/fixtures'
import { incidentsApi } from '../features/incidents/api'
import { monitorsApi } from '../features/monitors/api'
import type { Monitor } from '../features/monitors/types'
import { router } from '../app/router'
import MonitorDetailView from './MonitorDetailView.vue'

vi.mock('../features/monitors/api', () => ({
  monitorsApi: {
    get: vi.fn(),
    metrics: vi.fn(),
    checks: vi.fn(),
    list: vi.fn(),
    summary: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    pause: vi.fn(),
    resume: vi.fn(),
    delete: vi.fn(),
  },
}))
vi.mock('../features/incidents/api', () => ({ incidentsApi: { list: vi.fn() } }))

const monitor: Monitor = {
  id: '33333333-3333-3333-3333-333333333333',
  name: 'Public API',
  url: 'https://example.com/health',
  interval_seconds: 60,
  timeout_ms: 5000,
  expected_status_min: 200,
  expected_status_max: 399,
  enabled: true,
  show_public: false,
  status: 'DOWN',
  last_check_at: null,
  last_latency_ms: null,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

function mountDetail() {
  const pinia = createPinia()
  setActivePinia(pinia)
  useAuthStore().applyIdentity(identityFixture)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return mount(MonitorDetailView, {
    global: { plugins: [pinia, router, [VueQueryPlugin, { queryClient }]] },
  })
}

describe('MonitorDetailView', () => {
  beforeEach(async () => {
    setActivePinia(createPinia())
    useAuthStore().applyIdentity(identityFixture)
    vi.clearAllMocks()
    vi.mocked(monitorsApi.get).mockResolvedValue(monitor)
    vi.mocked(monitorsApi.metrics).mockResolvedValue({
      availability_24h: 0.6,
      average_latency_ms: 250,
      p95_latency_ms: 385,
      series: [],
    })
    vi.mocked(monitorsApi.checks).mockResolvedValue({
      items: [
        {
          id: 'check-1',
          checked_at: '2026-01-01T00:00:00Z',
          success: false,
          http_status: 500,
          latency_ms: 300,
          error_summary: 'unexpected HTTP status',
        },
      ],
      total: 5,
      page: 2,
      page_size: 2,
      total_pages: 3,
    })
    vi.mocked(incidentsApi.list).mockResolvedValue({
      items: [
        {
          id: 'incident-1',
          monitor_id: monitor.id,
          opened_at: '2026-01-01T00:00:00Z',
          resolved_at: null,
          duration_ms: null,
          reason: 'request timeout',
          status: 'OPEN',
        },
      ],
      total: 1,
      page: 1,
      page_size: 10,
      total_pages: 1,
    })
    await router.push(`/monitors/${monitor.id}`)
  })

  it('renders metrics and both histories', async () => {
    const wrapper = mountDetail()
    await flushPromises()

    expect(wrapper.text()).toContain('Public API')
    expect(wrapper.text()).toContain('DOWN')
    expect(wrapper.text()).toContain('60.0%')
    expect(wrapper.text()).toContain('250 ms')
    expect(wrapper.text()).toContain('385 ms')
    expect(wrapper.text()).toContain('unexpected HTTP status')
    expect(wrapper.text()).toContain('request timeout')
    expect(wrapper.text()).toContain('OPEN')

    await wrapper.get('#next-checks').trigger('click')
    await flushPromises()
    expect(monitorsApi.checks).toHaveBeenLastCalledWith(
      identityFixture.organizations[0]!.id,
      monitor.id,
      2,
      10,
    )
  })

  it('uses empty values instead of fabricating 100 percent', async () => {
    vi.mocked(monitorsApi.metrics).mockResolvedValue({
      availability_24h: null,
      average_latency_ms: null,
      p95_latency_ms: null,
      series: [],
    })
    vi.mocked(monitorsApi.checks).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 10,
      total_pages: 0,
    })
    vi.mocked(incidentsApi.list).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 10,
      total_pages: 0,
    })
    const wrapper = mountDetail()
    await flushPromises()

    expect(wrapper.text()).not.toContain('100%')
    expect(wrapper.text()).toContain('No checks yet.')
    expect(wrapper.text()).toContain('No incidents yet.')
  })

  it('paginates incident history', async () => {
    vi.mocked(incidentsApi.list).mockResolvedValue({
      items: [
        {
          id: 'incident-1',
          monitor_id: monitor.id,
          opened_at: '2026-01-01T00:00:00Z',
          resolved_at: null,
          duration_ms: null,
          reason: 'request timeout',
          status: 'OPEN',
        },
      ],
      total: 11,
      page: 1,
      page_size: 10,
      total_pages: 2,
    })
    const wrapper = mountDetail()
    await flushPromises()

    const next = wrapper.get('nav[aria-label="Incident history pages"]').findAll('button')[1]!
    await next.trigger('click')
    await flushPromises()

    expect(incidentsApi.list).toHaveBeenLastCalledWith(
      identityFixture.organizations[0]!.id,
      monitor.id,
      2,
      10,
    )
  })

  it('shows an error when the monitor cannot load', async () => {
    vi.mocked(monitorsApi.get).mockRejectedValue(
      new ApiError(404, 'monitor_not_found', 'Monitor not found.'),
    )
    const wrapper = mountDetail()
    await flushPromises()
    expect(wrapper.text()).toContain('Unable to load monitor.')
  })

  it('polls monitor data every 15 seconds', async () => {
    vi.useFakeTimers()
    const wrapper = mountDetail()
    await flushPromises()
    vi.mocked(monitorsApi.metrics).mockClear()

    await vi.advanceTimersByTimeAsync(15000)
    await flushPromises()
    expect(monitorsApi.metrics).toHaveBeenCalledTimes(1)
    wrapper.unmount()
    vi.useRealTimers()
  })
})
