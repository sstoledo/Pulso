<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { useRoute } from 'vue-router'

import { ApiError } from '../app/api'
import { statusApi } from '../features/status/api'

const route = useRoute()
const slug = computed(() => String(route.params.slug))

const pageQuery = useQuery({
  queryKey: ['public-status', slug],
  queryFn: () => statusApi.get(slug.value),
  retry: false,
})

const page = computed(() => pageQuery.data.value)
const notFound = computed(
  () => pageQuery.error.value instanceof ApiError && pageQuery.error.value.status === 404,
)

function availability(value: number | null): string {
  return value === null ? 'No data' : `${(value * 100).toFixed(1)}%`
}

function date(value: string | null): string {
  return value ? new Date(value).toLocaleString() : '—'
}

function duration(value: number | null): string {
  if (value === null) return '—'
  const seconds = Math.floor(value / 1000)
  if (seconds < 60) return `${seconds}s`
  return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
}

function statusClass(status: string): string {
  if (status === 'UP') return 'bg-emerald-100 text-emerald-800'
  if (status === 'DOWN') return 'bg-red-100 text-red-800'
  return 'bg-slate-100 text-slate-700'
}
</script>

<template>
  <section aria-labelledby="status-page-title">
    <div
      v-if="pageQuery.isPending.value"
      class="rounded-lg bg-white p-8 text-center text-slate-600"
    >
      Loading status…
    </div>
    <div
      v-else-if="notFound"
      role="alert"
      class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-700"
    >
      <h1 id="status-page-title" class="text-2xl font-bold">Status page not found</h1>
      <p class="mt-2 text-sm text-slate-500">This page does not exist or is not published.</p>
    </div>
    <div
      v-else-if="pageQuery.isError.value"
      role="alert"
      class="rounded-lg border border-red-200 bg-red-50 p-8 text-center text-red-700"
    >
      Unable to load this status page.
    </div>
    <template v-else-if="page">
      <h1 id="status-page-title" class="text-3xl font-bold tracking-tight">{{ page.title }}</h1>
      <p class="mt-1 text-sm text-slate-500">Service status and recent incidents</p>

      <section aria-labelledby="public-monitors-title" class="mt-8">
        <h2 id="public-monitors-title" class="text-xl font-semibold">Services</h2>
        <div
          v-if="page.monitors.length === 0"
          class="mt-3 rounded-lg border border-dashed border-slate-300 bg-white p-6 text-center text-slate-600"
        >
          No public monitors yet.
        </div>
        <ul v-else class="mt-3 grid gap-4 sm:grid-cols-2">
          <li
            v-for="monitor in page.monitors"
            :key="monitor.name"
            class="rounded-lg border border-slate-200 bg-white p-4"
          >
            <div class="flex items-center justify-between gap-3">
              <p class="font-medium">{{ monitor.name }}</p>
              <span
                :class="statusClass(monitor.status)"
                class="rounded-full px-3 py-1 text-xs font-semibold"
              >
                {{ monitor.status }}
              </span>
            </div>
            <p class="mt-2 text-sm text-slate-500">
              Availability (24h):
              <span class="font-semibold">{{ availability(monitor.availability_24h) }}</span>
            </p>
          </li>
        </ul>
      </section>

      <section aria-labelledby="public-incidents-title" class="mt-8">
        <h2 id="public-incidents-title" class="text-xl font-semibold">Recent incidents</h2>
        <div
          v-if="page.incidents.length === 0"
          class="mt-3 rounded-lg border border-dashed border-slate-300 bg-white p-6 text-center text-slate-600"
        >
          No incidents reported.
        </div>
        <div v-else class="mt-3 overflow-x-auto rounded-xl border border-slate-200 bg-white">
          <table class="w-full text-left text-sm">
            <thead class="bg-slate-50 text-slate-600">
              <tr>
                <th class="px-4 py-3">Service</th>
                <th class="px-4 py-3">Opened</th>
                <th class="px-4 py-3">Resolved</th>
                <th class="px-4 py-3">Duration</th>
                <th class="px-4 py-3">Status</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="incident in page.incidents"
                :key="incident.monitor_name + incident.opened_at"
                class="border-t border-slate-100"
              >
                <td class="px-4 py-3">{{ incident.monitor_name }}</td>
                <td class="px-4 py-3">{{ date(incident.opened_at) }}</td>
                <td class="px-4 py-3">{{ date(incident.resolved_at) }}</td>
                <td class="px-4 py-3">{{ duration(incident.duration_ms) }}</td>
                <td class="px-4 py-3">{{ incident.status }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>
  </section>
</template>
