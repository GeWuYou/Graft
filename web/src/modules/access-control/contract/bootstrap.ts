export const ACCESS_CONTROL_ROUTE_PATH = {
  ROOT: '/access-control',
  OVERVIEW: '/access-control/overview',
  USERS: '/access-control/users',
  ROLES: '/access-control/roles',
  PERMISSIONS: '/access-control/permissions',
  LEGACY_USERS: '/users',
  LEGACY_ROLES: '/roles',
  LEGACY_PERMISSIONS: '/permissions',
} as const;

export const ACCESS_CONTROL_BOOTSTRAP_ROUTE = {
  OVERVIEW: {
    menuPath: ACCESS_CONTROL_ROUTE_PATH.OVERVIEW,
    routeName: 'AccessControlOverview',
  },
} as const;
