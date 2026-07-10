import type { TaskStatus } from '../types/task';

export function taskStatusTheme(status: TaskStatus) {
  if (status === 'success') return 'success';
  if (status === 'failed' || status === 'needs_attention') return 'danger';
  if (status === 'running') return 'primary';
  if (status === 'cancelled') return 'warning';
  return 'default';
}
