import { describe, expect, it } from 'vitest';

import { AUTH_ROUTE_NAME, AUTH_ROUTE_PATH } from '@/contracts/auth/routes';

import router from './index';

describe('router static runtime surface', () => {
  it('does not register starter demo homepage or grouped result routes', () => {
    const registeredPaths = router.getRoutes().map((route) => route.path);

    expect(registeredPaths).not.toContain('/dashboard');
    expect(registeredPaths).not.toContain('/result');
    expect(registeredPaths).toContain('/result/404');
    expect(registeredPaths).toContain('/');
    expect(registeredPaths).toContain(AUTH_ROUTE_PATH.LOGIN);
    expect(
      router.getRoutes().some((route) => route.path === AUTH_ROUTE_PATH.LOGIN && route.name === AUTH_ROUTE_NAME.LOGIN),
    ).toBe(true);
  });
});
