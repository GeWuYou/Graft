// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { transformBootstrapMenusToRoutes, transformGlobalRegistrationsToRoutes } from './bootstrap';

describe('transformBootstrapMenusToRoutes', () => {
  it('只为当前 web 已接入的 bootstrap 菜单生成动态路由', () => {
    const routes = transformBootstrapMenusToRoutes([
      {
        code: 'access-control.root',
        order: 10,
        title_key: 'menu.access_control.title',
        title: '访问控制',
        path: '/access-control',
        icon: 'secured',
        permission: '',
      },
      {
        code: 'access-control.overview',
        order: 1,
        title_key: 'menu.access_control.overview.title',
        title: '概览',
        path: '/access-control/overview',
        icon: 'dashboard',
        permission: '',
      },
      {
        code: 'user.list',
        order: 2,
        title_key: 'menu.access_control.users.title',
        title: '用户管理',
        path: '/access-control/users',
        icon: 'usergroup',
        permission: 'user.read',
      },
      {
        code: 'role.list',
        order: 3,
        title_key: 'menu.access_control.roles.title',
        title: '角色管理',
        path: '/access-control/roles',
        icon: 'secured',
        permission: 'role.read',
      },
      {
        code: 'unknown.feature',
        title: '未知功能',
        path: '/unknown',
        icon: 'app',
        permission: '',
      },
      {
        code: 'audit.overview',
        order: 1,
        title_key: 'menu.audit.overview.title',
        title: '概览',
        path: '/audit/overview',
        icon: 'dashboard',
        permission: 'audit.read',
      },
      {
        code: 'audit.logs',
        order: 2,
        title_key: 'menu.audit.logs.title',
        title: '审计日志',
        path: '/audit/logs',
        icon: 'history',
        permission: 'audit.read',
      },
    ]);

    expect(routes).toHaveLength(3);
    expect(routes[0]?.path).toBe('/access-control');
    expect(routes[0]?.name).toBe('BootstrapGroupAccessControl');
    expect(routes[0]?.meta?.titleKey).toBe('menu.access_control.title');
    expect(routes[0]?.children?.[0]?.path).toBe('overview');
    expect(routes[0]?.children?.[0]?.name).toBe('AccessControlOverviewIndex');
    expect(routes[0]?.children?.[0]?.meta?.icon).toBe('dashboard');
    expect(routes[0]?.children?.[1]?.path).toBe('users');
    expect(routes[0]?.children?.[1]?.name).toBe('UserListIndex');
    expect(routes[0]?.children?.[1]?.meta?.titleKey).toBe('menu.access_control.users.title');
    expect(routes[0]?.children?.[1]?.meta?.icon).toBe('usergroup');
    expect(routes[0]?.children?.[1]?.meta?.pageKind).toBe('list');
    expect(routes[0]?.children?.[2]?.path).toBe('roles');
    expect(routes[0]?.children?.[2]?.name).toBe('RoleListIndex');
    expect(routes[0]?.children?.[2]?.meta?.titleKey).toBe('menu.access_control.roles.title');
    expect(routes[0]?.children?.[2]?.meta?.icon).toBe('secured');
    expect(routes[1]?.path).toBe('/audit/overview');
    expect(routes[1]?.name).toBe('AuditOverview');
    expect(routes[1]?.meta?.titleKey).toBe('menu.audit.overview.title');
    expect(routes[1]?.meta?.orderNo).toBe(1);
    expect(routes[1]?.meta?.domain).toBe('audit');
    expect(routes[1]?.meta?.dashboard).toBe(true);
    expect(routes[1]?.meta?.pageKind).toBe('overview');
    expect(routes[1]?.meta?.tabTitle?.['zh-CN']).toBe('安全审计 - 概览');
    expect(routes[1]?.children?.[0]?.name).toBe('AuditOverviewIndex');
    expect(routes[2]?.path).toBe('/audit/logs');
    expect(routes[2]?.name).toBe('AuditLogList');
    expect(routes[2]?.meta?.titleKey).toBe('menu.audit.logs.title');
    expect(routes[2]?.meta?.orderNo).toBe(2);
    expect(routes[2]?.meta?.domain).toBe('audit');
    expect(routes[2]?.meta?.pageKind).toBe('list');
    expect(routes[2]?.children?.[0]?.name).toBe('AuditLogListIndex');
  });

  it('按后端返回的规范化访问控制菜单生成分组路由', () => {
    const routes = transformBootstrapMenusToRoutes([
      {
        code: 'access-control.root',
        order: 10,
        title_key: 'menu.access_control.title',
        title: '访问控制',
        path: '/access-control',
        icon: 'secured',
        permission: '',
      },
      {
        code: 'access-control.overview',
        order: 1,
        title_key: 'menu.access_control.overview.title',
        title: '概览',
        path: '/access-control/overview',
        icon: 'dashboard',
        permission: '',
      },
      {
        code: 'user.list',
        order: 2,
        title_key: 'menu.access_control.users.title',
        title: '用户管理',
        path: '/access-control/users',
        icon: 'usergroup',
        permission: 'user.read',
      },
      {
        code: 'role.list',
        order: 3,
        title_key: 'menu.access_control.roles.title',
        title: '角色管理',
        path: '/access-control/roles',
        icon: 'secured',
        permission: 'role.read',
      },
      {
        code: 'permission.list',
        order: 4,
        title_key: 'menu.access_control.permissions.title',
        title: '权限管理',
        path: '/access-control/permissions',
        icon: 'lock-on',
        permission: 'permission.read',
      },
    ]);

    expect(routes).toHaveLength(1);
    expect(routes[0]?.children?.map((child) => child.path)).toEqual(['overview', 'users', 'roles', 'permissions']);
    expect(routes[0]?.children?.[0]?.name).toBe('AccessControlOverviewIndex');
    expect(routes[0]?.children?.[3]?.meta?.icon).toBe('lock-on');
    expect(routes[0]?.children?.[0]?.meta?.domain).toBe('rbac');
    expect(routes[0]?.children?.[0]?.meta?.dashboard).toBe(true);
    expect(routes[0]?.children?.[2]?.meta?.pageKind).toBe('list');
  });

  it('为服务管理模块合成显式父级导航并保持 canonical IA 顺序', () => {
    const routes = transformBootstrapMenusToRoutes([
      {
        code: 'server.section',
        order: 20,
        title_key: 'menu.server.title',
        title: '服务管理',
        path: '/server',
        icon: 'server',
        permission: '',
      },
      {
        code: 'server.overview',
        order: 1,
        title_key: 'menu.server.overview.title',
        title: '概览',
        path: '/server/overview',
        icon: 'dashboard',
        permission: 'monitor.server-status.read',
      },
      {
        code: 'server.runtime',
        order: 2,
        title_key: 'menu.server.runtime.title',
        title: '运行时',
        path: '/server/runtime',
        icon: 'time',
        permission: 'monitor.server-status.read',
      },
      {
        code: 'server.dependencies',
        order: 3,
        title_key: 'menu.server.dependencies.title',
        title: '依赖服务',
        path: '/server/dependencies',
        icon: 'data-base',
        permission: 'monitor.server-status.read',
      },
      {
        code: 'server.modules',
        order: 4,
        title_key: 'menu.server.modules.title',
        title: '模块概览',
        path: '/server/modules',
        icon: 'app',
        permission: 'monitor.server-status.read',
      },
    ]);

    expect(routes).toHaveLength(1);
    expect(routes[0]?.path).toBe('/server');
    expect(routes[0]?.redirect).toBe('/server/overview');
    expect(routes[0]?.name).toBe('BootstrapGroupServer');
    expect(routes[0]?.meta?.titleKey).toBe('menu.server.title');
    expect(routes[0]?.meta?.orderNo).toBe(20);
    expect(routes[0]?.children?.map((child) => child.path)).toEqual(['overview', 'runtime', 'dependencies', 'modules']);
    expect(routes[0]?.children?.[0]?.name).toBe('MonitorServerStatusOverviewIndex');
    expect(routes[0]?.children?.[0]?.meta?.orderNo).toBe(1);
    expect(routes[0]?.children?.[0]?.meta?.domain).toBe('monitor');
    expect(routes[0]?.children?.[0]?.meta?.dashboard).toBe(true);
    expect(routes[0]?.children?.[1]?.name).toBe('MonitorServerStatusRuntimeIndex');
    expect(routes[0]?.children?.[1]?.meta?.orderNo).toBe(2);
    expect(routes[0]?.children?.[1]?.meta?.pageKind).toBe('runtime');
    expect(routes[0]?.children?.[2]?.name).toBe('MonitorServerStatusDependenciesIndex');
    expect(routes[0]?.children?.[2]?.meta?.titleKey).toBe('menu.server.dependencies.title');
    expect(routes[0]?.children?.[2]?.meta?.orderNo).toBe(3);
    expect(routes[0]?.children?.[3]?.name).toBe('MonitorModuleRuntimeOverviewIndex');
    expect(routes[0]?.children?.[3]?.meta?.titleKey).toBe('menu.server.modules.title');
    expect(routes[0]?.children?.[3]?.meta?.breadcrumbTitle?.['zh-CN']).toBe('模块运行时');
    expect(routes[0]?.children?.[3]?.meta?.tabTitle?.['en-US']).toBe('Service Management - Module Runtime');
    expect(routes[0]?.children?.[3]?.meta?.orderNo).toBe(4);
    expect(routes[0]?.children?.[3]?.meta?.pageSurface).toBe('paged-table');
  });

  it('为日志中心访问日志页保留父级 breadcrumb 标题并使用组合 tab 标题', () => {
    const routes = transformBootstrapMenusToRoutes([
      {
        code: 'log-center.root',
        order: 210,
        title_key: 'menu.logCenter.title',
        title: '日志中心',
        path: '/logs',
        icon: 'list',
        permission: '',
      },
      {
        code: 'access-log.list',
        order: 211,
        title_key: 'menu.accessLog.title',
        title: '访问日志',
        path: '/logs/access',
        icon: 'search',
        permission: 'access-log.read',
      },
    ]);

    expect(routes).toHaveLength(1);
    expect(routes[0]?.path).toBe('/logs');
    expect(routes[0]?.name).toBe('BootstrapGroupLogs');
    expect(routes[0]?.meta?.titleKey).toBe('menu.logCenter.title');
    expect(routes[0]?.children?.[0]?.path).toBe('access');
    expect(routes[0]?.children?.[0]?.name).toBe('AccessLogListIndex');
    expect(routes[0]?.children?.[0]?.meta?.titleKey).toBe('menu.accessLog.title');
    expect(routes[0]?.children?.[0]?.meta?.pageKind).toBe('list');
    expect(routes[0]?.children?.[0]?.meta?.breadcrumbTitle?.['zh-CN']).toBe('访问日志');
    expect(routes[0]?.children?.[0]?.meta?.tabTitle?.['zh-CN']).toBe('日志中心 - 访问日志');
    expect(routes[0]?.children?.[0]?.meta?.tabTitle?.['en-US']).toBe('Log Center - Access Logs');
  });

  it('规范化尾随斜杠后仍能正确挂载父子菜单', () => {
    const routes = transformBootstrapMenusToRoutes([
      {
        code: 'server.section',
        order: 20,
        title_key: 'menu.server.title',
        title: '服务管理',
        path: '/server/',
        icon: 'server',
        permission: '',
      },
      {
        code: 'server.overview',
        order: 1,
        title_key: 'menu.server.overview.title',
        title: '概览',
        path: '/server/overview/',
        icon: 'dashboard',
        permission: 'monitor.server-status.read',
      },
      {
        code: 'server.runtime',
        order: 2,
        title_key: 'menu.server.runtime.title',
        title: '运行时',
        path: '/server/runtime/',
        icon: 'time',
        permission: 'monitor.server-status.read',
      },
      {
        code: 'server.dependencies',
        order: 3,
        title_key: 'menu.server.dependencies.title',
        title: '依赖服务',
        path: '/server/dependencies/',
        icon: 'data-base',
        permission: 'monitor.server-status.read',
      },
      {
        code: 'server.modules',
        order: 4,
        title_key: 'menu.server.modules.title',
        title: '模块概览',
        path: '/server/modules/',
        icon: 'app',
        permission: 'monitor.server-status.read',
      },
    ]);

    expect(routes).toHaveLength(1);
    expect(routes[0]?.path).toBe('/server');
    expect(routes[0]?.children?.map((child) => child.path)).toEqual(['overview', 'runtime', 'dependencies', 'modules']);
  });

  it('为服务管理下的公告管理页保留子级 breadcrumb 并派生父级 tab 标题', () => {
    const routes = transformBootstrapMenusToRoutes([
      {
        code: 'server.section',
        order: 20,
        title_key: 'menu.server.title',
        title: '服务管理',
        path: '/server',
        icon: 'server',
        permission: '',
      },
      {
        code: 'server.announcements',
        order: 5,
        title_key: 'menu.server.announcements.title',
        title: '公告管理',
        path: '/server/announcements',
        icon: 'notification',
        permission: 'announcement.read',
      },
    ]);

    expect(routes).toHaveLength(1);
    expect(routes[0]?.path).toBe('/server');
    expect(routes[0]?.children).toHaveLength(1);
    expect(routes[0]?.children?.[0]?.path).toBe('announcements');
    expect(routes[0]?.children?.[0]?.name).toBe('AnnouncementManagementIndex');
    expect(routes[0]?.children?.[0]?.meta?.titleKey).toBe('menu.server.announcements.title');
    expect(routes[0]?.children?.[0]?.meta?.breadcrumbTitle?.['zh-CN']).toBe('公告管理');
    expect(routes[0]?.children?.[0]?.meta?.tabTitle?.['zh-CN']).toBe('服务管理 - 公告管理');
    expect(routes[0]?.children?.[0]?.meta?.tabTitle?.['en-US']).toBe('Service Management - Announcements');
  });

  it('registers menu-hidden global routes at their canonical URL without index redirects', () => {
    const routes = transformGlobalRegistrationsToRoutes([
      {
        path: '/notifications',
        routeName: 'NotificationList',
        loadPage: async () => ({}),
        meta: {
          hiddenMenu: true,
          title: {
            'zh-CN': '通知中心',
            'en-US': 'Notification Center',
          },
          titleKey: 'menu.notification.title',
        },
      },
    ]);

    expect(routes).toHaveLength(1);
    expect(routes[0]?.path).toBe('/notifications');
    expect(routes[0]?.redirect).toBeUndefined();
    expect(routes[0]?.name).toBe('NotificationList');
    expect(routes[0]?.meta?.hiddenMenu).toBe(true);
    expect(routes[0]?.meta?.single).toBe(true);
    expect(routes[0]?.children?.[0]?.path).toBe('');
    expect(routes[0]?.children?.[0]?.name).toBe('NotificationListIndex');
    expect(routes[0]?.children?.[0]?.meta?.hiddenMenu).toBe(true);
    expect(routes[0]?.children?.[0]?.meta?.hiddenBreadcrumb).toBe(true);
  });
});
