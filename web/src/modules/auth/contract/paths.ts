export const AUTH_API_PATH = {
  BOOTSTRAP: '/api/auth/bootstrap',
  CHANGE_PASSWORD: '/api/auth/change-password',
  COMPLETE_REQUIRED_PASSWORD_CHANGE: '/api/auth/complete-required-password-change',
  LOGIN: '/api/auth/login',
  LOGOUT: '/api/auth/logout',
  REFRESH: '/api/auth/refresh',
  SESSIONS: '/api/auth/sessions',
  SESSION_REVOKE_TEMPLATE: '/api/auth/sessions/{sessionID}/revoke',
  SESSIONS_REVOKE_ALL: '/api/auth/sessions/revoke-all',
  SESSIONS_REVOKE_OTHERS: '/api/auth/sessions/revoke-others',
} as const;
