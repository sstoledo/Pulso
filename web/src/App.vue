<script setup lang="ts">
import { useRouter } from 'vue-router'

import { useAuthStore } from './app/auth'

const auth = useAuthStore()
const router = useRouter()

async function logout() {
  await auth.logout()
  await router.push({ name: 'home' })
}
</script>

<template>
  <div class="min-h-screen bg-slate-50 text-slate-900">
    <header class="border-b border-slate-200 bg-white">
      <div class="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
        <RouterLink
          to="/"
          class="text-xl font-bold tracking-tight text-emerald-700"
          aria-label="Pulso home"
        >
          Pulso
        </RouterLink>
        <nav aria-label="Main navigation" class="flex items-center gap-4 text-sm font-medium">
          <template v-if="auth.isAuthenticated">
            <RouterLink to="/monitors" class="text-slate-600 hover:text-emerald-700"
              >Monitors</RouterLink
            >
            <span class="text-slate-500">{{ auth.user?.name }}</span>
            <button type="button" class="text-emerald-700" @click="logout">Log out</button>
          </template>
          <template v-else>
            <RouterLink to="/login" class="text-slate-600 hover:text-emerald-700"
              >Log in</RouterLink
            >
            <RouterLink to="/register" class="text-emerald-700">Create account</RouterLink>
          </template>
        </nav>
      </div>
    </header>
    <main class="mx-auto max-w-7xl px-6 py-10">
      <RouterView />
    </main>
  </div>
</template>
