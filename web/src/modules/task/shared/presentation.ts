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

type TaskFailureAttributionFields = {
  failure_stage?: string | null;
  failure_class?: string | null;
};

// taskFailureAttribution 把有界失败阶段与稳定失败分类渲染为本地化摘要；
// 未归类的字段保持缺席，不伪造默认文案。
export function taskFailureAttribution(failure: TaskFailureAttributionFields, translate: (key: string) => string) {
  const stage = failure.failure_stage?.trim();
  const failureClass = failure.failure_class?.trim();
  return [
    stage ? translate(`task.detail.failureStages.${stage}`) : '',
    failureClass ? translate(`task.detail.failureClasses.${failureClass}`) : '',
  ]
    .filter(Boolean)
    .join(' · ');
}
