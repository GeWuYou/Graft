import type { components } from '@/contracts/openapi/generated/schema';

export type ScheduledTaskLastRun = components['schemas']['scheduled-task-last-run'];
export type ScheduledTaskItem = components['schemas']['scheduled-task-item'];
export type ScheduledTaskHTTPConfig = components['schemas']['scheduled-task-http-config'];
export type CreateScheduledTaskRequest = components['schemas']['create-scheduled-task-request'];
export type UpdateScheduledTaskRequest = components['schemas']['update-scheduled-task-request'];
export type ScheduledTaskListResponse = components['schemas']['scheduled-task-list-response'];
export type ScheduledTaskRunItem = components['schemas']['scheduled-task-run-item'];
export type ScheduledTaskRunListResponse = components['schemas']['scheduled-task-run-list-response'];

export type ScheduledTaskStatus = ScheduledTaskItem['status'];
export type ScheduledTaskType = ScheduledTaskItem['task_type'];
export type ScheduledTaskRunStatus = ScheduledTaskRunItem['status'];
export type ScheduledTaskRunTriggerType = ScheduledTaskRunItem['trigger_type'];

export type ScheduledTaskRunListQuery = {
  limit?: number;
  offset?: number;
};
