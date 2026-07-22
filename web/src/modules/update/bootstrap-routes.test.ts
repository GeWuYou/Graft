import { describe, expect, it } from 'vitest';

import { updateBootstrapRouteRegistrations } from './bootstrap-routes';

describe('platform update bootstrap route registrations', () => {
  it('declares the Platform Updates menu route once', () => {
    expect(updateBootstrapRouteRegistrations).toEqual([
      expect.objectContaining({
        menuPath: '/platform/updates',
        routeName: 'PlatformUpdateCenter',
      }),
    ]);
  });
});
