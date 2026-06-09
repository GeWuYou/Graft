// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { readdirSync, readFileSync } from 'node:fs';
import { extname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT_DIR = fileURLToPath(new URL('..', import.meta.url));
const SRC_DIR = join(ROOT_DIR, 'src');

const SCANNED_EXTENSIONS = new Set(['.vue', '.less', '.css', '.scss', '.sass', '.ts', '.tsx']);
const STYLE_EXTENSIONS = new Set(['.less', '.css', '.scss', '.sass']);
const EXCLUDED_DIRS = new Set([
  'node_modules',
  'dist',
  'coverage',
  'mock',
  '__mocks__',
  '__tests__',
  'tests',
  'generated',
  'ai-libs',
  'assets',
]);
const DENSITY_PROPERTIES = new Set([
  'gap',
  'row-gap',
  'column-gap',
  'padding',
  'padding-top',
  'padding-right',
  'padding-bottom',
  'padding-left',
  'padding-block',
  'padding-block-start',
  'padding-block-end',
  'padding-inline',
  'padding-inline-start',
  'padding-inline-end',
  'margin',
  'margin-top',
  'margin-right',
  'margin-bottom',
  'margin-left',
  'margin-block',
  'margin-block-start',
  'margin-block-end',
  'margin-inline',
  'margin-inline-start',
  'margin-inline-end',
]);
const DENSITY_PROPERTY_PATTERN =
  /(?:^|[;{\n\r])\s*(?<property>gap|row-gap|column-gap|padding(?:-(?:top|right|bottom|left|block|block-start|block-end|inline|inline-start|inline-end))?|margin(?:-(?:top|right|bottom|left|block|block-start|block-end|inline|inline-start|inline-end))?)\s*:\s*(?<value>[^;{}]+);?/g;
const FIXED_SPACING_VALUE_PATTERN = /(?:^|[\s,(])(?<value>-?\d+(?:\.\d+)?(?:px|r?em))(?:\b|[),])/;
const CSS_SELECTOR_PATTERN = /^\s*([^{};@]+)\s*\{/;
const VUE_T_SPACE_TAG_PATTERN = /<t-space\b(?<attrs>[^>]*)>/gi;
const VUE_ATTR_PATTERN = /(?<name>[:@]?[A-Za-z][\w:-]*)(?:\s*=\s*(?<quote>["'])(?<value>.*?)\k<quote>)?/gs;
const NUMERIC_LITERAL_PATTERN = /^-?\d+(?:\.\d+)?(?:px|r?em)?$/;
const NUMERIC_ARRAY_PATTERN = /^\[\s*-?\d+(?:\.\d+)?(?:px|r?em)?(?:\s*,\s*-?\d+(?:\.\d+)?(?:px|r?em)?)+\s*\]$/;
const STYLE_STRING_PATTERN = /\bstyle\s*=\s*(?<quote>["'`])(?<value>.*?)\k<quote>/gs;
const JS_STRING_PATTERN =
  /(?<quote>["'`])(?<value>[^"'`]*(?:gap|padding|margin)\s*:\s*-?\d+(?:\.\d+)?px[^"'`]*)\k<quote>/g;

type FindingKind = 'css-declaration' | 'inline-style' | 't-space-size';

type AllowlistEntry = {
  file: string;
  context: string;
  property: string;
  value: string;
  kind: FindingKind;
  reason: string;
};

type Finding = {
  file: string;
  line: number;
  context: string;
  property: string;
  value: string;
  kind: FindingKind;
  detail: string;
};

const ALLOWLIST: AllowlistEntry[] = [
  // Keep fixed spacing exceptions selector-specific and explain why they do not participate in page information density.
];

function walk(dir: string): string[] {
  const entries = readdirSync(dir, { withFileTypes: true });
  const files: string[] = [];

  for (const entry of entries) {
    if (EXCLUDED_DIRS.has(entry.name)) {
      continue;
    }

    const fullPath = join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walk(fullPath));
      continue;
    }

    if (!shouldScanFile(fullPath)) {
      continue;
    }

    files.push(fullPath);
  }

  return files;
}

function shouldScanFile(file: string): boolean {
  const normalized = relative(ROOT_DIR, file).replaceAll('\\', '/');

  if (
    !SCANNED_EXTENSIONS.has(extname(file)) ||
    /\.d\.ts$/.test(normalized) ||
    /\.(?:test|spec)\.(?:ts|tsx)$/.test(normalized)
  ) {
    return false;
  }

  if (normalized.startsWith('src/contracts/openapi/generated/')) {
    return false;
  }

  return true;
}

function selectorBeforeLine(lines: string[], lineIndex: number): string {
  for (let index = lineIndex; index >= 0; index -= 1) {
    const match = lines[index].match(CSS_SELECTOR_PATTERN);
    if (match) {
      return match[1].trim();
    }
  }

  return '<unknown>';
}

function lineNumberForIndex(source: string, index: number): number {
  return source.slice(0, index).split('\n').length;
}

function preserveStyleBlocksOnly(source: string): string {
  const output = source.replace(/[^\n]/g, ' ').split('');

  for (const match of source.matchAll(/<style\b[^>]*>[\s\S]*?<\/style>/gi)) {
    const start = match.index ?? 0;
    const block = match[0];
    for (let offset = 0; offset < block.length; offset += 1) {
      output[start + offset] = block[offset];
    }
  }

  return output.join('');
}

function normalizeValue(value: string): string {
  return value.replace(/\s+/g, ' ').trim();
}

function isDensityAwareValue(value: string): boolean {
  const normalized = normalizeValue(value);
  const components = splitCssValueComponents(normalized);

  return components.length > 0 && components.every(isDensityAwareValueComponent);
}

function splitCssValueComponents(value: string): string[] {
  const components: string[] = [];
  let current = '';
  let depth = 0;

  for (const character of value) {
    if (/\s/.test(character) && depth === 0) {
      if (current) {
        components.push(current);
        current = '';
      }
      continue;
    }

    if (character === '(') {
      depth += 1;
    } else if (character === ')' && depth > 0) {
      depth -= 1;
    }

    current += character;
  }

  if (current) {
    components.push(current);
  }

  return components;
}

function isDensityAwareValueComponent(value: string): boolean {
  return (
    value === '0' ||
    /^var\(\s*--(?:td-comp|graft-density)-[\w-]+(?:\s*,[^)]*)?\)$/.test(value) ||
    /^calc\([^)]*var\(\s*--graft-theme-density-scale\s*\)[^)]*\)$/.test(value) ||
    isDensityAwareCalcValue(value)
  );
}

function isDensityAwareCalcValue(value: string): boolean {
  const calcMatch = value.match(/^calc\((?<expression>.+)\)$/);
  if (!calcMatch?.groups?.expression) {
    return false;
  }

  const withoutAcceptedFunctions = calcMatch.groups.expression
    .replace(/var\(\s*--(?:td-comp|graft-density)-[\w-]+(?:\s*,[^)]*)?\)/g, '')
    .replace(/env\(\s*safe-area-inset-(?:top|right|bottom|left)\s*,\s*0px\s*\)/g, '');

  return /^[\s+\-*/().0px]*$/.test(withoutAcceptedFunctions);
}

function fixedSpacingToken(value: string): string {
  return value.match(FIXED_SPACING_VALUE_PATTERN)?.groups?.value ?? '';
}

function isAllowed(finding: Finding): boolean {
  return ALLOWLIST.some(
    (entry) =>
      entry.file === finding.file &&
      entry.context === finding.context &&
      entry.property === finding.property &&
      entry.value === finding.value &&
      entry.kind === finding.kind,
  );
}

function pushFinding(findings: Finding[], finding: Finding) {
  if (isAllowed(finding)) {
    return;
  }

  if (
    findings.some(
      (existing) =>
        existing.file === finding.file &&
        existing.line === finding.line &&
        existing.context === finding.context &&
        existing.property === finding.property &&
        existing.value === finding.value &&
        existing.kind === finding.kind &&
        existing.detail === finding.detail,
    )
  ) {
    return;
  }

  findings.push(finding);
}

function collectCssDeclarationFindings(source: string, rel: string, lines: string[], findings: Finding[]) {
  for (const match of source.matchAll(DENSITY_PROPERTY_PATTERN)) {
    const property = match.groups?.property;
    const rawValue = match.groups?.value ?? '';

    if (!property || !DENSITY_PROPERTIES.has(property) || isDensityAwareValue(rawValue)) {
      continue;
    }

    const value = fixedSpacingToken(rawValue);
    if (!value) {
      continue;
    }

    const index = match.index ?? 0;
    const line = lineNumberForIndex(source, index);
    const context = selectorBeforeLine(lines, line - 1);

    pushFinding(findings, {
      file: rel,
      line,
      context,
      property,
      value,
      kind: 'css-declaration',
      detail: `${property}: ${normalizeValue(rawValue)}`,
    });
  }
}

function collectInlineStyleFindings(source: string, rel: string, findings: Finding[]) {
  const styleMatches = [...source.matchAll(STYLE_STRING_PATTERN), ...source.matchAll(JS_STRING_PATTERN)].sort(
    (left, right) => (left.index ?? 0) - (right.index ?? 0),
  );

  for (const styleMatch of styleMatches) {
    const styleValue = styleMatch.groups?.value ?? '';

    for (const declarationMatch of styleValue.matchAll(DENSITY_PROPERTY_PATTERN)) {
      const property = declarationMatch.groups?.property;
      const rawValue = declarationMatch.groups?.value ?? '';

      if (!property || !DENSITY_PROPERTIES.has(property) || isDensityAwareValue(rawValue)) {
        continue;
      }

      const value = fixedSpacingToken(rawValue);
      if (!value) {
        continue;
      }

      const index = (styleMatch.index ?? 0) + (declarationMatch.index ?? 0);
      const line = lineNumberForIndex(source, index);

      pushFinding(findings, {
        file: rel,
        line,
        context: 'inline style',
        property,
        value,
        kind: 'inline-style',
        detail: `${property}: ${normalizeValue(rawValue)}`,
      });
    }
  }
}

function parseAttributes(attrs: string): Array<{ name: string; value: string }> {
  const parsed: Array<{ name: string; value: string }> = [];

  for (const match of attrs.matchAll(VUE_ATTR_PATTERN)) {
    const name = match.groups?.name;
    const value = match.groups?.value;
    if (!name || value === undefined) {
      continue;
    }

    parsed.push({ name, value: normalizeValue(value) });
  }

  return parsed;
}

function isFixedSpaceSize(value: string): boolean {
  if (isDensityAwareValue(value)) {
    return false;
  }

  return NUMERIC_LITERAL_PATTERN.test(value) || NUMERIC_ARRAY_PATTERN.test(value);
}

function collectTSpaceFindings(source: string, rel: string, findings: Finding[]) {
  for (const tagMatch of source.matchAll(VUE_T_SPACE_TAG_PATTERN)) {
    const attrs = tagMatch.groups?.attrs ?? '';
    for (const attr of parseAttributes(attrs)) {
      if (attr.name !== 'size' && attr.name !== ':size' && attr.name !== 'v-bind:size') {
        continue;
      }

      if (!isFixedSpaceSize(attr.value)) {
        continue;
      }

      const line = lineNumberForIndex(source, tagMatch.index ?? 0);
      pushFinding(findings, {
        file: rel,
        line,
        context: '<t-space>',
        property: attr.name,
        value: attr.value,
        kind: 't-space-size',
        detail: `${attr.name}="${attr.value}"`,
      });
    }
  }
}

function collectFindings(): Finding[] {
  const findings: Finding[] = [];

  for (const file of walk(SRC_DIR)) {
    const rel = relative(ROOT_DIR, file).replaceAll('\\', '/');
    const source = readFileSync(file, 'utf8');
    const lines = source.split('\n');
    const extension = extname(file);

    if (file.endsWith('.vue')) {
      collectCssDeclarationFindings(preserveStyleBlocksOnly(source), rel, lines, findings);
    } else if (STYLE_EXTENSIONS.has(extension)) {
      collectCssDeclarationFindings(source, rel, lines, findings);
    }
    if (file.endsWith('.vue') || extension === '.ts' || extension === '.tsx') {
      collectInlineStyleFindings(source, rel, findings);
    }

    if (file.endsWith('.vue')) {
      collectTSpaceFindings(source, rel, findings);
    }
  }

  return findings;
}

const findings = collectFindings();

if (findings.length === 0) {
  process.stdout.write('Density governance: no non-allowlisted fixed density spacing found.\n');
} else {
  process.stdout.write('Density governance findings:\n');
  for (const finding of findings) {
    process.stdout.write(
      `- ${finding.file}:${finding.line} ${finding.context} -> ${finding.detail} [${finding.kind}]\n`,
    );
  }
  process.stdout.write(
    '\nUse TDesign component spacing tokens, Graft density tokens, or calc(...var(--graft-theme-density-scale)...). Add a file/context/property/value allowlist entry only for fixed spacing that is not density-sensitive, with a concrete reason.\n',
  );
  process.exitCode = 1;
}
