import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import {
  buildScheduledTaskDetailApiPath,
  buildScheduledTaskRunApiPath,
  buildScheduledTaskRunsApiPath,
  SCHEDULED_TASK_API_PATH,
} from '../contract/paths';
import type {
  ScheduledTaskItem,
  ScheduledTaskListResponse,
  ScheduledTaskRunItem,
  ScheduledTaskRunListQuery,
  ScheduledTaskRunListResponse,
} from '../types/scheduled-task';

type ScheduledTaskListPath = (typeof SCHEDULED_TASK_API_PATH)['LIST'];
type GetScheduledTasksOperation = paths[ScheduledTaskListPath]['get'];
type GetScheduledTasksEnvelope = GetScheduledTasksOperation['responses'][200]['content']['application/json'];
type GetScheduledTasksData = NonNullable<GetScheduledTasksEnvelope['data']>;

type ScheduledTaskDetailPath = (typeof SCHEDULED_TASK_API_PATH)['DETAIL'];
type GetScheduledTaskOperation = paths[ScheduledTaskDetailPath]['get'];
type GetScheduledTaskEnvelope = GetScheduledTaskOperation['responses'][200]['content']['application/json'];
type GetScheduledTaskData = NonNullable<GetScheduledTaskEnvelope['data']>;
type GetScheduledTaskPathParams = GetScheduledTaskOperation['parameters']['path'];

type ScheduledTaskRunsPath = (typeof SCHEDULED_TASK_API_PATH)['RUNS'];
type GetScheduledTaskRunsOperation = paths[ScheduledTaskRunsPath]['get'];
type GetScheduledTaskRunsEnvelope = GetScheduledTaskRunsOperation['responses'][200]['content']['application/json'];
type GetScheduledTaskRunsData = NonNullable<GetScheduledTaskRunsEnvelope['data']>;
type GetScheduledTaskRunsPathParams = GetScheduledTaskRunsOperation['parameters']['path'];
type GetScheduledTaskRunsQuery = NonNullable<GetScheduledTaskRunsOperation['parameters']['query']>;

type ScheduledTaskRunPath = (typeof SCHEDULED_TASK_API_PATH)['RUN'];
type PostScheduledTaskRunOperation = paths[ScheduledTaskRunPath]['post'];
type PostScheduledTaskRunEnvelope = PostScheduledTaskRunOperation['responses'][200]['content']['application/json'];
type PostScheduledTaskRunData = NonNullable<PostScheduledTaskRunEnvelope['data']>;
type PostScheduledTaskRunPathParams = PostScheduledTaskRunOperation['parameters']['path'];

export function getScheduledTasks() {
  return request.get<GetScheduledTasksData>({
    url: SCHEDULED_TASK_API_PATH.LIST,
  }) as Promise<ScheduledTaskListResponse>;
}

export function getScheduledTask(taskKey: GetScheduledTaskPathParams['key']) {
  return request.get<GetScheduledTaskData>({
    url: buildScheduledTaskDetailApiPath(taskKey),
  }) as Promise<ScheduledTaskItem>;
}

export function getScheduledTaskRuns(
  taskKey: GetScheduledTaskRunsPathParams['key'],
  query?: ScheduledTaskRunListQuery,
) {
  return request.get<GetScheduledTaskRunsData>({
    url: buildScheduledTaskRunsApiPath(taskKey),
    params: query as GetScheduledTaskRunsQuery | undefined,
  }) as Promise<ScheduledTaskRunListResponse>;
}

export function runScheduledTask(taskKey: PostScheduledTaskRunPathParams['key']) {
  return request.post<PostScheduledTaskRunData>({
    url: buildScheduledTaskRunApiPath(taskKey),
  }) as Promise<ScheduledTaskRunItem>;
}
