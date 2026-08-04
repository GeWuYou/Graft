import { describe, expect, it } from 'vitest';

import {
  buildBootstrapRouteRegistrationMap,
  buildGlobalRouteRegistrations,
  getBootstrapRouteRegistration,
  getGlobalRouteRegistrations,
  resolveModuleRegistrationModulePaths,
} from './index';
import type { WebModuleRegistration } from './types';

describe('module registration aggregation', () => {
  it('exposes the actual module bootstrap registration map', () => {
    expect(getBootstrapRouteRegistration('/security/overview')?.routeName).toBe('SecurityOverview');
    expect(getBootstrapRouteRegistration('/security/users')?.routeName).toBe('UserList');
    expect(getBootstrapRouteRegistration('/security/roles')?.routeName).toBe('RoleList');
    expect(getBootstrapRouteRegistration('/security/permissions')?.routeName).toBe('PermissionList');
    expect(getBootstrapRouteRegistration('/observability/overview')?.routeName).toBe('MonitorServerStatusOverview');
    expect(getBootstrapRouteRegistration('/observability/service-status')?.routeName).toBe(
      'MonitorServerStatusRuntime',
    );
    expect(getBootstrapRouteRegistration('/observability/dependencies')?.routeName).toBe(
      'MonitorServerStatusDependencies',
    );
    expect(getBootstrapRouteRegistration('/observability/modules')?.routeName).toBe('MonitorModuleRuntimeOverview');
    expect(getBootstrapRouteRegistration('/platform/scheduled-tasks')?.routeName).toBe('ScheduledTaskList');
    expect(getBootstrapRouteRegistration('/platform/system-config')?.routeName).toBe('SystemConfigList');
    expect(getBootstrapRouteRegistration('/platform/network')?.routeName).toBe('PlatformNetworkConnectivity');
    expect(getBootstrapRouteRegistration('/infrastructure/images')?.routeName).toBe('DockerImageList');
    expect(getBootstrapRouteRegistration('/audit/overview')).toBeUndefined();
    expect(getBootstrapRouteRegistration('/security/audit')?.routeName).toBe('AuditLogList');
    expect(getBootstrapRouteRegistration('/notifications')).toBeUndefined();
    expect(getGlobalRouteRegistrations().find((route) => route.path === '/notifications')?.routeName).toBe(
      'NotificationList',
    );
    expect(getGlobalRouteRegistrations().find((route) => route.path === '/platform/network/outbound')?.routeName).toBe(
      'PlatformNetworkOutbound',
    );
    expect(getGlobalRouteRegistrations().find((route) => route.path === '/platform/network/:targetId')?.routeName).toBe(
      'PlatformNetworkConnectivityDiagnostics',
    );
  });

  it('rejects duplicate menu paths', () => {
    const registrations: WebModuleRegistration[] = [
      {
        moduleId: 'user',
        bootstrapRoutes: [
          {
            menuPath: '/security/users',
            routeName: 'UserList',
            loadPage: async () => ({}),
          },
        ],
      },
      {
        moduleId: 'audit',
        bootstrapRoutes: [
          {
            menuPath: '/security/users',
            routeName: 'AuditList',
            loadPage: async () => ({}),
          },
        ],
      },
    ];

    expect(() => buildBootstrapRouteRegistrationMap(registrations)).toThrow(/duplicate bootstrap route registration/);
  });

  it('rejects duplicate stable route names and derived child route names', () => {
    const duplicateParentNameRegistrations: WebModuleRegistration[] = [
      {
        moduleId: 'user',
        bootstrapRoutes: [
          {
            menuPath: '/security/users',
            routeName: 'List',
            loadPage: async () => ({}),
          },
        ],
      },
      {
        moduleId: 'rbac',
        bootstrapRoutes: [
          {
            menuPath: '/security/roles',
            routeName: 'List',
            loadPage: async () => ({}),
          },
        ],
      },
    ];

    expect(() => buildBootstrapRouteRegistrationMap(duplicateParentNameRegistrations)).toThrow(
      /duplicate bootstrap route name \(parent\)/,
    );

    const duplicateChildNameRegistrations: WebModuleRegistration[] = [
      {
        moduleId: 'user',
        bootstrapRoutes: [
          {
            menuPath: '/security/users',
            routeName: 'RoleIndex',
            loadPage: async () => ({}),
          },
        ],
      },
      {
        moduleId: 'rbac',
        bootstrapRoutes: [
          {
            menuPath: '/security/roles',
            routeName: 'Role',
            loadPage: async () => ({}),
          },
        ],
      },
    ];

    expect(() => buildBootstrapRouteRegistrationMap(duplicateChildNameRegistrations)).toThrow(
      /duplicate bootstrap route name \(child\)/,
    );
  });

  it('rejects stable route name collisions across bootstrap and global registrations', () => {
    const duplicateCrossRegistryRegistrations: WebModuleRegistration[] = [
      {
        moduleId: 'notification',
        bootstrapRoutes: [],
        globalRoutes: [
          {
            path: '/notifications',
            routeName: 'NotificationList',
            loadPage: async () => ({}),
            meta: {},
          },
        ],
      },
      {
        moduleId: 'audit',
        bootstrapRoutes: [
          {
            menuPath: '/audit/overview',
            routeName: 'NotificationList',
            loadPage: async () => ({}),
          },
        ],
      },
    ];

    expect(() => buildBootstrapRouteRegistrationMap(duplicateCrossRegistryRegistrations)).toThrow(
      /duplicate bootstrap route name \(parent\)/,
    );
    expect(() => buildGlobalRouteRegistrations(duplicateCrossRegistryRegistrations)).toThrow(
      /duplicate bootstrap route name \(parent\)/,
    );
  });

  it('only treats directories with bootstrap route declarations as web modules', () => {
    expect(
      resolveModuleRegistrationModulePaths(
        ['./user/index.ts', './rbac/index.ts', './shared/index.ts'],
        ['./user/bootstrap-routes.ts', './rbac/bootstrap-routes.ts'],
      ),
    ).toEqual(['./user/index.ts', './rbac/index.ts']);
  });

  it('rejects duplicate global route paths and route names', () => {
    const duplicatePathRegistrations: WebModuleRegistration[] = [
      {
        moduleId: 'notification',
        bootstrapRoutes: [],
        globalRoutes: [
          {
            path: '/notifications',
            routeName: 'NotificationList',
            loadPage: async () => ({}),
            meta: {},
          },
        ],
      },
      {
        moduleId: 'audit',
        bootstrapRoutes: [],
        globalRoutes: [
          {
            path: '/notifications',
            routeName: 'AuditNotificationList',
            loadPage: async () => ({}),
            meta: {},
          },
        ],
      },
    ];

    expect(() => buildGlobalRouteRegistrations(duplicatePathRegistrations)).toThrow(/duplicate global route path/);

    const duplicateNameRegistrations: WebModuleRegistration[] = [
      {
        moduleId: 'notification',
        bootstrapRoutes: [],
        globalRoutes: [
          {
            path: '/notifications',
            routeName: 'NotificationList',
            loadPage: async () => ({}),
            meta: {},
          },
        ],
      },
      {
        moduleId: 'audit',
        bootstrapRoutes: [],
        globalRoutes: [
          {
            path: '/audit-notifications',
            routeName: 'NotificationList',
            loadPage: async () => ({}),
            meta: {},
          },
        ],
      },
    ];

    expect(() => buildGlobalRouteRegistrations(duplicateNameRegistrations)).toThrow(
      /duplicate bootstrap route name \(parent\)/,
    );

    const duplicateBootstrapChildNameRegistrations: WebModuleRegistration[] = [
      {
        moduleId: 'notification',
        bootstrapRoutes: [],
        globalRoutes: [
          {
            path: '/notifications',
            routeName: 'AuditOverview',
            loadPage: async () => ({}),
            meta: {},
          },
        ],
      },
      {
        moduleId: 'audit',
        bootstrapRoutes: [
          {
            menuPath: '/audit/overview',
            routeName: 'AuditOverviewIndex',
            loadPage: async () => ({}),
          },
        ],
      },
    ];

    expect(() => buildGlobalRouteRegistrations(duplicateBootstrapChildNameRegistrations)).toThrow(
      /duplicate bootstrap route name \(parent\)/,
    );

    const duplicateGlobalChildNameRegistrations: WebModuleRegistration[] = [
      {
        moduleId: 'notification',
        bootstrapRoutes: [],
        globalRoutes: [
          {
            path: '/notifications',
            routeName: 'NotificationListIndex',
            loadPage: async () => ({}),
            meta: {},
          },
        ],
      },
      {
        moduleId: 'audit',
        bootstrapRoutes: [
          {
            menuPath: '/audit/overview',
            routeName: 'NotificationList',
            loadPage: async () => ({}),
          },
        ],
      },
    ];

    expect(() => buildGlobalRouteRegistrations(duplicateGlobalChildNameRegistrations)).toThrow(
      /duplicate bootstrap route name \(child\)/,
    );
  });

  it('returns defensive copies of global route registrations', () => {
    const firstRoutes = getGlobalRouteRegistrations();
    const originalLength = firstRoutes.length;

    firstRoutes.pop();

    expect(getGlobalRouteRegistrations()).toHaveLength(originalLength);
  });
});
