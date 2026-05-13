import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createMemoryHistory, createRouter } from 'vue-router';

import { useAuthStore } from '@/stores/auth';
import {
  createI18nPlugin,
  createTDesignStubs,
  createTestingPinia,
} from '@/test/helpers';

vi.mock('tdesign-vue-next', async () => {
  const actual =
    await vi.importActual<typeof import('tdesign-vue-next')>(
      'tdesign-vue-next',
    );

  return {
    ...actual,
    MessagePlugin: {
      success: vi.fn(),
      warning: vi.fn(),
    },
  };
});

import { MessagePlugin } from 'tdesign-vue-next';

import LoginPage from './LoginPage.vue';

describe('LoginPage', () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.clearAllMocks();
  });

  it('logs in and redirects to the requested page', async () => {
    const pinia = createTestingPinia();
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/login',
          name: 'login',
          component: LoginPage,
        },
        {
          path: '/dashboard',
          name: 'dashboard',
          component: {
            template: '<div>dashboard</div>',
          },
        },
      ],
    });

    await router.push('/login?redirect=/dashboard');
    await router.isReady();

    const wrapper = mount(LoginPage, {
      global: {
        plugins: [pinia, router, createI18nPlugin(pinia)],
        stubs: createTDesignStubs(),
      },
    });

    const inputs = wrapper.findAll('input');

    await inputs[0].setValue('admin');
    await inputs[1].setValue('secret');
    await wrapper.find('form').trigger('submit.prevent');
    await flushPromises();

    expect(useAuthStore().isAuthenticated).toBe(true);
    expect(router.currentRoute.value.fullPath).toBe('/dashboard');
    expect(MessagePlugin.success).toHaveBeenCalledTimes(1);
  });

  it('warns when credentials are incomplete', async () => {
    const pinia = createTestingPinia();
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/login',
          name: 'login',
          component: LoginPage,
        },
      ],
    });

    await router.push('/login');
    await router.isReady();

    const wrapper = mount(LoginPage, {
      global: {
        plugins: [pinia, router, createI18nPlugin(pinia)],
        stubs: createTDesignStubs(),
      },
    });

    const inputs = wrapper.findAll('input');

    await inputs[0].setValue('');
    await inputs[1].setValue('');
    await wrapper.find('form').trigger('submit.prevent');

    expect(useAuthStore().isAuthenticated).toBe(false);
    expect(MessagePlugin.warning).toHaveBeenCalledTimes(1);
  });
});
