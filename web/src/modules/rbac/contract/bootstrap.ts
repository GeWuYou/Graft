export const RBAC_BOOTSTRAP_ROUTE = {
  ROLE_LIST: {
    menuPath: '/roles',
    routeName: 'RoleList',
  },
} as const;

export type RbacBootstrapRouteName = (typeof RBAC_BOOTSTRAP_ROUTE)[keyof typeof RBAC_BOOTSTRAP_ROUTE]['routeName'];
