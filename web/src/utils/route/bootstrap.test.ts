import { describe, expect, it } from 'vitest';

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
    icon: 'secured',
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
    icon: 'usergroup',
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
    icon: 'secured',
    permission: 'role.read',
  },
  {
    code: 'domain.build',
    kind: 'group' as const,
    order: 30,
    title_key: 'menu.domain.build',
    title: 'Build',
    icon: 'tools',
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
        icon: 'infrastructure',
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
        icon: 'docker',
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
        icon: 'container',
        permission: 'container.view',
      },
    ]);

    expect(navigation[0]?.children?.[0]?.meta?.navigationSection?.key).toBe('runtime');
    expect(navigation[0]?.children?.[0]?.children?.[0]?.path).toBe('container.list');
    const routes = transformBootstrapMenusToRoutes([
      {
        code: 'domain.infrastructure',
        kind: 'group',
        order: 20,
        title: 'Infrastructure',
        icon: 'infrastructure',
        permission: '',
      },
      {
        code: 'docker',
        parent_code: 'domain.infrastructure',
        kind: 'group',
        order: 50,
        title: 'Docker',
        icon: 'docker',
        permission: '',
      },
      {
        code: 'container.list',
        parent_code: 'docker',
        kind: 'entry',
        order: 51,
        title: 'Containers',
        path: '/infrastructure/docker/containers',
        icon: 'container',
        permission: 'container.view',
      },
    ]);
    expect(routes).toHaveLength(1);
    expect(routes[0]?.meta?.navigationAncestors?.map((ancestor) => ancestor.code)).toEqual([
      'domain.infrastructure',
      'docker',
    ]);
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
        icon: 'usergroup',
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
