import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import {
  APPLICATION_API_PATH,
  buildApplicationConfigurationApiPath,
  buildApplicationDestroyApiPath,
  buildApplicationDetailApiPath,
  buildApplicationFilesAnnotationApiPath,
  buildApplicationFilesApiPath,
  buildApplicationFilesContentApiPath,
  buildApplicationFilesEntriesApiPath,
  buildApplicationFilesRenameApiPath,
  buildApplicationLifecycleConfigurationApiPath,
  buildApplicationLogsApiPath,
  buildApplicationOverviewApiPath,
  buildApplicationPublishedTemplateApiPath,
  buildApplicationRedeployApiPath,
  buildApplicationRestartApiPath,
  buildApplicationSavedViewApiPath,
  buildApplicationServicesApiPath,
  buildApplicationStopApiPath,
  buildApplicationTemplateApiPath,
  buildApplicationTemplateArchiveApiPath,
  buildApplicationTemplateCloneApiPath,
  buildApplicationTemplatePublishApiPath,
  buildApplicationTemplateVersionApiPath,
  buildApplicationTemplateWithdrawApiPath,
  buildApplicationUnregisterApiPath,
  buildApplicationUpApiPath,
} from '../contract/paths';
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

type ApplicationListPath = (typeof APPLICATION_API_PATH)['LIST'];
type GetApplicationListOperation = paths[ApplicationListPath]['get'];
type GetApplicationListEnvelope = GetApplicationListOperation['responses'][200]['content']['application/json'];
type GetApplicationListData = NonNullable<GetApplicationListEnvelope['data']>;
type GetApplicationListQuery = NonNullable<GetApplicationListOperation['parameters']['query']>;

type ApplicationDetailPath = (typeof APPLICATION_API_PATH)['DETAIL'];
type GetApplicationDetailOperation = paths[ApplicationDetailPath]['get'];
type GetApplicationDetailEnvelope = GetApplicationDetailOperation['responses'][200]['content']['application/json'];
type GetApplicationDetailData = NonNullable<GetApplicationDetailEnvelope['data']>;
type GetApplicationDetailPathParams = GetApplicationDetailOperation['parameters']['path'];

type ApplicationOverviewPath = (typeof APPLICATION_API_PATH)['OVERVIEW'];
type GetApplicationOverviewOperation = paths[ApplicationOverviewPath]['get'];
type GetApplicationOverviewEnvelope = GetApplicationOverviewOperation['responses'][200]['content']['application/json'];
type GetApplicationOverviewData = NonNullable<GetApplicationOverviewEnvelope['data']>;
type GetApplicationOverviewPathParams = GetApplicationOverviewOperation['parameters']['path'];

type ApplicationLogsPath = (typeof APPLICATION_API_PATH)['LOGS'];
type GetApplicationLogsOperation = paths[ApplicationLogsPath]['get'];
type GetApplicationLogsEnvelope = GetApplicationLogsOperation['responses'][200]['content']['application/json'];
type GetApplicationLogsData = NonNullable<GetApplicationLogsEnvelope['data']>;
type GetApplicationLogsPathParams = GetApplicationLogsOperation['parameters']['path'];
type GetApplicationLogsQuery = NonNullable<GetApplicationLogsOperation['parameters']['query']>;

type ApplicationServicesPath = (typeof APPLICATION_API_PATH)['SERVICES'];
type GetApplicationServicesOperation = paths[ApplicationServicesPath]['get'];
type GetApplicationServicesEnvelope = GetApplicationServicesOperation['responses'][200]['content']['application/json'];
type GetApplicationServicesData = NonNullable<GetApplicationServicesEnvelope['data']>;
type GetApplicationServicesPathParams = GetApplicationServicesOperation['parameters']['path'];

type ApplicationConfigurationPath = (typeof APPLICATION_API_PATH)['CONFIGURATION'];
type GetApplicationConfigurationOperation = paths[ApplicationConfigurationPath]['get'];
type GetApplicationConfigurationEnvelope =
  GetApplicationConfigurationOperation['responses'][200]['content']['application/json'];
type GetApplicationConfigurationData = NonNullable<GetApplicationConfigurationEnvelope['data']>;
type GetApplicationConfigurationPathParams = GetApplicationConfigurationOperation['parameters']['path'];

type ApplicationCreationMethodsPath = (typeof APPLICATION_API_PATH)['CREATION_METHODS'];
type GetApplicationCreationMethodsOperation = paths[ApplicationCreationMethodsPath]['get'];
type GetApplicationCreationMethodsEnvelope =
  GetApplicationCreationMethodsOperation['responses'][200]['content']['application/json'];
type GetApplicationCreationMethodsData = NonNullable<GetApplicationCreationMethodsEnvelope['data']>;

type ApplicationComposeRuntimeTargetsPath = (typeof APPLICATION_API_PATH)['COMPOSE_RUNTIME_TARGETS'];
type GetApplicationComposeRuntimeTargetsOperation = paths[ApplicationComposeRuntimeTargetsPath]['get'];
type GetApplicationComposeRuntimeTargetsData = NonNullable<
  GetApplicationComposeRuntimeTargetsOperation['responses'][200]['content']['application/json']['data']
>;

type ApplicationDiscoveryCandidatesPath = (typeof APPLICATION_API_PATH)['DISCOVERY_CANDIDATES'];
type GetApplicationDiscoveryCandidatesOperation = paths[ApplicationDiscoveryCandidatesPath]['get'];
type GetApplicationDiscoveryCandidatesEnvelope =
  GetApplicationDiscoveryCandidatesOperation['responses'][200]['content']['application/json'];
type GetApplicationDiscoveryCandidatesData = NonNullable<GetApplicationDiscoveryCandidatesEnvelope['data']>;

type ApplicationCreatePath = (typeof APPLICATION_API_PATH)['CREATE'];
type ApplicationCreateOperation = paths[ApplicationCreatePath]['post'];
type ApplicationCreateEnvelope = ApplicationCreateOperation['responses'][201]['content']['application/json'];
type ApplicationCreateData = NonNullable<ApplicationCreateEnvelope['data']>;
type ApplicationCreatePayload = ApplicationCreateOperation['requestBody']['content']['application/json'];

type ApplicationApplicationNameAvailabilityPath = (typeof APPLICATION_API_PATH)['APPLICATION_NAME_AVAILABILITY'];
type ApplicationApplicationNameAvailabilityOperation = paths[ApplicationApplicationNameAvailabilityPath]['post'];
type ApplicationApplicationNameAvailabilityData = NonNullable<
  ApplicationApplicationNameAvailabilityOperation['responses'][200]['content']['application/json']['data']
>;

type ApplicationTemplatesPath = (typeof APPLICATION_API_PATH)['TEMPLATES'];
type GetApplicationTemplatesOperation = paths[ApplicationTemplatesPath]['get'];
type GetApplicationTemplatesData = NonNullable<
  GetApplicationTemplatesOperation['responses'][200]['content']['application/json']['data']
>;
type ApplicationManagedTemplatesPath = (typeof APPLICATION_API_PATH)['TEMPLATES_MANAGE'];
type GetApplicationManagedTemplatesOperation = paths[ApplicationManagedTemplatesPath]['get'];
type GetApplicationManagedTemplatesData = NonNullable<
  GetApplicationManagedTemplatesOperation['responses'][200]['content']['application/json']['data']
>;
type ApplicationUpOperation = paths[(typeof APPLICATION_API_PATH)['UP']]['post'];
type ApplicationUpEnvelope = ApplicationUpOperation['responses'][202]['content']['application/json'];
type ApplicationUpData = NonNullable<ApplicationUpEnvelope['data']>;
type ApplicationUpPathParams = ApplicationUpOperation['parameters']['path'];

type ApplicationStopOperation = paths[(typeof APPLICATION_API_PATH)['STOP']]['post'];
type ApplicationStopEnvelope = ApplicationStopOperation['responses'][202]['content']['application/json'];
type ApplicationStopData = NonNullable<ApplicationStopEnvelope['data']>;
type ApplicationStopPathParams = ApplicationStopOperation['parameters']['path'];

type ApplicationRestartOperation = paths[(typeof APPLICATION_API_PATH)['RESTART']]['post'];
type ApplicationRestartEnvelope = ApplicationRestartOperation['responses'][202]['content']['application/json'];
type ApplicationRestartData = NonNullable<ApplicationRestartEnvelope['data']>;
type ApplicationRestartPathParams = ApplicationRestartOperation['parameters']['path'];

type ApplicationRedeployOperation = paths[(typeof APPLICATION_API_PATH)['REDEPLOY']]['post'];
type ApplicationRedeployEnvelope = ApplicationRedeployOperation['responses'][202]['content']['application/json'];
type ApplicationRedeployData = NonNullable<ApplicationRedeployEnvelope['data']>;
type ApplicationRedeployPathParams = ApplicationRedeployOperation['parameters']['path'];

type ApplicationUnregisterOperation = paths[(typeof APPLICATION_API_PATH)['UNREGISTER']]['post'];
type ApplicationUnregisterEnvelope = ApplicationUnregisterOperation['responses'][200]['content']['application/json'];
type ApplicationUnregisterData = NonNullable<ApplicationUnregisterEnvelope['data']>;
type ApplicationUnregisterPathParams = ApplicationUnregisterOperation['parameters']['path'];

type ApplicationDestroyOperation = paths[(typeof APPLICATION_API_PATH)['DESTROY']]['post'];
type ApplicationDestroyEnvelope = ApplicationDestroyOperation['responses'][200]['content']['application/json'];
type ApplicationDestroyData = NonNullable<ApplicationDestroyEnvelope['data']>;
type ApplicationDestroyPayload = ApplicationDestroyOperation['requestBody']['content']['application/json'];
type ApplicationDestroyPathParams = ApplicationDestroyOperation['parameters']['path'];

type ApplicationBatchActionsOperation = paths[(typeof APPLICATION_API_PATH)['BATCH_ACTIONS']]['post'];
type ApplicationBatchActionsEnvelope =
  ApplicationBatchActionsOperation['responses'][200]['content']['application/json'];
type ApplicationBatchActionsData = NonNullable<ApplicationBatchActionsEnvelope['data']>;
type ApplicationBatchActionsPayload = ApplicationBatchActionsOperation['requestBody']['content']['application/json'];
type ApplicationSavedViewsOperation = paths[(typeof APPLICATION_API_PATH)['SAVED_VIEWS']]['get'];
type ApplicationSavedViewsData = NonNullable<
  ApplicationSavedViewsOperation['responses'][200]['content']['application/json']['data']
>;
type ApplicationCreateSavedViewOperation = paths[(typeof APPLICATION_API_PATH)['SAVED_VIEWS']]['post'];
type ApplicationCreateSavedViewData = NonNullable<
  ApplicationCreateSavedViewOperation['responses'][201]['content']['application/json']['data']
>;
type ApplicationSavedViewOperation = paths[(typeof APPLICATION_API_PATH)['SAVED_VIEW']]['put'];
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
    url: APPLICATION_API_PATH.LIST,
    params: normalizeApplicationListQuery(query),
  }) as Promise<ApplicationListResponseWithLifecycle>;
}

/** 后端未返回视图数组时按空集合处理，避免可选的保存视图阻断项目列表。 */
export async function getApplicationSavedViews(): Promise<ApplicationSavedView[]> {
  const data = await request.get<ApplicationSavedViewsData>({ url: APPLICATION_API_PATH.SAVED_VIEWS });
  return data.items ?? [];
}

export function postApplicationSavedView(payload: ApplicationSavedViewRequest) {
  return request.post<ApplicationCreateSavedViewData>({
    url: APPLICATION_API_PATH.SAVED_VIEWS,
    data: payload,
  }) as Promise<ApplicationSavedView>;
}

export function putApplicationSavedView(viewId: number, payload: ApplicationSavedViewRequest) {
  return request.put<ApplicationUpdateSavedViewData>({
    url: buildApplicationSavedViewApiPath(viewId),
    data: payload,
  }) as Promise<ApplicationSavedView>;
}

export function deleteApplicationSavedView(viewId: number) {
  return request.delete({ url: buildApplicationSavedViewApiPath(viewId) });
}

export function getApplication(applicationId: GetApplicationDetailPathParams['applicationId']) {
  return request.get<GetApplicationDetailData>({
    url: buildApplicationDetailApiPath(applicationId),
  }) as Promise<ApplicationDetailResponseWithLifecycle>;
}

export function getApplicationOverview(applicationId: GetApplicationOverviewPathParams['applicationId']) {
  return request.get<GetApplicationOverviewData>({
    url: buildApplicationOverviewApiPath(applicationId),
  }) as Promise<ApplicationOverviewResponse>;
}

export function getApplicationLogs(
  applicationId: GetApplicationLogsPathParams['applicationId'],
  query?: GetApplicationLogsQuery,
) {
  return request.get<GetApplicationLogsData>({
    url: buildApplicationLogsApiPath(applicationId),
    params: query,
  }) as Promise<ApplicationLogResponse>;
}

export function getApplicationServices(applicationId: GetApplicationServicesPathParams['applicationId']) {
  return request.get<GetApplicationServicesData>({
    url: buildApplicationServicesApiPath(applicationId),
  }) as Promise<ApplicationServicesResponse>;
}

export function postApplicationWorkspaceEntry(applicationId: string, payload: ApplicationWorkspaceEntry) {
  return request.post({ url: buildApplicationFilesEntriesApiPath(applicationId), data: payload });
}

export function postApplicationWorkspaceRename(applicationId: string, payload: ApplicationWorkspaceRenameRequest) {
  return request.post({ url: buildApplicationFilesRenameApiPath(applicationId), data: payload });
}

export function deleteApplicationWorkspaceEntry(applicationId: string, query: { path: string; recursive?: boolean }) {
  return request.delete({ url: buildApplicationFilesEntriesApiPath(applicationId), params: query });
}

export function getApplicationConfiguration(applicationId: GetApplicationConfigurationPathParams['applicationId']) {
  return request.get<GetApplicationConfigurationData>({
    url: buildApplicationConfigurationApiPath(applicationId),
  }) as Promise<ApplicationConfigurationMetadataResponse>;
}

export function getApplicationFiles(id: string, query?: ApplicationWorkspaceFilesQuery) {
  return request.get<ApplicationWorkspaceFilesResponse>({
    url: buildApplicationFilesApiPath(id),
    params: query,
  }) as Promise<ApplicationWorkspaceFilesResponse>;
}

export function getApplicationFileContent(id: string, query: ApplicationWorkspaceFileContentQuery) {
  return request.get<ApplicationWorkspaceFileContentResponse>({
    url: buildApplicationFilesContentApiPath(id),
    params: query,
  }) as Promise<ApplicationWorkspaceFileContentResponse>;
}

export function putApplicationFileContent(
  id: string,
  query: ApplicationWorkspaceFileContentQuery,
  payload: ApplicationWorkspaceFileSaveRequest,
) {
  return request.put<ApplicationWorkspaceFileSaveResponse>({
    url: buildApplicationFilesContentApiPath(id),
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
    url: buildApplicationFilesAnnotationApiPath(id),
    params: query,
    data: payload,
  }) as Promise<ApplicationWorkspaceFileAnnotationResponse>;
}

export function getApplicationCreationMethods() {
  return request.get<GetApplicationCreationMethodsData>({
    url: APPLICATION_API_PATH.CREATION_METHODS,
  }) as Promise<ApplicationCreationMethodCatalogResponse>;
}

export function getApplicationComposeRuntimeTargets() {
  return request.get<GetApplicationComposeRuntimeTargetsData>({
    url: APPLICATION_API_PATH.COMPOSE_RUNTIME_TARGETS,
  }) as Promise<ApplicationComposeRuntimeTargetCatalogResponse>;
}

export function getApplicationDiscoveryCandidates() {
  return request.get<GetApplicationDiscoveryCandidatesData>({
    url: APPLICATION_API_PATH.DISCOVERY_CANDIDATES,
  }) as Promise<ApplicationDiscoveryCandidatesResponse>;
}

export function postApplicationCreate(payload: ApplicationCreateRequest) {
  return postApplicationAction<ApplicationCreateData>(
    APPLICATION_API_PATH.CREATE,
    payload as ApplicationCreatePayload,
  ) as Promise<ApplicationCreateResponse>;
}

export function postApplicationApplicationNameAvailability(payload: ApplicationApplicationNameAvailabilityRequest) {
  return request.post<ApplicationApplicationNameAvailabilityData>({
    url: APPLICATION_API_PATH.APPLICATION_NAME_AVAILABILITY,
    data: payload,
  }) as Promise<ApplicationApplicationNameAvailabilityResponse>;
}

export function getApplicationTemplates() {
  return request.get<GetApplicationTemplatesData>({
    url: APPLICATION_API_PATH.TEMPLATES,
  }) as Promise<ApplicationTemplateCatalogListResponse>;
}

export function getApplicationTemplateCatalog(query: ApplicationTemplateCatalogQuery) {
  return request.get<GetApplicationTemplatesData>({
    url: APPLICATION_API_PATH.TEMPLATES,
    params: query,
  }) as Promise<ApplicationTemplateCatalogListResponse>;
}

export function getPublishedApplicationTemplate(templateId: string): Promise<ApplicationTemplate> {
  return request.get<ApplicationTemplate>({ url: buildApplicationPublishedTemplateApiPath(templateId) });
}

export function getPublishedApplicationTemplateVersion(templateVersionId: string): Promise<ApplicationTemplate> {
  return request.get<ApplicationTemplate>({ url: buildApplicationTemplateVersionApiPath(templateVersionId) });
}

/** 管理目录会返回草稿与归档项，只能由模板管理页面在已授权上下文中消费。 */
export async function getApplicationManagedTemplates(): Promise<ApplicationTemplateListResponse> {
  return request.get<GetApplicationManagedTemplatesData>({ url: APPLICATION_API_PATH.TEMPLATES_MANAGE });
}

export async function getApplicationTemplate(templateId: string): Promise<ApplicationTemplate> {
  return request.get<ApplicationTemplate>({
    url: buildApplicationTemplateApiPath(templateId),
  });
}

export async function postApplicationTemplate(payload: ApplicationTemplateDraftRequest) {
  return request.post<ApplicationTemplate>({
    url: APPLICATION_API_PATH.TEMPLATES,
    data: payload,
  });
}

export async function putApplicationTemplate(templateId: string, payload: ApplicationTemplateDraftRequest) {
  return request.put<ApplicationTemplate>({
    url: buildApplicationTemplateApiPath(templateId),
    data: payload,
  });
}

export async function postApplicationTemplateClone(templateId: string, displayName: string) {
  return request.post<ApplicationTemplate>({
    url: buildApplicationTemplateCloneApiPath(templateId),
    data: { display_name: displayName },
  });
}

export async function postApplicationTemplatePublish(templateId: string) {
  return request.post<ApplicationTemplate>({
    url: buildApplicationTemplatePublishApiPath(templateId),
  });
}

export async function postApplicationTemplateWithdraw(templateId: string) {
  return request.post<ApplicationTemplate>({
    url: buildApplicationTemplateWithdrawApiPath(templateId),
  });
}

export function postApplicationTemplateArchive(templateId: string) {
  return request.post({ url: buildApplicationTemplateArchiveApiPath(templateId) });
}

export function deleteApplicationTemplate(templateId: string) {
  return request.delete({ url: buildApplicationTemplateApiPath(templateId) });
}

function postApplicationAction<T>(url: string, data?: unknown) {
  return request.post<T>({
    url,
    data,
  });
}

export function postApplicationUp(applicationId: ApplicationUpPathParams['applicationId']) {
  return postApplicationAction<ApplicationUpData>(
    buildApplicationUpApiPath(applicationId),
  ) as Promise<ApplicationTaskReceipt>;
}

export function postApplicationStop(applicationId: ApplicationStopPathParams['applicationId']) {
  return postApplicationAction<ApplicationStopData>(
    buildApplicationStopApiPath(applicationId),
  ) as Promise<ApplicationTaskReceipt>;
}

export function postApplicationRestart(applicationId: ApplicationRestartPathParams['applicationId']) {
  return postApplicationAction<ApplicationRestartData>(
    buildApplicationRestartApiPath(applicationId),
  ) as Promise<ApplicationTaskReceipt>;
}

export function postApplicationRedeploy(applicationId: ApplicationRedeployPathParams['applicationId']) {
  return postApplicationAction<ApplicationRedeployData>(
    buildApplicationRedeployApiPath(applicationId),
  ) as Promise<ApplicationTaskReceipt>;
}

export function putApplicationLifecycleConfiguration(
  id: string,
  payload: ApplicationLifecycleConfigurationUpdateRequest,
) {
  return request.put<ApplicationLifecycleConfigurationSavedResponse>({
    url: buildApplicationLifecycleConfigurationApiPath(id),
    data: payload,
  }) as Promise<ApplicationLifecycleConfigurationSavedResponse>;
}

export function postApplicationUnregister(applicationId: ApplicationUnregisterPathParams['applicationId']) {
  return postApplicationAction<ApplicationUnregisterData>(
    buildApplicationUnregisterApiPath(applicationId),
  ) as Promise<ApplicationActionResponse>;
}

export function postApplicationDestroy(
  applicationId: ApplicationDestroyPathParams['applicationId'],
  payload: ApplicationDestroyRequest,
) {
  return postApplicationAction<ApplicationDestroyData>(
    buildApplicationDestroyApiPath(applicationId),
    payload as ApplicationDestroyPayload,
  ) as Promise<ApplicationActionResponse>;
}

export function postApplicationBatchActions(payload: ApplicationBatchActionRequest) {
  return postApplicationAction<ApplicationBatchActionsData>(
    APPLICATION_API_PATH.BATCH_ACTIONS,
    payload as ApplicationBatchActionsPayload,
  ) as Promise<ApplicationBatchActionResponse>;
}
