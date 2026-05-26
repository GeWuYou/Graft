/**
 * USER_ROUTE_PATH 定义用户管理模块的 canonical 前端路由入口。
 *
 * `LEGACY_LIST` 仅用于兼容旧菜单或旧跳转路径；当上下游不再产出旧路径时应移除。
 */
export const USER_ROUTE_PATH = {
  LIST: '/access-control/users',
  LEGACY_LIST: '/users',
} as const;

/**
 * USER_API_PATH 定义用户管理模块访问 `server` 的稳定接口路径契约。
 *
 * @param userId 工厂函数中的用户主键 ID，必须对应目标用户记录。
 */
export const USER_API_PATH = {
  USERS: '/api/users',
  USER_BY_ID_TEMPLATE: '/api/users/{id}',
  USER_UPDATE_TEMPLATE: '/api/users/{id}/update',
  USER_STATUS_TEMPLATE: '/api/users/{id}/status',
  USER_RESET_PASSWORD_TEMPLATE: '/api/users/{id}/reset-password',
  USER_DELETE_TEMPLATE: '/api/users/{id}/delete',
  USER_SESSIONS_TEMPLATE: '/api/users/{id}/sessions',
  USER_SESSIONS_REVOKE_ALL_TEMPLATE: '/api/users/{id}/sessions/revoke-all',
  USER_SESSION_REVOKE_TEMPLATE: '/api/users/{id}/sessions/{sessionID}/revoke',
  /** USER_BY_ID 返回读取指定用户详情的接口路径。 */
  USER_BY_ID: (userId: number) => `/api/users/${userId}`,
  /** USER_UPDATE 返回更新指定用户资料的接口路径。 */
  USER_UPDATE: (userId: number) => `/api/users/${userId}/update`,
  /** USER_STATUS 返回更新指定用户启停状态的接口路径。 */
  USER_STATUS: (userId: number) => `/api/users/${userId}/status`,
  /** USER_RESET_PASSWORD 返回重置指定用户密码的接口路径。 */
  USER_RESET_PASSWORD: (userId: number) => `/api/users/${userId}/reset-password`,
  /** USER_DELETE 返回删除指定用户的接口路径。 */
  USER_DELETE: (userId: number) => `/api/users/${userId}/delete`,
} as const;
