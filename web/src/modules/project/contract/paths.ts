export const PROJECT_ROUTE_PATH = {
  LIST: '/ops/projects',
  IMPORT: '/ops/projects/import',
  CREATE: '/ops/projects/create',
  CREATE_DISCOVERY: '/ops/projects/create/discovery',
  CREATE_MANAGED: '/ops/projects/create/managed',
  CREATE_GIT: '/ops/projects/create/git',
  CREATE_TEMPLATE: '/ops/projects/create/template',
  CREATE_REMOTE_HOST: '/ops/projects/create/remote-host',
  DETAIL: '/ops/projects/:id',
  CONFIGURATION_WORKSPACE: '/ops/projects/:id/configuration',
} as const;

export const PROJECT_API_PATH = {
  LIST: '/api/ops/projects',
  BATCH_ACTIONS: '/api/ops/projects/batch-actions',
  IMPORT_RUNTIME_CANDIDATES: '/api/ops/projects/import/runtime-candidates',
  IMPORT_RUNTIME_INSPECT: '/api/ops/projects/import/runtime-inspect',
  IMPORT_DIRECTORY_SOURCES: '/api/ops/projects/import/directory-sources',
  IMPORT_DIRECTORIES: '/api/ops/projects/import/directories',
  IMPORT_INSPECT: '/api/ops/projects/import/inspect',
  IMPORT_VALIDATE: '/api/ops/projects/import/validate',
  IMPORT: '/api/ops/projects/import',
  SOURCES: '/api/ops/projects/sources',
  DISCOVERY_CANDIDATES: '/api/ops/projects/discovery-candidates',
  MANAGED_ROOT: '/api/ops/projects/managed/root',
  CREATE_VALIDATE: '/api/ops/projects/create/managed/validate',
  CREATE: '/api/ops/projects/create/managed',
  DETAIL: '/api/ops/projects/{id}',
  OVERVIEW: '/api/ops/projects/{id}/overview',
  LOGS: '/api/ops/projects/{id}/logs',
  SERVICES: '/api/ops/projects/{id}/services',
  CONFIGURATION: '/api/ops/projects/{id}/configuration',
  CONFIGURATION_PREVIEW: '/api/ops/projects/{id}/configuration/preview',
  CONFIGURATION_FILE: '/api/ops/projects/{id}/configuration/files/{fileId}',
  FILES: '/api/ops/projects/{id}/files',
  FILES_CONTENT: '/api/ops/projects/{id}/files/content',
  FILES_ANNOTATION: '/api/ops/projects/{id}/files/annotation',
  LIFECYCLE_CONFIGURATION: '/api/ops/projects/{id}/lifecycle-configuration',
  REFRESH: '/api/ops/projects/{id}/refresh',
  DEPLOY: '/api/ops/projects/{id}/deploy',
  UP: '/api/ops/projects/{id}/up',
  STOP: '/api/ops/projects/{id}/stop',
  RESTART: '/api/ops/projects/{id}/restart',
  REDEPLOY: '/api/ops/projects/{id}/redeploy',
  UNREGISTER: '/api/ops/projects/{id}/unregister',
  DESTROY: '/api/ops/projects/{id}/destroy',
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

export function buildProjectOverviewApiPath(id: number) {
  return PROJECT_API_PATH.OVERVIEW.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectLogsApiPath(id: number) {
  return PROJECT_API_PATH.LOGS.replace('{id}', encodeProjectPathParam(id));
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
 * 生成项目文件接口路径。
 *
 * @param id - 项目 ID
 * @returns 替换了 `{id}` 占位符的文件接口路径
 */
export function buildProjectFilesApiPath(id: number) {
  return PROJECT_API_PATH.FILES.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectFilesContentApiPath(id: number) {
  return PROJECT_API_PATH.FILES_CONTENT.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectFilesAnnotationApiPath(id: number) {
  return PROJECT_API_PATH.FILES_ANNOTATION.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectLifecycleConfigurationApiPath(id: number) {
  return PROJECT_API_PATH.LIFECYCLE_CONFIGURATION.replace('{id}', encodeProjectPathParam(id));
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
export function buildProjectStopApiPath(id: number) {
  return PROJECT_API_PATH.STOP.replace('{id}', encodeProjectPathParam(id));
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
 * 构建项目重新部署接口路径。
 *
 * @param id - 项目 ID
 * @returns 替换了 `id` 占位符的重新部署接口路径
 */
export function buildProjectRedeployApiPath(id: number) {
  return PROJECT_API_PATH.REDEPLOY.replace('{id}', encodeProjectPathParam(id));
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

/**
 * 构建项目销毁接口路径。
 *
 * @param id - 项目 ID
 * @returns 替换了 `id` 占位符的项目销毁接口路径
 */
export function buildProjectDestroyApiPath(id: number) {
  return PROJECT_API_PATH.DESTROY.replace('{id}', encodeProjectPathParam(id));
}

/**
 * 构建项目导入目录浏览接口路径。
 *
 * @returns 项目导入目录浏览接口路径
 */
export function buildProjectImportDirectoriesApiPath() {
  return PROJECT_API_PATH.IMPORT_DIRECTORIES;
}
