import { describe, expect, it } from 'vitest';

import { containerBootstrapRouteRegistrations } from './bootstrap-routes';

describe('container bootstrap route registrations', () => {
  it('uses the canonical container management route identity', () => {
    expect(containerBootstrapRouteRegistrations).toContainEqual(
      expect.objectContaining({
        menuPath: '/infrastructure/docker/containers',
        routeName: 'ContainerList',
      }),
    );
  });

  it('keeps menu title ownership with the bootstrap menu while deriving tab and breadcrumb titles locally', () => {
    const containerListRoute = containerBootstrapRouteRegistrations.find(
      (route) => route.routeName === 'ContainerList',
    );
    expect(containerListRoute?.meta).toMatchObject({
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
    expect(containerListRoute?.meta).not.toHaveProperty('title');
    expect(containerListRoute?.meta).not.toHaveProperty('titleKey');
  });

  it('registers Docker volumes as a server-menu-backed management entry', () => {
    expect(containerBootstrapRouteRegistrations).toContainEqual(
      expect.objectContaining({
        menuPath: '/infrastructure/docker/volumes',
        routeName: 'DockerVolumeList',
      }),
    );
  });

  it('registers the detail page as a menu-hidden global route', async () => {
    const { containerGlobalRouteRegistrations } = await import('./bootstrap-routes');

    expect(containerGlobalRouteRegistrations.find((route) => route.routeName === 'ContainerDetail')).toMatchObject({
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
    expect(containerGlobalRouteRegistrations.find((route) => route.routeName === 'DockerVolumeDetail')).toMatchObject({
      path: '/infrastructure/docker/volumes/:id',
      pageRouteName: 'DockerVolumeDetailIndex',
      routeName: 'DockerVolumeDetail',
      meta: { hiddenMenu: true, pageKind: 'detail' },
    });
  });
});
