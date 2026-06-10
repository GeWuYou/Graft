// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { positionForIndex, preserveLineStructure } from '../text-utils';
import type { I18nGovernanceRule, RuleViolation, SourceFile } from '../types';

const UNSAFE_INTL_DATETIME_PATTERN =
  /\bnew\s+Intl\.DateTimeFormat\s*\(\s*(?:undefined|void\s+0)\b|\bIntl\.DateTimeFormat\s*\(\s*(?:undefined|void\s+0)\b/g;
const DATE_TIME_LOCALE_METHOD_PATTERN =
  /(?:\bnew\s+Date\s*\([^)]*\)|\b[A-Za-z_$][\w$]*(?:\.[A-Za-z_$][\w$]*)*)\s*\.\s*(toLocaleDateString|toLocaleTimeString)\s*\(\s*(?:\)|(?:undefined|void\s+0)\b)/g;
const DATE_LOCALE_STRING_PATTERN = /\bnew\s+Date\s*\([^)]*\)\s*\.\s*toLocaleString\s*\(\s*\)/g;
const EXPLICIT_UNDEFINED_LOCALE_STRING_PATTERN =
  /(?:\bnew\s+Date\s*\([^)]*\)|\b[A-Za-z_$][\w$]*(?:\.[A-Za-z_$][\w$]*)*)\s*\.\s*toLocaleString\s*\(\s*(?:undefined|void\s+0)\b/g;

function collectDatetimeViolations(file: SourceFile): RuleViolation[] {
  const violations: RuleViolation[] = [];
  const source = preserveLineStructure(file.source);

  for (const match of source.matchAll(UNSAFE_INTL_DATETIME_PATTERN)) {
    const position = positionForIndex(file.lineStarts, match.index ?? 0);
    violations.push({
      ruleId: 'no-unsafe-datetime-locale',
      severity: 'error',
      filePath: file.relativePath,
      line: position.line,
      column: position.column,
      message: 'visible datetime formatting must pass the active locale instead of undefined',
      suggestion: 'Pass the active vue-i18n locale or use a locale-aware shared datetime formatter.',
    });
  }

  for (const match of source.matchAll(DATE_TIME_LOCALE_METHOD_PATTERN)) {
    const position = positionForIndex(file.lineStarts, match.index ?? 0);
    violations.push({
      ruleId: 'no-unsafe-datetime-locale',
      severity: 'error',
      filePath: file.relativePath,
      line: position.line,
      column: position.column,
      message: `${match[1]} must pass the active locale or use a locale-aware shared formatter`,
      suggestion: 'Pass the active vue-i18n locale or use a locale-aware shared datetime formatter.',
    });
  }

  for (const match of source.matchAll(DATE_LOCALE_STRING_PATTERN)) {
    const position = positionForIndex(file.lineStarts, match.index ?? 0);
    violations.push({
      ruleId: 'no-unsafe-datetime-locale',
      severity: 'error',
      filePath: file.relativePath,
      line: position.line,
      column: position.column,
      message: 'toLocaleString must pass the active locale or use a locale-aware shared formatter',
      suggestion: 'Pass the active vue-i18n locale or use a locale-aware shared datetime formatter.',
    });
  }

  for (const match of source.matchAll(EXPLICIT_UNDEFINED_LOCALE_STRING_PATTERN)) {
    const position = positionForIndex(file.lineStarts, match.index ?? 0);
    violations.push({
      ruleId: 'no-unsafe-datetime-locale',
      severity: 'error',
      filePath: file.relativePath,
      line: position.line,
      column: position.column,
      message: 'toLocaleString must pass the active locale or use a locale-aware shared formatter',
      suggestion: 'Pass the active vue-i18n locale or use a locale-aware shared datetime formatter.',
    });
  }

  return violations;
}

export const noUnsafeDatetimeLocaleRule: I18nGovernanceRule = {
  id: 'no-unsafe-datetime-locale',
  description: 'Blocks visible datetime formatting that depends on the host runtime locale.',
  defaultSeverity: 'error',
  appliesTo: ['vue', 'ts', 'tsx'],
  check(context) {
    return context.sourceFiles.flatMap((file) => collectDatetimeViolations(file));
  },
};
