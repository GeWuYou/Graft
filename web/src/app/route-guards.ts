import type { Pinia } from 'pinia';
import type { RouteLocationNormalized, Router } from 'vue-router';

import { useAuthStore } from '@/stores/auth';
import { useNavigationStore } from '@/stores/navigation';

const UNAUTHORIZED_ROUTE_NAME = 'unauthorized';

/**
 * Centralizes the shell's current routing assumptions:
 * authentication is session-based, and permission checks rely on route meta until
 * the backend menu + permission payload is available.
 */
export function setupRouteGuards(router: Router, pinia: Pinia) {
  router.beforeEach((to: RouteLocationNormalized) => {
    const authStore = useAuthStore(pinia);
    const navigationStore = useNavigationStore(pinia);
    const requiresAuth = to.matched.some((record) => record.meta.requiresAuth);
    const requiredPermission = to.meta.permission;

    if (to.name === 'login' && authStore.isAuthenticated) {
      return { name: 'dashboard' };
    }

    if (requiresAuth && !authStore.isAuthenticated) {
      return {
        name: 'login',
        query: {
          redirect: to.fullPath,
        },
      };
    }

    if (typeof requiredPermission === 'string' && !authStore.hasPermission(requiredPermission)) {
      const fallbackPath = navigationStore.firstAccessiblePath(authStore.permissions);

      if (to.name === UNAUTHORIZED_ROUTE_NAME) {
        return true;
      }

      return {
        name: UNAUTHORIZED_ROUTE_NAME,
        query: {
          from: to.fullPath,
          fallback: fallbackPath || '/dashboard',
        },
      };
    }

    navigationStore.setActivePath(to.path);
    return true;
  });

  router.afterEach((to: RouteLocationNormalized) => {
    const navigationStore = useNavigationStore(pinia);
    navigationStore.setActivePath(to.path);
  });
}
