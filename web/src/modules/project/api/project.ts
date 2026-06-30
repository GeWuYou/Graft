import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import {
  buildProjectConfigurationApiPath,
  buildProjectConfigurationFileApiPath,
  buildProjectConfigurationPreviewApiPath,
  buildProjectDetailApiPath,
  buildProjectDownApiPath,
  buildProjectRefreshApiPath,
  buildProjectRestartApiPath,
  buildProjectServicesApiPath,
  buildProjectUnregisterApiPath,
  buildProjectUpApiPath,
  PROJECT_API_PATH,
} from '../contract/paths';
import type {
  ProjectActionResponse,
  ProjectConfigurationFileResponse,
  ProjectConfigurationMetadataResponse,
  ProjectConfigurationPreviewResponse,
  ProjectDetailResponse,
  ProjectListQuery,
  ProjectListResponse,
  ProjectServicesResponse,
} from '../types/project';

type ProjectListPath = (typeof PROJECT_API_PATH)['LIST'];
type GetProjectListOperation = paths[ProjectListPath]['get'];
type GetProjectListEnvelope = GetProjectListOperation['responses'][200]['content']['application/json'];
type GetProjectListData = NonNullable<GetProjectListEnvelope['data']>;
type GetProjectListQuery = NonNullable<GetProjectListOperation['parameters']['query']>;

type ProjectDetailPath = (typeof PROJECT_API_PATH)['DETAIL'];
type GetProjectDetailOperation = paths[ProjectDetailPath]['get'];
type GetProjectDetailEnvelope = GetProjectDetailOperation['responses'][200]['content']['application/json'];
type GetProjectDetailData = NonNullable<GetProjectDetailEnvelope['data']>;
type GetProjectDetailPathParams = GetProjectDetailOperation['parameters']['path'];

type ProjectServicesPath = (typeof PROJECT_API_PATH)['SERVICES'];
type GetProjectServicesOperation = paths[ProjectServicesPath]['get'];
type GetProjectServicesEnvelope = GetProjectServicesOperation['responses'][200]['content']['application/json'];
type GetProjectServicesData = NonNullable<GetProjectServicesEnvelope['data']>;
type GetProjectServicesPathParams = GetProjectServicesOperation['parameters']['path'];

type ProjectConfigurationPath = (typeof PROJECT_API_PATH)['CONFIGURATION'];
type GetProjectConfigurationOperation = paths[ProjectConfigurationPath]['get'];
type GetProjectConfigurationEnvelope =
  GetProjectConfigurationOperation['responses'][200]['content']['application/json'];
type GetProjectConfigurationData = NonNullable<GetProjectConfigurationEnvelope['data']>;
type GetProjectConfigurationPathParams = GetProjectConfigurationOperation['parameters']['path'];

type ProjectConfigurationPreviewPath = (typeof PROJECT_API_PATH)['CONFIGURATION_PREVIEW'];
type GetProjectConfigurationPreviewOperation = paths[ProjectConfigurationPreviewPath]['get'];
type GetProjectConfigurationPreviewEnvelope =
  GetProjectConfigurationPreviewOperation['responses'][200]['content']['application/json'];
type GetProjectConfigurationPreviewData = NonNullable<GetProjectConfigurationPreviewEnvelope['data']>;
type GetProjectConfigurationPreviewPathParams = GetProjectConfigurationPreviewOperation['parameters']['path'];

type ProjectConfigurationFilePath = (typeof PROJECT_API_PATH)['CONFIGURATION_FILE'];
type GetProjectConfigurationFileOperation = paths[ProjectConfigurationFilePath]['get'];
type GetProjectConfigurationFileEnvelope =
  GetProjectConfigurationFileOperation['responses'][200]['content']['application/json'];
type GetProjectConfigurationFileData = NonNullable<GetProjectConfigurationFileEnvelope['data']>;
type GetProjectConfigurationFilePathParams = GetProjectConfigurationFileOperation['parameters']['path'];

type ProjectRefreshOperation = paths[(typeof PROJECT_API_PATH)['REFRESH']]['post'];
type ProjectRefreshEnvelope = ProjectRefreshOperation['responses'][200]['content']['application/json'];
type ProjectRefreshData = NonNullable<ProjectRefreshEnvelope['data']>;
type ProjectRefreshPathParams = ProjectRefreshOperation['parameters']['path'];

type ProjectUpOperation = paths[(typeof PROJECT_API_PATH)['UP']]['post'];
type ProjectUpEnvelope = ProjectUpOperation['responses'][200]['content']['application/json'];
type ProjectUpData = NonNullable<ProjectUpEnvelope['data']>;
type ProjectUpPathParams = ProjectUpOperation['parameters']['path'];

type ProjectDownOperation = paths[(typeof PROJECT_API_PATH)['DOWN']]['post'];
type ProjectDownEnvelope = ProjectDownOperation['responses'][200]['content']['application/json'];
type ProjectDownData = NonNullable<ProjectDownEnvelope['data']>;
type ProjectDownPathParams = ProjectDownOperation['parameters']['path'];

type ProjectRestartOperation = paths[(typeof PROJECT_API_PATH)['RESTART']]['post'];
type ProjectRestartEnvelope = ProjectRestartOperation['responses'][200]['content']['application/json'];
type ProjectRestartData = NonNullable<ProjectRestartEnvelope['data']>;
type ProjectRestartPathParams = ProjectRestartOperation['parameters']['path'];

type ProjectUnregisterOperation = paths[(typeof PROJECT_API_PATH)['UNREGISTER']]['post'];
type ProjectUnregisterEnvelope = ProjectUnregisterOperation['responses'][200]['content']['application/json'];
type ProjectUnregisterData = NonNullable<ProjectUnregisterEnvelope['data']>;
type ProjectUnregisterPathParams = ProjectUnregisterOperation['parameters']['path'];

function normalizeProjectListQuery(query?: ProjectListQuery): GetProjectListQuery | undefined {
  if (!query) {
    return undefined;
  }

  return query satisfies GetProjectListQuery;
}

export function getProjects(query?: ProjectListQuery) {
  return request.get<GetProjectListData>({
    url: PROJECT_API_PATH.LIST,
    params: normalizeProjectListQuery(query),
  }) as Promise<ProjectListResponse>;
}

export function getProject(id: GetProjectDetailPathParams['id']) {
  return request.get<GetProjectDetailData>({
    url: buildProjectDetailApiPath(id),
  }) as Promise<ProjectDetailResponse>;
}

export function getProjectServices(id: GetProjectServicesPathParams['id']) {
  return request.get<GetProjectServicesData>({
    url: buildProjectServicesApiPath(id),
  }) as Promise<ProjectServicesResponse>;
}

export function getProjectConfiguration(id: GetProjectConfigurationPathParams['id']) {
  return request.get<GetProjectConfigurationData>({
    url: buildProjectConfigurationApiPath(id),
  }) as Promise<ProjectConfigurationMetadataResponse>;
}

export function getProjectConfigurationPreview(id: GetProjectConfigurationPreviewPathParams['id']) {
  return request.get<GetProjectConfigurationPreviewData>({
    url: buildProjectConfigurationPreviewApiPath(id),
  }) as Promise<ProjectConfigurationPreviewResponse>;
}

export function getProjectConfigurationFile(
  id: GetProjectConfigurationFilePathParams['id'],
  fileId: GetProjectConfigurationFilePathParams['fileId'],
) {
  return request.get<GetProjectConfigurationFileData>({
    url: buildProjectConfigurationFileApiPath(id, fileId),
  }) as Promise<ProjectConfigurationFileResponse>;
}

function postProjectAction<T>(url: string, data?: unknown) {
  return request.post<T>({
    url,
    data,
  });
}

export function postProjectRefresh(id: ProjectRefreshPathParams['id']) {
  return postProjectAction<ProjectRefreshData>(buildProjectRefreshApiPath(id)) as Promise<ProjectActionResponse>;
}

export function postProjectUp(id: ProjectUpPathParams['id']) {
  return postProjectAction<ProjectUpData>(buildProjectUpApiPath(id)) as Promise<ProjectActionResponse>;
}

export function postProjectDown(id: ProjectDownPathParams['id']) {
  return postProjectAction<ProjectDownData>(buildProjectDownApiPath(id)) as Promise<ProjectActionResponse>;
}

export function postProjectRestart(id: ProjectRestartPathParams['id']) {
  return postProjectAction<ProjectRestartData>(buildProjectRestartApiPath(id)) as Promise<ProjectActionResponse>;
}

export function postProjectUnregister(id: ProjectUnregisterPathParams['id']) {
  return postProjectAction<ProjectUnregisterData>(buildProjectUnregisterApiPath(id)) as Promise<ProjectActionResponse>;
}
