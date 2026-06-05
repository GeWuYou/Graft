export const ROOT_ENTRY_PATH = '/';

export const APP_RESULT_ROUTE_PATH = {
  FORBIDDEN: '/result/403',
  NOT_FOUND: '/result/404',
  SERVER_ERROR: '/result/500',
  SUCCESS: '/result/success',
  FAIL: '/result/fail',
  NETWORK_ERROR: '/result/network-error',
  MAINTENANCE: '/result/maintenance',
  BROWSER_INCOMPATIBLE: '/result/browser-incompatible',
} as const;

export const APP_RESULT_ROUTE_NAME = {
  FORBIDDEN: 'Result403',
  NOT_FOUND: 'Result404',
  SERVER_ERROR: 'Result500',
  SUCCESS: 'ResultSuccess',
  FAIL: 'ResultFail',
  NETWORK_ERROR: 'ResultNetworkError',
  MAINTENANCE: 'ResultMaintenance',
  BROWSER_INCOMPATIBLE: 'ResultBrowserIncompatible',
} as const;
