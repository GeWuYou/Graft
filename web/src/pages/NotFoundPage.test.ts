import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it } from 'vitest';
import { createMemoryHistory, createRouter } from 'vue-router';

import {
  createI18nPlugin,
  createTDesignStubs,
  createTestingPinia,
} from '@/test/helpers';

import NotFoundPage from './NotFoundPage.vue';

describe('NotFoundPage', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  function createTestRouter() {
    return createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/missing',
          name: 'not-found',
          component: NotFoundPage,
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
  }

  async function mountPage() {
    const pinia = createTestingPinia();
    const router = createTestRouter();

    await router.push('/missing');
    await router.isReady();

    const wrapper = mount(NotFoundPage, {
      global: {
        plugins: [pinia, router, createI18nPlugin(pinia)],
        stubs: createTDesignStubs(),
      },
    });

    return { router, wrapper };
  }

  it('navigates to the dashboard entrypoint', async () => {
    const { router, wrapper } = await mountPage();

    await wrapper.findAll('button')[0].trigger('click');
    await flushPromises();
    expect(router.currentRoute.value.fullPath).toBe('/dashboard');
  });

  it('navigates to the login entrypoint', async () => {
    const { router, wrapper } = await mountPage();

    await wrapper.findAll('button')[1].trigger('click');
    await flushPromises();
    expect(router.currentRoute.value.fullPath).toBe('/login');
  });
});
