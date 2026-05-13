import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it } from 'vitest';
import { createMemoryHistory, createRouter } from 'vue-router';

import { useAuthStore } from '@/stores/auth';
import {
  createI18nPlugin,
  createTDesignStubs,
  createTestingPinia,
} from '@/test/helpers';

import UnauthorizedPage from './UnauthorizedPage.vue';

describe('UnauthorizedPage', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('falls back to the dashboard when the query path is unsafe', async () => {
    const pinia = createTestingPinia();
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/unauthorized',
          name: 'unauthorized',
          component: UnauthorizedPage,
        },
        {
          path: '/dashboard',
          name: 'dashboard',
          component: {
            template: '<div>dashboard</div>',
          },
        },
        {
          path: '/login',
          name: 'login',
          component: {
            template: '<div>login</div>',
          },
        },
      ],
    });

    await router.push('/unauthorized?fallback=https://example.com');
    await router.isReady();

    const wrapper = mount(UnauthorizedPage, {
      global: {
        plugins: [pinia, router, createI18nPlugin(pinia)],
        stubs: createTDesignStubs(),
      },
    });

    await wrapper.find('button').trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.fullPath).toBe('/dashboard');
  });

  it('logs out before switching account', async () => {
    const pinia = createTestingPinia();
    const authStore = useAuthStore(pinia);
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/unauthorized',
          name: 'unauthorized',
          component: UnauthorizedPage,
        },
        {
          path: '/login',
          name: 'login',
          component: {
            template: '<div>login</div>',
          },
        },
      ],
    });

    authStore.login('admin');
    await router.push('/unauthorized');
    await router.isReady();

    const wrapper = mount(UnauthorizedPage, {
      global: {
        plugins: [pinia, router, createI18nPlugin(pinia)],
        stubs: createTDesignStubs(),
      },
    });

    await wrapper.findAll('button')[1].trigger('click');
    await flushPromises();

    expect(authStore.isAuthenticated).toBe(false);
    expect(router.currentRoute.value.fullPath).toBe('/login');
  });
});
