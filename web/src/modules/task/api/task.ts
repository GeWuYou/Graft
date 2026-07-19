import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type { TaskDetail, TaskListQuery, TaskListResponse, TaskLogResponse } from '../types/task';

type TaskListPath = typeof OPENAPI_RUNTIME_PATH.listTasks;
type ListTasksOperation = paths[TaskListPath]['get'];
type ListTasksData = NonNullable<ListTasksOperation['responses'][200]['content']['application/json']['data']>;
type ListTasksQuery = NonNullable<ListTasksOperation['parameters']['query']>;

type TaskDetailPath = typeof OPENAPI_RUNTIME_PATH.getTask;
type GetTaskOperation = paths[TaskDetailPath]['get'];
type GetTaskData = NonNullable<GetTaskOperation['responses'][200]['content']['application/json']['data']>;
type GetTaskPath = GetTaskOperation['parameters']['path'];

type TaskLogsPath = typeof OPENAPI_RUNTIME_PATH.listTaskLogs;
type ListTaskLogsOperation = paths[TaskLogsPath]['get'];
type ListTaskLogsData = NonNullable<ListTaskLogsOperation['responses'][200]['content']['application/json']['data']>;
type ListTaskLogsQuery = NonNullable<ListTaskLogsOperation['parameters']['query']>;

type CancelTaskPath = typeof OPENAPI_RUNTIME_PATH.cancelTask;
type CancelTaskOperation = paths[CancelTaskPath]['post'];
type CancelTaskData = NonNullable<CancelTaskOperation['responses'][200]['content']['application/json']['data']>;

type RetryTaskStagePath = typeof OPENAPI_RUNTIME_PATH.retryTaskStage;
type RetryTaskStageOperation = paths[RetryTaskStagePath]['post'];
type RetryTaskStageData = NonNullable<RetryTaskStageOperation['responses'][202]['content']['application/json']['data']>;

export function getTasks(query?: TaskListQuery) {
  return request.get<ListTasksData>({
    url: OPENAPI_RUNTIME_PATH.listTasks,
    params: query as ListTasksQuery | undefined,
  }) as Promise<TaskListResponse>;
}

export function getTask(taskId: GetTaskPath['taskId']) {
  return request.get<GetTaskData>({ url: buildOpenApiRuntimePath('getTask', { taskId }) }) as Promise<TaskDetail>;
}

export function getTaskLogs(taskId: GetTaskPath['taskId'], query?: ListTaskLogsQuery) {
  return request.get<ListTaskLogsData>({
    url: buildOpenApiRuntimePath('listTaskLogs', { taskId }),
    params: query,
  }) as Promise<TaskLogResponse>;
}

export function cancelTask(taskId: CancelTaskOperation['parameters']['path']['taskId']) {
  return request.post<CancelTaskData>({
    url: buildOpenApiRuntimePath('cancelTask', { taskId }),
  }) as Promise<TaskDetail>;
}

export function retryTaskStage(taskId: number, stageId: number) {
  return request.post<RetryTaskStageData>({
    url: buildOpenApiRuntimePath('retryTaskStage', { taskId, stageId }),
  }) as Promise<TaskDetail>;
}
