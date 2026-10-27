<script setup lang="ts">
import { keepPreviousData, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, ref } from 'vue'

import { ApiError } from '../app/api'
import { useAuthStore } from '../app/auth'
import { monitorsApi } from '../features/monitors/api'
import MonitorForm from '../features/monitors/MonitorForm.vue'
import type { Monitor, MonitorInput } from '../features/monitors/types'

const auth = useAuthStore()
const queryClient = useQueryClient()
const organizationID = computed(() => auth.organizations[0]?.id ?? '')
const page = ref(1)
const formOpen = ref(false)
const editing = ref<Monitor | null>(null)
const submitting = ref(false)
const actionError = ref('')

const { data, isPending, isError } = useQuery({
  queryKey: ['monitors', organizationID, page],
  queryFn: () => monitorsApi.list(organizationID.value, page.value, 10),
  placeholderData: keepPreviousData,
  refetchInterval: 15000,
})
const { data: summary } = useQuery({
  queryKey: ['monitor-summary', organizationID],
  queryFn: () => monitorsApi.summary(organizationID.value),
  refetchInterval: 15000,
})

function openCreate() {
  editing.value = null
  actionError.value = ''
  formOpen.value = true
}

function openEdit(monitor: Monitor) {
  editing.value = monitor
  actionError.value = ''
  formOpen.value = true
}

async function invalidate() {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ['monitors'] }),
    queryClient.invalidateQueries({ queryKey: ['monitor-summary'] }),
  ])
}

async function save(input: MonitorInput) {
  submitting.value = true
  actionError.value = ''
  try {
    if (editing.value) {
      await monitorsApi.update(organizationID.value, editing.value.id, input)
    } else {
      await monitorsApi.create(organizationID.value, input)
    }
    await invalidate()
    formOpen.value = false
    editing.value = null
  } catch (error) {
    actionError.value = error instanceof ApiError ? error.message : 'Unable to save the monitor.'
  } finally {
    submitting.value = false
  }
}

async function setEnabled(monitor: Monitor, enabled: boolean) {
  actionError.value = ''
  try {
    if (enabled) {
      await monitorsApi.resume(organizationID.value, monitor.id)
    } else {
      await monitorsApi.pause(organizationID.value, monitor.id)
    }
    await invalidate()
  } catch (error) {
    actionError.value = error instanceof ApiError ? error.message : 'Unable to update the monitor.'
  }
}

async function remove(monitor: Monitor) {
  if (!window.confirm(`Delete ${monitor.name}? This cannot be undone.`)) {
    return
  }
  actionError.value = ''
  try {
    await monitorsApi.delete(organizationID.value, monitor.id)
    await invalidate()
  } catch (error) {
    actionError.value = error instanceof ApiError ? error.message : 'Unable to delete the monitor.'
  }
}

function formatDate(value: string | null): string {
  return value ? new Date(value).toLocaleString() : '—'
}
</script>

<template>
  <section aria-labelledby="monitors-title">
    <div class="flex items-center justify-between">
      <div>
        <p class="text-sm font-medium text-emerald-700">{{ auth.organizations[0]?.name }}</p>
        <h1 id="monitors-title" class="mt-1 text-3xl font-bold tracking-tight">Monitors</h1>
      </div>
      <button
        id="create-monitor"
        type="button"
        class="rounded-md bg-emerald-700 px-4 py-2 font-medium text-white"
        @click="openCreate"
      >
        Create monitor
      </button>
    </div>

    <div class="mt-6 grid grid-cols-3 gap-4">
      <div
        v-for="status in ['UP', 'DOWN', 'UNKNOWN']"
        :key="status"
        class="rounded-lg border border-slate-200 bg-white p-4"
      >
        <p class="text-sm font-medium text-slate-500">{{ status }}</p>
        <p class="mt-1 text-2xl font-bold">
          {{ summary?.[status.toLowerCase() as 'up' | 'down' | 'unknown'] ?? 0 }}
        </p>
      </div>
    </div>

    <MonitorForm
      v-if="formOpen"
      :monitor="editing"
      :submitting="submitting"
      :server-error="actionError"
      class="mt-8"
      @save="save"
      @cancel="formOpen = false"
    />
    <p v-if="actionError && !formOpen" role="alert" class="mt-6 text-sm text-red-700">
      {{ actionError }}
    </p>

    <div v-if="isPending" class="mt-8 rounded-lg bg-white p-8 text-center text-slate-600">
      Loading monitors…
    </div>
    <div
      v-else-if="isError"
      role="alert"
      class="mt-8 rounded-lg border border-red-200 bg-red-50 p-8 text-center text-red-700"
    >
      Unable to load monitors.
    </div>
    <div
      v-else-if="!data || data.total === 0"
      class="mt-8 rounded-xl border border-dashed border-slate-300 bg-white p-8 text-center"
    >
      <h2 class="text-lg font-semibold">No monitors yet</h2>
      <p class="mt-2 text-slate-600">Create your first monitor to start checking a service.</p>
    </div>
    <template v-else>
      <div class="mt-8 overflow-x-auto rounded-xl border border-slate-200 bg-white">
        <table class="w-full text-left text-sm">
          <thead class="border-b border-slate-200 bg-slate-50 text-slate-600">
            <tr>
              <th scope="col" class="px-4 py-3">Name</th>
              <th scope="col" class="px-4 py-3">Status</th>
              <th scope="col" class="px-4 py-3">Last latency</th>
              <th scope="col" class="px-4 py-3">Last check</th>
              <th scope="col" class="px-4 py-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="monitor in data.items"
              :key="monitor.id"
              class="border-b border-slate-100 last:border-0"
            >
              <td class="px-4 py-3">
                <p class="font-medium">{{ monitor.name }}</p>
                <p class="max-w-xs truncate text-xs text-slate-500">{{ monitor.url }}</p>
              </td>
              <td class="px-4 py-3">
                <span class="rounded-full bg-slate-100 px-2 py-1 text-xs font-semibold">{{
                  monitor.status
                }}</span>
              </td>
              <td class="px-4 py-3">
                {{ monitor.last_latency_ms === null ? '—' : `${monitor.last_latency_ms} ms` }}
              </td>
              <td class="px-4 py-3">{{ formatDate(monitor.last_check_at) }}</td>
              <td class="space-x-2 whitespace-nowrap px-4 py-3">
                <button
                  data-action="edit-monitor"
                  type="button"
                  class="text-emerald-700"
                  @click="openEdit(monitor)"
                >
                  Edit
                </button>
                <button
                  v-if="monitor.enabled"
                  data-action="pause-monitor"
                  type="button"
                  class="text-amber-700"
                  @click="setEnabled(monitor, false)"
                >
                  Pause
                </button>
                <button
                  v-else
                  data-action="resume-monitor"
                  type="button"
                  class="text-emerald-700"
                  @click="setEnabled(monitor, true)"
                >
                  Resume
                </button>
                <button
                  data-action="delete-monitor"
                  type="button"
                  class="text-red-700"
                  @click="remove(monitor)"
                >
                  Delete
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <nav aria-label="Monitor pages" class="mt-4 flex items-center justify-between text-sm">
        <span>Page {{ data.page }} of {{ data.total_pages }} · {{ data.total }} monitors</span>
        <div class="space-x-2">
          <button
            id="previous-page"
            type="button"
            :disabled="page <= 1"
            class="rounded border px-3 py-1 disabled:opacity-50"
            @click="page--"
          >
            Previous
          </button>
          <button
            id="next-page"
            type="button"
            :disabled="page >= data.total_pages"
            class="rounded border px-3 py-1 disabled:opacity-50"
            @click="page++"
          >
            Next
          </button>
        </div>
      </nav>
    </template>
  </section>
</template>
