export const RBAC_API_PATH = {
  ROLES: '/api/roles',
  PERMISSIONS: '/api/permissions',
  ROLE_UPDATE_TEMPLATE: '/api/roles/{id}/update',
  ROLE_PERMISSIONS_TEMPLATE: '/api/roles/{id}/permissions',
  ROLE_PERMISSION_ASSIGN_TEMPLATE: '/api/roles/{id}/permissions/assign',
  USER_ROLES_TEMPLATE: '/api/users/{id}/roles',
  USER_ROLE_ASSIGN_TEMPLATE: '/api/users/{id}/roles/assign',
  ROLE_UPDATE: (roleId: number) => `/api/roles/${roleId}/update`,
  ROLE_PERMISSIONS: (roleId: number) => `/api/roles/${roleId}/permissions`,
  ROLE_PERMISSION_ASSIGN: (roleId: number) => `/api/roles/${roleId}/permissions/assign`,
  USER_ROLES: (userId: number) => `/api/users/${userId}/roles`,
  USER_ROLE_ASSIGN: (userId: number) => `/api/users/${userId}/roles/assign`,
} as const;
