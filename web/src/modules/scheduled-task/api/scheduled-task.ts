import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import {
  buildScheduledTaskDetailApiPath,
  buildScheduledTaskDisableApiPath,
  buildScheduledTaskEnableApiPath,
  buildScheduledTaskRunApiPath,
  buildScheduledTaskRunDetailApiPath,
  buildScheduledTaskRunsApiPath,
  SCHEDULED_TASK_API_PATH,
} from '../contract/paths';
import type {
  CreateScheduledTaskRequest,
  ScheduledTaskItem,
  ScheduledTaskJobDefinitionListResponse,
  ScheduledTaskListQuery,
  ScheduledTaskListResponse,
  ScheduledTaskRunItem,
  ScheduledTaskRunListQuery,
  ScheduledTaskRunListResponse,
  UpdateScheduledTaskRequest,
} from '../types/scheduled-task';

type ScheduledTaskListPath = (typeof SCHEDULED_TASK_API_PATH)['LIST'];
type GetScheduledTasksOperation = paths[ScheduledTaskListPath]['get'];
type GetScheduledTasksEnvelope = GetScheduledTasksOperation['responses'][200]['content']['application/json'];
type GetScheduledTasksData = NonNullable<GetScheduledTasksEnvelope['data']>;
type GetScheduledTasksQuery = NonNullable<GetScheduledTasksOperation['parameters']['query']>;

type ScheduledTaskJobsPath = (typeof SCHEDULED_TASK_API_PATH)['JOBS'];
type GetScheduledTaskJobsOperation = paths[ScheduledTaskJobsPath]['get'];
type GetScheduledTaskJobsEnvelope = GetScheduledTaskJobsOperation['responses'][200]['content']['application/json'];
type GetScheduledTaskJobsData = NonNullable<GetScheduledTaskJobsEnvelope['data']>;

type ScheduledTaskDetailPath = (typeof SCHEDULED_TASK_API_PATH)['DETAIL'];
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

type ScheduledTaskEnablePath = (typeof SCHEDULED_TASK_API_PATH)['ENABLE'];
type PostScheduledTaskEnableOperation = paths[ScheduledTaskEnablePath]['post'];
type PostScheduledTaskEnableEnvelope =
  PostScheduledTaskEnableOperation['responses'][200]['content']['application/json'];
type PostScheduledTaskEnableData = NonNullable<PostScheduledTaskEnableEnvelope['data']>;
type PostScheduledTaskEnablePathParams = PostScheduledTaskEnableOperation['parameters']['path'];

type ScheduledTaskDisablePath = (typeof SCHEDULED_TASK_API_PATH)['DISABLE'];
type PostScheduledTaskDisableOperation = paths[ScheduledTaskDisablePath]['post'];
type PostScheduledTaskDisableEnvelope =
  PostScheduledTaskDisableOperation['responses'][200]['content']['application/json'];
type PostScheduledTaskDisableData = NonNullable<PostScheduledTaskDisableEnvelope['data']>;
type PostScheduledTaskDisablePathParams = PostScheduledTaskDisableOperation['parameters']['path'];

type ScheduledTaskRunsPath = (typeof SCHEDULED_TASK_API_PATH)['RUNS'];
type GetScheduledTaskRunsOperation = paths[ScheduledTaskRunsPath]['get'];
type GetScheduledTaskRunsEnvelope = GetScheduledTaskRunsOperation['responses'][200]['content']['application/json'];
type GetScheduledTaskRunsData = NonNullable<GetScheduledTaskRunsEnvelope['data']>;
type GetScheduledTaskRunsPathParams = GetScheduledTaskRunsOperation['parameters']['path'];
type GetScheduledTaskRunsQuery = NonNullable<GetScheduledTaskRunsOperation['parameters']['query']>;

type ScheduledTaskRunDetailPath = (typeof SCHEDULED_TASK_API_PATH)['RUN_DETAIL'];
type GetScheduledTaskRunOperation = paths[ScheduledTaskRunDetailPath]['get'];
type GetScheduledTaskRunEnvelope = GetScheduledTaskRunOperation['responses'][200]['content']['application/json'];
type GetScheduledTaskRunData = NonNullable<GetScheduledTaskRunEnvelope['data']>;
type GetScheduledTaskRunPathParams = GetScheduledTaskRunOperation['parameters']['path'];

type ScheduledTaskRunPath = (typeof SCHEDULED_TASK_API_PATH)['RUN'];
type PostScheduledTaskRunOperation = paths[ScheduledTaskRunPath]['post'];
type PostScheduledTaskRunEnvelope = PostScheduledTaskRunOperation['responses'][200]['content']['application/json'];
type PostScheduledTaskRunData = NonNullable<PostScheduledTaskRunEnvelope['data']>;
type PostScheduledTaskRunPathParams = PostScheduledTaskRunOperation['parameters']['path'];

export function getScheduledTasks(query?: ScheduledTaskListQuery) {
  return request.get<GetScheduledTasksData>({
    url: SCHEDULED_TASK_API_PATH.LIST,
    params: query as GetScheduledTasksQuery | undefined,
  }) as Promise<ScheduledTaskListResponse>;
}

export function getScheduledTaskJobs() {
  return request.get<GetScheduledTaskJobsData>({
    url: SCHEDULED_TASK_API_PATH.JOBS,
  }) as Promise<ScheduledTaskJobDefinitionListResponse>;
}

export function getScheduledTask(taskKey: GetScheduledTaskPathParams['taskKey']) {
  return request.get<GetScheduledTaskData>({
    url: buildScheduledTaskDetailApiPath(taskKey),
  }) as Promise<ScheduledTaskItem>;
}

export function createScheduledTask(payload: CreateScheduledTaskRequest) {
  return request.post<PostScheduledTaskData>({
    url: SCHEDULED_TASK_API_PATH.LIST,
    data: payload as PostScheduledTaskBody,
  }) as Promise<ScheduledTaskItem>;
}

export function updateScheduledTask(
  taskKey: PutScheduledTaskPathParams['taskKey'],
  payload: UpdateScheduledTaskRequest,
) {
  return request.put<PutScheduledTaskData>({
    url: buildScheduledTaskDetailApiPath(taskKey),
    data: payload as PutScheduledTaskBody,
  }) as Promise<ScheduledTaskItem>;
}

export function deleteScheduledTask(taskKey: DeleteScheduledTaskPathParams['taskKey']) {
  return request.delete<Record<string, never>>({
    url: buildScheduledTaskDetailApiPath(taskKey),
  });
}

export function enableScheduledTask(taskKey: PostScheduledTaskEnablePathParams['taskKey']) {
  return request.post<PostScheduledTaskEnableData>({
    url: buildScheduledTaskEnableApiPath(taskKey),
  }) as Promise<ScheduledTaskItem>;
}

export function disableScheduledTask(taskKey: PostScheduledTaskDisablePathParams['taskKey']) {
  return request.post<PostScheduledTaskDisableData>({
    url: buildScheduledTaskDisableApiPath(taskKey),
  }) as Promise<ScheduledTaskItem>;
}

export function getScheduledTaskRuns(
  taskKey: GetScheduledTaskRunsPathParams['taskKey'],
  query?: ScheduledTaskRunListQuery,
) {
  return request.get<GetScheduledTaskRunsData>({
    url: buildScheduledTaskRunsApiPath(taskKey),
    params: query as GetScheduledTaskRunsQuery | undefined,
  }) as Promise<ScheduledTaskRunListResponse>;
}

export function getScheduledTaskRun(runId: GetScheduledTaskRunPathParams['runId']) {
  return request.get<GetScheduledTaskRunData>({
    url: buildScheduledTaskRunDetailApiPath(runId),
  }) as Promise<ScheduledTaskRunItem>;
}

export function runScheduledTask(taskKey: PostScheduledTaskRunPathParams['taskKey']) {
  return request.post<PostScheduledTaskRunData>({
    url: buildScheduledTaskRunApiPath(taskKey),
  }) as Promise<ScheduledTaskRunItem>;
}
