<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'

import { ApiError } from '../../app/api'
import { useAuthStore } from '../../app/auth'
import { validEmail } from './validation'

const auth = useAuthStore()
const router = useRouter()
const form = reactive({ name: '', email: '', password: '' })
const errors = reactive<Record<string, string>>({})
const serverError = ref('')
const submitting = ref(false)

async function submit() {
  serverError.value = ''
  Object.keys(errors).forEach((key) => delete errors[key])
  if (!form.name.trim() || form.name.trim().length > 80) {
    errors.name = 'Enter between 1 and 80 characters.'
  }
  if (!validEmail(form.email)) {
    errors.email = 'Enter a valid email address.'
  }
  if (form.password.length < 12 || form.password.length > 128) {
    errors.password = 'Enter between 12 and 128 characters.'
  }
  if (Object.keys(errors).length > 0) {
    return
  }

  submitting.value = true
  try {
    await auth.register({
      name: form.name.trim(),
      email: form.email.trim(),
      password: form.password,
    })
    await router.push({ name: 'monitors' })
  } catch (error) {
    if (error instanceof ApiError && error.details) {
      Object.assign(errors, error.details)
    }
    serverError.value = error instanceof ApiError ? error.message : 'Unable to register. Try again.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section
    aria-labelledby="register-title"
    class="mx-auto max-w-md rounded-xl border border-slate-200 bg-white p-8 shadow-sm"
  >
    <h1 id="register-title" class="text-2xl font-bold">Create your account</h1>
    <p class="mt-2 text-sm text-slate-600">
      Your owner organization will be created automatically.
    </p>
    <form class="mt-6 space-y-5" novalidate @submit.prevent="submit">
      <div>
        <label for="name" class="block text-sm font-medium">Name</label>
        <input
          id="name"
          v-model="form.name"
          autocomplete="name"
          class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
        />
        <p v-if="errors.name" class="mt-1 text-sm text-red-700">{{ errors.name }}</p>
      </div>
      <div>
        <label for="email" class="block text-sm font-medium">Email</label>
        <input
          id="email"
          v-model="form.email"
          type="email"
          autocomplete="email"
          class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
        />
        <p v-if="errors.email" class="mt-1 text-sm text-red-700">{{ errors.email }}</p>
      </div>
      <div>
        <label for="password" class="block text-sm font-medium">Password</label>
        <input
          id="password"
          v-model="form.password"
          type="password"
          autocomplete="new-password"
          class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
        />
        <p v-if="errors.password" class="mt-1 text-sm text-red-700">{{ errors.password }}</p>
        <p v-else class="mt-1 text-xs text-slate-500">Use 12–128 characters.</p>
      </div>
      <p v-if="serverError" role="alert" class="text-sm text-red-700">{{ serverError }}</p>
      <button
        type="submit"
        :disabled="submitting"
        class="w-full rounded-md bg-emerald-700 px-4 py-2 font-medium text-white hover:bg-emerald-800 disabled:opacity-60"
      >
        {{ submitting ? 'Creating account…' : 'Create account' }}
      </button>
    </form>
    <p class="mt-5 text-center text-sm text-slate-600">
      Already registered?
      <RouterLink to="/login" class="font-medium text-emerald-700">Log in</RouterLink>
    </p>
  </section>
</template>
