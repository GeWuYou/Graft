import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import {
  buildTaskCancelApiPath,
  buildTaskDetailApiPath,
  buildTaskLogsApiPath,
  buildTaskRetryStageApiPath,
  TASK_API_PATH,
} from '../contract/paths';
import type { TaskDetail, TaskListQuery, TaskListResponse, TaskLogResponse } from '../types/task';

type TaskListPath = (typeof TASK_API_PATH)['LIST'];
type ListTasksOperation = paths[TaskListPath]['get'];
type ListTasksData = NonNullable<ListTasksOperation['responses'][200]['content']['application/json']['data']>;
type ListTasksQuery = NonNullable<ListTasksOperation['parameters']['query']>;

type TaskDetailPath = (typeof TASK_API_PATH)['DETAIL'];
type GetTaskOperation = paths[TaskDetailPath]['get'];
type GetTaskData = NonNullable<GetTaskOperation['responses'][200]['content']['application/json']['data']>;
type GetTaskPath = GetTaskOperation['parameters']['path'];

type TaskLogsPath = (typeof TASK_API_PATH)['LOGS'];
type ListTaskLogsOperation = paths[TaskLogsPath]['get'];
type ListTaskLogsData = NonNullable<ListTaskLogsOperation['responses'][200]['content']['application/json']['data']>;
type ListTaskLogsQuery = NonNullable<ListTaskLogsOperation['parameters']['query']>;

type CancelTaskPath = (typeof TASK_API_PATH)['CANCEL'];
type CancelTaskOperation = paths[CancelTaskPath]['post'];
type CancelTaskData = NonNullable<CancelTaskOperation['responses'][200]['content']['application/json']['data']>;

type RetryTaskStagePath = (typeof TASK_API_PATH)['RETRY_STAGE'];
type RetryTaskStageOperation = paths[RetryTaskStagePath]['post'];
type RetryTaskStageData = NonNullable<RetryTaskStageOperation['responses'][202]['content']['application/json']['data']>;

export function getTasks(query?: TaskListQuery) {
  return request.get<ListTasksData>({
    url: TASK_API_PATH.LIST,
    params: query as ListTasksQuery | undefined,
  }) as Promise<TaskListResponse>;
}

export function getTask(taskId: GetTaskPath['taskId']) {
  return request.get<GetTaskData>({ url: buildTaskDetailApiPath(taskId) }) as Promise<TaskDetail>;
}

export function getTaskLogs(taskId: GetTaskPath['taskId'], query?: ListTaskLogsQuery) {
  return request.get<ListTaskLogsData>({
    url: buildTaskLogsApiPath(taskId),
    params: query,
  }) as Promise<TaskLogResponse>;
}

export function cancelTask(taskId: CancelTaskOperation['parameters']['path']['taskId']) {
  return request.post<CancelTaskData>({ url: buildTaskCancelApiPath(taskId) }) as Promise<TaskDetail>;
}

export function retryTaskStage(taskId: number, stageId: number) {
  return request.post<RetryTaskStageData>({
    url: buildTaskRetryStageApiPath(taskId, stageId),
  }) as Promise<TaskDetail>;
}
