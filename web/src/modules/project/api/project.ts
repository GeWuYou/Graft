import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import {
  buildProjectConfigurationApiPath,
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
  buildProjectSavedViewApiPath,
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
  ProjectComposeRuntimeTargetCatalogResponse,
  ProjectConfigurationMetadataResponse,
  ProjectCreateRequest,
  ProjectCreateResponse,
  ProjectCreationMethodCatalogResponse,
  ProjectDeployResponse,
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

type ProjectTemplateCreatePath = (typeof PROJECT_API_PATH)['CREATE_TEMPLATE'];
type ProjectTemplateCreateOperation = paths[ProjectTemplateCreatePath]['post'];
type ProjectTemplateCreateData = NonNullable<
  ProjectTemplateCreateOperation['responses'][201]['content']['application/json']['data']
>;
type ProjectDeployOperation = paths[(typeof PROJECT_API_PATH)['DEPLOY']]['post'];
type ProjectDeployEnvelope = ProjectDeployOperation['responses'][200]['content']['application/json'];
type ProjectDeployData = NonNullable<ProjectDeployEnvelope['data']>;
type ProjectDeployPathParams = ProjectDeployOperation['parameters']['path'];

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
 * 获取项目已保存视图列表。
 *
 * @returns 项目已保存视图数组；响应缺少列表数据时返回空数组。
 */
export async function getProjectSavedViews(): Promise<ProjectSavedView[]> {
  const data = await request.get<ProjectSavedViewsData>({ url: PROJECT_API_PATH.SAVED_VIEWS });
  return data.items ?? [];
}

/**
 * 创建项目已保存视图。
 *
 * @param payload - 已保存视图的创建数据
 * @returns 创建后的项目已保存视图
 */
export function postProjectSavedView(payload: ProjectSavedViewRequest) {
  return request.post<ProjectCreateSavedViewData>({
    url: PROJECT_API_PATH.SAVED_VIEWS,
    data: payload,
  }) as Promise<ProjectSavedView>;
}

/**
 * 更新指定的项目已保存视图。
 *
 * @param viewId - 要更新的已保存视图 ID
 * @param payload - 已保存视图的更新数据
 * @returns 更新后的项目已保存视图
 */
export function putProjectSavedView(viewId: number, payload: ProjectSavedViewRequest) {
  return request.put<ProjectUpdateSavedViewData>({
    url: buildProjectSavedViewApiPath(viewId),
    data: payload,
  }) as Promise<ProjectSavedView>;
}

/**
 * 删除指定的项目已保存视图。
 *
 * @param viewId - 要删除的已保存视图标识
 * @returns 删除请求的响应结果
 */
export function deleteProjectSavedView(viewId: number) {
  return request.delete({ url: buildProjectSavedViewApiPath(viewId) });
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
 * 获取项目工作区文件列表。
 *
 * @param id - 项目 ID
 * @param query - 文件列表查询条件
 * @returns 项目工作区文件列表响应
 */
export function getProjectFiles(id: string, query?: ProjectWorkspaceFilesQuery) {
  return request.get<ProjectWorkspaceFilesResponse>({
    url: buildProjectFilesApiPath(id),
    params: query,
  }) as Promise<ProjectWorkspaceFilesResponse>;
}

/**
 * 获取项目工作区文件的内容。
 *
 * @param id - 项目标识
 * @param query - 文件内容查询参数
 * @returns 项目工作区文件内容响应
 */
export function getProjectFileContent(id: string, query: ProjectWorkspaceFileContentQuery) {
  return request.get<ProjectWorkspaceFileContentResponse>({
    url: buildProjectFilesContentApiPath(id),
    params: query,
  }) as Promise<ProjectWorkspaceFileContentResponse>;
}

/**
 * 保存项目工作区文件内容。
 *
 * @param id - 项目标识
 * @param query - 文件内容查询参数
 * @param payload - 要保存的文件内容
 * @returns 文件保存结果
 */
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

/**
 * 更新项目文件的注释信息。
 *
 * @param id - 项目 ID
 * @param query - 文件定位查询条件
 * @param payload - 注释内容
 * @returns 保存后的文件注释信息
 */
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

/**
 * 获取项目创建方式目录。
 *
 * @returns 项目创建方式目录信息。
 */
export function getProjectCreationMethods() {
  return request.get<GetProjectCreationMethodsData>({
    url: PROJECT_API_PATH.CREATION_METHODS,
  }) as Promise<ProjectCreationMethodCatalogResponse>;
}

/** Returns runtime targets eligible for Compose workspaces. */
export function getProjectComposeRuntimeTargets() {
  return request.get<GetProjectComposeRuntimeTargetsData>({
    url: PROJECT_API_PATH.COMPOSE_RUNTIME_TARGETS,
  }) as Promise<ProjectComposeRuntimeTargetCatalogResponse>;
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
 * 使用模板创建项目。
 *
 * @param payload - 模板创建请求数据
 * @returns 创建项目的响应
 */
export function postProjectCreateTemplate(payload: ProjectTemplateCreateRequest) {
  return postProjectAction<ProjectTemplateCreateData>(
    PROJECT_API_PATH.CREATE_TEMPLATE,
    payload,
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
 * 启动指定项目。
 *
 * @param id - 项目 ID
 * @returns 项目启动任务回执
 */
export function postProjectUp(id: ProjectUpPathParams['id']) {
  return postProjectAction<ProjectUpData>(buildProjectUpApiPath(id)) as Promise<ProjectTaskReceipt>;
}

/**
 * 停止指定项目。
 *
 * @param id - 项目 ID
 * @returns 项目任务回执
 */
export function postProjectStop(id: ProjectStopPathParams['id']) {
  return postProjectAction<ProjectStopData>(buildProjectStopApiPath(id)) as Promise<ProjectTaskReceipt>;
}

/**
 * 重启指定项目。
 *
 * @param id - 项目 ID
 * @returns 重启操作的响应结果
 */
export function postProjectRestart(id: ProjectRestartPathParams['id']) {
  return postProjectAction<ProjectRestartData>(buildProjectRestartApiPath(id)) as Promise<ProjectTaskReceipt>;
}

/**
 * 重新部署指定项目。
 *
 * @param id - 项目 ID
 * @returns 重新部署操作的响应结果
 */
export function postProjectRedeploy(id: ProjectRedeployPathParams['id']) {
  return postProjectAction<ProjectRedeployData>(buildProjectRedeployApiPath(id)) as Promise<ProjectTaskReceipt>;
}

/**
 * 更新项目的生命周期配置。
 *
 * @param id - 项目标识
 * @param payload - 要保存的生命周期配置
 * @returns 保存后的项目生命周期配置响应
 */
export function putProjectLifecycleConfiguration(id: string, payload: ProjectLifecycleConfigurationUpdateRequest) {
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
