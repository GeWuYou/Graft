export const SCHEDULED_TASK_ROUTE_PATH = {
  LIST: '/server/scheduled-tasks',
} as const;

export const SCHEDULED_TASK_API_PATH = {
  LIST: '/api/scheduled-tasks',
  DETAIL: '/api/scheduled-tasks/{taskKey}',
  ENABLE: '/api/scheduled-tasks/{taskKey}/enable',
  DISABLE: '/api/scheduled-tasks/{taskKey}/disable',
  RUNS: '/api/scheduled-tasks/{taskKey}/runs',
  RUN_DETAIL: '/api/scheduled-tasks/runs/{runId}',
  RUN: '/api/scheduled-tasks/{taskKey}/run',
} as const;

export function buildScheduledTaskDetailApiPath(taskKey: string) {
  return SCHEDULED_TASK_API_PATH.DETAIL.replace('{taskKey}', encodeURIComponent(taskKey));
}

export function buildScheduledTaskEnableApiPath(taskKey: string) {
  return SCHEDULED_TASK_API_PATH.ENABLE.replace('{taskKey}', encodeURIComponent(taskKey));
}

export function buildScheduledTaskDisableApiPath(taskKey: string) {
  return SCHEDULED_TASK_API_PATH.DISABLE.replace('{taskKey}', encodeURIComponent(taskKey));
}

export function buildScheduledTaskRunsApiPath(taskKey: string) {
  return SCHEDULED_TASK_API_PATH.RUNS.replace('{taskKey}', encodeURIComponent(taskKey));
}

export function buildScheduledTaskRunDetailApiPath(runId: number) {
  return SCHEDULED_TASK_API_PATH.RUN_DETAIL.replace('{runId}', String(runId));
}

export function buildScheduledTaskRunApiPath(taskKey: string) {
  return SCHEDULED_TASK_API_PATH.RUN.replace('{taskKey}', encodeURIComponent(taskKey));
}
