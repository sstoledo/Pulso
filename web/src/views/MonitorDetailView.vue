<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { useRoute } from 'vue-router'

import { useAuthStore } from '../app/auth'
import { monitorsApi } from '../features/monitors/api'

const auth = useAuthStore()
const route = useRoute()
const organizationID = computed(() => auth.organizations[0]?.id ?? '')
const monitorID = computed(() => String(route.params.id))

const monitorQuery = useQuery({
  queryKey: ['monitor', organizationID, monitorID],
  queryFn: () => monitorsApi.get(organizationID.value, monitorID.value),
  refetchInterval: 15000,
})
const metricsQuery = useQuery({
  queryKey: ['monitor-metrics', organizationID, monitorID],
  queryFn: () => monitorsApi.metrics(organizationID.value, monitorID.value),
  refetchInterval: 15000,
})

const loading = computed(() => monitorQuery.isPending.value || metricsQuery.isPending.value)
const failed = computed(() => monitorQuery.isError.value || metricsQuery.isError.value)
const monitor = computed(() => monitorQuery.data.value)
const metrics = computed(() => metricsQuery.data.value)

function availability(value: number | null): string {
  return value === null ? '—' : `${(value * 100).toFixed(1)}%`
}

function milliseconds(value: number | null): string {
  return value === null ? '—' : `${value} ms`
}
</script>

<template>
  <section aria-labelledby="monitor-title">
    <RouterLink to="/monitors" class="text-sm text-emerald-700">← Monitors</RouterLink>
    <div v-if="loading" class="mt-8 rounded-lg bg-white p-8 text-center text-slate-600">
      Loading monitor…
    </div>
    <div
      v-else-if="failed"
      role="alert"
      class="mt-8 rounded-lg border border-red-200 bg-red-50 p-8 text-center text-red-700"
    >
      Unable to load monitor.
    </div>
    <template v-else-if="monitor && metrics">
      <div class="mt-2 flex items-center justify-between">
        <div>
          <h1 id="monitor-title" class="text-3xl font-bold tracking-tight">{{ monitor.name }}</h1>
          <p class="mt-1 text-sm text-slate-500">{{ monitor.url }}</p>
        </div>
        <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-semibold">{{
          monitor.status
        }}</span>
      </div>

      <div class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div class="rounded-lg border border-slate-200 bg-white p-4">
          <p class="text-sm text-slate-500">Current status</p>
          <p class="mt-1 text-xl font-bold">{{ monitor.status }}</p>
        </div>
        <div class="rounded-lg border border-slate-200 bg-white p-4">
          <p class="text-sm text-slate-500">Availability (24h)</p>
          <p class="mt-1 text-xl font-bold">{{ availability(metrics.availability_24h) }}</p>
        </div>
        <div class="rounded-lg border border-slate-200 bg-white p-4">
          <p class="text-sm text-slate-500">Average latency</p>
          <p class="mt-1 text-xl font-bold">{{ milliseconds(metrics.average_latency_ms) }}</p>
        </div>
        <div class="rounded-lg border border-slate-200 bg-white p-4">
          <p class="text-sm text-slate-500">p95 latency</p>
          <p class="mt-1 text-xl font-bold">{{ milliseconds(metrics.p95_latency_ms) }}</p>
        </div>
      </div>
    </template>
  </section>
</template>
