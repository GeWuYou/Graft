export const APPLICATION_ROUTE_PATH = {
  LIST: '/applications',
  CREATE_IMPORT: '/applications/create/import',
  CREATE: '/applications/create',
  CREATE_RUNTIME_TARGET: '/applications/create/runtime-target',
  CREATE_SOURCE: '/applications/create/source',
  CREATE_DISCOVERY: '/applications/create/discovery',
  CREATE_BLANK: '/applications/create/blank',
  CREATE_TEMPLATE: '/applications/create/template',
  DETAIL: '/applications/:applicationId',
  CONFIGURATION_WORKSPACE: '/applications/:applicationId/configuration',
} as const;

export const APPLICATION_API_PATH = {
  LIST: '/api/ops/applications',
  SAVED_VIEWS: '/api/ops/applications/saved-views',
  SAVED_VIEW: '/api/ops/applications/saved-views/{viewId}',
  BATCH_ACTIONS: '/api/ops/applications/batch-actions',
  IMPORT_RUNTIME_CANDIDATES: '/api/ops/applications/import/runtime-candidates',
  IMPORT_RUNTIME_INSPECT: '/api/ops/applications/import/runtime-inspect',
  IMPORT_DIRECTORY_SOURCES: '/api/ops/applications/import/directory-sources',
  IMPORT_DIRECTORIES: '/api/ops/applications/import/directories',
  IMPORT_INSPECT: '/api/ops/applications/import/inspect',
  IMPORT_VALIDATE: '/api/ops/applications/import/validate',
  IMPORT: '/api/ops/applications/import',
  CREATION_METHODS: '/api/ops/applications/creation-methods',
  COMPOSE_RUNTIME_TARGETS: '/api/ops/applications/create/runtime-targets',
  CREATE_WORKSPACE_DEFAULTS: '/api/ops/applications/create/workspace-defaults',
  DISCOVERY_CANDIDATES: '/api/ops/applications/discovery-candidates',
  MANAGED_ROOT: '/api/ops/applications/managed/root',
  CREATE_VALIDATE: '/api/ops/applications/create/managed/validate',
  APPLICATION_NAME_AVAILABILITY: '/api/ops/applications/create/application-name/availability',
  CREATE: '/api/ops/applications/create/managed',
  CREATE_TEMPLATE_VALIDATE: '/api/ops/applications/create/template/validate',
  CREATE_TEMPLATE: '/api/ops/applications/create/template',
  DETAIL: '/api/ops/applications/{applicationId}',
  OVERVIEW: '/api/ops/applications/{applicationId}/overview',
  LOGS: '/api/ops/applications/{applicationId}/logs',
  SERVICES: '/api/ops/applications/{applicationId}/services',
  CONFIGURATION: '/api/ops/applications/{applicationId}/configuration',
  CONFIGURATION_PREVIEW: '/api/ops/applications/{applicationId}/configuration/preview',
  CONFIGURATION_FILE: '/api/ops/applications/{applicationId}/configuration/files/{fileId}',
  FILES: '/api/ops/applications/{applicationId}/files',
  FILES_CONTENT: '/api/ops/applications/{applicationId}/files/content',
  FILES_ANNOTATION: '/api/ops/applications/{applicationId}/files/annotation',
  FILES_ENTRIES: '/api/ops/applications/{applicationId}/files/entries',
  FILES_RENAME: '/api/ops/applications/{applicationId}/files/rename',
  LIFECYCLE_CONFIGURATION: '/api/ops/applications/{applicationId}/lifecycle-configuration',
  REFRESH: '/api/ops/applications/{applicationId}/refresh',
  UP: '/api/ops/applications/{applicationId}/up',
  STOP: '/api/ops/applications/{applicationId}/stop',
  RESTART: '/api/ops/applications/{applicationId}/restart',
  REDEPLOY: '/api/ops/applications/{applicationId}/redeploy',
  UNREGISTER: '/api/ops/applications/{applicationId}/unregister',
  DESTROY: '/api/ops/applications/{applicationId}/destroy',
} as const;

function encodeApplicationPathParam(value: string | number) {
  return encodeURIComponent(String(value));
}

export function buildApplicationSavedViewApiPath(viewId: number) {
  return APPLICATION_API_PATH.SAVED_VIEW.replace('{viewId}', encodeApplicationPathParam(viewId));
}

export function buildApplicationDetailApiPath(applicationId: string) {
  return APPLICATION_API_PATH.DETAIL.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationOverviewApiPath(applicationId: string) {
  return APPLICATION_API_PATH.OVERVIEW.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationLogsApiPath(applicationId: string) {
  return APPLICATION_API_PATH.LOGS.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationServicesApiPath(applicationId: string) {
  return APPLICATION_API_PATH.SERVICES.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationConfigurationApiPath(applicationId: string) {
  return APPLICATION_API_PATH.CONFIGURATION.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationFilesApiPath(applicationId: string) {
  return APPLICATION_API_PATH.FILES.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationFilesContentApiPath(applicationId: string) {
  return APPLICATION_API_PATH.FILES_CONTENT.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationFilesAnnotationApiPath(applicationId: string) {
  return APPLICATION_API_PATH.FILES_ANNOTATION.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationFilesEntriesApiPath(applicationId: string) {
  return APPLICATION_API_PATH.FILES_ENTRIES.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationFilesRenameApiPath(applicationId: string) {
  return APPLICATION_API_PATH.FILES_RENAME.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationLifecycleConfigurationApiPath(applicationId: string) {
  return APPLICATION_API_PATH.LIFECYCLE_CONFIGURATION.replace(
    '{applicationId}',
    encodeApplicationPathParam(applicationId),
  );
}

export function buildApplicationUpApiPath(applicationId: string) {
  return APPLICATION_API_PATH.UP.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationStopApiPath(applicationId: string) {
  return APPLICATION_API_PATH.STOP.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationRestartApiPath(applicationId: string) {
  return APPLICATION_API_PATH.RESTART.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationRedeployApiPath(applicationId: string) {
  return APPLICATION_API_PATH.REDEPLOY.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationUnregisterApiPath(applicationId: string) {
  return APPLICATION_API_PATH.UNREGISTER.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationDestroyApiPath(applicationId: string) {
  return APPLICATION_API_PATH.DESTROY.replace('{applicationId}', encodeApplicationPathParam(applicationId));
}

export function buildApplicationImportDirectoriesApiPath() {
  return APPLICATION_API_PATH.IMPORT_DIRECTORIES;
}
