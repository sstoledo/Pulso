<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'

import { useAuthStore } from '../app/auth'
import { incidentsApi } from '../features/incidents/api'
import { monitorsApi } from '../features/monitors/api'
import LatencyChart from '../features/monitors/LatencyChart.vue'

const auth = useAuthStore()
const route = useRoute()
const organizationID = computed(() => auth.organizations[0]?.id ?? '')
const monitorID = computed(() => String(route.params.id))
const page = ref(1)
const incidentPage = ref(1)

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
const checksQuery = useQuery({
  queryKey: ['monitor-checks', organizationID, monitorID, page],
  queryFn: () => monitorsApi.checks(organizationID.value, monitorID.value, page.value, 10),
  refetchInterval: 15000,
})
const incidentsQuery = useQuery({
  queryKey: ['monitor-incidents', organizationID, monitorID, incidentPage],
  queryFn: () => incidentsApi.list(organizationID.value, monitorID.value, incidentPage.value, 10),
  refetchInterval: 15000,
})

const loading = computed(
  () =>
    monitorQuery.isPending.value ||
    metricsQuery.isPending.value ||
    checksQuery.isPending.value ||
    incidentsQuery.isPending.value,
)
const failed = computed(
  () =>
    monitorQuery.isError.value ||
    metricsQuery.isError.value ||
    checksQuery.isError.value ||
    incidentsQuery.isError.value,
)
const monitor = computed(() => monitorQuery.data.value)
const metrics = computed(() => metricsQuery.data.value)
const checks = computed(() => checksQuery.data.value)
const incidents = computed(() => incidentsQuery.data.value)

function availability(value: number | null): string {
  return value === null ? '—' : `${(value * 100).toFixed(1)}%`
}

function milliseconds(value: number | null): string {
  return value === null ? '—' : `${value} ms`
}

function duration(value: number | null): string {
  if (value === null) return '—'
  const seconds = Math.floor(value / 1000)
  if (seconds < 60) return `${seconds}s`
  return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
}

function date(value: string | null): string {
  return value ? new Date(value).toLocaleString() : '—'
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
    <template v-else-if="monitor && metrics && checks && incidents">
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

      <LatencyChart :series="metrics.series" />

      <section aria-labelledby="checks-title" class="mt-8">
        <h2 id="checks-title" class="text-xl font-semibold">Check history</h2>
        <div
          v-if="checks.total === 0"
          class="mt-3 rounded-lg border border-dashed border-slate-300 bg-white p-6 text-center text-slate-600"
        >
          No checks yet.
        </div>
        <div v-else class="mt-3 overflow-x-auto rounded-xl border border-slate-200 bg-white">
          <table class="w-full text-left text-sm">
            <thead class="bg-slate-50 text-slate-600">
              <tr>
                <th class="px-4 py-3">Time</th>
                <th class="px-4 py-3">Result</th>
                <th class="px-4 py-3">HTTP</th>
                <th class="px-4 py-3">Latency</th>
                <th class="px-4 py-3">Error</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="result in checks.items" :key="result.id" class="border-t border-slate-100">
                <td class="px-4 py-3">{{ date(result.checked_at) }}</td>
                <td class="px-4 py-3">{{ result.success ? 'SUCCESS' : 'FAILURE' }}</td>
                <td class="px-4 py-3">{{ result.http_status ?? '—' }}</td>
                <td class="px-4 py-3">{{ result.latency_ms }} ms</td>
                <td class="px-4 py-3">{{ result.error_summary ?? '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <nav
          v-if="checks.total_pages > 0"
          aria-label="Check history pages"
          class="mt-3 flex items-center gap-3 text-sm"
        >
          <button
            id="previous-checks"
            type="button"
            :disabled="page <= 1"
            class="rounded border px-3 py-1 disabled:opacity-50"
            @click="page--"
          >
            Previous
          </button>
          <span>Page {{ checks.page }} of {{ checks.total_pages }}</span>
          <button
            id="next-checks"
            type="button"
            :disabled="page >= checks.total_pages"
            class="rounded border px-3 py-1 disabled:opacity-50"
            @click="page++"
          >
            Next
          </button>
        </nav>
      </section>

      <section aria-labelledby="incidents-title" class="mt-8">
        <h2 id="incidents-title" class="text-xl font-semibold">Incident history</h2>
        <div
          v-if="incidents.total === 0"
          class="mt-3 rounded-lg border border-dashed border-slate-300 bg-white p-6 text-center text-slate-600"
        >
          No incidents yet.
        </div>
        <div v-else class="mt-3 overflow-x-auto rounded-xl border border-slate-200 bg-white">
          <table class="w-full text-left text-sm">
            <thead class="bg-slate-50 text-slate-600">
              <tr>
                <th class="px-4 py-3">Opened</th>
                <th class="px-4 py-3">Resolved</th>
                <th class="px-4 py-3">Duration</th>
                <th class="px-4 py-3">Status</th>
                <th class="px-4 py-3">Reason</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="incident in incidents.items"
                :key="incident.id"
                class="border-t border-slate-100"
              >
                <td class="px-4 py-3">{{ date(incident.opened_at) }}</td>
                <td class="px-4 py-3">{{ date(incident.resolved_at) }}</td>
                <td class="px-4 py-3">{{ duration(incident.duration_ms) }}</td>
                <td class="px-4 py-3">{{ incident.status }}</td>
                <td class="px-4 py-3">{{ incident.reason }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <nav
          v-if="incidents.total_pages > 0"
          aria-label="Incident history pages"
          class="mt-3 flex items-center gap-3 text-sm"
        >
          <button
            id="previous-incidents"
            type="button"
            :disabled="incidentPage <= 1"
            class="rounded border px-3 py-1 disabled:opacity-50"
            @click="incidentPage--"
          >
            Previous
          </button>
          <span>Page {{ incidents.page }} of {{ incidents.total_pages }}</span>
          <button
            id="next-incidents"
            type="button"
            :disabled="incidentPage >= incidents.total_pages"
            class="rounded border px-3 py-1 disabled:opacity-50"
            @click="incidentPage++"
          >
            Next
          </button>
        </nav>
      </section>
    </template>
  </section>
</template>
