import { describe, expect, it } from 'vitest';

import { containerBootstrapRouteRegistrations, containerGlobalRouteRegistrations } from './bootstrap-routes';

describe('container bootstrap route registrations', () => {
  it('uses the canonical container management route identity', () => {
    expect(containerBootstrapRouteRegistrations).toHaveLength(1);
    expect(containerBootstrapRouteRegistrations[0]).toMatchObject({
      menuPath: '/infrastructure/docker/containers',
      routeName: 'ContainerList',
    });
  });

  it('keeps menu title ownership with the bootstrap menu while deriving tab and breadcrumb titles locally', () => {
    expect(containerBootstrapRouteRegistrations[0]?.meta).toMatchObject({
      tabGroup: 'infrastructure',
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

  it('registers the detail page as a menu-hidden global route', () => {
    expect(containerGlobalRouteRegistrations).toHaveLength(3);
    expect(containerGlobalRouteRegistrations[2]).toMatchObject({
      path: '/infrastructure/docker/containers/:id',
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

  it('registers image management at its canonical Docker child menu route', () => {
    expect(containerGlobalRouteRegistrations[0]).toMatchObject({
      path: '/infrastructure/images',
      pageRouteName: 'DockerImageListIndex',
      routeName: 'DockerImageList',
      navigationParentPath: '/infrastructure/docker/containers',
      meta: { titleKey: 'container.route.images.title' },
    });
  });
});
