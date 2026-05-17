export const RBAC_API_PATH = {
  ROLES: '/api/roles',
  PERMISSIONS: '/api/permissions',
  ROLE_UPDATE: (roleId: number) => `/api/roles/${roleId}/update`,
  ROLE_PERMISSIONS: (roleId: number) => `/api/roles/${roleId}/permissions`,
  ROLE_PERMISSION_ASSIGN: (roleId: number) => `/api/roles/${roleId}/permissions/assign`,
  USER_ROLE_ASSIGN: (userId: number) => `/api/users/${userId}/roles/assign`,
} as const;
