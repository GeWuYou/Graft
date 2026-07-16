export const TASK_API_PATH = {
  LIST: '/api/tasks',
  DETAIL: '/api/tasks/{taskId}',
  LOGS: '/api/tasks/{taskId}/logs',
  CANCEL: '/api/tasks/{taskId}/cancel',
  RETRY_STAGE: '/api/tasks/{taskId}/stages/{stageId}/retry',
} as const;

function encodeTaskPathParam(value: number) {
  return encodeURIComponent(String(value));
}

export function buildTaskDetailApiPath(taskId: number) {
  return TASK_API_PATH.DETAIL.replace('{taskId}', encodeTaskPathParam(taskId));
}

export function buildTaskLogsApiPath(taskId: number) {
  return TASK_API_PATH.LOGS.replace('{taskId}', encodeTaskPathParam(taskId));
}

export function buildTaskCancelApiPath(taskId: number) {
  return TASK_API_PATH.CANCEL.replace('{taskId}', encodeTaskPathParam(taskId));
}

export function buildTaskRetryStageApiPath(taskId: number, stageId: number) {
  return TASK_API_PATH.RETRY_STAGE.replace('{taskId}', encodeTaskPathParam(taskId)).replace(
    '{stageId}',
    encodeTaskPathParam(stageId),
  );
}
