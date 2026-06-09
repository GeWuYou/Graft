// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const RBAC_BOOTSTRAP_ROUTE = {
  ROLE_LIST: {
    menuPath: '/access-control/roles',
    routeName: 'RoleList',
  },
  PERMISSION_LIST: {
    menuPath: '/access-control/permissions',
    routeName: 'PermissionList',
  },
} as const;

export type RbacBootstrapRouteName = (typeof RBAC_BOOTSTRAP_ROUTE)[keyof typeof RBAC_BOOTSTRAP_ROUTE]['routeName'];
