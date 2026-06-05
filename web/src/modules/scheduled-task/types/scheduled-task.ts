import type { components } from '@/contracts/openapi/generated/schema';

export type ScheduledTaskLastRun = components['schemas']['scheduled-task-last-run'];
export type ScheduledTaskItem = components['schemas']['scheduled-task-item'];
export type ScheduledTaskListResponse = components['schemas']['scheduled-task-list-response'];
export type ScheduledTaskRunItem = components['schemas']['scheduled-task-run-item'];
export type ScheduledTaskRunListResponse = components['schemas']['scheduled-task-run-list-response'];

export type ScheduledTaskStatus = ScheduledTaskItem['status'];
export type ScheduledTaskRunStatus = ScheduledTaskRunItem['status'];
export type ScheduledTaskRunTriggerType = ScheduledTaskRunItem['trigger_type'];

export type ScheduledTaskRunListQuery = {
  limit?: number;
  offset?: number;
};
