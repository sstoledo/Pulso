import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import LatencyChart from './LatencyChart.vue'

const { init } = vi.hoisted(() => ({
  init: vi.fn(() => ({
    setOption: vi.fn(),
    resize: vi.fn(),
    dispose: vi.fn(),
  })),
}))

vi.mock('echarts/core', () => ({ init, use: vi.fn() }))

describe('LatencyChart', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('passes actual latency and success values to ECharts', () => {
    const series = [
      { checked_at: '2026-01-01T00:00:00Z', latency_ms: 100, success: true, http_status: 200 },
      { checked_at: '2026-01-01T00:01:00Z', latency_ms: 300, success: false, http_status: 500 },
      { checked_at: '2026-01-01T00:02:00Z', latency_ms: 50, success: false, http_status: null },
    ]
    mount(LatencyChart, { props: { series } })
    const chart = init.mock.results[0]!.value
    const option = chart.setOption.mock.calls[0][0]

    expect(option.series[0].data).toEqual([
      ['2026-01-01T00:00:00Z', 100, true],
      ['2026-01-01T00:01:00Z', 300, false],
      ['2026-01-01T00:02:00Z', 50, false],
    ])
    expect(option.series[0].data[2][1]).not.toBe(0)
  })

  it('does not initialize a chart without data', () => {
    const wrapper = mount(LatencyChart, { props: { series: [] } })
    expect(wrapper.text()).toContain('No latency data.')
    expect(init).not.toHaveBeenCalled()
  })

  it('resizes and disposes with the component lifecycle', async () => {
    const wrapper = mount(LatencyChart, {
      props: {
        series: [
          { checked_at: '2026-01-01T00:00:00Z', latency_ms: 100, success: true, http_status: 200 },
        ],
      },
    })
    const chart = init.mock.results[0]!.value
    window.dispatchEvent(new Event('resize'))
    expect(chart.resize).toHaveBeenCalled()
    wrapper.unmount()
    expect(chart.dispose).toHaveBeenCalled()
  })
})
