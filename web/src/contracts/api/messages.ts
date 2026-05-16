export const MESSAGE_KEY = {
  AUTH_FORBIDDEN: 'auth.forbidden',
} as const;

export type MessageKey = (typeof MESSAGE_KEY)[keyof typeof MESSAGE_KEY];
