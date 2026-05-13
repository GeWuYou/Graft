import { beforeEach, describe, expect, it } from 'vitest';
import { createMemoryHistory, createRouter } from 'vue-router';

import { useAuthStore } from '@/stores/auth';
import { createTestingPinia } from '@/test/helpers';

import { setupRouteGuards } from './route-guards';

const routes = [
  {
    path: '/login',
    name: 'login',
    component: {
      template: '<div>login</div>',
    },
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: {
      template: '<div>dashboard</div>',
    },
    meta: {
      requiresAuth: true,
      permission: 'dashboard.view',
    },
  },
  {
    path: '/reports',
    name: 'reports',
    component: {
      template: '<div>reports</div>',
    },
    meta: {
      requiresAuth: true,
      permission: 'reports.view',
    },
  },
  {
    path: '/unauthorized',
    name: 'unauthorized',
    component: {
      template: '<div>unauthorized</div>',
    },
    meta: {
      requiresAuth: true,
    },
  },
];

describe('setupRouteGuards', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('redirects unauthenticated users to login with the original path', async () => {
    const pinia = createTestingPinia();
    const router = createRouter({
      history: createMemoryHistory(),
      routes,
    });

    setupRouteGuards(router, pinia);
    await router.push('/reports');
    await router.isReady();

    expect(router.currentRoute.value.name).toBe('login');
    expect(router.currentRoute.value.query.redirect).toBe('/reports');
  });

  it('redirects authenticated users without permission to the unauthorized page', async () => {
    const pinia = createTestingPinia();
    const authStore = useAuthStore(pinia);
    const router = createRouter({
      history: createMemoryHistory(),
      routes,
    });

    authStore.login('admin');
    setupRouteGuards(router, pinia);
    await router.push('/reports');
    await router.isReady();

    expect(router.currentRoute.value.name).toBe('unauthorized');
    expect(router.currentRoute.value.query.from).toBe('/reports');
    expect(router.currentRoute.value.query.fallback).toBe('/dashboard');
  });
});
