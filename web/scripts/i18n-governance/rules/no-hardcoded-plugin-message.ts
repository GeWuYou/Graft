import {
  isTechnicalString,
  normalizeText,
  parseStringLiteral,
  positionForIndex,
  preserveLineStructure,
} from '../text-utils';
import type { I18nGovernanceRule, RuleViolation, SourceFile } from '../types';

function addPluginViolation(violations: RuleViolation[], file: SourceFile, index: number, value: string) {
  const text = normalizeText(value);
  if (!text || isTechnicalString(text)) return;
  const position = positionForIndex(file.lineStarts, index);
  violations.push({
    ruleId: 'no-hardcoded-plugin-message',
    severity: 'error',
    filePath: file.relativePath,
    line: position.line,
    column: position.column,
    message: 'Hard-coded TDesign plugin message',
    excerpt: text,
    suggestion: "Use t('...') before calling MessagePlugin/NotificationPlugin/DialogPlugin.",
  });
}

function collectPluginLiteralCalls(file: SourceFile): RuleViolation[] {
  const violations: RuleViolation[] = [];
  const source = preserveLineStructure(file.source);
  const callPattern = /\b(?:MessagePlugin|NotificationPlugin|DialogPlugin)(?:\.\w+)?\s*\(\s*(['"`])/g;
  for (const match of source.matchAll(callPattern)) {
    const quoteIndex = (match.index ?? 0) + match[0].length - 1;
    const parsed = parseStringLiteral(source, quoteIndex);
    if (parsed && !parsed.hasInterpolation) addPluginViolation(violations, file, quoteIndex, parsed.value);
  }
  return violations;
}

export const noHardcodedPluginMessageRule: I18nGovernanceRule = {
  id: 'no-hardcoded-plugin-message',
  description: 'Blocks hard-coded copy passed directly to TDesign message, notification, and dialog plugins.',
  defaultSeverity: 'error',
  appliesTo: ['ts', 'tsx', 'vue'],
  check(context) {
    return context.sourceFiles.flatMap((file) => collectPluginLiteralCalls(file));
  },
};
