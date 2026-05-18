import { describe, expect, it } from 'vitest';

import {
  buildBootstrapRouteRegistrationMap,
  getBootstrapRouteRegistration,
  resolveModuleRegistrationModulePaths,
} from './index';
import type { WebModuleRegistration } from './types';

describe('module registration aggregation', () => {
  it('exposes the actual module bootstrap registration map', () => {
    expect(getBootstrapRouteRegistration('/users')?.routeName).toBe('UserList');
    expect(getBootstrapRouteRegistration('/roles')?.routeName).toBe('RoleList');
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

  it('only treats directories with bootstrap route declarations as web modules', () => {
    expect(
      resolveModuleRegistrationModulePaths(
        ['./user/index.ts', './rbac/index.ts', './shared/index.ts'],
        ['./user/bootstrap-routes.ts', './rbac/bootstrap-routes.ts'],
      ),
    ).toEqual(['./user/index.ts', './rbac/index.ts']);
  });
});
