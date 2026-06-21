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
    expect(getBootstrapRouteRegistration('/access-control/overview')?.routeName).toBe('AccessControlOverview');
    expect(getBootstrapRouteRegistration('/access-control/users')?.routeName).toBe('UserList');
    expect(getBootstrapRouteRegistration('/access-control/roles')?.routeName).toBe('RoleList');
    expect(getBootstrapRouteRegistration('/access-control/permissions')?.routeName).toBe('PermissionList');
    expect(getBootstrapRouteRegistration('/server/overview')?.routeName).toBe('MonitorServerStatusOverview');
    expect(getBootstrapRouteRegistration('/server/runtime')?.routeName).toBe('MonitorServerStatusRuntime');
    expect(getBootstrapRouteRegistration('/server/dependencies')?.routeName).toBe('MonitorServerStatusDependencies');
    expect(getBootstrapRouteRegistration('/server/modules')?.routeName).toBe('MonitorModuleRuntimeOverview');
    expect(getBootstrapRouteRegistration('/server/scheduled-tasks')?.routeName).toBe('ScheduledTaskList');
    expect(getBootstrapRouteRegistration('/server/system-config')?.routeName).toBe('SystemConfigList');
    expect(getBootstrapRouteRegistration('/audit/overview')?.routeName).toBe('AuditOverview');
    expect(getBootstrapRouteRegistration('/audit/logs')?.routeName).toBe('AuditLogList');
    expect(getBootstrapRouteRegistration('/notifications')).toBeUndefined();
    expect(getGlobalRouteRegistrations().find((route) => route.path === '/notifications')?.routeName).toBe(
      'NotificationList',
    );
  });

  it('rejects duplicate menu paths', () => {
    const registrations: WebModuleRegistration[] = [
      {
        moduleId: 'user',
        bootstrapRoutes: [
          {
            menuPath: '/users',
            routeName: 'UserList',
            loadPage: async () => ({}),
          },
        ],
      },
      {
        moduleId: 'audit',
        bootstrapRoutes: [
          {
            menuPath: '/users',
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
            menuPath: '/users',
            routeName: 'List',
            loadPage: async () => ({}),
          },
        ],
      },
      {
        moduleId: 'rbac',
        bootstrapRoutes: [
          {
            menuPath: '/roles',
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
            menuPath: '/users',
            routeName: 'RoleIndex',
            loadPage: async () => ({}),
          },
        ],
      },
      {
        moduleId: 'rbac',
        bootstrapRoutes: [
          {
            menuPath: '/roles',
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
