import { describe, expect, it } from 'vitest';

import { networkBootstrapRouteRegistrations } from './bootstrap-routes';

describe('network bootstrap route registrations', () => {
  it('registers the platform-owned outbound network settings route', () => {
    expect(networkBootstrapRouteRegistrations).toEqual([
      expect.objectContaining({
        menuPath: '/platform/network',
        routeName: 'PlatformNetworkOutbound',
        meta: expect.objectContaining({ pageKind: 'detail', pageSurface: 'form-detail' }),
      }),
    ]);
  });
});
