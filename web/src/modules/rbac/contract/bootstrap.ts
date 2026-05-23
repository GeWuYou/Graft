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
