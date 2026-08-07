export const ROOT_ENTRY_PATH = '/';

export const APP_RESULT_ROUTE_PATH = {
  FORBIDDEN: '/result/403',
  NOT_FOUND: '/result/404',
  SERVER_ERROR: '/result/500',
  SUCCESS: '/result/success',
  FAIL: '/result/fail',
  NETWORK_ERROR: '/result/network-error',
  SERVICE_UNAVAILABLE: '/result/service-unavailable',
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
  SERVICE_UNAVAILABLE: 'ResultServiceUnavailable',
  MAINTENANCE: 'ResultMaintenance',
  BROWSER_INCOMPATIBLE: 'ResultBrowserIncompatible',
} as const;

function isServiceUnavailableRoutePath(path: string | null | undefined): path is string {
  return (
    typeof path === 'string' &&
    (path === APP_RESULT_ROUTE_PATH.SERVICE_UNAVAILABLE ||
      path.startsWith(`${APP_RESULT_ROUTE_PATH.SERVICE_UNAVAILABLE}?`) ||
      path.startsWith(`${APP_RESULT_ROUTE_PATH.SERVICE_UNAVAILABLE}#`))
  );
}

export function resolveRecoveryRoutePath(
  requestedPath: string | null | undefined,
  pendingPath: string | null | undefined,
): string {
  if (typeof requestedPath === 'string' && requestedPath.length > 0 && !isServiceUnavailableRoutePath(requestedPath)) {
    return requestedPath;
  }
  if (typeof pendingPath === 'string' && pendingPath.length > 0 && !isServiceUnavailableRoutePath(pendingPath)) {
    return pendingPath;
  }
  return ROOT_ENTRY_PATH;
}
