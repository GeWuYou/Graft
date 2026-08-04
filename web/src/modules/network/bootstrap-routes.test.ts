import { describe, expect, it } from 'vitest';

import { networkBootstrapRouteRegistrations, networkGlobalRouteRegistrations } from './bootstrap-routes';

describe('network bootstrap route registrations', () => {
  it('uses Connectivity as the Network entry and exposes policy and diagnostics as visible global details', () => {
    expect(networkBootstrapRouteRegistrations).toEqual([
      expect.objectContaining({ menuPath: '/platform/network', routeName: 'PlatformNetworkConnectivity' }),
    ]);
    expect(networkGlobalRouteRegistrations).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          path: '/platform/network/outbound',
          routeName: 'PlatformNetworkOutbound',
          navigationParentPath: '/platform/network',
          meta: expect.objectContaining({
            hidden: false,
            hiddenMenu: true,
            tabTitle: { 'en-US': 'Outbound Network', 'zh-CN': '出站网络' },
          }),
        }),
        expect.objectContaining({
          path: '/platform/network/:targetId',
          routeName: 'PlatformNetworkConnectivityDiagnostics',
          navigationParentPath: '/platform/network',
          meta: expect.objectContaining({
            hidden: false,
            hiddenMenu: true,
            tabTitle: { 'en-US': 'Connectivity Diagnostics', 'zh-CN': '连通性诊断' },
          }),
        }),
      ]),
    );
  });
});
