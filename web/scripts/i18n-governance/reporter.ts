import type { RuleViolation, Severity } from './types';

const MAX_DISPLAYED = 50;

function severityRank(severity: Severity) {
  return severity === 'error' ? 0 : 1;
}

export function sortViolations(violations: RuleViolation[]): RuleViolation[] {
  return [...violations].sort((left, right) => {
    const severityDelta = severityRank(left.severity) - severityRank(right.severity);
    if (severityDelta !== 0) return severityDelta;
    if (left.filePath !== right.filePath) return left.filePath.localeCompare(right.filePath);
    if (left.line !== right.line) return left.line - right.line;
    if ((left.column ?? 0) !== (right.column ?? 0)) return (left.column ?? 0) - (right.column ?? 0);
    return left.ruleId.localeCompare(right.ruleId);
  });
}

export function formatViolations(violations: RuleViolation[]): string {
  if (violations.length === 0) {
    return 'No hard-coded UI text or locale governance issues found.\n';
  }

  const sorted = sortViolations(violations);
  const errors = sorted.filter((violation) => violation.severity === 'error').length;
  const warnings = sorted.length - errors;
  const output: string[] = [`Found ${errors} i18n error(s) and ${warnings} warning(s):`];

  for (const violation of sorted.slice(0, MAX_DISPLAYED)) {
    const column = violation.column ? `:${violation.column}` : '';
    output.push(
      `- [${violation.severity}] ${violation.filePath}:${violation.line}${column} ${violation.ruleId}: ${violation.message}`,
    );
    if (violation.excerpt) output.push(`  excerpt: ${violation.excerpt}`);
    if (violation.suggestion) output.push(`  suggestion: ${violation.suggestion}`);
  }

  if (sorted.length > MAX_DISPLAYED) {
    output.push(`... ${sorted.length - MAX_DISPLAYED} more issue(s) hidden. Fix the first ${MAX_DISPLAYED} and rerun.`);
  }

  output.push(`Summary: ${sorted.length} total, ${errors} error(s), ${warnings} warning(s).`);
  return `${output.join('\n')}\n`;
}

export function hasBlockingViolations(violations: RuleViolation[]): boolean {
  return violations.some((violation) => violation.severity === 'error');
}
