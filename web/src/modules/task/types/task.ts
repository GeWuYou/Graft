import type { components } from '@/contracts/openapi/generated/schema';
import type { StructuredLogEntry } from '@/shared/observability';

export type TaskStatus = components['schemas']['task-status'];
export type TaskStageStatus = components['schemas']['task-stage-status'];
export type TaskSummary = components['schemas']['task-summary'];
export type TaskDetail = components['schemas']['task-detail'];
export type TaskStage = components['schemas']['task-stage'];
export type TaskLogEntry = components['schemas']['task-log-entry'];
export type TaskReceipt = components['schemas']['task-receipt'];
export type TaskListResponse = components['schemas']['task-list-response'];
export type TaskLogResponse = components['schemas']['task-log-response'];

export type TaskListQuery = {
  limit?: number;
  offset?: number;
  owner_id?: string;
  owner_type?: string;
  status?: TaskStatus;
  type?: string;
};

/**
 * 将任务日志条目转换为结构化日志条目。
 *
 * @param entries - 待转换的任务日志条目
 * @returns 字段名称转换后的结构化日志条目数组
 */
export function taskLogEntriesToStructured(entries: TaskLogEntry[]): StructuredLogEntry[] {
  return entries.map((entry) => ({
    line: entry.line,
    occurredAt: entry.occurred_at,
    stream: entry.stream === 'stderr' ? 'stderr' : 'stdout',
  }));
}
