import { describe, expect, it } from 'vitest';

import { registryBootstrapRouteRegistrations, registryGlobalRouteRegistrations } from './bootstrap-routes';

describe('registry bootstrap route registrations', () => {
  it('registers the canonical registry list with localized list titles', () => {
    expect(registryBootstrapRouteRegistrations).toContainEqual(
      expect.objectContaining({
        menuPath: '/infrastructure/registries',
        routeName: 'RegistryConnectionList',
        meta: expect.objectContaining({
          pageKind: 'list',
          pageSurface: 'paged-table',
          semanticTitle: {
            'zh-CN': '镜像仓库',
            'en-US': 'Image Registries',
          },
          breadcrumbTitle: {
            'zh-CN': '镜像仓库',
            'en-US': 'Image Registries',
          },
        }),
      }),
    );
  });

  it('declares the mounted detail page route used for its runtime tab title', () => {
    expect(registryGlobalRouteRegistrations).toContainEqual(
      expect.objectContaining({
        pageRouteName: 'RegistryConnectionDetailIndex',
        routeName: 'RegistryConnectionDetail',
      }),
    );
  });
});
