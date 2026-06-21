import { formatViolations, hasBlockingViolations } from './reporter';
import { rules } from './rules';
import { createScanContext } from './scanner';
import type { RuleViolation } from './types';

function dedupeViolations(violations: RuleViolation[]): RuleViolation[] {
  const seen = new Set<string>();
  const deduped: RuleViolation[] = [];

  for (const violation of violations) {
    const key = [
      violation.ruleId,
      violation.filePath,
      violation.line,
      violation.column ?? 0,
      violation.excerpt ?? violation.message,
    ].join('\0');
    if (seen.has(key)) continue;
    seen.add(key);
    deduped.push(violation);
  }

  return deduped;
}

export function runI18nGovernance() {
  const context = createScanContext();
  const violations = dedupeViolations(rules.flatMap((rule) => rule.check(context)));
  process.stdout.write(formatViolations(violations));
  if (hasBlockingViolations(violations)) {
    process.exitCode = 1;
  }
}

runI18nGovernance();
