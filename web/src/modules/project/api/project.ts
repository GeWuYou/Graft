import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import {
  buildProjectConfigurationApiPath,
  buildProjectConfigurationPreviewApiPath,
  buildProjectConfigurationValidateApiPath,
  buildProjectDeployApiPath,
  buildProjectDestroyApiPath,
  buildProjectDetailApiPath,
  buildProjectFilesAnnotationApiPath,
  buildProjectFilesApiPath,
  buildProjectFilesContentApiPath,
  buildProjectLifecycleConfigurationApiPath,
  buildProjectLogsApiPath,
  buildProjectOverviewApiPath,
  buildProjectRedeployApiPath,
  buildProjectRestartApiPath,
  buildProjectServicesApiPath,
  buildProjectStopApiPath,
  buildProjectUnregisterApiPath,
  buildProjectUpApiPath,
  PROJECT_API_PATH,
} from '../contract/paths';
import type {
  ProjectActionResponse,
  ProjectBatchActionRequest,
  ProjectBatchActionResponse,
  ProjectConfigurationMetadataResponse,
  ProjectConfigurationPreviewResponse,
  ProjectConfigurationValidateResponse,
  ProjectCreateRequest,
  ProjectCreateResponse,
  ProjectCreateValidateRequest,
  ProjectCreateValidateResponse,
  ProjectDeployResponse,
  ProjectDestroyRequest,
  ProjectDetailResponseWithLifecycle,
  ProjectDiscoveryCandidatesResponse,
  ProjectLifecycleConfigurationSavedResponse,
  ProjectLifecycleConfigurationUpdateRequest,
  ProjectListQuery,
  ProjectListResponseWithLifecycle,
  ProjectLogResponse,
  ProjectManagedRootResponse,
  ProjectOverviewResponse,
  ProjectServicesResponse,
  ProjectSourceCatalogResponse,
  ProjectWorkspaceFileAnnotationRequest,
  ProjectWorkspaceFileAnnotationResponse,
  ProjectWorkspaceFileContentQuery,
  ProjectWorkspaceFileContentResponse,
  ProjectWorkspaceFileSaveRequest,
  ProjectWorkspaceFileSaveResponse,
  ProjectWorkspaceFilesQuery,
  ProjectWorkspaceFilesResponse,
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

type ProjectConfigurationPreviewPath = (typeof PROJECT_API_PATH)['CONFIGURATION_PREVIEW'];
type GetProjectConfigurationPreviewOperation = paths[ProjectConfigurationPreviewPath]['get'];
type GetProjectConfigurationPreviewEnvelope =
  GetProjectConfigurationPreviewOperation['responses'][200]['content']['application/json'];
type GetProjectConfigurationPreviewData = NonNullable<GetProjectConfigurationPreviewEnvelope['data']>;
type GetProjectConfigurationPreviewPathParams = GetProjectConfigurationPreviewOperation['parameters']['path'];

type ProjectConfigurationValidatePath = (typeof PROJECT_API_PATH)['CONFIGURATION_VALIDATE'];
type ProjectConfigurationValidateOperation = paths[ProjectConfigurationValidatePath]['post'];
type ProjectConfigurationValidateEnvelope =
  ProjectConfigurationValidateOperation['responses'][200]['content']['application/json'];
type ProjectConfigurationValidateData = NonNullable<ProjectConfigurationValidateEnvelope['data']>;
type ProjectConfigurationValidatePathParams = ProjectConfigurationValidateOperation['parameters']['path'];

type ProjectManagedRootPath = (typeof PROJECT_API_PATH)['MANAGED_ROOT'];
type GetProjectManagedRootOperation = paths[ProjectManagedRootPath]['get'];
type GetProjectManagedRootEnvelope = GetProjectManagedRootOperation['responses'][200]['content']['application/json'];
type GetProjectManagedRootData = NonNullable<GetProjectManagedRootEnvelope['data']>;

type ProjectSourcesPath = (typeof PROJECT_API_PATH)['SOURCES'];
type GetProjectSourcesOperation = paths[ProjectSourcesPath]['get'];
type GetProjectSourcesEnvelope = GetProjectSourcesOperation['responses'][200]['content']['application/json'];
type GetProjectSourcesData = NonNullable<GetProjectSourcesEnvelope['data']>;

type ProjectDiscoveryCandidatesPath = (typeof PROJECT_API_PATH)['DISCOVERY_CANDIDATES'];
type GetProjectDiscoveryCandidatesOperation = paths[ProjectDiscoveryCandidatesPath]['get'];
type GetProjectDiscoveryCandidatesEnvelope =
  GetProjectDiscoveryCandidatesOperation['responses'][200]['content']['application/json'];
type GetProjectDiscoveryCandidatesData = NonNullable<GetProjectDiscoveryCandidatesEnvelope['data']>;

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

type ProjectDeployOperation = paths[(typeof PROJECT_API_PATH)['DEPLOY']]['post'];
type ProjectDeployEnvelope = ProjectDeployOperation['responses'][200]['content']['application/json'];
type ProjectDeployData = NonNullable<ProjectDeployEnvelope['data']>;
type ProjectDeployPathParams = ProjectDeployOperation['parameters']['path'];

type ProjectUpOperation = paths[(typeof PROJECT_API_PATH)['UP']]['post'];
type ProjectUpEnvelope = ProjectUpOperation['responses'][200]['content']['application/json'];
type ProjectUpData = NonNullable<ProjectUpEnvelope['data']>;
type ProjectUpPathParams = ProjectUpOperation['parameters']['path'];

type ProjectStopOperation = paths[(typeof PROJECT_API_PATH)['STOP']]['post'];
type ProjectStopEnvelope = ProjectStopOperation['responses'][200]['content']['application/json'];
type ProjectStopData = NonNullable<ProjectStopEnvelope['data']>;
type ProjectStopPathParams = ProjectStopOperation['parameters']['path'];

type ProjectRestartOperation = paths[(typeof PROJECT_API_PATH)['RESTART']]['post'];
type ProjectRestartEnvelope = ProjectRestartOperation['responses'][200]['content']['application/json'];
type ProjectRestartData = NonNullable<ProjectRestartEnvelope['data']>;
type ProjectRestartPathParams = ProjectRestartOperation['parameters']['path'];

type ProjectRedeployOperation = paths[(typeof PROJECT_API_PATH)['REDEPLOY']]['post'];
type ProjectRedeployEnvelope = ProjectRedeployOperation['responses'][200]['content']['application/json'];
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

/**
 * 规范化项目列表查询参数。
 *
 * @param query - 项目列表查询条件
 * @returns 传入的查询条件；未提供时返回 `undefined`
 */
function normalizeProjectListQuery(query?: ProjectListQuery): GetProjectListQuery | undefined {
  if (!query) {
    return undefined;
  }

  return query satisfies GetProjectListQuery;
}

/**
 * 获取项目列表。
 *
 * @param query - 列表查询条件
 * @returns 项目列表数据
 */
export function getProjects(query?: ProjectListQuery) {
  return request.get<GetProjectListData>({
    url: PROJECT_API_PATH.LIST,
    params: normalizeProjectListQuery(query),
  }) as Promise<ProjectListResponseWithLifecycle>;
}

/**
 * 获取项目详情。
 *
 * @param id - 项目 ID
 * @returns 项目详情数据
 */
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

/**
 * 获取项目的服务信息。
 *
 * @param id - 项目 ID
 * @returns 项目服务信息响应
 */
export function getProjectServices(id: GetProjectServicesPathParams['id']) {
  return request.get<GetProjectServicesData>({
    url: buildProjectServicesApiPath(id),
  }) as Promise<ProjectServicesResponse>;
}

/**
 * 获取项目配置元数据。
 *
 * @param id - 项目 ID
 * @returns 项目配置元数据。
 */
export function getProjectConfiguration(id: GetProjectConfigurationPathParams['id']) {
  return request.get<GetProjectConfigurationData>({
    url: buildProjectConfigurationApiPath(id),
  }) as Promise<ProjectConfigurationMetadataResponse>;
}

/**
 * 获取项目配置预览。
 *
 * @param id - 项目 ID
 * @returns 项目配置预览结果
 */
export function getProjectConfigurationPreview(id: GetProjectConfigurationPreviewPathParams['id']) {
  return request.get<GetProjectConfigurationPreviewData>({
    url: buildProjectConfigurationPreviewApiPath(id),
  }) as Promise<ProjectConfigurationPreviewResponse>;
}

export function getProjectFiles(id: number, query?: ProjectWorkspaceFilesQuery) {
  return request.get<ProjectWorkspaceFilesResponse>({
    url: buildProjectFilesApiPath(id),
    params: query,
  }) as Promise<ProjectWorkspaceFilesResponse>;
}

export function getProjectFileContent(id: number, query: ProjectWorkspaceFileContentQuery) {
  return request.get<ProjectWorkspaceFileContentResponse>({
    url: buildProjectFilesContentApiPath(id),
    params: query,
  }) as Promise<ProjectWorkspaceFileContentResponse>;
}

export function putProjectFileContent(
  id: number,
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
  id: number,
  query: ProjectWorkspaceFileContentQuery,
  payload: ProjectWorkspaceFileAnnotationRequest,
) {
  return request.put<ProjectWorkspaceFileAnnotationResponse>({
    url: buildProjectFilesAnnotationApiPath(id),
    params: query,
    data: payload,
  }) as Promise<ProjectWorkspaceFileAnnotationResponse>;
}

/**
 * 校验指定项目的配置。
 *
 * @param id - 项目 ID
 * @returns 配置校验结果
 */
export function postProjectConfigurationValidate(id: ProjectConfigurationValidatePathParams['id']) {
  return postProjectAction<ProjectConfigurationValidateData>(
    buildProjectConfigurationValidateApiPath(id),
  ) as Promise<ProjectConfigurationValidateResponse>;
}

/**
 * 获取项目可管理的根目录。
 *
 * @returns 项目可管理根目录的响应数据。
 */
export function getProjectManagedRoot() {
  return request.get<GetProjectManagedRootData>({
    url: PROJECT_API_PATH.MANAGED_ROOT,
  }) as Promise<ProjectManagedRootResponse>;
}

/**
 * 获取项目来源目录。
 *
 * @returns 项目来源目录信息。
 */
export function getProjectSources() {
  return request.get<GetProjectSourcesData>({
    url: PROJECT_API_PATH.SOURCES,
  }) as Promise<ProjectSourceCatalogResponse>;
}

/**
 * 获取项目的发现候选列表。
 *
 * @returns 项目发现候选列表。
 */
export function getProjectDiscoveryCandidates() {
  return request.get<GetProjectDiscoveryCandidatesData>({
    url: PROJECT_API_PATH.DISCOVERY_CANDIDATES,
  }) as Promise<ProjectDiscoveryCandidatesResponse>;
}

/**
 * 校验项目创建请求。
 *
 * @param payload - 项目创建校验请求内容
 * @returns 校验结果
 */
export function postProjectCreateValidate(payload: ProjectCreateValidateRequest) {
  return postProjectAction<ProjectCreateValidateData>(
    PROJECT_API_PATH.CREATE_VALIDATE,
    payload as ProjectCreateValidatePayload,
  ) as Promise<ProjectCreateValidateResponse>;
}

/**
 * 创建项目。
 *
 * @param payload - 创建项目所需的请求内容
 * @returns 创建结果响应
 */
export function postProjectCreate(payload: ProjectCreateRequest) {
  return postProjectAction<ProjectCreateData>(
    PROJECT_API_PATH.CREATE,
    payload as ProjectCreatePayload,
  ) as Promise<ProjectCreateResponse>;
}

/**
 * 发送项目相关的 POST 请求。
 *
 * @param url - 请求地址
 * @param data - 请求体
 * @returns 请求结果
 */
function postProjectAction<T>(url: string, data?: unknown) {
  return request.post<T>({
    url,
    data,
  });
}

/**
 * 部署指定项目。
 *
 * @param id - 项目 ID
 * @returns 部署操作的响应结果
 */
export function postProjectDeploy(id: ProjectDeployPathParams['id']) {
  return postProjectAction<ProjectDeployData>(buildProjectDeployApiPath(id)) as Promise<ProjectDeployResponse>;
}

/**
 * 执行项目启动操作。
 *
 * @param id - 项目 ID
 * @returns 项目操作响应
 */
export function postProjectUp(id: ProjectUpPathParams['id']) {
  return postProjectAction<ProjectUpData>(buildProjectUpApiPath(id)) as Promise<ProjectActionResponse>;
}

/**
 * 将指定项目下线。
 *
 * @param id - 项目 ID
 * @returns 项目操作响应结果
 */
export function postProjectStop(id: ProjectStopPathParams['id']) {
  return postProjectAction<ProjectStopData>(buildProjectStopApiPath(id)) as Promise<ProjectActionResponse>;
}

/**
 * 重启指定项目。
 *
 * @param id - 项目 ID
 * @returns 重启操作的响应结果
 */
export function postProjectRestart(id: ProjectRestartPathParams['id']) {
  return postProjectAction<ProjectRestartData>(buildProjectRestartApiPath(id)) as Promise<ProjectActionResponse>;
}

/**
 * 重新部署指定项目。
 *
 * @param id - 项目 ID
 * @returns 重新部署操作的响应结果
 */
export function postProjectRedeploy(id: ProjectRedeployPathParams['id']) {
  return postProjectAction<ProjectRedeployData>(buildProjectRedeployApiPath(id)) as Promise<ProjectActionResponse>;
}

export function putProjectLifecycleConfiguration(id: number, payload: ProjectLifecycleConfigurationUpdateRequest) {
  return request.put<ProjectLifecycleConfigurationSavedResponse>({
    url: buildProjectLifecycleConfigurationApiPath(id),
    data: payload,
  }) as Promise<ProjectLifecycleConfigurationSavedResponse>;
}

/**
 * 注销指定项目。
 *
 * @param id - 项目 ID
 * @returns 项目操作结果
 */
export function postProjectUnregister(id: ProjectUnregisterPathParams['id']) {
  return postProjectAction<ProjectUnregisterData>(buildProjectUnregisterApiPath(id)) as Promise<ProjectActionResponse>;
}

/**
 * 销毁指定项目。
 *
 * @param id - 项目 ID
 * @param payload - 销毁请求内容
 * @returns 销毁操作结果
 */
export function postProjectDestroy(id: ProjectDestroyPathParams['id'], payload: ProjectDestroyRequest) {
  return postProjectAction<ProjectDestroyData>(
    buildProjectDestroyApiPath(id),
    payload as ProjectDestroyPayload,
  ) as Promise<ProjectActionResponse>;
}

/**
 * 批量执行项目动作。
 *
 * @param payload - 批量动作请求体
 * @returns 批量动作结果
 */
export function postProjectBatchActions(payload: ProjectBatchActionRequest) {
  return postProjectAction<ProjectBatchActionsData>(
    PROJECT_API_PATH.BATCH_ACTIONS,
    payload as ProjectBatchActionsPayload,
  ) as Promise<ProjectBatchActionResponse>;
}
