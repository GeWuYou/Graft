import type { TaskStage, TaskStatus } from '../types/task';

export function taskStageExecutionLabel(
  stage: Pick<TaskStage, 'execution_mode' | 'execution_version'>,
  translate: (key: string) => string,
) {
  if (!stage.execution_mode) return '';
  return [translate(`task.executionModes.${stage.execution_mode}`), stage.execution_version]
    .filter(Boolean)
    .join(' · ');
}

export function taskStatusTheme(status: TaskStatus) {
  if (status === 'success') return 'success';
  if (status === 'failed' || status === 'needs_attention') return 'danger';
  if (status === 'running') return 'primary';
  if (status === 'cancelled') return 'warning';
  return 'default';
}
