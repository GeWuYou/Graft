export const TASK_API_PATH = {
  LIST: '/api/tasks',
  DETAIL: '/api/tasks/{taskId}',
  LOGS: '/api/tasks/{taskId}/logs',
  CANCEL: '/api/tasks/{taskId}/cancel',
  RETRY_STAGE: '/api/tasks/{taskId}/stages/{stageId}/retry',
} as const;

/**
 * 将任务路径参数编码为可安全嵌入 URL 的字符串。
 *
 * @param value - 要编码的数值
 * @returns 编码后的路径参数
 */
function encodeTaskPathParam(value: number) {
  return encodeURIComponent(String(value));
}

/**
 * 构建任务详情 API 路径。
 *
 * @param taskId - 任务标识
 * @returns 包含编码任务标识的任务详情路径
 */
export function buildTaskDetailApiPath(taskId: number) {
  return TASK_API_PATH.DETAIL.replace('{taskId}', encodeTaskPathParam(taskId));
}

/**
 * 构建任务日志接口的路径。
 *
 * @param taskId - 任务标识
 * @returns 包含编码任务标识的任务日志路径
 */
export function buildTaskLogsApiPath(taskId: number) {
  return TASK_API_PATH.LOGS.replace('{taskId}', encodeTaskPathParam(taskId));
}

/**
 * 构建取消任务的 API 路径。
 *
 * @param taskId - 要取消的任务标识
 * @returns 包含经过 URL 编码的任务标识的取消路径
 */
export function buildTaskCancelApiPath(taskId: number) {
  return TASK_API_PATH.CANCEL.replace('{taskId}', encodeTaskPathParam(taskId));
}

/**
 * 构建任务阶段重试接口的 API 路径。
 *
 * @param taskId - 任务标识
 * @param stageId - 阶段标识
 * @returns 包含任务标识和阶段标识的重试接口路径
 */
export function buildTaskRetryStageApiPath(taskId: number, stageId: number) {
  return TASK_API_PATH.RETRY_STAGE.replace('{taskId}', encodeTaskPathParam(taskId)).replace(
    '{stageId}',
    encodeTaskPathParam(stageId),
  );
}
