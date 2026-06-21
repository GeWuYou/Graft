import { positionForIndex } from '../text-utils';
import type { I18nGovernanceRule, RuleViolation, SourceFile } from '../types';

const RAW_RESULT_FIELD_PATTERNS = [
  'result_summary',
  'error_message',
  'structured.summary',
  'actionResultStructured.summary',
] as const;

function isScheduledTaskDisplayFile(file: SourceFile) {
  return (
    file.relativePath.startsWith('src/modules/scheduled-task/pages/') ||
    file.relativePath.startsWith('src/modules/scheduled-task/components/') ||
    file.relativePath.startsWith('src/modules/scheduled-task/shared/')
  );
}

function addViolation(violations: RuleViolation[], file: SourceFile, index: number, excerpt: string, message: string) {
  const position = positionForIndex(file.lineStarts, index);
  violations.push({
    ruleId: 'no-raw-scheduled-task-result-display',
    severity: 'error',
    filePath: file.relativePath,
    line: position.line,
    column: position.column,
    message,
    excerpt,
    suggestion: 'Render scheduled task run results through a localized presenter backed by result_json metrics.',
  });
}

function collectTemplateViolations(file: SourceFile): RuleViolation[] {
  const violations: RuleViolation[] = [];
  const interpolationPattern = /\{\{([\s\S]*?)\}\}/g;

  for (const interpolation of file.source.matchAll(interpolationPattern)) {
    const expression = interpolation[1] ?? '';
    const interpolationIndex = interpolation.index ?? 0;
    for (const field of RAW_RESULT_FIELD_PATTERNS) {
      const fieldIndex = expression.indexOf(field);
      if (fieldIndex === -1) continue;
      const sourceIndex = interpolationIndex + interpolation[0].indexOf(expression) + fieldIndex;
      addViolation(
        violations,
        file,
        sourceIndex,
        field,
        `scheduled task raw result field ${field} must not be rendered directly`,
      );
    }
  }

  return violations;
}

function collectBoundPropViolations(file: SourceFile): RuleViolation[] {
  const violations: RuleViolation[] = [];
  const boundPropPattern = /(?::[A-Za-z][\w:-]*|v-bind:[A-Za-z][\w:-]*)\s*=\s*(["'])(.*?)\1/gs;

  for (const prop of file.source.matchAll(boundPropPattern)) {
    const expression = prop[2] ?? '';
    const propIndex = prop.index ?? 0;
    for (const field of RAW_RESULT_FIELD_PATTERNS) {
      const fieldIndex = expression.indexOf(field);
      if (fieldIndex === -1) continue;
      const sourceIndex = propIndex + prop[0].indexOf(expression) + fieldIndex;
      addViolation(
        violations,
        file,
        sourceIndex,
        field,
        `scheduled task raw result field ${field} must not be passed directly to visible UI props`,
      );
    }
  }

  return violations;
}

function collectReturnViolations(file: SourceFile): RuleViolation[] {
  const violations: RuleViolation[] = [];
  const returnPattern =
    /\breturn\b(?:(?!\b(?:if|for|while|switch|function|const|let|var|type|interface|export|import)\b)[\s\S])*?\b(result_summary|error_message|structured\.summary|actionResultStructured\.summary)\b/g;

  for (const match of file.source.matchAll(returnPattern)) {
    if (match.index === undefined) continue;
    const field = match[1] ?? match[0];
    addViolation(
      violations,
      file,
      match.index + match[0].indexOf(field),
      field,
      `scheduled task raw result field ${field} must not be returned directly for display`,
    );
  }

  return violations;
}

export const noRawScheduledTaskResultDisplayRule: I18nGovernanceRule = {
  id: 'no-raw-scheduled-task-result-display',
  description: 'Blocks scheduled task raw API result summaries from being rendered as localized UI text.',
  defaultSeverity: 'error',
  appliesTo: ['vue', 'ts'],
  check(context) {
    return context.sourceFiles
      .filter(isScheduledTaskDisplayFile)
      .flatMap((file) => [
        ...collectTemplateViolations(file),
        ...collectBoundPropViolations(file),
        ...collectReturnViolations(file),
      ]);
  },
};
