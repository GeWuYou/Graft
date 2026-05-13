import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it } from 'vitest';
import { createMemoryHistory, createRouter } from 'vue-router';

import { useAuthStore } from '@/stores/auth';
import {
  createI18nPlugin,
  createTDesignStubs,
  createTestingPinia,
} from '@/test/helpers';

import BasicLayout from './BasicLayout.vue';

describe('BasicLayout', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('renders the current user and visible navigation items', async () => {
    const pinia = createTestingPinia();
    const authStore = useAuthStore(pinia);
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/dashboard',
          name: 'dashboard',
          component: {
            template: '<div>dashboard</div>',
          },
          meta: {
            hideInMenu: false,
            requiresAuth: true,
            title: '仪表盘',
            titleKey: 'navigation.dashboard',
          },
        },
      ],
    });

    authStore.login('admin');
    await router.push('/dashboard');
    await router.isReady();

    const wrapper = mount(BasicLayout, {
      global: {
        plugins: [pinia, router, createI18nPlugin(pinia)],
        stubs: {
          ...createTDesignStubs(),
          RouterView: {
            template: '<div data-stub="router-view" />',
          },
        },
      },
    });

    expect(wrapper.text()).toContain('admin');
    expect(wrapper.text()).toContain('仪表盘');
    expect(wrapper.find('[data-stub="router-view"]').exists()).toBe(true);
  });
});
