import { createPinia } from 'pinia'
import { VueQueryPlugin } from '@tanstack/vue-query'
import { createApp } from 'vue'

import App from './App.vue'
import { setUnauthorizedHandler } from './app/api'
import { useAuthStore } from './app/auth'
import { queryClient } from './app/query'
import { router } from './app/router'
import './style.css'

const pinia = createPinia()
const auth = useAuthStore(pinia)
setUnauthorizedHandler(() => auth.handleUnauthorized())

createApp(App).use(pinia).use(router).use(VueQueryPlugin, { queryClient }).mount('#app')
