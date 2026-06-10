// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import {
  isLikelyI18nKey,
  isTechnicalString,
  normalizeText,
  parseStringLiteral,
  positionForIndex,
  preserveLineStructure,
} from '../text-utils';
import type { I18nGovernanceRule, RuleViolation, SourceFile } from '../types';

type SchemaFallbackFinding = {
  file: string;
  line: number;
  message: string;
};

function collectServerSystemConfigSchemaFallbackViolations(file: SourceFile): RuleViolation[] {
  const findings: SchemaFallbackFinding[] = [];
  const source = preserveLineStructure(file.source);
  let index = 0;

  while (index < source.length) {
    const quote = source[index];
    if (quote !== '"' && quote !== "'" && quote !== '`') {
      index += 1;
      continue;
    }

    const parsed = parseStringLiteral(source, index);
    if (!parsed) {
      index += 1;
      continue;
    }

    const schema = parsePotentialSystemConfigSchema(parsed.value);
    if (schema) {
      const line = positionForIndex(file.lineStarts, index).line;
      collectSchemaNodeFallbackFindings(schema, file.relativePath, line, 'schema', findings);
    }

    index = parsed.endIndex;
  }

  return findings.map((finding) => ({
    ruleId: 'no-system-config-schema-fallback',
    severity: 'error',
    filePath: finding.file,
    line: finding.line,
    message: finding.message,
    suggestion:
      'Add matching x-i18n titleKey, descriptionKey, or placeholderKey fields and define those keys in web locale catalogs.',
  }));
}

function parsePotentialSystemConfigSchema(value: string): Record<string, unknown> | null {
  const trimmed = value.trim();

  if (
    !trimmed.startsWith('{') ||
    !trimmed.endsWith('}') ||
    !/"(?:type|properties)"\s*:/.test(trimmed) ||
    !/"(?:title|description|placeholder)"\s*:/.test(trimmed)
  ) {
    return null;
  }

  try {
    const parsed: unknown = JSON.parse(trimmed);
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null;
    return parsed as Record<string, unknown>;
  } catch {
    return null;
  }
}

function collectSchemaNodeFallbackFindings(
  node: unknown,
  file: string,
  line: number,
  path: string,
  findings: SchemaFallbackFinding[],
): void {
  if (!node || typeof node !== 'object') return;

  if (Array.isArray(node)) {
    node.forEach((child, index) => collectSchemaNodeFallbackFindings(child, file, line, `${path}[${index}]`, findings));
    return;
  }

  const objectNode = node as Record<string, unknown>;
  const i18nExtension = objectNode['x-i18n'];
  const i18nObject =
    i18nExtension && typeof i18nExtension === 'object' && !Array.isArray(i18nExtension)
      ? (i18nExtension as Record<string, unknown>)
      : {};

  for (const field of ['title', 'description', 'placeholder'] as const) {
    const value = objectNode[field];
    if (typeof value !== 'string') continue;

    const normalized = normalizeText(value);
    if (normalized.length === 0 || isTechnicalString(normalized)) continue;

    const keyField = `${field}Key`;
    const keyValue = i18nObject[keyField];
    if (typeof keyValue === 'string' && isLikelyI18nKey(keyValue)) continue;

    findings.push({
      file,
      line,
      message: `system config schema ${path}.${field} has visible fallback "${normalized}" without x-i18n.${keyField}`,
    });
  }

  for (const [key, child] of Object.entries(objectNode)) {
    if (key === 'x-i18n') continue;
    collectSchemaNodeFallbackFindings(child, file, line, `${path}.${key}`, findings);
  }
}

export const noSystemConfigSchemaFallbackRule: I18nGovernanceRule = {
  id: 'no-system-config-schema-fallback',
  description: 'Blocks server system config schema visible fallback copy without x-i18n key fields.',
  defaultSeverity: 'error',
  appliesTo: ['go'],
  check(context) {
    return context.serverFiles.flatMap((file) => collectServerSystemConfigSchemaFallbackViolations(file));
  },
};
