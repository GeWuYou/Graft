export const PROJECT_ROUTE_PATH = {
  LIST: '/ops/projects',
  CREATE: '/ops/projects/create',
  CREATE_MANAGED: '/ops/projects/create/managed',
  CREATE_GIT: '/ops/projects/create/git',
  CREATE_TEMPLATE: '/ops/projects/create/template',
  DETAIL: '/ops/projects/:id',
} as const;

export const PROJECT_API_PATH = {
  LIST: '/api/ops/projects',
  SOURCES: '/api/ops/projects/sources',
  MANAGED_ROOT: '/api/ops/projects/managed/root',
  CREATE_VALIDATE: '/api/ops/projects/create/managed/validate',
  CREATE: '/api/ops/projects/create/managed',
  DETAIL: '/api/ops/projects/{id}',
  SERVICES: '/api/ops/projects/{id}/services',
  CONFIGURATION: '/api/ops/projects/{id}/configuration',
  CONFIGURATION_PREVIEW: '/api/ops/projects/{id}/configuration/preview',
  CONFIGURATION_FILE: '/api/ops/projects/{id}/configuration/files/{fileId}',
  CONFIGURATION_DIFF: '/api/ops/projects/{id}/configuration/diff',
  CONFIGURATION_VALIDATE: '/api/ops/projects/{id}/configuration/validate',
  REFRESH: '/api/ops/projects/{id}/refresh',
  DEPLOY: '/api/ops/projects/{id}/deploy',
  UP: '/api/ops/projects/{id}/up',
  DOWN: '/api/ops/projects/{id}/down',
  RESTART: '/api/ops/projects/{id}/restart',
  UNREGISTER: '/api/ops/projects/{id}/unregister',
} as const;

/**
 * 将路径参数编码为可安全用于 URL 的字符串。
 *
 * @param value - 需要编码的值
 * @returns 编码后的路径参数
 */
function encodeProjectPathParam(value: string | number) {
  return encodeURIComponent(String(value));
}

/**
 * 构建项目详情 API 路径。
 *
 * @param id - 项目 ID
 * @returns 替换 `id` 占位符后的项目详情 API 路径
 */
export function buildProjectDetailApiPath(id: number) {
  return PROJECT_API_PATH.DETAIL.replace('{id}', encodeProjectPathParam(id));
}

/**
 * 构建项目服务接口路径。
 *
 * @param id - 项目 ID
 * @returns 替换了 `id` 占位符的项目服务接口路径
 */
export function buildProjectServicesApiPath(id: number) {
  return PROJECT_API_PATH.SERVICES.replace('{id}', encodeProjectPathParam(id));
}

/**
 * 构建项目配置 API 路径。
 *
 * @param id - 项目 ID
 * @returns 项目配置 API 路径
 */
export function buildProjectConfigurationApiPath(id: number) {
  return PROJECT_API_PATH.CONFIGURATION.replace('{id}', encodeProjectPathParam(id));
}

/**
 * 构建项目配置预览的 API 路径。
 *
 * @param id - 项目 ID
 * @returns 项目配置预览接口路径
 */
export function buildProjectConfigurationPreviewApiPath(id: number) {
  return PROJECT_API_PATH.CONFIGURATION_PREVIEW.replace('{id}', encodeProjectPathParam(id));
}

/**
 * 构建项目配置文件的 API 路径。
 *
 * @param id - 项目 ID
 * @param fileId - 配置文件 ID
 * @returns 替换为编码后的 `id` 和 `fileId` 的配置文件 API 路径
 */
export function buildProjectConfigurationFileApiPath(id: number, fileId: number) {
  return PROJECT_API_PATH.CONFIGURATION_FILE.replace('{id}', encodeProjectPathParam(id)).replace(
    '{fileId}',
    encodeProjectPathParam(fileId),
  );
}

/**
 * 构建项目配置差异接口路径。
 *
 * @param id - 项目 ID
 * @returns 替换 `id` 占位符后的配置差异接口路径
 */
export function buildProjectConfigurationDiffApiPath(id: number) {
  return PROJECT_API_PATH.CONFIGURATION_DIFF.replace('{id}', encodeProjectPathParam(id));
}

/**
 * 构建项目配置校验接口路径。
 *
 * @param id - 项目 ID
 * @returns 替换了 `id` 占位符的配置校验接口路径
 */
export function buildProjectConfigurationValidateApiPath(id: number) {
  return PROJECT_API_PATH.CONFIGURATION_VALIDATE.replace('{id}', encodeProjectPathParam(id));
}

/**
 * 构建项目刷新接口路径。
 *
 * @param id - 项目 ID
 * @returns 替换 `{id}` 后的刷新接口路径
 */
export function buildProjectRefreshApiPath(id: number) {
  return PROJECT_API_PATH.REFRESH.replace('{id}', encodeProjectPathParam(id));
}

/**
 * 构建项目部署接口路径。
 *
 * @param id - 项目 ID
 * @returns 项目部署接口 URL
 */
export function buildProjectDeployApiPath(id: number) {
  return PROJECT_API_PATH.DEPLOY.replace('{id}', encodeProjectPathParam(id));
}

/**
 * 构建项目上线 API 路径。
 *
 * @param id - 项目 ID
 * @returns 替换了 `id` 占位符的项目上线 API 路径
 */
export function buildProjectUpApiPath(id: number) {
  return PROJECT_API_PATH.UP.replace('{id}', encodeProjectPathParam(id));
}

/**
 * 构建项目下线操作的 API 路径。
 *
 * @param id - 项目 ID
 * @returns 替换了 `id` 占位符的下线 API 路径
 */
export function buildProjectDownApiPath(id: number) {
  return PROJECT_API_PATH.DOWN.replace('{id}', encodeProjectPathParam(id));
}

/**
 * 构建项目重启接口路径。
 *
 * @param id - 项目 ID
 * @returns 替换了 `{id}` 占位符的重启接口路径
 */
export function buildProjectRestartApiPath(id: number) {
  return PROJECT_API_PATH.RESTART.replace('{id}', encodeProjectPathParam(id));
}

/**
 * 构建项目注销接口路径。
 *
 * @param id - 项目 ID
 * @returns 替换了 `id` 占位符的项目注销接口路径
 */
export function buildProjectUnregisterApiPath(id: number) {
  return PROJECT_API_PATH.UNREGISTER.replace('{id}', encodeProjectPathParam(id));
}
