import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { VueQueryPlugin } from '@tanstack/vue-query'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import App from './App.vue'

describe('App', () => {
  it('renders the monitoring shell with application plugins', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/',
          component: { template: '<h1>Service monitoring</h1>' },
        },
      ],
    })
    await router.push('/')
    await router.isReady()

    const wrapper = mount(App, {
      global: {
        plugins: [router, createPinia(), VueQueryPlugin],
      },
    })

    expect(wrapper.get('header').text()).toContain('Pulso')
    expect(wrapper.get('main').text()).toContain('Service monitoring')
  })
})
