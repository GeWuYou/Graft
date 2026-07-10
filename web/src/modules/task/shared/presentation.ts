import type { TaskStatus } from '../types/task';

/**
 * 将任务状态映射为对应的主题键。
 *
 * @param status - 任务状态
 * @returns 与任务状态对应的主题键
 */
export function taskStatusTheme(status: TaskStatus) {
  if (status === 'success') return 'success';
  if (status === 'failed' || status === 'needs_attention') return 'danger';
  if (status === 'running') return 'primary';
  if (status === 'cancelled') return 'warning';
  return 'default';
}
