/**
 * USER_ROUTE_PATH 定义用户管理模块的 canonical 前端路由入口。
 */
export const USER_ROUTE_PATH = {
  LIST: '/security/users',
} as const;

/**
 * USER_API_PATH 定义用户管理模块访问 `server` 的稳定接口路径契约。
 *
 * @param userId 工厂函数中的用户主键 ID，必须对应目标用户记录。
 */
export const USER_API_PATH = {
  USERS: '/api/users',
  ROLES: '/api/roles',
  USER_BY_ID_TEMPLATE: '/api/users/{id}',
  USER_UPDATE_TEMPLATE: '/api/users/{id}/update',
  USER_STATUS_TEMPLATE: '/api/users/{id}/status',
  USER_RESET_PASSWORD_TEMPLATE: '/api/users/{id}/reset-password',
  USER_DELETE_TEMPLATE: '/api/users/{id}/delete',
  USER_SESSIONS_TEMPLATE: '/api/users/{id}/sessions',
  USER_SESSIONS_REVOKE_ALL_TEMPLATE: '/api/users/{id}/sessions/revoke-all',
  USER_SESSION_REVOKE_TEMPLATE: '/api/users/{id}/sessions/{sessionID}/revoke',
  USER_ROLES_TEMPLATE: '/api/users/{id}/roles',
  USER_ROLE_REPLACE_TEMPLATE: '/api/users/{id}/roles/replace',
  USER_ROLE_ADD_TEMPLATE: '/api/users/{id}/roles/add',
  USER_ROLE_REMOVE_TEMPLATE: '/api/users/{id}/roles/remove',
  BATCH_USER_ROLE_REPLACE: '/api/users/roles/replace',
  BATCH_USER_ROLE_ADD: '/api/users/roles/add',
  BATCH_USER_ROLE_REMOVE: '/api/users/roles/remove',
  USER_BY_ID: (userId: number) => `/api/users/${userId}`,
  USER_UPDATE: (userId: number) => `/api/users/${userId}/update`,
  USER_STATUS: (userId: number) => `/api/users/${userId}/status`,
  USER_RESET_PASSWORD: (userId: number) => `/api/users/${userId}/reset-password`,
  USER_DELETE: (userId: number) => `/api/users/${userId}/delete`,
  USER_ROLES: (userId: number) => `/api/users/${userId}/roles`,
  USER_ROLE_REPLACE: (userId: number) => `/api/users/${userId}/roles/replace`,
  USER_ROLE_ADD: (userId: number) => `/api/users/${userId}/roles/add`,
  USER_ROLE_REMOVE: (userId: number) => `/api/users/${userId}/roles/remove`,
} as const;
