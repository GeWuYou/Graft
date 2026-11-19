import type { TaskStage, TaskStatus } from '../types/task';
import { resolveTaskFailureMessage } from './failure-copy';

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

// 与服务端结算摘要上限（moduleapi.ExecutionDiagnosticSummaryMaxRunes）保持一致；
// 超长或多行的历史文本不属于受信摘要，渲染时保持缺席。
const taskDiagnosticSummaryMaxRunes = 256;

// taskDiagnosticSummary 渲染受信的服务端脱敏诊断摘要。结算边界已保证摘要单行、
// 有界且不含路径与凭据；这里再做一次形态复核，历史原始错误文本不会被透出。
export function taskDiagnosticSummary(message: string | null | undefined) {
  const summary = message?.trim() ?? '';
  if (
    !summary ||
    summary.includes('\n') ||
    summary.includes('\r') ||
    [...summary].length > taskDiagnosticSummaryMaxRunes
  ) {
    return '';
  }
  return summary;
}

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

// taskFailureDetail 组合稳定失败码文案、受信脱敏摘要与阶段/分类归因；各段独立缺席。
export function taskFailureDetail(
  failure: TaskFailureAttributionFields & { failure_code?: string | null; failure_message?: string | null },
  translate: (key: string) => string,
) {
  return [
    resolveTaskFailureMessage(failure.failure_code, failure.failure_message, translate),
    taskDiagnosticSummary(failure.failure_message),
    taskFailureAttribution(failure, translate),
  ]
    .filter(Boolean)
    .join(' · ');
}
