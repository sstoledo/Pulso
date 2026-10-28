<script setup lang="ts">
import * as echarts from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

echarts.use([LineChart, GridComponent, TooltipComponent, CanvasRenderer])

import type { MetricPoint } from './types'

const props = defineProps<{ series: MetricPoint[] }>()
const container = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | undefined

function render() {
  if (!container.value) return
  if (props.series.length === 0) {
    chart?.dispose()
    chart = undefined
    return
  }
  chart ??= echarts.init(container.value)
  chart.setOption({
    animation: false,
    grid: { left: 48, right: 24, top: 24, bottom: 40 },
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'time' },
    yAxis: { type: 'value', name: 'ms', min: 0 },
    series: [
      {
        name: 'Latency',
        type: 'line',
        data: props.series.map((point) => [point.checked_at, point.latency_ms, point.success]),
        showSymbol: true,
        symbolSize: 7,
        itemStyle: {
          color: (params: { data: [string, number, boolean] }) =>
            params.data[2] ? '#059669' : '#dc2626',
        },
      },
    ],
  })
}

function resize() {
  chart?.resize()
}

onMounted(() => {
  render()
  window.addEventListener('resize', resize)
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', resize)
  chart?.dispose()
})
watch(() => props.series, render, { deep: true })
</script>

<template>
  <section aria-labelledby="latency-chart-title" class="mt-8">
    <h2 id="latency-chart-title" class="text-xl font-semibold">Latency (24h)</h2>
    <p
      v-if="series.length === 0"
      class="mt-3 rounded-lg border border-dashed border-slate-300 bg-white p-6 text-center text-slate-600"
    >
      No latency data.
    </p>
    <div
      v-else
      ref="container"
      role="img"
      aria-label="Latency over the last 24 hours"
      class="mt-3 h-72 rounded-xl border border-slate-200 bg-white"
    />
  </section>
</template>
