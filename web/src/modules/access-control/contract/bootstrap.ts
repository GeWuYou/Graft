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
