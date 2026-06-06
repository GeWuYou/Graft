import type { components } from '@/contracts/openapi/generated/schema';

export type ScheduledTaskLastRun = components['schemas']['scheduled-task-last-run'];
export type ScheduledTaskItem = components['schemas']['scheduled-task-item'];
export type ScheduledTaskJobDefinitionItem = components['schemas']['scheduled-task-job-definition-item'];
export type ScheduledTaskJobDefinitionListResponse =
  components['schemas']['scheduled-task-job-definition-list-response'];
export type ScheduledTaskJobDefinitionResponse = ScheduledTaskJobDefinitionItem;
export type CreateScheduledTaskRequest = components['schemas']['create-scheduled-task-request'];
export type UpdateScheduledTaskRequest = components['schemas']['update-scheduled-task-request'];
export type ScheduledTaskListResponse = components['schemas']['scheduled-task-list-response'];
export type ScheduledTaskRunItem = components['schemas']['scheduled-task-run-item'];
export type ScheduledTaskRunListResponse = components['schemas']['scheduled-task-run-list-response'];

export type ScheduledTaskStatus = ScheduledTaskItem['status'];
export type ScheduledTaskJobKey = ScheduledTaskItem['job_key'];
export type ScheduledTaskRunStatus = ScheduledTaskRunItem['status'];
export type ScheduledTaskRunTriggerType = ScheduledTaskRunItem['trigger_type'];

export type ScheduledTaskListQuery = {
  limit?: number;
  offset?: number;
};

export type ScheduledTaskRunListQuery = {
  limit?: number;
  offset?: number;
};
