import { describe, expect, it } from 'vitest';
import type { RouteRecordRaw } from 'vue-router';

import { buildDashboardQuickActionLinks } from './sidebar-quick-actions';

function asRouteRecordRaw<T extends object>(route: T) {
  return route as unknown as RouteRecordRaw;
}

describe('buildDashboardQuickActionLinks', () => {
  it('keeps only visible sidebar leaf routes and preserves route title/icon truth', () => {
    const routes = [
      asRouteRecordRaw({
        path: '/access-control',
        name: 'BootstrapGroupAccessControl',
        meta: {
          titleKey: 'menu.accessControl',
          title: {
            'zh-CN': '访问控制',
            'en-US': 'Access Control',
          },
        },
        children: [
          asRouteRecordRaw({
            path: 'roles',
            name: 'RoleListIndex',
            meta: {
              icon: 'secured',
              orderNo: 20,
              permission: 'rbac.role.read',
              breadcrumbTitle: {
                'zh-CN': '角色管理',
                'en-US': 'Role Management',
              },
              tabTitle: {
                'zh-CN': '访问控制 - 角色管理',
                'en-US': 'Access Control - Role Management',
              },
              titleKey: 'rbac.role.list.title',
            },
          }),
          asRouteRecordRaw({
            path: 'permissions',
            name: 'PermissionListIndex',
            meta: {
              hiddenMenu: true,
              orderNo: 30,
              tabTitle: {
                'zh-CN': '访问控制 - 权限管理',
                'en-US': 'Access Control - Permissions',
              },
              titleKey: 'rbac.permission.list.title',
            },
          }),
        ],
      }),
      asRouteRecordRaw({
        path: '/ops/containers',
        name: 'ContainerList',
        meta: {
          icon: 'layers',
          orderNo: 10,
          single: true,
          title: {
            'zh-CN': '运维管理',
            'en-US': 'Operations',
          },
          breadcrumbTitle: {
            'zh-CN': '容器管理',
            'en-US': 'Container Management',
          },
          tabTitle: {
            'zh-CN': '运维管理 - 容器管理',
            'en-US': 'Operations - Container Management',
          },
          titleKey: 'container.route.list.title',
        },
        children: [
          asRouteRecordRaw({
            path: 'index',
            name: 'ContainerListIndex',
            meta: {
              hidden: true,
              titleKey: 'container.route.list.title',
            },
          }),
        ],
      }),
      asRouteRecordRaw({
        path: '/monitor',
        name: 'BootstrapGroupMonitor',
        meta: {
          titleKey: 'menu.server.title',
          orderNo: 5,
          title: {
            'zh-CN': '服务管理',
            'en-US': 'Service Management',
          },
        },
        children: [
          asRouteRecordRaw({
            path: 'overview',
            name: 'MonitorOverviewIndex',
            meta: {
              orderNo: 5,
              breadcrumbTitle: {
                'zh-CN': '服务概览',
                'en-US': 'Overview',
              },
              tabTitle: {
                'zh-CN': '服务管理 - 概览',
                'en-US': 'Service Management - Overview',
              },
              titleKey: 'monitor.route.overview.title',
            },
          }),
        ],
      }),
      asRouteRecordRaw({
        path: '/notifications',
        name: 'NotificationList',
        meta: {
          hiddenMenu: true,
          tabTitle: {
            'zh-CN': '通知中心',
            'en-US': 'Notifications',
          },
          titleKey: 'notification.route.list.title',
        },
      }),
    ] as RouteRecordRaw[];

    expect(buildDashboardQuickActionLinks(routes, 'zh-CN')).toEqual([
      {
        full_label: '服务管理 - 概览',
        group: '服务管理',
        group_key: 'menu.server.title',
        icon: undefined,
        id: 'MonitorOverviewIndex',
        module_key: 'monitor',
        order: 5,
        route_location: '/monitor/overview',
        title: '服务概览',
        title_key: 'monitor.route.overview.title',
      },
      {
        full_label: '运维管理 - 容器管理',
        group: '运维管理',
        group_key: 'container.route.list.title',
        icon: 'layers',
        id: 'ContainerList',
        module_key: 'container',
        order: 10,
        route_location: '/ops/containers',
        title: '容器管理',
        title_key: 'container.route.list.title',
      },
      {
        full_label: '访问控制 - 角色管理',
        group: '访问控制',
        group_key: 'menu.accessControl',
        icon: 'secured',
        id: 'RoleListIndex',
        module_key: 'rbac',
        order: 20,
        required_permissions: ['rbac.role.read'],
        route_location: '/access-control/roles',
        title: '角色管理',
        title_key: 'rbac.role.list.title',
      },
    ]);
  });

  it('uses the requested locale instead of hard-coding english titles', () => {
    const routes = [
      asRouteRecordRaw({
        path: '/monitor',
        name: 'BootstrapGroupMonitor',
        meta: {
          titleKey: 'menu.server.title',
          title: {
            'zh-CN': '服务管理',
            'en-US': 'Service Management',
          },
        },
        children: [
          asRouteRecordRaw({
            path: 'overview',
            name: 'MonitorOverviewIndex',
            meta: {
              breadcrumbTitle: {
                'zh-CN': '概览',
                'en-US': 'Overview',
              },
              tabTitle: {
                'zh-CN': '服务管理 - 概览',
                'en-US': 'Service Management - Overview',
              },
              titleKey: 'monitor.route.overview.title',
            },
          }),
        ],
      }),
    ] as RouteRecordRaw[];

    expect(buildDashboardQuickActionLinks(routes, 'en-US')).toEqual([
      {
        full_label: 'Service Management - Overview',
        group: 'Service Management',
        group_key: 'menu.server.title',
        icon: undefined,
        id: 'MonitorOverviewIndex',
        module_key: 'monitor',
        order: 0,
        route_location: '/monitor/overview',
        title: 'Overview',
        title_key: 'monitor.route.overview.title',
      },
    ]);
  });

  it('derives split title and group from a single top-level route without string splitting', () => {
    const routes = [
      asRouteRecordRaw({
        path: '/ops/containers',
        name: 'ContainerList',
        meta: {
          icon: 'layers',
          orderNo: 10,
          single: true,
          title: {
            'zh-CN': '运维管理',
            'en-US': 'Operations',
          },
          breadcrumbTitle: {
            'zh-CN': '容器管理',
            'en-US': 'Container Management',
          },
          tabTitle: {
            'zh-CN': '运维管理 - 容器管理',
            'en-US': 'Operations - Container Management',
          },
          titleKey: 'container.route.list.title',
        },
      }),
    ] as RouteRecordRaw[];

    expect(buildDashboardQuickActionLinks(routes, 'en-US')).toEqual([
      {
        full_label: 'Operations - Container Management',
        group: 'Operations',
        group_key: 'container.route.list.title',
        icon: 'layers',
        id: 'ContainerList',
        module_key: 'container',
        order: 10,
        route_location: '/ops/containers',
        title: 'Container Management',
        title_key: 'container.route.list.title',
      },
    ]);
  });

  it('keeps a single route as the quick action even when it has visible child routes', () => {
    const routes = [
      asRouteRecordRaw({
        path: '/ops/containers',
        name: 'ContainerList',
        meta: {
          icon: 'layers',
          orderNo: 10,
          single: true,
          title: {
            'zh-CN': '运维管理',
            'en-US': 'Operations',
          },
          breadcrumbTitle: {
            'zh-CN': '容器管理',
            'en-US': 'Container Management',
          },
          tabTitle: {
            'zh-CN': '运维管理 - 容器管理',
            'en-US': 'Operations - Container Management',
          },
          titleKey: 'container.route.list.title',
          permission: 'container.read',
        },
        children: [
          asRouteRecordRaw({
            path: 'runtime',
            name: 'ContainerRuntimeIndex',
            meta: {
              orderNo: 20,
              breadcrumbTitle: {
                'zh-CN': '运行时',
                'en-US': 'Runtime',
              },
              tabTitle: {
                'zh-CN': '运维管理 - 运行时',
                'en-US': 'Operations - Runtime',
              },
              titleKey: 'container.route.runtime.title',
            },
          }),
        ],
      }),
    ] as RouteRecordRaw[];

    expect(buildDashboardQuickActionLinks(routes, 'en-US')).toEqual([
      {
        full_label: 'Operations - Container Management',
        group: 'Operations',
        group_key: 'container.route.list.title',
        icon: 'layers',
        id: 'ContainerList',
        module_key: 'container',
        order: 10,
        required_permissions: ['container.read'],
        route_location: '/ops/containers',
        title: 'Container Management',
        title_key: 'container.route.list.title',
      },
    ]);
  });
});
