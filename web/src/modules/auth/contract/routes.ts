export const AUTH_ROUTE_NAME = {
  LOGIN: 'login',
  RESTRICTED_SESSION: 'RestrictedSession',
} as const;

export const AUTH_ROUTE_PATH = {
  LOGIN: '/login',
  RESTRICTED_SESSION: '/auth/restricted-session',
} as const;
