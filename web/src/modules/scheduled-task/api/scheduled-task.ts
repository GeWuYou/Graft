import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type {
  CreateScheduledTaskRequest,
  ScheduledTaskActionRequest,
  ScheduledTaskActionResult,
  ScheduledTaskItem,
  ScheduledTaskJobDefinitionListResponse,
  ScheduledTaskJobDefinitionResponse,
  ScheduledTaskListQuery,
  ScheduledTaskListResponse,
  ScheduledTaskRunItem,
  ScheduledTaskRunListQuery,
  ScheduledTaskRunListResponse,
  ScheduledTaskSavedView,
  ScheduledTaskSavedViewRequest,
  UpdateScheduledTaskRequest,
} from '../types/scheduled-task';

type ScheduledTaskListPath = typeof OPENAPI_RUNTIME_PATH.getScheduledTasks;
type GetScheduledTasksOperation = paths[ScheduledTaskListPath]['get'];
type GetScheduledTasksEnvelope = GetScheduledTasksOperation['responses'][200]['content']['application/json'];
type GetScheduledTasksData = NonNullable<GetScheduledTasksEnvelope['data']>;
type GetScheduledTasksQuery = NonNullable<GetScheduledTasksOperation['parameters']['query']>;

type ScheduledTaskJobDefinitionsPath = typeof OPENAPI_RUNTIME_PATH.getScheduledTaskJobDefinitions;
type GetScheduledTaskJobDefinitionsOperation = paths[ScheduledTaskJobDefinitionsPath]['get'];
type GetScheduledTaskJobDefinitionsEnvelope =
  GetScheduledTaskJobDefinitionsOperation['responses'][200]['content']['application/json'];
type GetScheduledTaskJobDefinitionsData = NonNullable<GetScheduledTaskJobDefinitionsEnvelope['data']>;

type ScheduledTaskJobDefinitionDetailPath = typeof OPENAPI_RUNTIME_PATH.getScheduledTaskJobDefinition;
type GetScheduledTaskJobDefinitionOperation = paths[ScheduledTaskJobDefinitionDetailPath]['get'];
type GetScheduledTaskJobDefinitionEnvelope =
  GetScheduledTaskJobDefinitionOperation['responses'][200]['content']['application/json'];
type GetScheduledTaskJobDefinitionData = NonNullable<GetScheduledTaskJobDefinitionEnvelope['data']>;
type GetScheduledTaskJobDefinitionPathParams = GetScheduledTaskJobDefinitionOperation['parameters']['path'];

type ScheduledTaskDetailPath = typeof OPENAPI_RUNTIME_PATH.getScheduledTask;
type GetScheduledTaskOperation = paths[ScheduledTaskDetailPath]['get'];
type GetScheduledTaskEnvelope = GetScheduledTaskOperation['responses'][200]['content']['application/json'];
type GetScheduledTaskData = NonNullable<GetScheduledTaskEnvelope['data']>;
type GetScheduledTaskPathParams = GetScheduledTaskOperation['parameters']['path'];

type PostScheduledTaskOperation = paths[ScheduledTaskListPath]['post'];
type PostScheduledTaskEnvelope = PostScheduledTaskOperation['responses'][200]['content']['application/json'];
type PostScheduledTaskData = NonNullable<PostScheduledTaskEnvelope['data']>;
type PostScheduledTaskBody = PostScheduledTaskOperation['requestBody']['content']['application/json'];

type PutScheduledTaskOperation = paths[ScheduledTaskDetailPath]['put'];
type PutScheduledTaskEnvelope = PutScheduledTaskOperation['responses'][200]['content']['application/json'];
type PutScheduledTaskData = NonNullable<PutScheduledTaskEnvelope['data']>;
type PutScheduledTaskPathParams = PutScheduledTaskOperation['parameters']['path'];
type PutScheduledTaskBody = PutScheduledTaskOperation['requestBody']['content']['application/json'];

type DeleteScheduledTaskOperation = paths[ScheduledTaskDetailPath]['delete'];
type DeleteScheduledTaskPathParams = DeleteScheduledTaskOperation['parameters']['path'];

type ScheduledTaskEnablePath = typeof OPENAPI_RUNTIME_PATH.postScheduledTaskEnable;
type PostScheduledTaskEnableOperation = paths[ScheduledTaskEnablePath]['post'];
type PostScheduledTaskEnableEnvelope =
  PostScheduledTaskEnableOperation['responses'][200]['content']['application/json'];
type PostScheduledTaskEnableData = NonNullable<PostScheduledTaskEnableEnvelope['data']>;
type PostScheduledTaskEnablePathParams = PostScheduledTaskEnableOperation['parameters']['path'];

type ScheduledTaskDisablePath = typeof OPENAPI_RUNTIME_PATH.postScheduledTaskDisable;
type PostScheduledTaskDisableOperation = paths[ScheduledTaskDisablePath]['post'];
type PostScheduledTaskDisableEnvelope =
  PostScheduledTaskDisableOperation['responses'][200]['content']['application/json'];
type PostScheduledTaskDisableData = NonNullable<PostScheduledTaskDisableEnvelope['data']>;
type PostScheduledTaskDisablePathParams = PostScheduledTaskDisableOperation['parameters']['path'];

type ScheduledTaskRunsPath = typeof OPENAPI_RUNTIME_PATH.getScheduledTaskRuns;
type GetScheduledTaskRunsOperation = paths[ScheduledTaskRunsPath]['get'];
type GetScheduledTaskRunsEnvelope = GetScheduledTaskRunsOperation['responses'][200]['content']['application/json'];
type GetScheduledTaskRunsData = NonNullable<GetScheduledTaskRunsEnvelope['data']>;
type GetScheduledTaskRunsPathParams = GetScheduledTaskRunsOperation['parameters']['path'];
type GetScheduledTaskRunsQuery = NonNullable<GetScheduledTaskRunsOperation['parameters']['query']>;

type ScheduledTaskRunDetailPath = typeof OPENAPI_RUNTIME_PATH.getScheduledTaskRun;
type GetScheduledTaskRunOperation = paths[ScheduledTaskRunDetailPath]['get'];
type GetScheduledTaskRunEnvelope = GetScheduledTaskRunOperation['responses'][200]['content']['application/json'];
type GetScheduledTaskRunData = NonNullable<GetScheduledTaskRunEnvelope['data']>;
type GetScheduledTaskRunPathParams = GetScheduledTaskRunOperation['parameters']['path'];

type ScheduledTaskRunPath = typeof OPENAPI_RUNTIME_PATH.postScheduledTaskRun;
type PostScheduledTaskRunOperation = paths[ScheduledTaskRunPath]['post'];
type PostScheduledTaskRunEnvelope = PostScheduledTaskRunOperation['responses'][200]['content']['application/json'];
type PostScheduledTaskRunData = NonNullable<PostScheduledTaskRunEnvelope['data']>;
type PostScheduledTaskRunPathParams = PostScheduledTaskRunOperation['parameters']['path'];

type ScheduledTaskActionPath = typeof OPENAPI_RUNTIME_PATH.postScheduledTaskAction;
type PostScheduledTaskActionOperation = paths[ScheduledTaskActionPath]['post'];
type PostScheduledTaskActionEnvelope =
  PostScheduledTaskActionOperation['responses'][200]['content']['application/json'];
type PostScheduledTaskActionData = NonNullable<PostScheduledTaskActionEnvelope['data']>;
type PostScheduledTaskActionPathParams = PostScheduledTaskActionOperation['parameters']['path'];
type PostScheduledTaskActionBody = NonNullable<
  PostScheduledTaskActionOperation['requestBody']
>['content']['application/json'];

type ScheduledTaskSavedViewsPath = typeof OPENAPI_RUNTIME_PATH.getScheduledTaskSavedViews;
type GetScheduledTaskSavedViewsOperation = paths[ScheduledTaskSavedViewsPath]['get'];
type GetScheduledTaskSavedViewsData = NonNullable<
  GetScheduledTaskSavedViewsOperation['responses'][200]['content']['application/json']['data']
>;
type PostScheduledTaskSavedViewOperation = paths[ScheduledTaskSavedViewsPath]['post'];
type PostScheduledTaskSavedViewData = NonNullable<
  PostScheduledTaskSavedViewOperation['responses'][201]['content']['application/json']['data']
>;
type ScheduledTaskSavedViewPath = typeof OPENAPI_RUNTIME_PATH.putScheduledTaskSavedView;
type PutScheduledTaskSavedViewOperation = paths[ScheduledTaskSavedViewPath]['put'];
type PutScheduledTaskSavedViewData = NonNullable<
  PutScheduledTaskSavedViewOperation['responses'][200]['content']['application/json']['data']
>;
type ScheduledTaskSavedViewBody = PostScheduledTaskSavedViewOperation['requestBody']['content']['application/json'];

export function getScheduledTasks(query?: ScheduledTaskListQuery) {
  return request.get<GetScheduledTasksData>({
    url: OPENAPI_RUNTIME_PATH.getScheduledTasks,
    params: query as GetScheduledTasksQuery | undefined,
  }) as Promise<ScheduledTaskListResponse>;
}

export function getScheduledTaskSavedViews() {
  return request
    .get<GetScheduledTaskSavedViewsData>({ url: OPENAPI_RUNTIME_PATH.getScheduledTaskSavedViews })
    .then((response) => response.items) as Promise<ScheduledTaskSavedView[]>;
}

export function postScheduledTaskSavedView(payload: ScheduledTaskSavedViewRequest) {
  return request.post<PostScheduledTaskSavedViewData>({
    url: OPENAPI_RUNTIME_PATH.postScheduledTaskSavedView,
    data: payload as ScheduledTaskSavedViewBody,
  }) as Promise<ScheduledTaskSavedView>;
}

export function putScheduledTaskSavedView(id: number, payload: ScheduledTaskSavedViewRequest) {
  return request.put<PutScheduledTaskSavedViewData>({
    url: buildOpenApiRuntimePath('putScheduledTaskSavedView', { viewId: id }),
    data: payload as ScheduledTaskSavedViewBody,
  }) as Promise<ScheduledTaskSavedView>;
}

export function deleteScheduledTaskSavedView(id: number) {
  return request.delete<Record<string, never>>({
    url: buildOpenApiRuntimePath('deleteScheduledTaskSavedView', { viewId: id }),
  });
}

export function getScheduledTaskJobDefinitions() {
  return request.get<GetScheduledTaskJobDefinitionsData>({
    url: OPENAPI_RUNTIME_PATH.getScheduledTaskJobDefinitions,
  }) as Promise<ScheduledTaskJobDefinitionListResponse>;
}

export function getScheduledTaskJobDefinition(jobKey: GetScheduledTaskJobDefinitionPathParams['jobKey']) {
  return request.get<GetScheduledTaskJobDefinitionData>({
    url: buildOpenApiRuntimePath('getScheduledTaskJobDefinition', { jobKey }),
  }) as Promise<ScheduledTaskJobDefinitionResponse>;
}

export function getScheduledTask(taskKey: GetScheduledTaskPathParams['taskKey']) {
  return request.get<GetScheduledTaskData>({
    url: buildOpenApiRuntimePath('getScheduledTask', { taskKey }),
  }) as Promise<ScheduledTaskItem>;
}

export function createScheduledTask(payload: CreateScheduledTaskRequest) {
  return request.post<PostScheduledTaskData>({
    url: OPENAPI_RUNTIME_PATH.postScheduledTask,
    data: payload as PostScheduledTaskBody,
  }) as Promise<ScheduledTaskItem>;
}

export function updateScheduledTask(
  taskKey: PutScheduledTaskPathParams['taskKey'],
  payload: UpdateScheduledTaskRequest,
) {
  return request.put<PutScheduledTaskData>({
    url: buildOpenApiRuntimePath('putScheduledTask', { taskKey }),
    data: payload as PutScheduledTaskBody,
  }) as Promise<ScheduledTaskItem>;
}

export function deleteScheduledTask(taskKey: DeleteScheduledTaskPathParams['taskKey']) {
  return request.delete<Record<string, never>>({
    url: buildOpenApiRuntimePath('deleteScheduledTask', { taskKey }),
  });
}

export function enableScheduledTask(taskKey: PostScheduledTaskEnablePathParams['taskKey']) {
  return request.post<PostScheduledTaskEnableData>({
    url: buildOpenApiRuntimePath('postScheduledTaskEnable', { taskKey }),
  }) as Promise<ScheduledTaskItem>;
}

export function disableScheduledTask(taskKey: PostScheduledTaskDisablePathParams['taskKey']) {
  return request.post<PostScheduledTaskDisableData>({
    url: buildOpenApiRuntimePath('postScheduledTaskDisable', { taskKey }),
  }) as Promise<ScheduledTaskItem>;
}

export function getScheduledTaskRuns(
  taskKey: GetScheduledTaskRunsPathParams['taskKey'],
  query?: ScheduledTaskRunListQuery,
) {
  return request.get<GetScheduledTaskRunsData>({
    url: buildOpenApiRuntimePath('getScheduledTaskRuns', { taskKey }),
    params: query as GetScheduledTaskRunsQuery | undefined,
  }) as Promise<ScheduledTaskRunListResponse>;
}

export function getScheduledTaskRun(runId: GetScheduledTaskRunPathParams['runId']) {
  return request.get<GetScheduledTaskRunData>({
    url: buildOpenApiRuntimePath('getScheduledTaskRun', { runId }),
  }) as Promise<ScheduledTaskRunItem>;
}

export function runScheduledTask(taskKey: PostScheduledTaskRunPathParams['taskKey']) {
  return request.post<PostScheduledTaskRunData>({
    url: buildOpenApiRuntimePath('postScheduledTaskRun', { taskKey }),
  }) as Promise<ScheduledTaskRunItem>;
}

export function executeScheduledTaskAction(
  taskKey: PostScheduledTaskActionPathParams['taskKey'],
  actionKey: PostScheduledTaskActionPathParams['actionKey'],
  payload?: ScheduledTaskActionRequest,
) {
  return request.post<PostScheduledTaskActionData>({
    url: buildOpenApiRuntimePath('postScheduledTaskAction', { taskKey, actionKey }),
    data: payload as PostScheduledTaskActionBody | undefined,
  }) as Promise<ScheduledTaskActionResult>;
}
