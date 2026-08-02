import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type {
  ApplicationActionResponse,
  ApplicationApplicationNameAvailabilityRequest,
  ApplicationApplicationNameAvailabilityResponse,
  ApplicationBatchActionRequest,
  ApplicationBatchActionResponse,
  ApplicationComposeRuntimeTargetCatalogResponse,
  ApplicationConfigurationMetadataResponse,
  ApplicationCreateRequest,
  ApplicationCreateResponse,
  ApplicationCreationMethodCatalogResponse,
  ApplicationDestroyRequest,
  ApplicationDetailResponseWithLifecycle,
  ApplicationDiscoveryCandidatesResponse,
  ApplicationLifecycleConfigurationSavedResponse,
  ApplicationLifecycleConfigurationUpdateRequest,
  ApplicationListQuery,
  ApplicationListResponseWithLifecycle,
  ApplicationLogResponse,
  ApplicationOverviewResponse,
  ApplicationSavedView,
  ApplicationSavedViewRequest,
  ApplicationServicesResponse,
  ApplicationTaskReceipt,
  ApplicationTemplate,
  ApplicationTemplateCatalogListResponse,
  ApplicationTemplateCatalogQuery,
  ApplicationTemplateDraftRequest,
  ApplicationTemplateListResponse,
  ApplicationTemplateSavedView,
  ApplicationTemplateSavedViewRequest,
  ApplicationWorkspaceEntry,
  ApplicationWorkspaceFileAnnotationRequest,
  ApplicationWorkspaceFileAnnotationResponse,
  ApplicationWorkspaceFileContentQuery,
  ApplicationWorkspaceFileContentResponse,
  ApplicationWorkspaceFileSaveRequest,
  ApplicationWorkspaceFileSaveResponse,
  ApplicationWorkspaceFilesQuery,
  ApplicationWorkspaceFilesResponse,
  ApplicationWorkspaceRenameRequest,
} from '../types/project';

type ApplicationListPath = typeof OPENAPI_RUNTIME_PATH.getApplications;
type GetApplicationListOperation = paths[ApplicationListPath]['get'];
type GetApplicationListEnvelope = GetApplicationListOperation['responses'][200]['content']['application/json'];
type GetApplicationListData = NonNullable<GetApplicationListEnvelope['data']>;
type GetApplicationListQuery = NonNullable<GetApplicationListOperation['parameters']['query']>;

type ApplicationDetailPath = typeof OPENAPI_RUNTIME_PATH.getApplication;
type GetApplicationDetailOperation = paths[ApplicationDetailPath]['get'];
type GetApplicationDetailEnvelope = GetApplicationDetailOperation['responses'][200]['content']['application/json'];
type GetApplicationDetailData = NonNullable<GetApplicationDetailEnvelope['data']>;
type GetApplicationDetailPathParams = GetApplicationDetailOperation['parameters']['path'];

type ApplicationOverviewPath = typeof OPENAPI_RUNTIME_PATH.getApplicationOverview;
type GetApplicationOverviewOperation = paths[ApplicationOverviewPath]['get'];
type GetApplicationOverviewEnvelope = GetApplicationOverviewOperation['responses'][200]['content']['application/json'];
type GetApplicationOverviewData = NonNullable<GetApplicationOverviewEnvelope['data']>;
type GetApplicationOverviewPathParams = GetApplicationOverviewOperation['parameters']['path'];

type ApplicationLogsPath = typeof OPENAPI_RUNTIME_PATH.getApplicationLogs;
type GetApplicationLogsOperation = paths[ApplicationLogsPath]['get'];
type GetApplicationLogsEnvelope = GetApplicationLogsOperation['responses'][200]['content']['application/json'];
type GetApplicationLogsData = NonNullable<GetApplicationLogsEnvelope['data']>;
type GetApplicationLogsPathParams = GetApplicationLogsOperation['parameters']['path'];
type GetApplicationLogsQuery = NonNullable<GetApplicationLogsOperation['parameters']['query']>;

type ApplicationServicesPath = typeof OPENAPI_RUNTIME_PATH.getApplicationServices;
type GetApplicationServicesOperation = paths[ApplicationServicesPath]['get'];
type GetApplicationServicesEnvelope = GetApplicationServicesOperation['responses'][200]['content']['application/json'];
type GetApplicationServicesData = NonNullable<GetApplicationServicesEnvelope['data']>;
type GetApplicationServicesPathParams = GetApplicationServicesOperation['parameters']['path'];

type ApplicationConfigurationPath = typeof OPENAPI_RUNTIME_PATH.getApplicationConfiguration;
type GetApplicationConfigurationOperation = paths[ApplicationConfigurationPath]['get'];
type GetApplicationConfigurationEnvelope =
  GetApplicationConfigurationOperation['responses'][200]['content']['application/json'];
type GetApplicationConfigurationData = NonNullable<GetApplicationConfigurationEnvelope['data']>;
type GetApplicationConfigurationPathParams = GetApplicationConfigurationOperation['parameters']['path'];

type ApplicationCreationMethodsPath = typeof OPENAPI_RUNTIME_PATH.getApplicationCreationMethods;
type GetApplicationCreationMethodsOperation = paths[ApplicationCreationMethodsPath]['get'];
type GetApplicationCreationMethodsEnvelope =
  GetApplicationCreationMethodsOperation['responses'][200]['content']['application/json'];
type GetApplicationCreationMethodsData = NonNullable<GetApplicationCreationMethodsEnvelope['data']>;

type ApplicationComposeRuntimeTargetsPath = typeof OPENAPI_RUNTIME_PATH.getApplicationComposeRuntimeTargets;
type GetApplicationComposeRuntimeTargetsOperation = paths[ApplicationComposeRuntimeTargetsPath]['get'];
type GetApplicationComposeRuntimeTargetsData = NonNullable<
  GetApplicationComposeRuntimeTargetsOperation['responses'][200]['content']['application/json']['data']
>;

type ApplicationDiscoveryCandidatesPath = typeof OPENAPI_RUNTIME_PATH.getApplicationDiscoveryCandidates;
type GetApplicationDiscoveryCandidatesOperation = paths[ApplicationDiscoveryCandidatesPath]['get'];
type GetApplicationDiscoveryCandidatesEnvelope =
  GetApplicationDiscoveryCandidatesOperation['responses'][200]['content']['application/json'];
type GetApplicationDiscoveryCandidatesData = NonNullable<GetApplicationDiscoveryCandidatesEnvelope['data']>;

type ApplicationCreatePath = typeof OPENAPI_RUNTIME_PATH.postApplicationCreate;
type ApplicationCreateOperation = paths[ApplicationCreatePath]['post'];
type ApplicationCreateEnvelope = ApplicationCreateOperation['responses'][201]['content']['application/json'];
type ApplicationCreateData = NonNullable<ApplicationCreateEnvelope['data']>;
type ApplicationCreatePayload = ApplicationCreateOperation['requestBody']['content']['application/json'];

type ApplicationApplicationNameAvailabilityPath = typeof OPENAPI_RUNTIME_PATH.postApplicationNameAvailability;
type ApplicationApplicationNameAvailabilityOperation = paths[ApplicationApplicationNameAvailabilityPath]['post'];
type ApplicationApplicationNameAvailabilityData = NonNullable<
  ApplicationApplicationNameAvailabilityOperation['responses'][200]['content']['application/json']['data']
>;

type ApplicationManagedTemplatesPath = typeof OPENAPI_RUNTIME_PATH.getApplicationManagedTemplates;
type GetApplicationManagedTemplatesOperation = paths[ApplicationManagedTemplatesPath]['get'];
type GetApplicationManagedTemplatesData = NonNullable<
  GetApplicationManagedTemplatesOperation['responses'][200]['content']['application/json']['data']
>;
type GetApplicationManagedTemplatesQuery = NonNullable<GetApplicationManagedTemplatesOperation['parameters']['query']>;
type ApplicationTemplateSavedViewsOperation =
  paths[typeof OPENAPI_RUNTIME_PATH.getApplicationTemplateSavedViews]['get'];
type ApplicationTemplateSavedViewsData = NonNullable<
  ApplicationTemplateSavedViewsOperation['responses'][200]['content']['application/json']['data']
>;
type ApplicationTemplateCreateSavedViewOperation =
  paths[typeof OPENAPI_RUNTIME_PATH.postApplicationTemplateSavedView]['post'];
type ApplicationTemplateCreateSavedViewData = NonNullable<
  ApplicationTemplateCreateSavedViewOperation['responses'][201]['content']['application/json']['data']
>;
type ApplicationTemplateSavedViewOperation = paths[typeof OPENAPI_RUNTIME_PATH.putApplicationTemplateSavedView]['put'];
type ApplicationTemplateUpdateSavedViewData = NonNullable<
  ApplicationTemplateSavedViewOperation['responses'][200]['content']['application/json']['data']
>;
type ApplicationUpOperation = paths[typeof OPENAPI_RUNTIME_PATH.postApplicationUp]['post'];
type ApplicationUpEnvelope = ApplicationUpOperation['responses'][202]['content']['application/json'];
type ApplicationUpData = NonNullable<ApplicationUpEnvelope['data']>;
type ApplicationUpPathParams = ApplicationUpOperation['parameters']['path'];

type ApplicationStopOperation = paths[typeof OPENAPI_RUNTIME_PATH.postApplicationStop]['post'];
type ApplicationStopEnvelope = ApplicationStopOperation['responses'][202]['content']['application/json'];
type ApplicationStopData = NonNullable<ApplicationStopEnvelope['data']>;
type ApplicationStopPathParams = ApplicationStopOperation['parameters']['path'];

type ApplicationRestartOperation = paths[typeof OPENAPI_RUNTIME_PATH.postApplicationRestart]['post'];
type ApplicationRestartEnvelope = ApplicationRestartOperation['responses'][202]['content']['application/json'];
type ApplicationRestartData = NonNullable<ApplicationRestartEnvelope['data']>;
type ApplicationRestartPathParams = ApplicationRestartOperation['parameters']['path'];

type ApplicationRedeployOperation = paths[typeof OPENAPI_RUNTIME_PATH.postApplicationRedeploy]['post'];
type ApplicationRedeployEnvelope = ApplicationRedeployOperation['responses'][202]['content']['application/json'];
type ApplicationRedeployData = NonNullable<ApplicationRedeployEnvelope['data']>;
type ApplicationRedeployPathParams = ApplicationRedeployOperation['parameters']['path'];

type ApplicationUnregisterOperation = paths[typeof OPENAPI_RUNTIME_PATH.postApplicationUnregister]['post'];
type ApplicationUnregisterEnvelope = ApplicationUnregisterOperation['responses'][200]['content']['application/json'];
type ApplicationUnregisterData = NonNullable<ApplicationUnregisterEnvelope['data']>;
type ApplicationUnregisterPathParams = ApplicationUnregisterOperation['parameters']['path'];

type ApplicationDestroyOperation = paths[typeof OPENAPI_RUNTIME_PATH.postApplicationDestroy]['post'];
type ApplicationDestroyEnvelope = ApplicationDestroyOperation['responses'][200]['content']['application/json'];
type ApplicationDestroyData = NonNullable<ApplicationDestroyEnvelope['data']>;
type ApplicationDestroyPayload = ApplicationDestroyOperation['requestBody']['content']['application/json'];
type ApplicationDestroyPathParams = ApplicationDestroyOperation['parameters']['path'];

type ApplicationBatchActionsOperation = paths[typeof OPENAPI_RUNTIME_PATH.postApplicationBatchActions]['post'];
type ApplicationBatchActionsEnvelope =
  ApplicationBatchActionsOperation['responses'][200]['content']['application/json'];
type ApplicationBatchActionsData = NonNullable<ApplicationBatchActionsEnvelope['data']>;
type ApplicationBatchActionsPayload = ApplicationBatchActionsOperation['requestBody']['content']['application/json'];
type ApplicationSavedViewsOperation = paths[typeof OPENAPI_RUNTIME_PATH.getApplicationSavedViews]['get'];
type ApplicationSavedViewsData = NonNullable<
  ApplicationSavedViewsOperation['responses'][200]['content']['application/json']['data']
>;
type ApplicationCreateSavedViewOperation = paths[typeof OPENAPI_RUNTIME_PATH.postApplicationSavedView]['post'];
type ApplicationCreateSavedViewData = NonNullable<
  ApplicationCreateSavedViewOperation['responses'][201]['content']['application/json']['data']
>;
type ApplicationSavedViewOperation = paths[typeof OPENAPI_RUNTIME_PATH.putApplicationSavedView]['put'];
type ApplicationUpdateSavedViewData = NonNullable<
  ApplicationSavedViewOperation['responses'][200]['content']['application/json']['data']
>;

function normalizeApplicationListQuery(query?: ApplicationListQuery): GetApplicationListQuery | undefined {
  if (!query) {
    return undefined;
  }

  return query satisfies GetApplicationListQuery;
}

export function getApplications(query?: ApplicationListQuery) {
  return request.get<GetApplicationListData>({
    url: OPENAPI_RUNTIME_PATH.getApplications,
    params: normalizeApplicationListQuery(query),
  }) as Promise<ApplicationListResponseWithLifecycle>;
}

/** 后端未返回视图数组时按空集合处理，避免可选的保存视图阻断项目列表。 */
export async function getApplicationSavedViews(): Promise<ApplicationSavedView[]> {
  const data = await request.get<ApplicationSavedViewsData>({ url: OPENAPI_RUNTIME_PATH.getApplicationSavedViews });
  return data.items ?? [];
}

export function postApplicationSavedView(payload: ApplicationSavedViewRequest) {
  return request.post<ApplicationCreateSavedViewData>({
    url: OPENAPI_RUNTIME_PATH.postApplicationSavedView,
    data: payload,
  }) as Promise<ApplicationSavedView>;
}

export function putApplicationSavedView(viewId: number, payload: ApplicationSavedViewRequest) {
  return request.put<ApplicationUpdateSavedViewData>({
    url: buildOpenApiRuntimePath('putApplicationSavedView', { viewId }),
    data: payload,
  }) as Promise<ApplicationSavedView>;
}

export function deleteApplicationSavedView(viewId: number) {
  return request.delete({ url: buildOpenApiRuntimePath('deleteApplicationSavedView', { viewId }) });
}

export function getApplication(applicationId: GetApplicationDetailPathParams['applicationId']) {
  return request.get<GetApplicationDetailData>({
    url: buildOpenApiRuntimePath('getApplication', { applicationId }),
  }) as Promise<ApplicationDetailResponseWithLifecycle>;
}

export function getApplicationOverview(applicationId: GetApplicationOverviewPathParams['applicationId']) {
  return request.get<GetApplicationOverviewData>({
    url: buildOpenApiRuntimePath('getApplicationOverview', { applicationId }),
  }) as Promise<ApplicationOverviewResponse>;
}

export function getApplicationLogs(
  applicationId: GetApplicationLogsPathParams['applicationId'],
  query?: GetApplicationLogsQuery,
) {
  return request.get<GetApplicationLogsData>({
    url: buildOpenApiRuntimePath('getApplicationLogs', { applicationId }),
    params: query,
  }) as Promise<ApplicationLogResponse>;
}

export function getApplicationServices(applicationId: GetApplicationServicesPathParams['applicationId']) {
  return request.get<GetApplicationServicesData>({
    url: buildOpenApiRuntimePath('getApplicationServices', { applicationId }),
  }) as Promise<ApplicationServicesResponse>;
}

export function postApplicationWorkspaceEntry(applicationId: string, payload: ApplicationWorkspaceEntry) {
  return request.post({
    url: buildOpenApiRuntimePath('postApplicationWorkspaceEntry', { applicationId }),
    data: payload,
  });
}

export function postApplicationWorkspaceRename(applicationId: string, payload: ApplicationWorkspaceRenameRequest) {
  return request.post({
    url: buildOpenApiRuntimePath('postApplicationWorkspaceEntryRename', { applicationId }),
    data: payload,
  });
}

export function deleteApplicationWorkspaceEntry(applicationId: string, query: { path: string; recursive?: boolean }) {
  return request.delete({
    url: buildOpenApiRuntimePath('deleteApplicationWorkspaceEntry', { applicationId }),
    params: query,
  });
}

export function getApplicationConfiguration(applicationId: GetApplicationConfigurationPathParams['applicationId']) {
  return request.get<GetApplicationConfigurationData>({
    url: buildOpenApiRuntimePath('getApplicationConfiguration', { applicationId }),
  }) as Promise<ApplicationConfigurationMetadataResponse>;
}

export function getApplicationFiles(id: string, query?: ApplicationWorkspaceFilesQuery) {
  return request.get<ApplicationWorkspaceFilesResponse>({
    url: buildOpenApiRuntimePath('getApplicationFiles', { applicationId: id }),
    params: query,
  }) as Promise<ApplicationWorkspaceFilesResponse>;
}

export function getApplicationFileContent(id: string, query: ApplicationWorkspaceFileContentQuery) {
  return request.get<ApplicationWorkspaceFileContentResponse>({
    url: buildOpenApiRuntimePath('getApplicationFileContent', { applicationId: id }),
    params: query,
  }) as Promise<ApplicationWorkspaceFileContentResponse>;
}

export function putApplicationFileContent(
  id: string,
  query: ApplicationWorkspaceFileContentQuery,
  payload: ApplicationWorkspaceFileSaveRequest,
) {
  return request.put<ApplicationWorkspaceFileSaveResponse>({
    url: buildOpenApiRuntimePath('putApplicationFileContent', { applicationId: id }),
    params: query,
    data: payload,
  }) as Promise<ApplicationWorkspaceFileSaveResponse>;
}

export function putApplicationFileAnnotation(
  id: string,
  query: ApplicationWorkspaceFileContentQuery,
  payload: ApplicationWorkspaceFileAnnotationRequest,
) {
  return request.put<ApplicationWorkspaceFileAnnotationResponse>({
    url: buildOpenApiRuntimePath('putApplicationFileAnnotation', { applicationId: id }),
    params: query,
    data: payload,
  }) as Promise<ApplicationWorkspaceFileAnnotationResponse>;
}

export function getApplicationCreationMethods() {
  return request.get<GetApplicationCreationMethodsData>({
    url: OPENAPI_RUNTIME_PATH.getApplicationCreationMethods,
  }) as Promise<ApplicationCreationMethodCatalogResponse>;
}

export function getApplicationComposeRuntimeTargets() {
  return request.get<GetApplicationComposeRuntimeTargetsData>({
    url: OPENAPI_RUNTIME_PATH.getApplicationComposeRuntimeTargets,
  }) as Promise<ApplicationComposeRuntimeTargetCatalogResponse>;
}

export function getApplicationDiscoveryCandidates() {
  return request.get<GetApplicationDiscoveryCandidatesData>({
    url: OPENAPI_RUNTIME_PATH.getApplicationDiscoveryCandidates,
  }) as Promise<ApplicationDiscoveryCandidatesResponse>;
}

export function postApplicationCreate(payload: ApplicationCreateRequest) {
  return postApplicationAction<ApplicationCreateData>(
    OPENAPI_RUNTIME_PATH.postApplicationCreate,
    payload as ApplicationCreatePayload,
  ) as Promise<ApplicationCreateResponse>;
}

export function postApplicationApplicationNameAvailability(payload: ApplicationApplicationNameAvailabilityRequest) {
  return request.post<ApplicationApplicationNameAvailabilityData>({
    url: OPENAPI_RUNTIME_PATH.postApplicationNameAvailability,
    data: payload,
  }) as Promise<ApplicationApplicationNameAvailabilityResponse>;
}

export function getApplicationTemplateCatalog(query: ApplicationTemplateCatalogQuery) {
  return request.get<ApplicationTemplateCatalogListResponse>({
    url: OPENAPI_RUNTIME_PATH.getApplicationTemplates,
    params: query,
  });
}

export function getPublishedApplicationTemplate(templateId: string): Promise<ApplicationTemplate> {
  return request.get<ApplicationTemplate>({
    url: buildOpenApiRuntimePath('getPublishedApplicationTemplate', { templateId }),
  });
}

export function getPublishedApplicationTemplateVersion(templateVersionId: string): Promise<ApplicationTemplate> {
  return request.get<ApplicationTemplate>({
    url: buildOpenApiRuntimePath('getPublishedApplicationTemplateVersion', { templateVersionId }),
  });
}

/** 管理目录会返回草稿与归档项，只能由模板管理页面在已授权上下文中消费。 */
export async function getApplicationManagedTemplates(
  query?: GetApplicationManagedTemplatesQuery,
): Promise<ApplicationTemplateListResponse> {
  return request.get<GetApplicationManagedTemplatesData>({
    url: OPENAPI_RUNTIME_PATH.getApplicationManagedTemplates,
    params: query,
  });
}

export async function getApplicationTemplateSavedViews(): Promise<ApplicationTemplateSavedView[]> {
  const data = await request.get<ApplicationTemplateSavedViewsData>({
    url: OPENAPI_RUNTIME_PATH.getApplicationTemplateSavedViews,
  });
  return data.items ?? [];
}

export function postApplicationTemplateSavedView(payload: ApplicationTemplateSavedViewRequest) {
  return request.post<ApplicationTemplateCreateSavedViewData>({
    url: OPENAPI_RUNTIME_PATH.postApplicationTemplateSavedView,
    data: payload,
  }) as Promise<ApplicationTemplateSavedView>;
}

export function putApplicationTemplateSavedView(viewId: number, payload: ApplicationTemplateSavedViewRequest) {
  return request.put<ApplicationTemplateUpdateSavedViewData>({
    url: buildOpenApiRuntimePath('putApplicationTemplateSavedView', { viewId }),
    data: payload,
  }) as Promise<ApplicationTemplateSavedView>;
}

export function deleteApplicationTemplateSavedView(viewId: number) {
  return request.delete({ url: buildOpenApiRuntimePath('deleteApplicationTemplateSavedView', { viewId }) });
}

export async function getApplicationTemplate(templateId: string): Promise<ApplicationTemplate> {
  return request.get<ApplicationTemplate>({
    url: buildOpenApiRuntimePath('getApplicationTemplate', { templateId }),
  });
}

export async function postApplicationTemplate(payload: ApplicationTemplateDraftRequest) {
  return request.post<ApplicationTemplate>({
    url: OPENAPI_RUNTIME_PATH.postApplicationTemplate,
    data: payload,
  });
}

export async function putApplicationTemplate(templateId: string, payload: ApplicationTemplateDraftRequest) {
  return request.put<ApplicationTemplate>({
    url: buildOpenApiRuntimePath('putApplicationTemplate', { templateId }),
    data: payload,
  });
}

export async function postApplicationTemplateClone(templateId: string, displayName: string) {
  return request.post<ApplicationTemplate>({
    url: buildOpenApiRuntimePath('postApplicationTemplateClone', { templateId }),
    data: { display_name: displayName },
  });
}

export async function postApplicationTemplatePublish(templateId: string) {
  return request.post<ApplicationTemplate>({
    url: buildOpenApiRuntimePath('postApplicationTemplatePublish', { templateId }),
  });
}

export async function postApplicationTemplateWithdraw(templateId: string) {
  return request.post<ApplicationTemplate>({
    url: buildOpenApiRuntimePath('postApplicationTemplateWithdraw', { templateId }),
  });
}

export function postApplicationTemplateArchive(templateId: string) {
  return request.post({ url: buildOpenApiRuntimePath('postApplicationTemplateArchive', { templateId }) });
}

export function deleteApplicationTemplate(templateId: string) {
  return request.delete({ url: buildOpenApiRuntimePath('deleteApplicationTemplate', { templateId }) });
}

function postApplicationAction<T>(url: string, data?: unknown) {
  return request.post<T>({
    url,
    data,
  });
}

export function postApplicationUp(applicationId: ApplicationUpPathParams['applicationId']) {
  return postApplicationAction<ApplicationUpData>(
    buildOpenApiRuntimePath('postApplicationUp', { applicationId }),
  ) as Promise<ApplicationTaskReceipt>;
}

export function postApplicationStop(applicationId: ApplicationStopPathParams['applicationId']) {
  return postApplicationAction<ApplicationStopData>(
    buildOpenApiRuntimePath('postApplicationStop', { applicationId }),
  ) as Promise<ApplicationTaskReceipt>;
}

export function postApplicationRestart(applicationId: ApplicationRestartPathParams['applicationId']) {
  return postApplicationAction<ApplicationRestartData>(
    buildOpenApiRuntimePath('postApplicationRestart', { applicationId }),
  ) as Promise<ApplicationTaskReceipt>;
}

export function postApplicationRedeploy(applicationId: ApplicationRedeployPathParams['applicationId']) {
  return postApplicationAction<ApplicationRedeployData>(
    buildOpenApiRuntimePath('postApplicationRedeploy', { applicationId }),
  ) as Promise<ApplicationTaskReceipt>;
}

export function putApplicationLifecycleConfiguration(
  id: string,
  payload: ApplicationLifecycleConfigurationUpdateRequest,
) {
  return request.put<ApplicationLifecycleConfigurationSavedResponse>({
    url: buildOpenApiRuntimePath('putApplicationLifecycleConfiguration', { applicationId: id }),
    data: payload,
  }) as Promise<ApplicationLifecycleConfigurationSavedResponse>;
}

export function postApplicationUnregister(applicationId: ApplicationUnregisterPathParams['applicationId']) {
  return postApplicationAction<ApplicationUnregisterData>(
    buildOpenApiRuntimePath('postApplicationUnregister', { applicationId }),
  ) as Promise<ApplicationActionResponse>;
}

export function postApplicationDestroy(
  applicationId: ApplicationDestroyPathParams['applicationId'],
  payload: ApplicationDestroyRequest,
) {
  return postApplicationAction<ApplicationDestroyData>(
    buildOpenApiRuntimePath('postApplicationDestroy', { applicationId }),
    payload as ApplicationDestroyPayload,
  ) as Promise<ApplicationActionResponse>;
}

export function postApplicationBatchActions(payload: ApplicationBatchActionRequest) {
  return postApplicationAction<ApplicationBatchActionsData>(
    OPENAPI_RUNTIME_PATH.postApplicationBatchActions,
    payload as ApplicationBatchActionsPayload,
  ) as Promise<ApplicationBatchActionResponse>;
}
