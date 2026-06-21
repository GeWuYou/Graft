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

  it('keeps menu title ownership with the bootstrap menu while deriving tab and breadcrumb titles locally', () => {
    expect(containerBootstrapRouteRegistrations[0]?.meta).toMatchObject({
      semanticTitle: {
        'zh-CN': '运维管理 - 容器管理',
        'en-US': 'Operations - Container Management',
      },
      tabTitle: {
        'zh-CN': '运维管理 - 容器管理',
        'en-US': 'Operations - Container Management',
      },
      breadcrumbTitle: {
        'zh-CN': '容器管理',
        'en-US': 'Container Management',
      },
    });
    expect(containerBootstrapRouteRegistrations[0]?.meta).not.toHaveProperty('title');
    expect(containerBootstrapRouteRegistrations[0]?.meta).not.toHaveProperty('titleKey');
  });

  it('registers the detail page as a menu-hidden global route', async () => {
    const { containerGlobalRouteRegistrations } = await import('./bootstrap-routes');

    expect(containerGlobalRouteRegistrations).toHaveLength(1);
    expect(containerGlobalRouteRegistrations[0]).toMatchObject({
      path: '/ops/containers/:id',
      pageRouteName: 'ContainerDetailIndex',
      routeName: 'ContainerDetail',
      meta: {
        hidden: false,
        hiddenMenu: true,
        pageKind: 'detail',
        titleKey: 'container.route.detail.title',
      },
    });
  });
});
