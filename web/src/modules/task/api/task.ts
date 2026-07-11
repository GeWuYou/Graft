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

export type TaskOwnerReference = Readonly<{
  ownerId: string;
  ownerType: string;
}>;

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

/**
 * 获取任务列表。
 *
 * @param query - 任务列表的筛选和分页参数
 * @returns 任务列表数据
 */
export function getTasks(query?: TaskListQuery) {
  return request.get<ListTasksData>({
    url: TASK_API_PATH.LIST,
    params: query as ListTasksQuery | undefined,
  }) as Promise<TaskListResponse>;
}

/**
 * 获取指定资源拥有的最新任务。
 *
 * @param owner - 任务拥有者的标识及类型
 * @returns 最新任务；没有匹配任务时返回 `null`
 */
export async function getLatestTaskForOwner(owner: TaskOwnerReference) {
  const response = await getTasks({ limit: 1, owner_id: owner.ownerId, owner_type: owner.ownerType });
  return response.items[0] ?? null;
}

/**
 * 获取指定任务的详细信息。
 *
 * @param taskId - 要查询的任务标识
 * @returns 指定任务的详细信息
 */
export function getTask(taskId: GetTaskPath['taskId']) {
  return request.get<GetTaskData>({ url: buildTaskDetailApiPath(taskId) }) as Promise<TaskDetail>;
}

/**
 * 获取指定任务的日志。
 *
 * @param taskId - 任务标识
 * @param query - 日志列表查询参数
 * @returns 任务日志数据
 */
export function getTaskLogs(taskId: GetTaskPath['taskId'], query?: ListTaskLogsQuery) {
  return request.get<ListTaskLogsData>({
    url: buildTaskLogsApiPath(taskId),
    params: query,
  }) as Promise<TaskLogResponse>;
}

/**
 * 取消指定任务。
 *
 * @param taskId - 要取消的任务标识
 * @returns 取消后的任务详情
 */
export function cancelTask(taskId: CancelTaskOperation['parameters']['path']['taskId']) {
  return request.post<CancelTaskData>({ url: buildTaskCancelApiPath(taskId) }) as Promise<TaskDetail>;
}

/**
 * 重试任务的指定阶段。
 *
 * @param taskId - 任务标识
 * @param stageId - 要重试的阶段标识
 * @returns 更新后的任务详情
 */
export function retryTaskStage(taskId: number, stageId: number) {
  return request.post<RetryTaskStageData>({
    url: buildTaskRetryStageApiPath(taskId, stageId),
  }) as Promise<TaskDetail>;
}
