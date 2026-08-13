import { describe, expect, it } from 'vitest';
import { createMemoryHistory, createRouter } from 'vue-router';

import { getBootstrapRouteRegistration } from '@/modules';

import {
  buildBootstrapNavigationTree,
  transformBootstrapMenusToRoutes,
  transformGlobalRegistrationsToRoutes,
} from './bootstrap';

const graph = [
  {
    code: 'domain.security',
    kind: 'group' as const,
    order: 60,
    title_key: 'menu.domain.security',
    title: 'Security',
    icon: 'security-domain',
    permission: '',
  },
  {
    code: 'user.list',
    parent_code: 'domain.security',
    kind: 'entry' as const,
    order: 1,
    title_key: 'menu.user_list.title',
    title: 'Users',
    path: '/security/users',
    icon: 'user-identity',
    permission: 'user.read',
  },
  {
    code: 'role.list',
    parent_code: 'domain.security',
    kind: 'entry' as const,
    order: 2,
    title_key: 'menu.role_list.title',
    title: 'Roles',
    path: '/security/roles',
    icon: 'access-policy',
    permission: 'role.read',
  },
  {
    code: 'domain.build',
    kind: 'group' as const,
    order: 30,
    title_key: 'menu.domain.build',
    title: 'Build',
    icon: 'build-domain',
    permission: '',
  },
] as const;

describe('bootstrap navigation graph', () => {
  it('preserves section metadata and nested provider groups without making them routes', () => {
    const navigation = buildBootstrapNavigationTree([
      {
        code: 'domain.infrastructure',
        kind: 'group',
        order: 20,
        title: 'Infrastructure',
        icon: 'infrastructure-domain',
        permission: '',
      },
      {
        code: 'docker',
        parent_code: 'domain.infrastructure',
        kind: 'group',
        order: 50,
        title: 'Docker',
        title_key: 'menu.docker.title',
        section_key: 'runtime',
        section_title_key: 'menu.section.runtime',
        icon: 'docker-provider',
        permission: '',
      },
      {
        code: 'container.list',
        parent_code: 'docker',
        kind: 'entry',
        order: 51,
        title: 'Containers',
        title_key: 'menu.container.title',
        path: '/infrastructure/docker/containers',
        icon: 'container-workload',
        permission: 'container.view',
      },
      {
        code: 'docker.image.list',
        parent_code: 'docker',
        kind: 'entry',
        order: 52,
        title: 'Images',
        title_key: 'menu.docker.image.title',
        path: '/infrastructure/images',
        icon: 'image-artifact',
        permission: 'container.view',
      },
    ]);

    expect(navigation[0]?.children?.[0]?.meta?.navigationSection?.key).toBe('runtime');
    expect(navigation[0]?.children?.[0]?.children?.map((item) => item.path)).toEqual([
      'container.list',
      'docker.image.list',
    ]);
    const routes = transformBootstrapMenusToRoutes([
      {
        code: 'domain.infrastructure',
        kind: 'group',
        order: 20,
        title: 'Infrastructure',
        icon: 'infrastructure-domain',
        permission: '',
      },
      {
        code: 'docker',
        parent_code: 'domain.infrastructure',
        kind: 'group',
        order: 50,
        title: 'Docker',
        icon: 'docker-provider',
        permission: '',
      },
      {
        code: 'container.list',
        parent_code: 'docker',
        kind: 'entry',
        order: 51,
        title: 'Containers',
        path: '/infrastructure/docker/containers',
        icon: 'container-workload',
        permission: 'container.view',
      },
      {
        code: 'docker.image.list',
        parent_code: 'docker',
        kind: 'entry',
        order: 52,
        title: 'Images',
        title_key: 'menu.docker.image.title',
        path: '/infrastructure/images',
        icon: 'image-artifact',
        permission: 'container.view',
      },
    ]);
    expect(routes).toHaveLength(2);
    expect(routes[0]?.meta?.navigationAncestors?.map((ancestor) => ancestor.code)).toEqual([
      'domain.infrastructure',
      'docker',
    ]);
    expect(routes[1]?.name).toBe('DockerImageList');
  });

  it('places shared image registries before runtime targets without creating a section route', () => {
    const navigation = buildBootstrapNavigationTree([
      {
        code: 'domain.infrastructure',
        kind: 'group',
        order: 20,
        title: 'Infrastructure',
        icon: 'infrastructure-domain',
        permission: '',
      },
      {
        code: 'registry.list',
        parent_code: 'domain.infrastructure',
        kind: 'entry',
        order: 30,
        title: '',
        title_key: 'menu.registries.title',
        section_key: 'shared-resources',
        section_title_key: 'menu.section.shared_resources',
        path: '/infrastructure/registries',
        icon: 'image-registry',
        permission: 'registry.read',
      },
      {
        code: 'runtime-target.list',
        parent_code: 'domain.infrastructure',
        kind: 'entry',
        order: 40,
        title: 'Runtime Targets',
        title_key: 'menu.runtimeTargets.title',
        path: '/infrastructure/runtime-targets',
        icon: 'runtime-target',
        permission: 'runtime_target.view',
      },
    ]);

    expect(navigation[0]?.children?.map((item) => item.path)).toEqual(['registry.list', 'runtime-target.list']);
    expect(navigation[0]?.children?.[0]?.meta?.navigationSection).toEqual({
      key: 'shared-resources',
      title: { 'zh-CN': '共享资源', 'en-US': 'Shared Resources' },
    });
    expect(navigation[0]?.children?.[0]?.meta?.navigationTargetPath).toBe('/infrastructure/registries');
  });

  it('builds visible navigation by explicit parent code and prunes empty groups', () => {
    const navigation = buildBootstrapNavigationTree(graph.map((item) => ({ ...item })));
    expect(navigation).toHaveLength(1);
    expect(navigation[0]?.path).toBe('domain.security');
    expect(navigation[0]?.meta?.navigationTargetPath).toBe('/security/users');
    expect(navigation[0]?.children?.map((item) => item.path)).toEqual(['user.list', 'role.list']);
    expect(navigation[0]?.children?.[0]?.meta?.navigationAncestors?.map((ancestor) => ancestor.code)).toEqual([
      'domain.security',
    ]);
  });

  it('creates router records only for registered entry resources', () => {
    const routes = transformBootstrapMenusToRoutes(graph.map((item) => ({ ...item })));
    expect(routes.map((route) => route.path)).toEqual(['/security/users', '/security/roles']);
    expect(routes.every((route) => route.name && !String(route.name).startsWith('BootstrapGroup'))).toBe(true);
    expect(routes[0]?.meta?.navigationTitle?.['en-US']).toBe('Security / Users');
  });

  it('keeps a visible template menu entry in the bootstrap navigation tree', () => {
    const menus = [
      {
        code: 'domain.application',
        kind: 'group' as const,
        order: 10,
        title: 'Application',
        icon: 'application-domain',
        permission: '',
      },
      {
        code: 'application.templates',
        parent_code: 'domain.application',
        kind: 'entry' as const,
        order: 53,
        title: 'Templates',
        path: '/applications/templates',
        icon: 'application-template',
        permission: 'application.template.manage',
      },
    ];

    const navigation = buildBootstrapNavigationTree(menus);
    expect(navigation[0]?.children?.map((item) => item.path)).toEqual(['application.templates']);

    const routes = transformBootstrapMenusToRoutes(menus);
    expect(routes[0]?.path).toBe('/applications/templates');
    expect(routes[0]?.name).toBe('ApplicationTemplates');
  });

  it('keeps Docker volume management visible when the backend returns its menu entry', () => {
    const menus = [
      {
        code: 'domain.infrastructure',
        kind: 'group' as const,
        order: 20,
        title: 'Infrastructure',
        icon: 'infrastructure-domain',
        permission: '',
      },
      {
        code: 'docker',
        parent_code: 'domain.infrastructure',
        kind: 'group' as const,
        order: 50,
        title: 'Docker',
        icon: 'docker-provider',
        permission: '',
      },
      {
        code: 'docker.volume.list',
        parent_code: 'docker',
        kind: 'entry' as const,
        order: 53,
        title: 'Volumes',
        title_key: 'menu.dockerVolume.title',
        path: '/infrastructure/docker/volumes',
        icon: 'persistent-volume',
        permission: 'container.view',
      },
    ];

    const navigation = buildBootstrapNavigationTree(menus);
    expect(navigation[0]?.children?.[0]?.children?.map((item) => item.path)).toEqual(['docker.volume.list']);

    const routes = transformBootstrapMenusToRoutes(menus);
    expect(routes).toHaveLength(1);
    expect(routes[0]?.path).toBe('/infrastructure/docker/volumes');
    expect(routes[0]?.name).toBe('DockerVolumeList');
  });

  it('keeps bootstrap-owned menu display metadata ahead of registration patches', () => {
    const registration = getBootstrapRouteRegistration('/security/users');
    const originalMeta = registration?.meta;
    const overriddenTitle = { 'zh-CN': '错误标题', 'en-US': 'Incorrect Title' };

    expect(registration).toBeDefined();
    registration!.meta = {
      ...originalMeta,
      title: overriddenTitle,
      titleKey: 'incorrect.title',
      icon: 'close',
      orderNo: 999,
    };

    try {
      const [route] = transformBootstrapMenusToRoutes(graph.map((item) => ({ ...item })));

      expect(route?.meta).toMatchObject({
        title: { 'en-US': 'Users' },
        titleKey: 'menu.user_list.title',
        icon: 'user-identity',
        orderNo: 1,
      });
    } finally {
      registration!.meta = originalMeta;
    }
  });

  it('keeps global routes out of menu navigation and preserves their breadcrumb policy', () => {
    const routes = transformGlobalRegistrationsToRoutes([
      {
        path: '/notifications',
        routeName: 'NotificationList',
        loadPage: () => import('@/modules/notification/pages/list/index.vue'),
        meta: { title: { 'zh-CN': '通知中心', 'en-US': 'Notifications' }, titleKey: 'notification.route.list.title' },
      },
    ]);
    expect(routes[0]?.path).toBe('/notifications');
    expect(routes[0]?.children?.[0]?.meta?.hiddenMenu).toBe(true);
    expect(routes[0]?.children?.[0]?.meta?.hiddenBreadcrumb).toBe(true);
    expect(routes[0]?.meta?.navigationTargetPath).toBeUndefined();
  });

  it('redirects named global route navigation to the page child while preserving route state', async () => {
    const routes = transformGlobalRegistrationsToRoutes([
      {
        path: '/platform/network/:targetId',
        routeName: 'PlatformNetworkConnectivityDiagnostics',
        loadPage: () => import('@/modules/network/pages/connectivity/diagnostics.vue'),
        meta: { title: { 'zh-CN': '连通性诊断', 'en-US': 'Connectivity Diagnostics' } },
      },
    ]);
    const router = createRouter({ history: createMemoryHistory(), routes });

    await router.push({
      name: 'PlatformNetworkConnectivityDiagnostics',
      params: { targetId: 'platform-update' },
      query: { source: 'connectivity' },
      hash: '#report',
    });
    await router.isReady();

    expect(router.currentRoute.value).toMatchObject({
      name: 'PlatformNetworkConnectivityDiagnosticsIndex',
      path: '/platform/network/platform-update',
      params: { targetId: 'platform-update' },
      query: { source: 'connectivity' },
      hash: '#report',
    });
  });

  it('uses an explicitly declared child route name when mounting a global page', async () => {
    const routes = transformGlobalRegistrationsToRoutes([
      {
        path: '/infrastructure/registries/:connectionRef',
        pageRouteName: 'RegistryConnectionDetailIndex',
        routeName: 'RegistryConnectionDetail',
        loadPage: () => import('@/modules/registry/pages/detail/index.vue'),
        meta: { title: { 'zh-CN': '镜像仓库详情', 'en-US': 'Image Registry Detail' } },
      },
    ]);
    const router = createRouter({ history: createMemoryHistory(), routes });

    await router.push('/infrastructure/registries/registry-a');

    expect(router.currentRoute.value.name).toBe('RegistryConnectionDetailIndex');
  });

  it('attaches an explicitly declared parent resource trail to global detail routes', () => {
    const routes = transformGlobalRegistrationsToRoutes(
      [
        {
          path: '/security/users/42',
          routeName: 'UserDetail',
          navigationParentPath: '/security/users',
          loadPage: () => import('@/modules/user/pages/index.vue'),
          meta: { title: { 'zh-CN': '用户详情', 'en-US': 'User Detail' } },
        },
      ],
      graph.map((item) => ({ ...item })),
    );
    expect(routes[0]?.meta?.navigationAncestors?.map((ancestor) => ancestor.code)).toEqual(['domain.security']);
    expect(routes[0]?.meta?.navigationTitle?.['en-US']).toBe('Security / User Detail');
    expect(routes[0]?.meta?.navigationTargetPath).toBe('/security/users');
  });
});
