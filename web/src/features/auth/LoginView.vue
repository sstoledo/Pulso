<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { ApiError } from '../../app/api'
import { useAuthStore } from '../../app/auth'
import { safeReturnPath, validEmail } from './validation'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const form = reactive({ email: '', password: '' })
const errors = reactive<Record<string, string>>({})
const serverError = ref('')
const submitting = ref(false)

async function submit() {
  serverError.value = ''
  Object.keys(errors).forEach((key) => delete errors[key])
  if (!validEmail(form.email)) {
    errors.email = 'Enter a valid email address.'
  }
  if (!form.password) {
    errors.password = 'Enter your password.'
  }
  if (Object.keys(errors).length > 0) {
    return
  }

  submitting.value = true
  try {
    await auth.login({ email: form.email.trim(), password: form.password })
    await router.push(safeReturnPath(route.query.redirect))
  } catch (error) {
    serverError.value = error instanceof ApiError ? error.message : 'Unable to log in. Try again.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section
    aria-labelledby="login-title"
    class="mx-auto max-w-md rounded-xl border border-slate-200 bg-white p-8 shadow-sm"
  >
    <h1 id="login-title" class="text-2xl font-bold">Log in to Pulso</h1>
    <p class="mt-2 text-sm text-slate-600">Access your organization's monitoring dashboard.</p>
    <form class="mt-6 space-y-5" novalidate @submit.prevent="submit">
      <div>
        <label for="email" class="block text-sm font-medium">Email</label>
        <input
          id="email"
          v-model="form.email"
          type="email"
          autocomplete="email"
          class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
          :aria-describedby="errors.email ? 'email-error' : undefined"
        />
        <p v-if="errors.email" id="email-error" class="mt-1 text-sm text-red-700">
          {{ errors.email }}
        </p>
      </div>
      <div>
        <label for="password" class="block text-sm font-medium">Password</label>
        <input
          id="password"
          v-model="form.password"
          type="password"
          autocomplete="current-password"
          class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
          :aria-describedby="errors.password ? 'password-error' : undefined"
        />
        <p v-if="errors.password" id="password-error" class="mt-1 text-sm text-red-700">
          {{ errors.password }}
        </p>
      </div>
      <p v-if="serverError" role="alert" class="text-sm text-red-700">{{ serverError }}</p>
      <button
        type="submit"
        :disabled="submitting"
        class="w-full rounded-md bg-emerald-700 px-4 py-2 font-medium text-white hover:bg-emerald-800 disabled:opacity-60"
      >
        {{ submitting ? 'Logging in…' : 'Log in' }}
      </button>
    </form>
    <p class="mt-5 text-center text-sm text-slate-600">
      New to Pulso?
      <RouterLink to="/register" class="font-medium text-emerald-700">Create an account</RouterLink>
    </p>
  </section>
</template>
