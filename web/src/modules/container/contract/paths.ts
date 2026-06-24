export const CONTAINER_ROUTE_PATH = {
  LIST: '/ops/containers',
  DETAIL: '/ops/containers/:id',
} as const;

export const CONTAINER_API_PATH = {
  LIST: '/api/ops/containers',
  DASHBOARD_SUMMARY: '/api/ops/containers/dashboard-summary',
  DETAIL: '/api/ops/containers/{id}',
  LOGS: '/api/ops/containers/{id}/logs',
  SHELL_SESSIONS: '/api/ops/containers/{id}/shell/sessions',
  SHELL_WS: '/api/ops/containers/{id}/shell/ws',
  MOUNTS_USAGE: '/api/ops/containers/{id}/mounts/usage',
  MOUNT_USAGE_REFRESH: '/api/ops/containers/{id}/mounts/{mountId}/usage/refresh',
  START: '/api/ops/containers/{id}/start',
  STOP: '/api/ops/containers/{id}/stop',
  RESTART: '/api/ops/containers/{id}/restart',
  REMOVE: '/api/ops/containers/{id}/remove',
  BATCH_ACTIONS: '/api/ops/containers/batch-actions',
} as const;

export function buildContainerDetailApiPath(containerId: string) {
  return CONTAINER_API_PATH.DETAIL.replace('{id}', encodeContainerPathParam(containerId));
}

/**
 * Builds the API path for retrieving logs of a specific container.
 *
 * @param containerId - The ID of the container
 * @returns The API path for fetching the container's logs
 */
export function buildContainerLogsApiPath(containerId: string) {
  return CONTAINER_API_PATH.LOGS.replace('{id}', encodeContainerPathParam(containerId));
}

/**
 * Builds the API path for accessing container shell sessions.
 *
 * @param containerId - The container identifier
 * @returns The container shell sessions API path
 */
export function buildContainerShellSessionsApiPath(containerId: string) {
  return CONTAINER_API_PATH.SHELL_SESSIONS.replace('{id}', encodeContainerPathParam(containerId));
}

/**
 * Constructs the API path for retrieving mount usage information for a container.
 *
 * @param containerId - The container identifier
 * @returns The API path for querying container mount usage
 */
export function buildContainerMountUsageApiPath(containerId: string) {
  return CONTAINER_API_PATH.MOUNTS_USAGE.replace('{id}', encodeContainerPathParam(containerId));
}

/**
 * Generates an API path for refreshing a container mount's usage.
 *
 * @param containerId - The container's identifier
 * @param mountId - The mount's identifier
 * @returns The API path for mount usage refresh with the container and mount IDs properly encoded
 */
export function buildContainerMountUsageRefreshApiPath(containerId: string, mountId: string) {
  return CONTAINER_API_PATH.MOUNT_USAGE_REFRESH.replace('{id}', encodeContainerPathParam(containerId)).replace(
    '{mountId}',
    encodeContainerPathParam(mountId),
  );
}

/**
 * Builds the API path for starting a container.
 *
 * @returns The API path for starting the container
 */
export function buildContainerStartApiPath(containerId: string) {
  return CONTAINER_API_PATH.START.replace('{id}', encodeContainerPathParam(containerId));
}

export function buildContainerStopApiPath(containerId: string) {
  return CONTAINER_API_PATH.STOP.replace('{id}', encodeContainerPathParam(containerId));
}

export function buildContainerRestartApiPath(containerId: string) {
  return CONTAINER_API_PATH.RESTART.replace('{id}', encodeContainerPathParam(containerId));
}

export function buildContainerRemoveApiPath(containerId: string) {
  return CONTAINER_API_PATH.REMOVE.replace('{id}', encodeContainerPathParam(containerId));
}

function encodeContainerPathParam(containerId: string) {
  return encodeURIComponent(containerId);
}
