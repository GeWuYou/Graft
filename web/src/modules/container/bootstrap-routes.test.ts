// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { containerBootstrapRouteRegistrations } from './bootstrap-routes';

describe('container bootstrap route registrations', () => {
  it('uses the canonical container management route identity', () => {
    expect(containerBootstrapRouteRegistrations).toHaveLength(1);
    expect(containerBootstrapRouteRegistrations[0]).toMatchObject({
      menuPath: '/ops/containers',
      routeName: 'ContainerList',
    });
  });

  it('registers the detail page as a menu-hidden global route', async () => {
    const { containerGlobalRouteRegistrations } = await import('./bootstrap-routes');

    expect(containerGlobalRouteRegistrations).toHaveLength(1);
    expect(containerGlobalRouteRegistrations[0]).toMatchObject({
      path: '/ops/containers/:id',
      pageRouteName: 'ContainerDetailIndex',
      routeName: 'ContainerDetail',
      meta: {
        hiddenMenu: true,
        pageKind: 'detail',
        titleKey: 'container.route.detail.title',
      },
    });
  });
});
