import { describe, expect, it } from 'vitest';

import { networkBootstrapRouteRegistrations } from './bootstrap-routes';

describe('network bootstrap route registrations', () => {
  it('registers outbound settings, connectivity health, and target diagnostics routes', () => {
    expect(networkBootstrapRouteRegistrations).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ menuPath: '/platform/network', routeName: 'PlatformNetworkOutbound' }),
        expect.objectContaining({
          menuPath: '/platform/network/connectivity',
          routeName: 'PlatformNetworkConnectivity',
        }),
        expect.objectContaining({
          menuPath: '/platform/network/connectivity/:targetId',
          routeName: 'PlatformNetworkConnectivityDiagnostics',
          meta: expect.objectContaining({ hidden: true }),
        }),
      ]),
    );
  });
});
