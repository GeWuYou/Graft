import { describe, expect, it } from 'vitest';

import { APP_RESULT_ROUTE_PATH } from '@/contracts/app/routes';
import { AUTH_ROUTE_NAME, AUTH_ROUTE_PATH } from '@/modules/auth/contract/routes';

import router from './index';

describe('router static runtime surface', () => {
  it('does not register starter demo homepage or grouped result routes', () => {
    const registeredPaths = router.getRoutes().map((route) => route.path);

    expect(registeredPaths).not.toContain('/dashboard');
    expect(registeredPaths).not.toContain('/result');
    expect(registeredPaths).toContain(APP_RESULT_ROUTE_PATH.FORBIDDEN);
    expect(registeredPaths).toContain(APP_RESULT_ROUTE_PATH.NOT_FOUND);
    expect(registeredPaths).toContain(APP_RESULT_ROUTE_PATH.SERVER_ERROR);
    expect(registeredPaths).toContain(APP_RESULT_ROUTE_PATH.SUCCESS);
    expect(registeredPaths).toContain(APP_RESULT_ROUTE_PATH.FAIL);
    expect(registeredPaths).toContain(APP_RESULT_ROUTE_PATH.NETWORK_ERROR);
    expect(registeredPaths).toContain(APP_RESULT_ROUTE_PATH.MAINTENANCE);
    expect(registeredPaths).toContain(APP_RESULT_ROUTE_PATH.BROWSER_INCOMPATIBLE);
    expect(registeredPaths).toContain('/');
    expect(registeredPaths).toContain(AUTH_ROUTE_PATH.LOGIN);
    expect(
      router.getRoutes().some((route) => route.path === AUTH_ROUTE_PATH.LOGIN && route.name === AUTH_ROUTE_NAME.LOGIN),
    ).toBe(true);
  });

  it('keeps the root entry renderable and leaves catch-all available for dynamic route recovery', () => {
    const rootRoute = router.getRoutes().find((route) => route.path === '/' && route.name === 'RootEntry');
    const catchAllRoute = router.getRoutes().find((route) => route.path === '/:pathMatch(.*)*');

    expect(rootRoute?.name).toBe('RootEntry');
    expect(rootRoute?.meta.hidden).not.toBe(true);
    expect(catchAllRoute?.name).toBe('404Page');
    expect(catchAllRoute?.redirect).toBeUndefined();
  });
});
