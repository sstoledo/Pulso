import { createRouter, createWebHistory } from 'vue-router'

import LoginView from '../features/auth/LoginView.vue'
import RegisterView from '../features/auth/RegisterView.vue'
import HomeView from '../views/HomeView.vue'
import MonitorsView from '../views/MonitorsView.vue'
import MonitorDetailView from '../views/MonitorDetailView.vue'
import PublicStatusView from '../views/PublicStatusView.vue'
import { useAuthStore } from './auth'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView,
    },
    {
      path: '/login',
      name: 'login',
      component: LoginView,
      meta: { guestOnly: true },
    },
    {
      path: '/register',
      name: 'register',
      component: RegisterView,
      meta: { guestOnly: true },
    },
    {
      path: '/monitors',
      name: 'monitors',
      component: MonitorsView,
      meta: { requiresAuth: true },
    },
    {
      path: '/monitors/:id',
      name: 'monitor-detail',
      component: MonitorDetailView,
      meta: { requiresAuth: true },
    },
    {
      path: '/status/:slug',
      name: 'public-status',
      component: PublicStatusView,
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (auth.status === 'unknown') {
    await auth.restore()
  }
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.meta.guestOnly && auth.isAuthenticated) {
    return { name: 'monitors' }
  }
})
