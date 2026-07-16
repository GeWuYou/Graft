export const PROJECT_ROUTE_PATH = {
  LIST: '/applications/projects',
  CREATE_IMPORT: '/applications/projects/create/import',
  CREATE: '/applications/projects/create',
  CREATE_RUNTIME_TARGET: '/applications/projects/create/runtime-target',
  CREATE_SOURCE: '/applications/projects/create/source',
  CREATE_DISCOVERY: '/applications/projects/create/discovery',
  CREATE_BLANK: '/applications/projects/create/blank',
  CREATE_TEMPLATE: '/applications/projects/create/template',
  DETAIL: '/applications/projects/:id',
  CONFIGURATION_WORKSPACE: '/applications/projects/:id/configuration',
} as const;

export const PROJECT_API_PATH = {
  LIST: '/api/ops/projects',
  SAVED_VIEWS: '/api/ops/projects/saved-views',
  SAVED_VIEW: '/api/ops/projects/saved-views/{viewId}',
  BATCH_ACTIONS: '/api/ops/projects/batch-actions',
  IMPORT_RUNTIME_CANDIDATES: '/api/ops/projects/import/runtime-candidates',
  IMPORT_RUNTIME_INSPECT: '/api/ops/projects/import/runtime-inspect',
  IMPORT_DIRECTORY_SOURCES: '/api/ops/projects/import/directory-sources',
  IMPORT_DIRECTORIES: '/api/ops/projects/import/directories',
  IMPORT_INSPECT: '/api/ops/projects/import/inspect',
  IMPORT_VALIDATE: '/api/ops/projects/import/validate',
  IMPORT: '/api/ops/projects/import',
  CREATION_METHODS: '/api/ops/projects/creation-methods',
  COMPOSE_RUNTIME_TARGETS: '/api/ops/projects/create/runtime-targets',
  CREATE_WORKSPACE_DEFAULTS: '/api/ops/projects/create/workspace-defaults',
  DISCOVERY_CANDIDATES: '/api/ops/projects/discovery-candidates',
  MANAGED_ROOT: '/api/ops/projects/managed/root',
  CREATE_VALIDATE: '/api/ops/projects/create/managed/validate',
  APPLICATION_NAME_AVAILABILITY: '/api/ops/projects/create/application-name/availability',
  CREATE: '/api/ops/projects/create/managed',
  CREATE_TEMPLATE_VALIDATE: '/api/ops/projects/create/template/validate',
  CREATE_TEMPLATE: '/api/ops/projects/create/template',
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
  FILES_ENTRIES: '/api/ops/projects/{id}/files/entries',
  FILES_RENAME: '/api/ops/projects/{id}/files/rename',
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

function encodeProjectPathParam(value: string | number) {
  return encodeURIComponent(String(value));
}

export function buildProjectSavedViewApiPath(viewId: number) {
  return PROJECT_API_PATH.SAVED_VIEW.replace('{viewId}', encodeProjectPathParam(viewId));
}

export function buildProjectDetailApiPath(id: string | number) {
  return PROJECT_API_PATH.DETAIL.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectOverviewApiPath(id: string | number) {
  return PROJECT_API_PATH.OVERVIEW.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectLogsApiPath(id: string | number) {
  return PROJECT_API_PATH.LOGS.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectServicesApiPath(id: string | number) {
  return PROJECT_API_PATH.SERVICES.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectConfigurationApiPath(id: string | number) {
  return PROJECT_API_PATH.CONFIGURATION.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectFilesApiPath(id: string | number) {
  return PROJECT_API_PATH.FILES.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectFilesContentApiPath(id: string | number) {
  return PROJECT_API_PATH.FILES_CONTENT.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectFilesAnnotationApiPath(id: string | number) {
  return PROJECT_API_PATH.FILES_ANNOTATION.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectFilesEntriesApiPath(id: string | number) {
  return PROJECT_API_PATH.FILES_ENTRIES.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectFilesRenameApiPath(id: string | number) {
  return PROJECT_API_PATH.FILES_RENAME.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectLifecycleConfigurationApiPath(id: string | number) {
  return PROJECT_API_PATH.LIFECYCLE_CONFIGURATION.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectDeployApiPath(id: string | number) {
  return PROJECT_API_PATH.DEPLOY.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectUpApiPath(id: string | number) {
  return PROJECT_API_PATH.UP.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectStopApiPath(id: string | number) {
  return PROJECT_API_PATH.STOP.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectRestartApiPath(id: string | number) {
  return PROJECT_API_PATH.RESTART.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectRedeployApiPath(id: string | number) {
  return PROJECT_API_PATH.REDEPLOY.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectUnregisterApiPath(id: string | number) {
  return PROJECT_API_PATH.UNREGISTER.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectDestroyApiPath(id: string | number) {
  return PROJECT_API_PATH.DESTROY.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectImportDirectoriesApiPath() {
  return PROJECT_API_PATH.IMPORT_DIRECTORIES;
}
