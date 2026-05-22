export const USER_ROUTE_PATH = {
  LIST: '/access-control/users',
  LEGACY_LIST: '/users',
} as const;

export const USER_API_PATH = {
  USERS: '/api/users',
  USER_UPDATE: (userId: number) => `/api/users/${userId}/update`,
  USER_STATUS: (userId: number) => `/api/users/${userId}/status`,
  USER_RESET_PASSWORD: (userId: number) => `/api/users/${userId}/reset-password`,
  USER_DELETE: (userId: number) => `/api/users/${userId}/delete`,
} as const;
