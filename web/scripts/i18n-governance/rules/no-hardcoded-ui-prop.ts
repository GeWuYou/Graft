// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import * as ts from 'typescript';

import { KEY_FIELDS, UI_COPY_FIELDS } from '../config';
import {
  hasCjk,
  hasVisibleLetters,
  isTechnicalString,
  normalizeText,
  parseStringLiteral,
  positionForIndex,
  preserveLineStructure,
} from '../text-utils';
import type { I18nGovernanceRule, RuleViolation, SourceFile } from '../types';

const LOCALIZED_TITLE_FIELDS = new Set(['semanticTitle', 'breadcrumbTitle', 'tabTitle']);
const LOCALE_LITERAL_FIELDS = new Set(['en', 'enUS', 'en-US', 'zhCN', 'zh_CN', 'zh-CN']);
const UI_COPY_LOGICAL_OPERATORS = new Set<ts.SyntaxKind>([
  ts.SyntaxKind.AmpersandAmpersandToken,
  ts.SyntaxKind.BarBarToken,
  ts.SyntaxKind.QuestionQuestionToken,
]);

type ExpressionLiteral = {
  index: number;
  value: string;
  hasInterpolation: boolean;
};

function normalizeAttributeName(name: string): string {
  return name
    .replace(/^:/, '')
    .replace(/^v-bind:/, '')
    .replace(/-([a-z])/g, (_, letter: string) => letter.toUpperCase());
}

function isBoundAttribute(name: string): boolean {
  return name.startsWith(':') || name.startsWith('v-bind:');
}

function addViolation(violations: RuleViolation[], file: SourceFile, index: number, field: string, value: string) {
  const text = normalizeText(value);
  if (!text || text === field || isTechnicalString(text)) return;
  const position = positionForIndex(file.lineStarts, index);
  violations.push({
    ruleId: 'no-hardcoded-ui-prop',
    severity: 'error',
    filePath: file.relativePath,
    line: position.line,
    column: position.column,
    message: `Hard-coded UI copy in ${field}`,
    excerpt: text,
    suggestion: LOCALIZED_TITLE_FIELDS.has(field.split('.')[0] ?? field)
      ? 'Move route title copy to locale catalogs and reference it through the route/menu title key boundary.'
      : `Use t('...') or provide a ${field}Key plus localized fallback.`,
  });
}

function templateExpressionValue(node: ts.TemplateExpression): string {
  return [
    node.head.text,
    ...node.templateSpans.map((span) => `\${${span.expression.getText()}}${span.literal.text}`),
  ].join('');
}

function collectExpressionLiterals(expression: string): ExpressionLiteral[] {
  const prefix = 'const __i18nExpression = (';
  const source = `${prefix}${expression});`;
  const sourceFile = ts.createSourceFile(
    'i18n-governance-expression.ts',
    source,
    ts.ScriptTarget.Latest,
    true,
    ts.ScriptKind.TSX,
  );
  const literals: ExpressionLiteral[] = [];

  function relativeIndex(node: ts.Node): number {
    return Math.max(0, node.getStart(sourceFile) - prefix.length);
  }

  function inspectExpression(node: ts.Expression): void {
    if (ts.isParenthesizedExpression(node) || ts.isAsExpression(node) || ts.isSatisfiesExpression(node)) {
      inspectExpression(node.expression);
      return;
    }

    if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
      literals.push({ index: relativeIndex(node), value: node.text, hasInterpolation: false });
      return;
    }

    if (ts.isTemplateExpression(node)) {
      const value = templateExpressionValue(node);
      if (hasVisibleLetters(value)) {
        literals.push({ index: relativeIndex(node), value, hasInterpolation: true });
      }
      return;
    }

    if (ts.isConditionalExpression(node)) {
      inspectExpression(node.whenTrue);
      inspectExpression(node.whenFalse);
      return;
    }

    if (ts.isBinaryExpression(node) && UI_COPY_LOGICAL_OPERATORS.has(node.operatorToken.kind)) {
      inspectExpression(node.left);
      inspectExpression(node.right);
    }
  }

  for (const statement of sourceFile.statements) {
    if (!ts.isVariableStatement(statement)) continue;
    for (const declaration of statement.declarationList.declarations) {
      if (declaration.initializer) inspectExpression(declaration.initializer);
    }
  }

  return literals;
}

function collectObjectFieldViolations(file: SourceFile): RuleViolation[] {
  const violations: RuleViolation[] = [];
  const source = preserveLineStructure(file.source);
  const names = [...UI_COPY_FIELDS].join('|');
  const pattern = new RegExp(`(^|[,({]\\s*)(['"]?)(${names})\\2\\s*[:=]\\s*(['"\`])`, 'gm');

  for (const match of source.matchAll(pattern)) {
    const field = match[3];
    if (!field || KEY_FIELDS.has(field)) continue;
    const quoteIndex = (match.index ?? 0) + match[0].length - 1;
    const parsed = parseStringLiteral(source, quoteIndex);
    if (!parsed) continue;
    if (parsed.hasInterpolation && !hasCjk(parsed.value)) continue;
    addViolation(violations, file, quoteIndex, field, parsed.value);
  }

  return violations;
}

function collectConditionalFieldViolations(file: SourceFile): RuleViolation[] {
  const violations: RuleViolation[] = [];
  const source = preserveLineStructure(file.source);
  const names = [...UI_COPY_FIELDS].join('|');
  const pattern = new RegExp(`\\b(${names})\\s*:\\s*[^\\n?:]+\\?\\s*(['"\`])`, 'gm');

  for (const match of source.matchAll(pattern)) {
    const field = match[1];
    if (!field || KEY_FIELDS.has(field)) continue;
    const quoteIndex = (match.index ?? 0) + match[0].lastIndexOf(match[2] ?? "'");
    const first = parseStringLiteral(source, quoteIndex);
    if (first) addViolation(violations, file, quoteIndex, field, first.value);
    if (first) {
      const tail = source.slice(first.endIndex, Math.min(first.endIndex + 120, source.length));
      const secondMatch = tail.match(/:\s*(['"`])/);
      if (secondMatch?.index !== undefined) {
        const secondQuote = first.endIndex + secondMatch.index + secondMatch[0].length - 1;
        const second = parseStringLiteral(source, secondQuote);
        if (second) addViolation(violations, file, secondQuote, field, second.value);
      }
    }
  }

  return violations;
}

function collectLogicalFallbackViolations(file: SourceFile): RuleViolation[] {
  const violations: RuleViolation[] = [];
  const source = preserveLineStructure(file.source);
  const names = [...UI_COPY_FIELDS].join('|');
  const pattern = new RegExp(`\\b(${names})\\s*[:=]\\s*[^\\n|]+\\|\\|\\s*(['"\`])`, 'gm');

  for (const match of source.matchAll(pattern)) {
    const field = match[1];
    if (!field || KEY_FIELDS.has(field)) continue;
    const quoteIndex = (match.index ?? 0) + match[0].length - 1;
    const parsed = parseStringLiteral(source, quoteIndex);
    if (!parsed) continue;
    addViolation(violations, file, quoteIndex, field, parsed.value);
  }

  return violations;
}

function collectLocalizedTitleObjectViolations(file: SourceFile): RuleViolation[] {
  const violations: RuleViolation[] = [];
  const source = preserveLineStructure(file.source);
  const titlePattern = /\b(semanticTitle|breadcrumbTitle|tabTitle)\s*:\s*{/g;

  for (const titleMatch of source.matchAll(titlePattern)) {
    const field = titleMatch[1];
    const objectStart = (titleMatch.index ?? 0) + titleMatch[0].length;
    const objectEnd = source.indexOf('}', objectStart);
    if (objectEnd === -1 || objectEnd - objectStart > 500) continue;
    const objectSource = source.slice(objectStart, objectEnd);
    for (const localeMatch of objectSource.matchAll(
      /(?:['"](zh-CN|en-US)['"]|\[\s*LOCALE\.(ZH_CN|EN_US)\s*\])\s*:\s*(['"`])/g,
    )) {
      const locale = localeMatch[1] ?? (localeMatch[2] === 'ZH_CN' ? 'zh-CN' : 'en-US');
      const quoteIndex = objectStart + (localeMatch.index ?? 0) + localeMatch[0].length - 1;
      const parsed = parseStringLiteral(source, quoteIndex);
      if (!parsed) continue;
      addViolation(violations, file, quoteIndex, `${field}.${locale}`, parsed.value);
    }
  }

  return violations;
}

function collectLocaleLiteralFieldViolations(file: SourceFile): RuleViolation[] {
  const violations: RuleViolation[] = [];
  const source = preserveLineStructure(file.source);
  const pattern = /(^|[,({]\s*)(['"]?)(en|enUS|en-US|zhCN|zh_CN|zh-CN)\2\s*:\s*(['"`])/gm;

  for (const match of source.matchAll(pattern)) {
    const field = match[3];
    if (!field || !LOCALE_LITERAL_FIELDS.has(field)) continue;
    const quoteIndex = (match.index ?? 0) + match[0].length - 1;
    const parsed = parseStringLiteral(source, quoteIndex);
    if (!parsed) continue;
    addViolation(violations, file, quoteIndex, field, parsed.value);
  }

  return violations;
}

function collectComputedLocaleLiteralFieldViolations(file: SourceFile): RuleViolation[] {
  const violations: RuleViolation[] = [];
  const source = preserveLineStructure(file.source);

  for (const match of source.matchAll(/\[\s*LOCALE\.(ZH_CN|EN_US)\s*\]\s*:\s*(['"`])/g)) {
    const field = match[1] === 'ZH_CN' ? 'zh-CN' : 'en-US';
    const quoteIndex = (match.index ?? 0) + match[0].length - 1;
    const parsed = parseStringLiteral(source, quoteIndex);
    if (!parsed) continue;
    if (
      parsed.hasInterpolation &&
      !hasCjk(parsed.value) &&
      !/[A-Za-z]/.test(parsed.value.replace(/\$\{[^}]+\}/g, ''))
    ) {
      continue;
    }
    addViolation(violations, file, quoteIndex, field, parsed.value);
  }

  return violations;
}

function collectTemplateAttributeViolations(file: SourceFile): RuleViolation[] {
  if (file.kind !== 'vue') return [];
  const violations: RuleViolation[] = [];
  const tagPattern = /<[^!/\s][^>]*>/g;
  const attrPattern = /([:@#]?[A-Za-z][\w:-]*)\s*=\s*(["'])(.*?)\2/gs;

  for (const tagMatch of file.source.matchAll(tagPattern)) {
    const tag = tagMatch[0];
    const tagIndex = tagMatch.index ?? 0;
    for (const attrMatch of tag.matchAll(attrPattern)) {
      const rawName = attrMatch[1];
      const value = attrMatch[3] ?? '';
      const field = normalizeAttributeName(rawName ?? '');
      if (!UI_COPY_FIELDS.has(field) || KEY_FIELDS.has(field)) continue;
      const valueIndex = tagIndex + (attrMatch.index ?? 0) + attrMatch[0].indexOf(value);
      if (isBoundAttribute(rawName ?? '')) {
        const trimmed = value.trim();
        if (!trimmed) continue;
        const expressionIndex = valueIndex + value.indexOf(trimmed);
        for (const literal of collectExpressionLiterals(trimmed)) {
          if (literal.hasInterpolation && !hasCjk(literal.value)) continue;
          addViolation(violations, file, expressionIndex + literal.index, rawName ?? field, literal.value);
        }
        continue;
      }
      if (value.includes('{{') || value.includes('${')) continue;
      addViolation(violations, file, valueIndex, rawName ?? field, value);
    }
  }

  return violations;
}

function collectTemplateLiteralCjkViolations(file: SourceFile): RuleViolation[] {
  const violations: RuleViolation[] = [];
  const source = preserveLineStructure(file.source);
  for (const match of source.matchAll(/`/g)) {
    const quoteIndex = match.index ?? 0;
    const parsed = parseStringLiteral(source, quoteIndex);
    if (!parsed || !parsed.hasInterpolation || !hasCjk(parsed.value)) continue;
    addViolation(violations, file, quoteIndex, 'templateLiteral', parsed.value);
  }
  return violations;
}

export const noHardcodedUiPropRule: I18nGovernanceRule = {
  id: 'no-hardcoded-ui-prop',
  description: 'Blocks hard-coded visible UI copy in known props, object fields, and CJK template literals.',
  defaultSeverity: 'error',
  appliesTo: ['vue', 'ts', 'tsx'],
  check(context) {
    return context.sourceFiles.flatMap((file) => [
      ...collectObjectFieldViolations(file),
      ...collectConditionalFieldViolations(file),
      ...collectLogicalFallbackViolations(file),
      ...collectLocalizedTitleObjectViolations(file),
      ...collectLocaleLiteralFieldViolations(file),
      ...collectComputedLocaleLiteralFieldViolations(file),
      ...collectTemplateAttributeViolations(file),
      ...collectTemplateLiteralCjkViolations(file),
    ]);
  },
};
