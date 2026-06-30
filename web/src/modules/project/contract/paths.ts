export const PROJECT_ROUTE_PATH = {
  LIST: '/ops/projects',
  CREATE: '/ops/projects/create',
  DETAIL: '/ops/projects/:id',
} as const;

export const PROJECT_API_PATH = {
  LIST: '/api/ops/projects',
  MANAGED_ROOT: '/api/ops/projects/managed-root',
  CREATE_VALIDATE: '/api/ops/projects/create/validate',
  CREATE: '/api/ops/projects/create',
  DETAIL: '/api/ops/projects/{id}',
  SERVICES: '/api/ops/projects/{id}/services',
  CONFIGURATION: '/api/ops/projects/{id}/configuration',
  CONFIGURATION_PREVIEW: '/api/ops/projects/{id}/configuration/preview',
  CONFIGURATION_FILE: '/api/ops/projects/{id}/configuration/files/{fileId}',
  REFRESH: '/api/ops/projects/{id}/refresh',
  UP: '/api/ops/projects/{id}/up',
  DOWN: '/api/ops/projects/{id}/down',
  RESTART: '/api/ops/projects/{id}/restart',
  UNREGISTER: '/api/ops/projects/{id}/unregister',
} as const;

function encodeProjectPathParam(value: string | number) {
  return encodeURIComponent(String(value));
}

export function buildProjectDetailApiPath(id: number) {
  return PROJECT_API_PATH.DETAIL.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectServicesApiPath(id: number) {
  return PROJECT_API_PATH.SERVICES.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectConfigurationApiPath(id: number) {
  return PROJECT_API_PATH.CONFIGURATION.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectConfigurationPreviewApiPath(id: number) {
  return PROJECT_API_PATH.CONFIGURATION_PREVIEW.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectConfigurationFileApiPath(id: number, fileId: number) {
  return PROJECT_API_PATH.CONFIGURATION_FILE.replace('{id}', encodeProjectPathParam(id)).replace(
    '{fileId}',
    encodeProjectPathParam(fileId),
  );
}

export function buildProjectRefreshApiPath(id: number) {
  return PROJECT_API_PATH.REFRESH.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectUpApiPath(id: number) {
  return PROJECT_API_PATH.UP.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectDownApiPath(id: number) {
  return PROJECT_API_PATH.DOWN.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectRestartApiPath(id: number) {
  return PROJECT_API_PATH.RESTART.replace('{id}', encodeProjectPathParam(id));
}

export function buildProjectUnregisterApiPath(id: number) {
  return PROJECT_API_PATH.UNREGISTER.replace('{id}', encodeProjectPathParam(id));
}
