import { describe, expect, it } from 'vitest';

import { containerBootstrapRouteRegistrations, containerGlobalRouteRegistrations } from './bootstrap-routes';

describe('container bootstrap route registrations', () => {
  it('uses the canonical container management route identity', () => {
    expect(containerBootstrapRouteRegistrations).toHaveLength(3);
    expect(containerBootstrapRouteRegistrations).toContainEqual(
      expect.objectContaining({
        menuPath: '/infrastructure/docker/containers',
        routeName: 'ContainerList',
      }),
    );
    expect(containerBootstrapRouteRegistrations).toContainEqual(
      expect.objectContaining({
        menuPath: '/infrastructure/images',
        routeName: 'DockerImageList',
      }),
    );
    expect(containerBootstrapRouteRegistrations).toContainEqual(
      expect.objectContaining({
        menuPath: '/infrastructure/docker/volumes',
        routeName: 'DockerVolumeList',
      }),
    );
  });

  it('registers Docker network management as a visible bootstrap route', () => {
    expect(containerBootstrapRouteRegistrations[1]).toMatchObject({
      menuPath: '/infrastructure/docker/networks',
      routeName: 'DockerNetworkList',
      meta: { pageKind: 'list', tabGroup: 'infrastructure' },
    });
  });

  it('keeps menu title ownership with the bootstrap menu while deriving tab and breadcrumb titles locally', () => {
    const containerRoute = containerBootstrapRouteRegistrations.find((route) => route.routeName === 'ContainerList');
    const imageRoute = containerBootstrapRouteRegistrations.find((route) => route.routeName === 'DockerImageList');
    const volumeRoute = containerBootstrapRouteRegistrations.find((route) => route.routeName === 'DockerVolumeList');

    expect(containerRoute?.meta).toMatchObject({
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
    expect(containerRoute?.meta).not.toHaveProperty('title');
    expect(containerRoute?.meta).not.toHaveProperty('titleKey');
    expect(imageRoute?.meta).toMatchObject({
      tabGroup: 'infrastructure',
    });
    expect(imageRoute?.meta?.semanticTitle).toBeDefined();
    expect(imageRoute?.meta?.tabTitle).toBeDefined();
    expect(imageRoute?.meta?.breadcrumbTitle).toBeDefined();
    expect(imageRoute?.meta).not.toHaveProperty('title');
    expect(imageRoute?.meta).not.toHaveProperty('titleKey');
    expect(volumeRoute?.meta).toMatchObject({
      tabGroup: 'infrastructure',
      pageKind: 'list',
    });
    expect(volumeRoute?.meta).not.toHaveProperty('title');
    expect(volumeRoute?.meta).not.toHaveProperty('titleKey');
  });

  it('registers the detail page as a menu-hidden global route', () => {
    expect(containerGlobalRouteRegistrations).toHaveLength(2);
    const detailRoute = containerGlobalRouteRegistrations.find((route) => route.routeName === 'ContainerDetail');
    expect(detailRoute).toMatchObject({
      path: '/infrastructure/docker/containers/:id',
      routeName: 'ContainerDetail',
      meta: {
        hidden: false,
        hiddenMenu: true,
        pageKind: 'detail',
        tabGroup: 'infrastructure',
        titleKey: 'container.route.detail.title',
      },
    });
    expect(containerGlobalRouteRegistrations).not.toContainEqual(
      expect.objectContaining({ routeName: 'DockerVolumeDetail' }),
    );
  });

  it('registers image management as a visible Docker child menu route', () => {
    const imageRoute = containerBootstrapRouteRegistrations.find((route) => route.routeName === 'DockerImageList');
    expect(imageRoute).toMatchObject({
      menuPath: '/infrastructure/images',
      routeName: 'DockerImageList',
      meta: {
        tabGroup: 'infrastructure',
        pageKind: 'list',
      },
    });
    expect(containerGlobalRouteRegistrations).not.toContainEqual(
      expect.objectContaining({ path: '/infrastructure/images' }),
    );
  });
});
