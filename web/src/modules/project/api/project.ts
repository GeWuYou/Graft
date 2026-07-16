import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import {
  buildProjectConfigurationApiPath,
  buildProjectDestroyApiPath,
  buildProjectDetailApiPath,
  buildProjectFilesAnnotationApiPath,
  buildProjectFilesApiPath,
  buildProjectFilesContentApiPath,
  buildProjectFilesEntriesApiPath,
  buildProjectFilesRenameApiPath,
  buildProjectLifecycleConfigurationApiPath,
  buildProjectLogsApiPath,
  buildProjectOverviewApiPath,
  buildProjectRedeployApiPath,
  buildProjectRestartApiPath,
  buildProjectSavedViewApiPath,
  buildProjectServicesApiPath,
  buildProjectStopApiPath,
  buildProjectUnregisterApiPath,
  buildProjectUpApiPath,
  PROJECT_API_PATH,
} from '../contract/paths';
import type {
  ProjectActionResponse,
  ProjectApplicationNameAvailabilityRequest,
  ProjectApplicationNameAvailabilityResponse,
  ProjectBatchActionRequest,
  ProjectBatchActionResponse,
  ProjectComposeRuntimeTargetCatalogResponse,
  ProjectConfigurationMetadataResponse,
  ProjectCreateRequest,
  ProjectCreateResponse,
  ProjectCreationMethodCatalogResponse,
  ProjectDestroyRequest,
  ProjectDetailResponseWithLifecycle,
  ProjectDiscoveryCandidatesResponse,
  ProjectLifecycleConfigurationSavedResponse,
  ProjectLifecycleConfigurationUpdateRequest,
  ProjectListQuery,
  ProjectListResponseWithLifecycle,
  ProjectLogResponse,
  ProjectOverviewResponse,
  ProjectSavedView,
  ProjectSavedViewRequest,
  ProjectServicesResponse,
  ProjectTaskReceipt,
  ProjectTemplateCreateRequest,
  ProjectWorkspaceDefaultsResponse,
  ProjectWorkspaceEntry,
  ProjectWorkspaceFileAnnotationRequest,
  ProjectWorkspaceFileAnnotationResponse,
  ProjectWorkspaceFileContentQuery,
  ProjectWorkspaceFileContentResponse,
  ProjectWorkspaceFileSaveRequest,
  ProjectWorkspaceFileSaveResponse,
  ProjectWorkspaceFilesQuery,
  ProjectWorkspaceFilesResponse,
  ProjectWorkspaceRenameRequest,
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

type ProjectOverviewPath = (typeof PROJECT_API_PATH)['OVERVIEW'];
type GetProjectOverviewOperation = paths[ProjectOverviewPath]['get'];
type GetProjectOverviewEnvelope = GetProjectOverviewOperation['responses'][200]['content']['application/json'];
type GetProjectOverviewData = NonNullable<GetProjectOverviewEnvelope['data']>;
type GetProjectOverviewPathParams = GetProjectOverviewOperation['parameters']['path'];

type ProjectLogsPath = (typeof PROJECT_API_PATH)['LOGS'];
type GetProjectLogsOperation = paths[ProjectLogsPath]['get'];
type GetProjectLogsEnvelope = GetProjectLogsOperation['responses'][200]['content']['application/json'];
type GetProjectLogsData = NonNullable<GetProjectLogsEnvelope['data']>;
type GetProjectLogsPathParams = GetProjectLogsOperation['parameters']['path'];
type GetProjectLogsQuery = NonNullable<GetProjectLogsOperation['parameters']['query']>;

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

type ProjectCreationMethodsPath = (typeof PROJECT_API_PATH)['CREATION_METHODS'];
type GetProjectCreationMethodsOperation = paths[ProjectCreationMethodsPath]['get'];
type GetProjectCreationMethodsEnvelope =
  GetProjectCreationMethodsOperation['responses'][200]['content']['application/json'];
type GetProjectCreationMethodsData = NonNullable<GetProjectCreationMethodsEnvelope['data']>;

type ProjectComposeRuntimeTargetsPath = (typeof PROJECT_API_PATH)['COMPOSE_RUNTIME_TARGETS'];
type GetProjectComposeRuntimeTargetsOperation = paths[ProjectComposeRuntimeTargetsPath]['get'];
type GetProjectComposeRuntimeTargetsData = NonNullable<
  GetProjectComposeRuntimeTargetsOperation['responses'][200]['content']['application/json']['data']
>;

type ProjectDiscoveryCandidatesPath = (typeof PROJECT_API_PATH)['DISCOVERY_CANDIDATES'];
type GetProjectDiscoveryCandidatesOperation = paths[ProjectDiscoveryCandidatesPath]['get'];
type GetProjectDiscoveryCandidatesEnvelope =
  GetProjectDiscoveryCandidatesOperation['responses'][200]['content']['application/json'];
type GetProjectDiscoveryCandidatesData = NonNullable<GetProjectDiscoveryCandidatesEnvelope['data']>;

type ProjectCreatePath = (typeof PROJECT_API_PATH)['CREATE'];
type ProjectCreateOperation = paths[ProjectCreatePath]['post'];
type ProjectCreateEnvelope = ProjectCreateOperation['responses'][201]['content']['application/json'];
type ProjectCreateData = NonNullable<ProjectCreateEnvelope['data']>;
type ProjectCreatePayload = ProjectCreateOperation['requestBody']['content']['application/json'];

type ProjectApplicationNameAvailabilityPath = (typeof PROJECT_API_PATH)['APPLICATION_NAME_AVAILABILITY'];
type ProjectApplicationNameAvailabilityOperation = paths[ProjectApplicationNameAvailabilityPath]['post'];
type ProjectApplicationNameAvailabilityData = NonNullable<
  ProjectApplicationNameAvailabilityOperation['responses'][200]['content']['application/json']['data']
>;

type ProjectTemplateCreatePath = (typeof PROJECT_API_PATH)['CREATE_TEMPLATE'];
type ProjectTemplateCreateOperation = paths[ProjectTemplateCreatePath]['post'];
type ProjectTemplateCreateData = NonNullable<
  ProjectTemplateCreateOperation['responses'][201]['content']['application/json']['data']
>;
type ProjectUpOperation = paths[(typeof PROJECT_API_PATH)['UP']]['post'];
type ProjectUpEnvelope = ProjectUpOperation['responses'][202]['content']['application/json'];
type ProjectUpData = NonNullable<ProjectUpEnvelope['data']>;
type ProjectUpPathParams = ProjectUpOperation['parameters']['path'];

type ProjectStopOperation = paths[(typeof PROJECT_API_PATH)['STOP']]['post'];
type ProjectStopEnvelope = ProjectStopOperation['responses'][202]['content']['application/json'];
type ProjectStopData = NonNullable<ProjectStopEnvelope['data']>;
type ProjectStopPathParams = ProjectStopOperation['parameters']['path'];

type ProjectRestartOperation = paths[(typeof PROJECT_API_PATH)['RESTART']]['post'];
type ProjectRestartEnvelope = ProjectRestartOperation['responses'][202]['content']['application/json'];
type ProjectRestartData = NonNullable<ProjectRestartEnvelope['data']>;
type ProjectRestartPathParams = ProjectRestartOperation['parameters']['path'];

type ProjectRedeployOperation = paths[(typeof PROJECT_API_PATH)['REDEPLOY']]['post'];
type ProjectRedeployEnvelope = ProjectRedeployOperation['responses'][202]['content']['application/json'];
type ProjectRedeployData = NonNullable<ProjectRedeployEnvelope['data']>;
type ProjectRedeployPathParams = ProjectRedeployOperation['parameters']['path'];

type ProjectUnregisterOperation = paths[(typeof PROJECT_API_PATH)['UNREGISTER']]['post'];
type ProjectUnregisterEnvelope = ProjectUnregisterOperation['responses'][200]['content']['application/json'];
type ProjectUnregisterData = NonNullable<ProjectUnregisterEnvelope['data']>;
type ProjectUnregisterPathParams = ProjectUnregisterOperation['parameters']['path'];

type ProjectDestroyOperation = paths[(typeof PROJECT_API_PATH)['DESTROY']]['post'];
type ProjectDestroyEnvelope = ProjectDestroyOperation['responses'][200]['content']['application/json'];
type ProjectDestroyData = NonNullable<ProjectDestroyEnvelope['data']>;
type ProjectDestroyPayload = ProjectDestroyOperation['requestBody']['content']['application/json'];
type ProjectDestroyPathParams = ProjectDestroyOperation['parameters']['path'];

type ProjectBatchActionsOperation = paths[(typeof PROJECT_API_PATH)['BATCH_ACTIONS']]['post'];
type ProjectBatchActionsEnvelope = ProjectBatchActionsOperation['responses'][200]['content']['application/json'];
type ProjectBatchActionsData = NonNullable<ProjectBatchActionsEnvelope['data']>;
type ProjectBatchActionsPayload = ProjectBatchActionsOperation['requestBody']['content']['application/json'];
type ProjectSavedViewsOperation = paths[(typeof PROJECT_API_PATH)['SAVED_VIEWS']]['get'];
type ProjectSavedViewsData = NonNullable<
  ProjectSavedViewsOperation['responses'][200]['content']['application/json']['data']
>;
type ProjectCreateSavedViewOperation = paths[(typeof PROJECT_API_PATH)['SAVED_VIEWS']]['post'];
type ProjectCreateSavedViewData = NonNullable<
  ProjectCreateSavedViewOperation['responses'][201]['content']['application/json']['data']
>;
type ProjectSavedViewOperation = paths[(typeof PROJECT_API_PATH)['SAVED_VIEW']]['put'];
type ProjectUpdateSavedViewData = NonNullable<
  ProjectSavedViewOperation['responses'][200]['content']['application/json']['data']
>;

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
  }) as Promise<ProjectListResponseWithLifecycle>;
}

/** 后端未返回视图数组时按空集合处理，避免可选的保存视图阻断项目列表。 */
export async function getProjectSavedViews(): Promise<ProjectSavedView[]> {
  const data = await request.get<ProjectSavedViewsData>({ url: PROJECT_API_PATH.SAVED_VIEWS });
  return data.items ?? [];
}

export function postProjectSavedView(payload: ProjectSavedViewRequest) {
  return request.post<ProjectCreateSavedViewData>({
    url: PROJECT_API_PATH.SAVED_VIEWS,
    data: payload,
  }) as Promise<ProjectSavedView>;
}

export function putProjectSavedView(viewId: number, payload: ProjectSavedViewRequest) {
  return request.put<ProjectUpdateSavedViewData>({
    url: buildProjectSavedViewApiPath(viewId),
    data: payload,
  }) as Promise<ProjectSavedView>;
}

export function deleteProjectSavedView(viewId: number) {
  return request.delete({ url: buildProjectSavedViewApiPath(viewId) });
}

export function getProject(id: GetProjectDetailPathParams['id']) {
  return request.get<GetProjectDetailData>({
    url: buildProjectDetailApiPath(id),
  }) as Promise<ProjectDetailResponseWithLifecycle>;
}

export function getProjectOverview(id: GetProjectOverviewPathParams['id']) {
  return request.get<GetProjectOverviewData>({
    url: buildProjectOverviewApiPath(id),
  }) as Promise<ProjectOverviewResponse>;
}

export function getProjectLogs(id: GetProjectLogsPathParams['id'], query?: GetProjectLogsQuery) {
  return request.get<GetProjectLogsData>({
    url: buildProjectLogsApiPath(id),
    params: query,
  }) as Promise<ProjectLogResponse>;
}

export function getProjectServices(id: GetProjectServicesPathParams['id']) {
  return request.get<GetProjectServicesData>({
    url: buildProjectServicesApiPath(id),
  }) as Promise<ProjectServicesResponse>;
}

export function getProjectWorkspaceDefaults() {
  return request.get<ProjectWorkspaceDefaultsResponse>({ url: PROJECT_API_PATH.CREATE_WORKSPACE_DEFAULTS });
}

export function postProjectWorkspaceEntry(id: string | number, payload: ProjectWorkspaceEntry) {
  return request.post({ url: buildProjectFilesEntriesApiPath(id), data: payload });
}

export function postProjectWorkspaceRename(id: string | number, payload: ProjectWorkspaceRenameRequest) {
  return request.post({ url: buildProjectFilesRenameApiPath(id), data: payload });
}

export function deleteProjectWorkspaceEntry(id: string | number, query: { path: string; recursive?: boolean }) {
  return request.delete({ url: buildProjectFilesEntriesApiPath(id), params: query });
}

export function getProjectConfiguration(id: GetProjectConfigurationPathParams['id']) {
  return request.get<GetProjectConfigurationData>({
    url: buildProjectConfigurationApiPath(id),
  }) as Promise<ProjectConfigurationMetadataResponse>;
}

export function getProjectFiles(id: string, query?: ProjectWorkspaceFilesQuery) {
  return request.get<ProjectWorkspaceFilesResponse>({
    url: buildProjectFilesApiPath(id),
    params: query,
  }) as Promise<ProjectWorkspaceFilesResponse>;
}

export function getProjectFileContent(id: string, query: ProjectWorkspaceFileContentQuery) {
  return request.get<ProjectWorkspaceFileContentResponse>({
    url: buildProjectFilesContentApiPath(id),
    params: query,
  }) as Promise<ProjectWorkspaceFileContentResponse>;
}

export function putProjectFileContent(
  id: string,
  query: ProjectWorkspaceFileContentQuery,
  payload: ProjectWorkspaceFileSaveRequest,
) {
  return request.put<ProjectWorkspaceFileSaveResponse>({
    url: buildProjectFilesContentApiPath(id),
    params: query,
    data: payload,
  }) as Promise<ProjectWorkspaceFileSaveResponse>;
}

export function putProjectFileAnnotation(
  id: string,
  query: ProjectWorkspaceFileContentQuery,
  payload: ProjectWorkspaceFileAnnotationRequest,
) {
  return request.put<ProjectWorkspaceFileAnnotationResponse>({
    url: buildProjectFilesAnnotationApiPath(id),
    params: query,
    data: payload,
  }) as Promise<ProjectWorkspaceFileAnnotationResponse>;
}

export function getProjectCreationMethods() {
  return request.get<GetProjectCreationMethodsData>({
    url: PROJECT_API_PATH.CREATION_METHODS,
  }) as Promise<ProjectCreationMethodCatalogResponse>;
}

export function getProjectComposeRuntimeTargets() {
  return request.get<GetProjectComposeRuntimeTargetsData>({
    url: PROJECT_API_PATH.COMPOSE_RUNTIME_TARGETS,
  }) as Promise<ProjectComposeRuntimeTargetCatalogResponse>;
}

export function getProjectDiscoveryCandidates() {
  return request.get<GetProjectDiscoveryCandidatesData>({
    url: PROJECT_API_PATH.DISCOVERY_CANDIDATES,
  }) as Promise<ProjectDiscoveryCandidatesResponse>;
}

export function postProjectCreate(payload: ProjectCreateRequest) {
  return postProjectAction<ProjectCreateData>(
    PROJECT_API_PATH.CREATE,
    payload as ProjectCreatePayload,
  ) as Promise<ProjectCreateResponse>;
}

export function postProjectApplicationNameAvailability(payload: ProjectApplicationNameAvailabilityRequest) {
  return request.post<ProjectApplicationNameAvailabilityData>({
    url: PROJECT_API_PATH.APPLICATION_NAME_AVAILABILITY,
    data: payload,
  }) as Promise<ProjectApplicationNameAvailabilityResponse>;
}

export function postProjectCreateTemplate(payload: ProjectTemplateCreateRequest) {
  return postProjectAction<ProjectTemplateCreateData>(
    PROJECT_API_PATH.CREATE_TEMPLATE,
    payload,
  ) as Promise<ProjectCreateResponse>;
}

function postProjectAction<T>(url: string, data?: unknown) {
  return request.post<T>({
    url,
    data,
  });
}

export function postProjectUp(id: ProjectUpPathParams['id']) {
  return postProjectAction<ProjectUpData>(buildProjectUpApiPath(id)) as Promise<ProjectTaskReceipt>;
}

export function postProjectStop(id: ProjectStopPathParams['id']) {
  return postProjectAction<ProjectStopData>(buildProjectStopApiPath(id)) as Promise<ProjectTaskReceipt>;
}

export function postProjectRestart(id: ProjectRestartPathParams['id']) {
  return postProjectAction<ProjectRestartData>(buildProjectRestartApiPath(id)) as Promise<ProjectTaskReceipt>;
}

export function postProjectRedeploy(id: ProjectRedeployPathParams['id']) {
  return postProjectAction<ProjectRedeployData>(buildProjectRedeployApiPath(id)) as Promise<ProjectTaskReceipt>;
}

export function putProjectLifecycleConfiguration(id: string, payload: ProjectLifecycleConfigurationUpdateRequest) {
  return request.put<ProjectLifecycleConfigurationSavedResponse>({
    url: buildProjectLifecycleConfigurationApiPath(id),
    data: payload,
  }) as Promise<ProjectLifecycleConfigurationSavedResponse>;
}

export function postProjectUnregister(id: ProjectUnregisterPathParams['id']) {
  return postProjectAction<ProjectUnregisterData>(buildProjectUnregisterApiPath(id)) as Promise<ProjectActionResponse>;
}

export function postProjectDestroy(id: ProjectDestroyPathParams['id'], payload: ProjectDestroyRequest) {
  return postProjectAction<ProjectDestroyData>(
    buildProjectDestroyApiPath(id),
    payload as ProjectDestroyPayload,
  ) as Promise<ProjectActionResponse>;
}

export function postProjectBatchActions(payload: ProjectBatchActionRequest) {
  return postProjectAction<ProjectBatchActionsData>(
    PROJECT_API_PATH.BATCH_ACTIONS,
    payload as ProjectBatchActionsPayload,
  ) as Promise<ProjectBatchActionResponse>;
}
