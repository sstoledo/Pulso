<script setup lang="ts">
import { useAuthStore } from '../app/auth'

const auth = useAuthStore()
</script>

<template>
  <section aria-labelledby="monitors-title">
    <p class="text-sm font-medium text-emerald-700">{{ auth.organizations[0]?.name }}</p>
    <h1 id="monitors-title" class="mt-1 text-3xl font-bold tracking-tight">Monitors</h1>
    <div class="mt-6 rounded-xl border border-dashed border-slate-300 bg-white p-8 text-center">
      <h2 class="text-lg font-semibold">No monitors yet</h2>
      <p class="mt-2 text-slate-600">Monitor management will be available in the next delivery.</p>
    </div>
  </section>
</template>
