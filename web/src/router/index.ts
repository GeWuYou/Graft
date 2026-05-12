import type { Pinia } from 'pinia';
import type { Router } from 'vue-router';

import { createRouter, createWebHistory } from 'vue-router';

import { setupRouteGuards } from '@/app/route-guards';

import { staticRoutes } from './routes';

export function createAppRouter(pinia: Pinia): Router {
  const router = createRouter({
    history: createWebHistory(import.meta.env.BASE_URL),
    routes: staticRoutes,
    scrollBehavior() {
      return { top: 0 };
    },
  });

  setupRouteGuards(router, pinia);

  return router;
}
