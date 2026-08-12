import { describe, expect, it } from 'vitest';

import { registryBootstrapRouteRegistrations } from './bootstrap-routes';

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
});
