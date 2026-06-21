import { positionForIndex } from '../text-utils';
import type { I18nGovernanceRule, RuleViolation, SourceFile } from '../types';

const TECHNICAL_FIELD_PATTERNS = [
  'event_type',
  'resource_type',
  'delivery_type',
  'category',
  'source',
  'source_module',
  'level',
  'severity',
  'status',
  'target_type',
  'target_ref',
] as const;

const BLOCKED_LITERAL_PATTERNS = [
  'Scheduled task succeeded',
  'Scheduled task Access log retention cleanup succeeded.',
  'task_succeeded',
  'scheduled_task_run',
  'USER',
] as const;

const ALLOWED_FUNCTIONS = ['resolveNotification', 'notificationSeverityTheme', 'notificationStatusTheme'] as const;

function isNotificationDisplayFile(file: SourceFile) {
  return (
    file.relativePath.startsWith('src/modules/notification/components/') ||
    file.relativePath.startsWith('src/modules/notification/domain/') ||
    file.relativePath.startsWith('src/modules/notification/pages/') ||
    file.relativePath.startsWith('src/modules/notification/shared/')
  );
}

function addViolation(violations: RuleViolation[], file: SourceFile, index: number, excerpt: string, message: string) {
  const position = positionForIndex(file.lineStarts, index);
  violations.push({
    ruleId: 'no-raw-notification-technical-display',
    severity: 'error',
    filePath: file.relativePath,
    line: position.line,
    column: position.column,
    message,
    excerpt,
    suggestion: 'Render notification display text through the notification presentation resolver or i18n formatter.',
  });
}

function isAllowedResolverCall(source: string, index: number) {
  const lineStart = source.lastIndexOf('\n', index) + 1;
  const prefix = source.slice(lineStart, index);
  return ALLOWED_FUNCTIONS.some((name) => prefix.includes(`${name}(`));
}

function collectFieldViolations(file: SourceFile): RuleViolation[] {
  const violations: RuleViolation[] = [];
  const interpolationPattern = /\{\{([\s\S]*?)\}\}/g;
  const interpolations = [...file.source.matchAll(interpolationPattern)];

  for (const field of TECHNICAL_FIELD_PATTERNS) {
    const pattern = new RegExp(`(?:item|row|notificationRow\\([^)]*\\))\\.${field}\\b`, 'g');
    for (const interpolation of interpolations) {
      const expression = interpolation[1] ?? '';
      const interpolationIndex = interpolation.index ?? 0;
      for (const match of expression.matchAll(pattern)) {
        if (match.index === undefined) continue;
        const sourceIndex = interpolationIndex + interpolation[0].indexOf(expression) + match.index;
        if (isAllowedResolverCall(expression, match.index)) continue;

        addViolation(
          violations,
          file,
          sourceIndex,
          match[0],
          `notification technical field ${field} must not be rendered directly`,
        );
      }
    }
  }
  return violations;
}

function collectLiteralViolations(file: SourceFile): RuleViolation[] {
  const violations: RuleViolation[] = [];
  if (file.relativePath.endsWith('/shared/presentation.ts')) return violations;

  for (const literal of BLOCKED_LITERAL_PATTERNS) {
    let index = file.source.indexOf(literal);
    while (index >= 0) {
      addViolation(violations, file, index, literal, `notification display layer must not hard-code ${literal}`);
      index = file.source.indexOf(literal, index + literal.length);
    }
  }
  return violations;
}

export const noRawNotificationTechnicalDisplayRule: I18nGovernanceRule = {
  id: 'no-raw-notification-technical-display',
  description: 'Blocks raw notification technical enums and hard-coded notification copy in display code.',
  defaultSeverity: 'error',
  appliesTo: ['vue', 'ts'],
  check(context) {
    return context.sourceFiles
      .filter(isNotificationDisplayFile)
      .flatMap((file) => [...collectFieldViolations(file), ...collectLiteralViolations(file)]);
  },
};
