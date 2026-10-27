import { createPinia } from 'pinia'
import { VueQueryPlugin } from '@tanstack/vue-query'
import { createApp } from 'vue'

import App from './App.vue'
import { queryClient } from './app/query'
import { router } from './app/router'
import './style.css'

createApp(App).use(createPinia()).use(router).use(VueQueryPlugin, { queryClient }).mount('#app')
