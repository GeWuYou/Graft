import { describe, expect, it } from 'vitest';

import { transformBootstrapMenusToRoutes } from './bootstrap';

describe('transformBootstrapMenusToRoutes', () => {
  it('只为当前 web 已接入的 bootstrap 菜单生成动态路由', () => {
    const routes = transformBootstrapMenusToRoutes([
      {
        code: 'role.list',
        title_key: 'menu.role_list.title',
        title: '角色管理',
        path: '/roles',
        icon: 'secured',
        permission: 'role.read',
      },
      {
        code: 'user.list',
        title_key: 'menu.user_list.title',
        title: '用户管理',
        path: '/users',
        icon: 'usergroup',
        permission: 'user.read',
      },
      {
        code: 'unknown.feature',
        title: '未知功能',
        path: '/unknown',
        icon: 'app',
        permission: '',
      },
    ]);

    expect(routes).toHaveLength(2);
    expect(routes[0]?.path).toBe('/roles');
    expect(routes[0]?.redirect).toBe('/roles/index');
    expect(routes[0]?.children?.[0]?.name).toBe('RoleListIndex');
    expect(routes[0]?.meta?.titleKey).toBe('menu.role_list.title');
    expect(routes[0]?.meta?.title).toEqual({ 'zh-CN': '角色管理', 'en-US': 'Role Management' });
    expect(routes[1]?.path).toBe('/users');
    expect(routes[1]?.redirect).toBe('/users/index');
    expect(routes[1]?.children?.[0]?.name).toBe('UserListIndex');
    expect(routes[1]?.meta?.titleKey).toBe('menu.user_list.title');
    expect(routes[1]?.meta?.title).toEqual({ 'zh-CN': '用户管理', 'en-US': 'User Management' });
  });

  it('为监控模块合成显式父级导航并避免 index 面包屑段', () => {
    const routes = transformBootstrapMenusToRoutes([
      {
        code: 'monitor.section',
        title_key: 'monitor.sectionTitle',
        title: '服务器管理',
        path: '/monitor',
        icon: 'server',
        permission: '',
      },
      {
        code: 'monitor.server-status',
        title_key: 'menu.monitor.server_status.title',
        title: '服务器状态',
        path: '/monitor/server-status',
        icon: 'activity',
        permission: '',
      },
      {
        code: 'monitor.server-status.overview',
        title_key: 'menu.monitor.server_status.overview.title',
        title: '概览',
        path: '/monitor/server-status/overview',
        icon: 'dashboard',
        permission: 'monitor.server-status.read',
      },
      {
        code: 'monitor.server-status.runtime',
        title_key: 'menu.monitor.server_status.runtime.title',
        title: '运行时',
        path: '/monitor/server-status/runtime',
        icon: 'time',
        permission: 'monitor.server-status.read',
      },
      {
        code: 'monitor.server-status.dependencies',
        title_key: 'menu.monitor.server_status.dependencies.title',
        title: '依赖服务',
        path: '/monitor/server-status/dependencies',
        icon: 'data-base',
        permission: 'monitor.server-status.read',
      },
    ]);

    expect(routes).toHaveLength(1);
    expect(routes[0]?.path).toBe('/monitor');
    expect(routes[0]?.redirect).toBe('/monitor/server-status');
    expect(routes[0]?.name).toBe('BootstrapGroupMonitor');
    expect(routes[0]?.meta?.titleKey).toBe('monitor.sectionTitle');
    expect(routes[0]?.children?.[0]?.path).toBe('server-status');
    expect(routes[0]?.children?.[0]?.name).toBe('BootstrapGroupMonitorServerStatus');
    expect(routes[0]?.children?.[0]?.redirect).toBe('overview');
    expect(routes[0]?.children?.[0]?.meta?.titleKey).toBe('menu.monitor.server_status.title');
    expect(routes[0]?.children?.[0]?.children).toHaveLength(3);
    expect(routes[0]?.children?.[0]?.children?.[0]?.path).toBe('overview');
    expect(routes[0]?.children?.[0]?.children?.[0]?.name).toBe('MonitorServerStatusOverviewIndex');
    expect(routes[0]?.children?.[0]?.children?.[0]?.meta?.hidden).toBeUndefined();
    expect(routes[0]?.children?.[0]?.children?.[0]?.meta?.titleKey).toBe('menu.monitor.server_status.overview.title');
    expect(routes[0]?.children?.[0]?.children?.[1]?.path).toBe('runtime');
    expect(routes[0]?.children?.[0]?.children?.[1]?.name).toBe('MonitorServerStatusRuntimeIndex');
    expect(routes[0]?.children?.[0]?.children?.[1]?.meta?.titleKey).toBe('menu.monitor.server_status.runtime.title');
    expect(routes[0]?.children?.[0]?.children?.[2]?.path).toBe('dependencies');
    expect(routes[0]?.children?.[0]?.children?.[2]?.name).toBe('MonitorServerStatusDependenciesIndex');
    expect(routes[0]?.children?.[0]?.children?.[2]?.meta?.titleKey).toBe(
      'menu.monitor.server_status.dependencies.title',
    );
  });

  it('规范化尾随斜杠后仍能正确挂载父子菜单', () => {
    const routes = transformBootstrapMenusToRoutes([
      {
        code: 'monitor.section',
        title_key: 'monitor.sectionTitle',
        title: '服务器管理',
        path: '/monitor/',
        icon: 'server',
        permission: '',
      },
      {
        code: 'monitor.server-status',
        title_key: 'menu.monitor.server_status.title',
        title: '服务器状态',
        path: '/monitor/server-status/',
        icon: 'activity',
        permission: '',
      },
      {
        code: 'monitor.server-status.overview',
        title_key: 'menu.monitor.server_status.overview.title',
        title: '概览',
        path: '/monitor/server-status/overview/',
        icon: 'dashboard',
        permission: 'monitor.server-status.read',
      },
      {
        code: 'monitor.server-status.runtime',
        title_key: 'menu.monitor.server_status.runtime.title',
        title: '运行时',
        path: '/monitor/server-status/runtime/',
        icon: 'time',
        permission: 'monitor.server-status.read',
      },
      {
        code: 'monitor.server-status.dependencies',
        title_key: 'menu.monitor.server_status.dependencies.title',
        title: '依赖服务',
        path: '/monitor/server-status/dependencies/',
        icon: 'data-base',
        permission: 'monitor.server-status.read',
      },
    ]);

    expect(routes).toHaveLength(1);
    expect(routes[0]?.path).toBe('/monitor');
    expect(routes[0]?.children?.[0]?.path).toBe('server-status');
    expect(routes[0]?.children?.[0]?.children?.map((child) => child.path)).toEqual([
      'overview',
      'runtime',
      'dependencies',
    ]);
  });
});
