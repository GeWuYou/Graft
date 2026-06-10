// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { UI_COPY_FIELDS } from '../config';
import {
  isLikelyI18nKey,
  isTechnicalString,
  normalizeText,
  parseStringLiteral,
  positionForIndex,
  preserveLineStructure,
} from '../text-utils';
import type { RuleContext, RuleViolation, SourceFile } from '../types';

type Finding = {
  file: string;
  line: number;
  text: string;
};

type LocaleFinding = {
  file: string;
  message: string;
};

// TODO(i18n-governance): split these remaining compatibility checks into named rules:
// - no-hardcoded-template-text
// - no-system-config-schema-fallback

function addFinding(findings: Finding[], file: string, line: number, text: string, fieldName?: string) {
  const normalized = normalizeText(text);
  if (normalized.length === 0 || normalized === fieldName || isTechnicalString(normalized)) return;

  findings.push({ file, line, text: normalized });
}

function findTemplateBlocks(source: string): Array<{ start: number; end: number }> {
  const blocks: Array<{ start: number; end: number }> = [];
  let searchIndex = 0;

  while (searchIndex < source.length) {
    const openMatch = source.slice(searchIndex).match(/<template(?:\s[^>]*)?>/i);
    if (!openMatch || openMatch.index === undefined) break;

    const openingTagStart = searchIndex + openMatch.index;
    const contentStart = openingTagStart + openMatch[0].length;
    let depth = 1;
    let index = contentStart;

    while (index < source.length) {
      const tagStart = source.indexOf('<', index);
      if (tagStart === -1) break;

      const tagEnd = findTagEnd(source, tagStart);
      if (tagEnd === -1) break;

      const tagText = source.slice(tagStart, tagEnd + 1);
      if (/^<\/template\s*>$/i.test(tagText)) {
        depth -= 1;
        if (depth === 0) {
          blocks.push({ start: contentStart, end: tagStart });
          searchIndex = tagEnd + 1;
          break;
        }
      } else if (/^<template(?:\s|>)/i.test(tagText) && !/\/>$/.test(tagText)) {
        depth += 1;
      }

      index = tagEnd + 1;
    }

    if (depth !== 0) break;
  }

  return blocks;
}

function findTagEnd(source: string, tagStart: number): number {
  let quote: '"' | "'" | null = null;

  for (let index = tagStart + 1; index < source.length; index += 1) {
    const char = source[index];
    if (quote) {
      if (char === quote && source[index - 1] !== '\\') quote = null;
      continue;
    }

    if (char === '"' || char === "'") {
      quote = char;
      continue;
    }

    if (char === '>') return index;
  }

  return -1;
}

function lineForIndex(file: SourceFile, index: number): number {
  return positionForIndex(file.lineStarts, index).line;
}

function collectTemplateTextFindings(file: SourceFile): Finding[] {
  if (file.kind !== 'vue') return [];

  const findings: Finding[] = [];
  for (const block of findTemplateBlocks(file.source)) {
    let index = block.start;
    while (index < block.end) {
      if (file.source.startsWith('<!--', index)) {
        const commentEnd = file.source.indexOf('-->', index + 4);
        index = commentEnd === -1 ? block.end : commentEnd + 3;
        continue;
      }

      if (file.source[index] === '<') {
        const tagEnd = findTagEnd(file.source, index);
        index = tagEnd === -1 ? block.end : tagEnd + 1;
        continue;
      }

      if (file.source.startsWith('{{', index)) {
        const interpolationEnd = file.source.indexOf('}}', index + 2);
        index = interpolationEnd === -1 ? block.end : interpolationEnd + 2;
        continue;
      }

      const textStart = index;
      while (index < block.end && file.source[index] !== '<' && !file.source.startsWith('{{', index)) index += 1;

      const rawText = file.source.slice(textStart, index);
      const previousTagStart = file.source.lastIndexOf('<', textStart);
      const previousTagEnd = previousTagStart === -1 ? -1 : findTagEnd(file.source, previousTagStart);
      const containingTag =
        previousTagStart !== -1 && previousTagEnd !== -1 && previousTagEnd < textStart
          ? file.source.slice(previousTagStart, previousTagEnd + 1)
          : '';
      if (/aria-hidden\s*=\s*(?:"true"|'true'|true)/.test(containingTag)) continue;

      addFinding(findings, file.relativePath, lineForIndex(file, textStart), rawText);
    }
  }

  return findings;
}

function normalizeTemplateAttributeName(name: string): string {
  return name.replace(/-([a-z])/g, (_, letter: string) => letter.toUpperCase());
}

function isBoundTemplateAttribute(name: string): boolean {
  return (
    name.startsWith(':') || name.startsWith('@') || name.startsWith('#') || name.startsWith('v-') || name.includes(':')
  );
}

function collectTemplateAttributeFindings(file: SourceFile): Finding[] {
  if (file.kind !== 'vue') return [];

  const findings: Finding[] = [];
  for (const block of findTemplateBlocks(file.source)) {
    let index = block.start;
    while (index < block.end) {
      const tagStart = file.source.indexOf('<', index);
      if (tagStart === -1 || tagStart >= block.end) break;

      if (file.source.startsWith('<!--', tagStart)) {
        const commentEnd = file.source.indexOf('-->', tagStart + 4);
        index = commentEnd === -1 ? block.end : commentEnd + 3;
        continue;
      }

      const tagEnd = findTagEnd(file.source, tagStart);
      if (tagEnd === -1) break;
      if (file.source[tagStart + 1] === '/') {
        index = tagEnd + 1;
        continue;
      }

      let cursor = tagStart + 1;
      while (cursor < tagEnd && !/[\s/>]/.test(file.source[cursor])) cursor += 1;

      while (cursor < tagEnd) {
        while (cursor < tagEnd && /\s/.test(file.source[cursor])) cursor += 1;
        if (cursor >= tagEnd || file.source[cursor] === '/') {
          cursor += 1;
          continue;
        }

        const attrNameStart = cursor;
        while (cursor < tagEnd && !/[\s=>]/.test(file.source[cursor])) cursor += 1;
        const attrName = file.source.slice(attrNameStart, cursor);

        while (cursor < tagEnd && /\s/.test(file.source[cursor])) cursor += 1;
        if (file.source[cursor] !== '=') continue;
        cursor += 1;

        while (cursor < tagEnd && /\s/.test(file.source[cursor])) cursor += 1;
        const quote = file.source[cursor];
        if (quote !== '"' && quote !== "'") {
          while (cursor < tagEnd && !/\s/.test(file.source[cursor])) cursor += 1;
          continue;
        }

        const valueStart = cursor + 1;
        cursor = valueStart;
        while (cursor < tagEnd && file.source[cursor] !== quote) cursor += file.source[cursor] === '\\' ? 2 : 1;
        const value = file.source.slice(valueStart, cursor);
        cursor += 1;

        const fieldName = normalizeTemplateAttributeName(attrName);
        if (
          isBoundTemplateAttribute(attrName) ||
          !UI_COPY_FIELDS.has(fieldName) ||
          value.includes('{{') ||
          value.includes('${')
        ) {
          continue;
        }

        addFinding(findings, file.relativePath, lineForIndex(file, valueStart), value, fieldName);
      }

      index = tagEnd + 1;
    }
  }

  return findings;
}

function collectUiFieldFindings(file: SourceFile): Finding[] {
  const findings: Finding[] = [];
  const strippedSource = preserveLineStructure(file.source);
  const fieldPattern =
    /(^|[,{(]\s*)(['"]?)(label|title|description|placeholder|content|header|emptyText|text|message)\2\s*:\s*(['"`])/gm;

  for (const match of strippedSource.matchAll(fieldPattern)) {
    const fieldName = match[3];
    if (!UI_COPY_FIELDS.has(fieldName)) continue;

    const quoteIndex = (match.index ?? 0) + match[0].length - 1;
    const parsed = parseStringLiteral(strippedSource, quoteIndex);
    if (!parsed || parsed.hasInterpolation) continue;

    addFinding(findings, file.relativePath, lineForIndex(file, quoteIndex), parsed.value, fieldName);
  }

  return findings;
}

function collectPluginStringFindings(file: SourceFile): Finding[] {
  const findings: Finding[] = [];
  const strippedSource = preserveLineStructure(file.source);
  const pluginPattern = /\b(?:MessagePlugin|NotificationPlugin|DialogPlugin)(?:\.\w+)?\s*\(\s*(['"`])/g;

  for (const match of strippedSource.matchAll(pluginPattern)) {
    const quoteIndex = (match.index ?? 0) + match[0].length - 1;
    const parsed = parseStringLiteral(strippedSource, quoteIndex);
    if (!parsed || parsed.hasInterpolation) continue;

    addFinding(findings, file.relativePath, lineForIndex(file, quoteIndex), parsed.value);
  }

  return findings;
}

function collectFindings(context: RuleContext): Finding[] {
  const findings = context.sourceFiles.flatMap((file) => [
    ...collectTemplateTextFindings(file),
    ...collectTemplateAttributeFindings(file),
    ...collectUiFieldFindings(file),
    ...collectPluginStringFindings(file),
  ]);

  return dedupeFindings(findings).sort((left, right) => {
    if (left.file !== right.file) return left.file.localeCompare(right.file);
    if (left.line !== right.line) return left.line - right.line;
    return left.text.localeCompare(right.text);
  });
}

function dedupeFindings(findings: Finding[]): Finding[] {
  const seen = new Set<string>();
  const deduped: Finding[] = [];

  for (const finding of findings) {
    const key = `${finding.file}:${finding.line}:${finding.text}`;
    if (seen.has(key)) continue;
    seen.add(key);
    deduped.push(finding);
  }

  return deduped;
}

function collectServerSystemConfigSchemaFallbackFindings(context: RuleContext): LocaleFinding[] {
  const findings: LocaleFinding[] = [];

  for (const file of context.serverFiles) {
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
        const line = lineForIndex(file, index);
        collectSchemaNodeFallbackFindings(schema, `${file.relativePath}:${line}`, 'schema', findings);
      }

      index = parsed.endIndex;
    }
  }

  return findings;
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

function collectSchemaNodeFallbackFindings(node: unknown, file: string, path: string, findings: LocaleFinding[]): void {
  if (!node || typeof node !== 'object') return;

  if (Array.isArray(node)) {
    node.forEach((child, index) => collectSchemaNodeFallbackFindings(child, file, `${path}[${index}]`, findings));
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
      message: `system config schema ${path}.${field} has visible fallback "${normalized}" without x-i18n.${keyField}`,
    });
  }

  for (const [key, child] of Object.entries(objectNode)) {
    if (key === 'x-i18n') continue;
    collectSchemaNodeFallbackFindings(child, file, `${path}.${key}`, findings);
  }
}

export function runLegacyRule(context: RuleContext): RuleViolation[] {
  const findings = collectFindings(context);
  const schemaFindings = collectServerSystemConfigSchemaFallbackFindings(context);
  const violations: RuleViolation[] = [];

  for (const finding of findings) {
    violations.push({
      ruleId: 'legacy.no-hardcoded-ui-text',
      severity: 'error',
      filePath: finding.file,
      line: finding.line,
      message: 'Found hard-coded UI text',
      excerpt: finding.text,
      suggestion: "Move visible copy into locale catalogs and render it with t('...').",
    });
  }

  for (const finding of schemaFindings) {
    violations.push({
      ruleId: 'legacy.locale-governance',
      severity: 'error',
      filePath: finding.file,
      line: 1,
      message: finding.message,
      suggestion: 'Keep zh-CN/en-US catalogs, ownership, and referenced keys aligned.',
    });
  }

  return violations;
}
