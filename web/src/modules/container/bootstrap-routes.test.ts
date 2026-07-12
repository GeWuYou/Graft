import { describe, expect, it } from 'vitest';

import { containerBootstrapRouteRegistrations } from './bootstrap-routes';

describe('container bootstrap route registrations', () => {
  it('uses the canonical container management route identity', () => {
    expect(containerBootstrapRouteRegistrations).toHaveLength(1);
    expect(containerBootstrapRouteRegistrations[0]).toMatchObject({
      menuPath: '/containers',
      routeName: 'ContainerList',
    });
  });

  it('keeps menu title ownership with the bootstrap menu while deriving tab and breadcrumb titles locally', () => {
    expect(containerBootstrapRouteRegistrations[0]?.meta).toMatchObject({
      tabGroup: 'infrastructure',
      navigationSection: {
        key: 'runtime',
        title: {
          'zh-CN': '运行时',
          'en-US': 'Runtime',
        },
      },
      semanticTitle: {
        'zh-CN': '容器管理',
        'en-US': 'Containers',
      },
      tabTitle: {
        'zh-CN': '容器管理',
        'en-US': 'Containers',
      },
      breadcrumbTitle: {
        'zh-CN': '容器管理',
        'en-US': 'Containers',
      },
    });
    expect(containerBootstrapRouteRegistrations[0]?.meta).not.toHaveProperty('title');
    expect(containerBootstrapRouteRegistrations[0]?.meta).not.toHaveProperty('titleKey');
  });

  it('registers the detail page as a menu-hidden global route', async () => {
    const { containerGlobalRouteRegistrations } = await import('./bootstrap-routes');

    expect(containerGlobalRouteRegistrations).toHaveLength(2);
    expect(containerGlobalRouteRegistrations[1]).toMatchObject({
      path: '/containers/:id',
      pageRouteName: 'ContainerDetailIndex',
      routeName: 'ContainerDetail',
      meta: {
        hidden: false,
        hiddenMenu: true,
        pageKind: 'detail',
        tabGroup: 'infrastructure',
        titleKey: 'container.route.detail.title',
      },
    });
  });
});
