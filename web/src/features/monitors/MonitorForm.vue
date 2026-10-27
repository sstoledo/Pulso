<script setup lang="ts">
import { reactive, watch } from 'vue'

import type { Monitor, MonitorInput } from './types'

const props = defineProps<{
  monitor?: Monitor | null
  submitting: boolean
  serverError: string
}>()
const emit = defineEmits<{
  save: [input: MonitorInput]
  cancel: []
}>()

const form = reactive<MonitorInput>({
  name: '',
  url: '',
  interval_seconds: 60,
  timeout_ms: 5000,
  expected_status_min: 200,
  expected_status_max: 399,
  show_public: false,
})
const errors = reactive<Record<string, string>>({})

watch(
  () => props.monitor,
  (monitor) => {
    Object.assign(form, {
      name: monitor?.name ?? '',
      url: monitor?.url ?? '',
      interval_seconds: monitor?.interval_seconds ?? 60,
      timeout_ms: monitor?.timeout_ms ?? 5000,
      expected_status_min: monitor?.expected_status_min ?? 200,
      expected_status_max: monitor?.expected_status_max ?? 399,
      show_public: monitor?.show_public ?? false,
    })
  },
  { immediate: true },
)

function submit() {
  Object.keys(errors).forEach((key) => delete errors[key])
  if (!form.name.trim() || form.name.trim().length > 100) {
    errors.name = 'Enter between 1 and 100 characters.'
  }
  try {
    const target = new URL(form.url)
    if (
      !['http:', 'https:'].includes(target.protocol) ||
      !target.host ||
      target.username ||
      target.password ||
      target.hash
    ) {
      errors.url = 'Enter an HTTP(S) URL without credentials or fragments.'
    }
  } catch {
    errors.url = 'Enter an HTTP(S) URL without credentials or fragments.'
  }
  if (![30, 60, 300].includes(form.interval_seconds)) {
    errors.interval_seconds = 'Choose 30, 60, or 300 seconds.'
  }
  if (form.timeout_ms < 100 || form.timeout_ms > 10000) {
    errors.timeout_ms = 'Enter between 100 and 10,000 milliseconds.'
  }
  if (form.expected_status_min < 100 || form.expected_status_min > 599) {
    errors.expected_status_min = 'Enter a status between 100 and 599.'
  }
  if (
    form.expected_status_max < 100 ||
    form.expected_status_max > 599 ||
    form.expected_status_min > form.expected_status_max
  ) {
    errors.expected_status_max =
      'Maximum status must be valid and greater than or equal to minimum status.'
  }
  if (Object.keys(errors).length === 0) {
    emit('save', { ...form, name: form.name.trim(), url: form.url.trim() })
  }
}
</script>

<template>
  <section
    aria-labelledby="monitor-form-title"
    class="mb-8 rounded-xl border border-slate-200 bg-white p-6 shadow-sm"
  >
    <h2 id="monitor-form-title" class="text-xl font-semibold">
      {{ monitor ? 'Edit monitor' : 'Create monitor' }}
    </h2>
    <form class="mt-5 grid gap-5 md:grid-cols-2" novalidate @submit.prevent="submit">
      <div>
        <label for="monitor-name" class="block text-sm font-medium">Name</label>
        <input
          id="monitor-name"
          v-model="form.name"
          class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
        />
        <p v-if="errors.name" class="mt-1 text-sm text-red-700">{{ errors.name }}</p>
      </div>
      <div>
        <label for="monitor-url" class="block text-sm font-medium">HTTP(S) URL</label>
        <input
          id="monitor-url"
          v-model="form.url"
          type="url"
          class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
        />
        <p v-if="errors.url" class="mt-1 text-sm text-red-700">{{ errors.url }}</p>
      </div>
      <div>
        <label for="monitor-interval" class="block text-sm font-medium">Check interval</label>
        <select
          id="monitor-interval"
          v-model.number="form.interval_seconds"
          class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
        >
          <option :value="30">30 seconds</option>
          <option :value="60">1 minute</option>
          <option :value="300">5 minutes</option>
        </select>
        <p v-if="errors.interval_seconds" class="mt-1 text-sm text-red-700">
          {{ errors.interval_seconds }}
        </p>
      </div>
      <div>
        <label for="monitor-timeout" class="block text-sm font-medium">Timeout (ms)</label>
        <input
          id="monitor-timeout"
          v-model.number="form.timeout_ms"
          type="number"
          min="100"
          max="10000"
          class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
        />
        <p v-if="errors.timeout_ms" class="mt-1 text-sm text-red-700">{{ errors.timeout_ms }}</p>
      </div>
      <div>
        <label for="monitor-status-min" class="block text-sm font-medium"
          >Expected status minimum</label
        >
        <input
          id="monitor-status-min"
          v-model.number="form.expected_status_min"
          type="number"
          min="100"
          max="599"
          class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
        />
        <p v-if="errors.expected_status_min" class="mt-1 text-sm text-red-700">
          {{ errors.expected_status_min }}
        </p>
      </div>
      <div>
        <label for="monitor-status-max" class="block text-sm font-medium"
          >Expected status maximum</label
        >
        <input
          id="monitor-status-max"
          v-model.number="form.expected_status_max"
          type="number"
          min="100"
          max="599"
          class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
        />
        <p v-if="errors.expected_status_max" class="mt-1 text-sm text-red-700">
          {{ errors.expected_status_max }}
        </p>
      </div>
      <label class="flex items-center gap-2 text-sm md:col-span-2">
        <input v-model="form.show_public" type="checkbox" class="h-4 w-4" />
        Show on the public status page
      </label>
      <p v-if="serverError" role="alert" class="text-sm text-red-700 md:col-span-2">
        {{ serverError }}
      </p>
      <div class="flex gap-3 md:col-span-2">
        <button
          type="submit"
          :disabled="submitting"
          class="rounded-md bg-emerald-700 px-4 py-2 font-medium text-white disabled:opacity-60"
        >
          {{ submitting ? 'Saving…' : monitor ? 'Save changes' : 'Create monitor' }}
        </button>
        <button
          type="button"
          class="rounded-md border border-slate-300 px-4 py-2 font-medium"
          @click="emit('cancel')"
        >
          Cancel
        </button>
      </div>
    </form>
  </section>
</template>
