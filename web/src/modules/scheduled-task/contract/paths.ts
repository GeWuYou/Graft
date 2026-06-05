export const SCHEDULED_TASK_ROUTE_PATH = {
  LIST: '/server/scheduled-tasks',
} as const;

export const SCHEDULED_TASK_API_PATH = {
  LIST: '/api/scheduled-tasks',
  DETAIL: '/api/scheduled-tasks/{key}',
  RUNS: '/api/scheduled-tasks/{key}/runs',
  RUN: '/api/scheduled-tasks/{key}:run',
} as const;

export function buildScheduledTaskDetailApiPath(taskKey: string) {
  return SCHEDULED_TASK_API_PATH.DETAIL.replace('{key}', encodeURIComponent(taskKey));
}

export function buildScheduledTaskRunsApiPath(taskKey: string) {
  return SCHEDULED_TASK_API_PATH.RUNS.replace('{key}', encodeURIComponent(taskKey));
}

export function buildScheduledTaskRunApiPath(taskKey: string) {
  return SCHEDULED_TASK_API_PATH.RUN.replace('{key}', encodeURIComponent(taskKey));
}
