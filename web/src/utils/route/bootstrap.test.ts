import { describe, expect, it } from 'vitest';

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
    path: '/users',
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
    path: '/roles',
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
  it('builds visible navigation by explicit parent code and prunes empty groups', () => {
    const navigation = buildBootstrapNavigationTree(graph.map((item) => ({ ...item })));
    expect(navigation).toHaveLength(1);
    expect(navigation[0]?.path).toBe('domain.security');
    expect(navigation[0]?.meta?.navigationTargetPath).toBe('/users');
    expect(navigation[0]?.children?.map((item) => item.path)).toEqual(['user.list', 'role.list']);
  });

  it('creates router records only for registered entry resources', () => {
    const routes = transformBootstrapMenusToRoutes(graph.map((item) => ({ ...item })));
    expect(routes.map((route) => route.path)).toEqual(['/users', '/roles']);
    expect(routes.every((route) => route.name && !String(route.name).startsWith('BootstrapGroup'))).toBe(true);
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
  });
});
