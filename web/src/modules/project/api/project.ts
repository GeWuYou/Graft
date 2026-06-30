import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import {
  buildProjectConfigurationApiPath,
  buildProjectConfigurationDiffApiPath,
  buildProjectConfigurationFileApiPath,
  buildProjectConfigurationPreviewApiPath,
  buildProjectConfigurationValidateApiPath,
  buildProjectDeployApiPath,
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
  ProjectConfigurationDiffRequest,
  ProjectConfigurationDiffResponse,
  ProjectConfigurationFileResponse,
  ProjectConfigurationMetadataResponse,
  ProjectConfigurationPreviewResponse,
  ProjectConfigurationValidateRequest,
  ProjectConfigurationValidateResponse,
  ProjectCreateRequest,
  ProjectCreateResponse,
  ProjectCreateValidateRequest,
  ProjectCreateValidateResponse,
  ProjectDeployRequest,
  ProjectDeployResponse,
  ProjectDetailResponse,
  ProjectListQuery,
  ProjectListResponse,
  ProjectManagedRootResponse,
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

type ProjectConfigurationDiffPath = (typeof PROJECT_API_PATH)['CONFIGURATION_DIFF'];
type ProjectConfigurationDiffOperation = paths[ProjectConfigurationDiffPath]['post'];
type ProjectConfigurationDiffEnvelope =
  ProjectConfigurationDiffOperation['responses'][200]['content']['application/json'];
type ProjectConfigurationDiffData = NonNullable<ProjectConfigurationDiffEnvelope['data']>;
type ProjectConfigurationDiffPayload = ProjectConfigurationDiffOperation['requestBody']['content']['application/json'];
type ProjectConfigurationDiffPathParams = ProjectConfigurationDiffOperation['parameters']['path'];

type ProjectConfigurationValidatePath = (typeof PROJECT_API_PATH)['CONFIGURATION_VALIDATE'];
type ProjectConfigurationValidateOperation = paths[ProjectConfigurationValidatePath]['post'];
type ProjectConfigurationValidateEnvelope =
  ProjectConfigurationValidateOperation['responses'][200]['content']['application/json'];
type ProjectConfigurationValidateData = NonNullable<ProjectConfigurationValidateEnvelope['data']>;
type ProjectConfigurationValidatePayload =
  ProjectConfigurationValidateOperation['requestBody']['content']['application/json'];
type ProjectConfigurationValidatePathParams = ProjectConfigurationValidateOperation['parameters']['path'];

type ProjectManagedRootPath = (typeof PROJECT_API_PATH)['MANAGED_ROOT'];
type GetProjectManagedRootOperation = paths[ProjectManagedRootPath]['get'];
type GetProjectManagedRootEnvelope = GetProjectManagedRootOperation['responses'][200]['content']['application/json'];
type GetProjectManagedRootData = NonNullable<GetProjectManagedRootEnvelope['data']>;

type ProjectCreateValidatePath = (typeof PROJECT_API_PATH)['CREATE_VALIDATE'];
type ProjectCreateValidateOperation = paths[ProjectCreateValidatePath]['post'];
type ProjectCreateValidateEnvelope = ProjectCreateValidateOperation['responses'][200]['content']['application/json'];
type ProjectCreateValidateData = NonNullable<ProjectCreateValidateEnvelope['data']>;
type ProjectCreateValidatePayload = ProjectCreateValidateOperation['requestBody']['content']['application/json'];

type ProjectCreatePath = (typeof PROJECT_API_PATH)['CREATE'];
type ProjectCreateOperation = paths[ProjectCreatePath]['post'];
type ProjectCreateEnvelope = ProjectCreateOperation['responses'][201]['content']['application/json'];
type ProjectCreateData = NonNullable<ProjectCreateEnvelope['data']>;
type ProjectCreatePayload = ProjectCreateOperation['requestBody']['content']['application/json'];

type ProjectRefreshOperation = paths[(typeof PROJECT_API_PATH)['REFRESH']]['post'];
type ProjectRefreshEnvelope = ProjectRefreshOperation['responses'][200]['content']['application/json'];
type ProjectRefreshData = NonNullable<ProjectRefreshEnvelope['data']>;
type ProjectRefreshPathParams = ProjectRefreshOperation['parameters']['path'];

type ProjectDeployOperation = paths[(typeof PROJECT_API_PATH)['DEPLOY']]['post'];
type ProjectDeployEnvelope = ProjectDeployOperation['responses'][200]['content']['application/json'];
type ProjectDeployData = NonNullable<ProjectDeployEnvelope['data']>;
type ProjectDeployPayload = ProjectDeployOperation['requestBody']['content']['application/json'];
type ProjectDeployPathParams = ProjectDeployOperation['parameters']['path'];

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

export function postProjectConfigurationDiff(
  id: ProjectConfigurationDiffPathParams['id'],
  payload: ProjectConfigurationDiffRequest,
) {
  return postProjectAction<ProjectConfigurationDiffData>(
    buildProjectConfigurationDiffApiPath(id),
    payload as ProjectConfigurationDiffPayload,
  ) as Promise<ProjectConfigurationDiffResponse>;
}

export function postProjectConfigurationValidate(
  id: ProjectConfigurationValidatePathParams['id'],
  payload: ProjectConfigurationValidateRequest,
) {
  return postProjectAction<ProjectConfigurationValidateData>(
    buildProjectConfigurationValidateApiPath(id),
    payload as ProjectConfigurationValidatePayload,
  ) as Promise<ProjectConfigurationValidateResponse>;
}

export function getProjectManagedRoot() {
  return request.get<GetProjectManagedRootData>({
    url: PROJECT_API_PATH.MANAGED_ROOT,
  }) as Promise<ProjectManagedRootResponse>;
}

export function postProjectCreateValidate(payload: ProjectCreateValidateRequest) {
  return postProjectAction<ProjectCreateValidateData>(
    PROJECT_API_PATH.CREATE_VALIDATE,
    payload as ProjectCreateValidatePayload,
  ) as Promise<ProjectCreateValidateResponse>;
}

export function postProjectCreate(payload: ProjectCreateRequest) {
  return postProjectAction<ProjectCreateData>(
    PROJECT_API_PATH.CREATE,
    payload as ProjectCreatePayload,
  ) as Promise<ProjectCreateResponse>;
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

export function postProjectDeploy(id: ProjectDeployPathParams['id'], payload: ProjectDeployRequest) {
  return postProjectAction<ProjectDeployData>(
    buildProjectDeployApiPath(id),
    payload as ProjectDeployPayload,
  ) as Promise<ProjectDeployResponse>;
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
