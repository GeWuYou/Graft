import { describe, expect, it } from 'vitest';

import { containerBootstrapRouteRegistrations, containerGlobalRouteRegistrations } from './bootstrap-routes';

describe('container bootstrap route registrations', () => {
  it('uses the canonical container management route identity', () => {
    expect(containerBootstrapRouteRegistrations).toHaveLength(2);
    expect(containerBootstrapRouteRegistrations[0]).toMatchObject({
      menuPath: '/infrastructure/docker/containers',
      routeName: 'ContainerList',
    });
    expect(containerBootstrapRouteRegistrations[1]).toMatchObject({
      menuPath: '/infrastructure/images',
      routeName: 'DockerImageList',
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
    expect(containerBootstrapRouteRegistrations[1]?.meta).toMatchObject({
      tabGroup: 'infrastructure',
    });
    expect(containerBootstrapRouteRegistrations[1]?.meta?.semanticTitle).toBeDefined();
    expect(containerBootstrapRouteRegistrations[1]?.meta?.tabTitle).toBeDefined();
    expect(containerBootstrapRouteRegistrations[1]?.meta?.breadcrumbTitle).toBeDefined();
    expect(containerBootstrapRouteRegistrations[1]?.meta).not.toHaveProperty('title');
    expect(containerBootstrapRouteRegistrations[1]?.meta).not.toHaveProperty('titleKey');
  });

  it('registers the detail page as a menu-hidden global route', () => {
    expect(containerGlobalRouteRegistrations).toHaveLength(2);
    expect(containerGlobalRouteRegistrations[1]).toMatchObject({
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

  it('registers image management as a visible Docker child menu route', () => {
    expect(containerBootstrapRouteRegistrations[1]).toMatchObject({
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
