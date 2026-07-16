export const CONTAINER_ROUTE_PATH = {
  LIST: '/infrastructure/docker/containers',
  DETAIL: '/infrastructure/docker/containers/:id',
  RESOURCES: '/infrastructure/docker/containers/resources',
} as const;

export const CONTAINER_API_PATH = {
  LIST: '/api/ops/containers',
  DASHBOARD_SUMMARY: '/api/ops/containers/dashboard-summary',
  DETAIL: '/api/ops/containers/{id}',
  EVENTS: '/api/ops/containers/{id}/events',
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
  DOCKER_IMAGES: '/api/ops/docker/images',
  DOCKER_NETWORKS: '/api/ops/docker/networks',
  DOCKER_VOLUMES: '/api/ops/docker/volumes',
  DOCKER_SYSTEM: '/api/ops/docker/system',
} as const;

/**
 * 构建容器详情接口的请求路径。
 *
 * @param containerId - 容器标识符
 * @returns 包含编码后容器标识符的容器详情接口路径
 */
export function buildContainerDetailApiPath(containerId: string) {
  return CONTAINER_API_PATH.DETAIL.replace('{id}', encodeContainerPathParam(containerId));
}

/**
 * 构建指定容器日志的 API 路径。
 *
 * @param containerId - 容器 ID
 * @returns 容器日志的 API 路径
 */
export function buildContainerLogsApiPath(containerId: string) {
  return CONTAINER_API_PATH.LOGS.replace('{id}', encodeContainerPathParam(containerId));
}

/**
 * 构造容器事件的 API 路径。
 *
 * @param containerId - 容器 ID
 * @returns 替换了 `{id}` 占位符的事件接口路径
 */
export function buildContainerEventsApiPath(containerId: string) {
  return CONTAINER_API_PATH.EVENTS.replace('{id}', encodeContainerPathParam(containerId));
}

/**
 * 构建容器终端会话接口路径。
 *
 * @param containerId - 容器标识
 * @returns 容器终端会话接口路径
 */
export function buildContainerShellSessionsApiPath(containerId: string) {
  return CONTAINER_API_PATH.SHELL_SESSIONS.replace('{id}', encodeContainerPathParam(containerId));
}

/**
 * 构建查询容器挂载使用量的接口路径。
 *
 * @param containerId - 容器标识
 * @returns 查询容器挂载使用量的接口路径
 */
export function buildContainerMountUsageApiPath(containerId: string) {
  return CONTAINER_API_PATH.MOUNTS_USAGE.replace('{id}', encodeContainerPathParam(containerId));
}

/**
 * 构建刷新容器挂载使用量的接口路径，并对容器和挂载标识进行编码。
 *
 * @param containerId - 容器标识
 * @param mountId - 挂载标识
 * @returns 刷新挂载使用量的接口路径
 */
export function buildContainerMountUsageRefreshApiPath(containerId: string, mountId: string) {
  return CONTAINER_API_PATH.MOUNT_USAGE_REFRESH.replace('{id}', encodeContainerPathParam(containerId)).replace(
    '{mountId}',
    encodeContainerPathParam(mountId),
  );
}

/**
 * 构建启动容器的接口路径。
 *
 * @param containerId - 容器标识
 * @returns 启动容器的接口路径
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
