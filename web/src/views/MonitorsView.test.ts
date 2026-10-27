import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '../app/api'
import { useAuthStore } from '../app/auth'
import { identityFixture } from '../test/fixtures'
import { monitorsApi } from '../features/monitors/api'
import type { Monitor } from '../features/monitors/types'
import MonitorsView from './MonitorsView.vue'

vi.mock('../features/monitors/api', () => ({
  monitorsApi: {
    list: vi.fn(),
    summary: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    pause: vi.fn(),
    resume: vi.fn(),
    delete: vi.fn(),
  },
}))

const monitorFixture: Monitor = {
  id: '33333333-3333-3333-3333-333333333333',
  name: 'Public API',
  url: 'https://example.com/health',
  interval_seconds: 60,
  timeout_ms: 5000,
  expected_status_min: 200,
  expected_status_max: 399,
  enabled: true,
  show_public: false,
  status: 'UNKNOWN',
  last_check_at: null,
  last_latency_ms: null,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

function listWith(items: Monitor[], total = items.length) {
  return { items, total, page: 1, page_size: 10, total_pages: total > 10 ? 2 : total > 0 ? 1 : 0 }
}

function mountView() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const auth = useAuthStore()
  auth.applyIdentity(identityFixture)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return mount(MonitorsView, { global: { plugins: [pinia, [VueQueryPlugin, { queryClient }]] } })
}

describe('MonitorsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(monitorsApi.summary).mockResolvedValue({ up: 0, down: 0, unknown: 0 })
    vi.mocked(monitorsApi.list).mockResolvedValue(listWith([]))
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('shows the empty state and creates a monitor with defaults', async () => {
    vi.mocked(monitorsApi.create).mockResolvedValue(monitorFixture)
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('No monitors yet')

    await wrapper.get('#create-monitor').trigger('click')
    await wrapper.get('#monitor-name').setValue('Public API')
    await wrapper.get('#monitor-url').setValue('https://example.com/health')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(monitorsApi.create).toHaveBeenCalledWith(identityFixture.organizations[0]!.id, {
      name: 'Public API',
      url: 'https://example.com/health',
      interval_seconds: 60,
      timeout_ms: 5000,
      expected_status_min: 200,
      expected_status_max: 399,
      show_public: false,
    })
  })

  it('validates the form before calling the API', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('#create-monitor').trigger('click')
    await wrapper.get('#monitor-name').setValue('Bad')
    await wrapper.get('#monitor-url').setValue('https://user:secret@example.com')

    await wrapper.get('form').trigger('submit')

    expect(wrapper.text()).toContain('Enter an HTTP(S) URL without credentials or fragments.')
    expect(monitorsApi.create).not.toHaveBeenCalled()
  })

  it('rejects an empty URL fragment', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('#create-monitor').trigger('click')
    await wrapper.get('#monitor-name').setValue('Bad')
    await wrapper.get('#monitor-url').setValue('https://example.com/#')

    await wrapper.get('form').trigger('submit')

    expect(wrapper.text()).toContain('Enter an HTTP(S) URL without credentials or fragments.')
    expect(monitorsApi.create).not.toHaveBeenCalled()
  })

  it('edits, pauses and deletes a monitor', async () => {
    vi.mocked(monitorsApi.list).mockResolvedValue(listWith([monitorFixture]))
    vi.mocked(monitorsApi.summary).mockResolvedValue({ up: 0, down: 0, unknown: 1 })
    vi.mocked(monitorsApi.update).mockResolvedValue(monitorFixture)
    vi.mocked(monitorsApi.pause).mockResolvedValue({ ...monitorFixture, enabled: false })
    vi.mocked(monitorsApi.delete).mockResolvedValue(undefined)
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('Public API')
    expect(wrapper.text()).toContain('UNKNOWN')

    await wrapper.get('[data-action="edit-monitor"]').trigger('click')
    await wrapper.get('#monitor-timeout').setValue(250)
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(monitorsApi.update).toHaveBeenCalledWith(
      identityFixture.organizations[0]!.id,
      monitorFixture.id,
      expect.objectContaining({ timeout_ms: 250 }),
    )

    await wrapper.get('[data-action="pause-monitor"]').trigger('click')
    await flushPromises()
    expect(monitorsApi.pause).toHaveBeenCalledWith(
      identityFixture.organizations[0]!.id,
      monitorFixture.id,
    )

    await wrapper.get('[data-action="delete-monitor"]').trigger('click')
    await flushPromises()
    expect(monitorsApi.delete).toHaveBeenCalledWith(
      identityFixture.organizations[0]!.id,
      monitorFixture.id,
    )
  })

  it('shows an error when monitors cannot load', async () => {
    vi.mocked(monitorsApi.list).mockRejectedValue(
      new ApiError(500, 'internal_error', 'An unexpected error occurred.'),
    )
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Unable to load monitors.')
  })

  it('polls every 15 seconds and paginates', async () => {
    vi.useFakeTimers()
    vi.mocked(monitorsApi.list).mockResolvedValue(listWith([monitorFixture], 11))
    const wrapper = mountView()
    await flushPromises()
    vi.mocked(monitorsApi.list).mockClear()

    await vi.advanceTimersByTimeAsync(15000)
    await flushPromises()
    expect(monitorsApi.list).toHaveBeenCalledTimes(1)

    await wrapper.get('#next-page').trigger('click')
    await flushPromises()
    expect(monitorsApi.list).toHaveBeenLastCalledWith(identityFixture.organizations[0]!.id, 2, 10)
  })
})
