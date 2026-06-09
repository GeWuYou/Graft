// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const ACCESS_CONTROL_ROUTE_PATH = {
  ROOT: '/access-control',
  OVERVIEW: '/access-control/overview',
  USERS: '/access-control/users',
  ROLES: '/access-control/roles',
  PERMISSIONS: '/access-control/permissions',
} as const;

export const ACCESS_CONTROL_BOOTSTRAP_ROUTE = {
  OVERVIEW: {
    menuPath: ACCESS_CONTROL_ROUTE_PATH.OVERVIEW,
    routeName: 'AccessControlOverview',
  },
} as const;
