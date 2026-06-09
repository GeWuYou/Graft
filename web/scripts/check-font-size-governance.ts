// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { readdirSync, readFileSync } from 'node:fs';
import { extname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT_DIR = fileURLToPath(new URL('..', import.meta.url));
const SRC_DIR = join(ROOT_DIR, 'src');

const SCANNED_EXTENSIONS = new Set(['.vue', '.less', '.css', '.scss', '.sass', '.ts', '.tsx']);
const EXCLUDED_DIRS = new Set(['node_modules', 'dist', 'coverage', 'mock', '__mocks__', 'ai-libs', 'assets']);
const FIXED_FONT_SIZE_PATTERN =
  /\bfont-size\s*:\s*(?:calc\([^;]*\b\d+(?:\.\d+)?(?:px|r?em)\b[^;]*\)|clamp\([^;]*\b\d+(?:\.\d+)?(?:px|r?em)\b[^;]*\)|\d+(?:\.\d+)?(?:px|r?em))\b[^;]*;?/g;
const FIXED_FONT_SHORTHAND_PATTERN = /\bfont\s*:\s*(?!var\()[^;{}]*\b\d+(?:\.\d+)?(?:px|r?em)\b[^;{}]*;?/g;
const CSS_SELECTOR_PATTERN = /^\s*([^{};@]+)\s*\{/;

type AllowlistEntry = {
  file: string;
  selector: string;
  property: 'font' | 'font-size';
  value: string;
  reason: string;
};

type Finding = {
  file: string;
  line: number;
  selector: string;
  declaration: string;
};

const ALLOWLIST: AllowlistEntry[] = [
  {
    file: 'src/layouts/components/Search.vue',
    selector: '.t-icon',
    property: 'font-size',
    value: '20px',
    reason: 'Header search icon follows the fixed TDesign icon glyph box rather than page text scale.',
  },
  {
    file: 'src/layouts/components/theme-workbench/ThemeWorkbenchPanel.vue',
    selector: '.nav-item__icon',
    property: 'font-size',
    value: '20px',
    reason: 'Theme workbench navigation icons use a fixed glyph box for stable rail alignment.',
  },
  {
    file: 'src/layouts/components/theme-workbench/ThemeWorkbenchPanel.vue',
    selector: '.brand-input__value',
    property: 'font-size',
    value: '12px',
    reason: 'Monospace color value preview keeps compact code-like text inside constrained controls.',
  },
  {
    file: 'src/layouts/components/theme-workbench/ThemeWorkbenchPanel.vue',
    selector: '.advanced-layout .advanced-group__icon',
    property: 'font-size',
    value: '20px',
    reason: 'Advanced group icon uses a fixed glyph box inside a fixed square action surface.',
  },
  {
    file: 'src/layouts/components/theme-workbench/ThemeTokenEditor.vue',
    selector: '.token-key',
    property: 'font-size',
    value: '12px',
    reason: 'Monospace token keys need a compact fixed code scale inside editor rows.',
  },
  {
    file: 'src/modules/monitor/pages/modules/index.vue',
    selector: '.module-runtime-detail__paths code',
    property: 'font-size',
    value: '12px',
    reason: 'Runtime path code blocks use compact monospace text for long path readability.',
  },
  {
    file: 'src/modules/monitor/pages/overview/index.less',
    selector: '.trend-info-trigger__icon',
    property: 'font-size',
    value: '16px',
    reason: 'Trend info trigger icon uses a fixed glyph size inside an icon-only control.',
  },
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

  if (!SCANNED_EXTENSIONS.has(extname(file)) || /\.d\.ts$/.test(normalized) || /\.test\.(?:ts|tsx)$/.test(normalized)) {
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

function propertyForDeclaration(declaration: string): AllowlistEntry['property'] {
  return declaration.trim().startsWith('font-size') ? 'font-size' : 'font';
}

function valueForDeclaration(declaration: string): string {
  return declaration.match(/\b\d+(?:\.\d+)?(?:px|r?em)\b/)?.[0] ?? '';
}

function isAllowed(finding: Finding): boolean {
  const property = propertyForDeclaration(finding.declaration);
  const value = valueForDeclaration(finding.declaration);

  return ALLOWLIST.some(
    (entry) =>
      entry.file === finding.file &&
      entry.selector === finding.selector &&
      entry.property === property &&
      entry.value === value,
  );
}

function collectFindings(): Finding[] {
  const findings: Finding[] = [];

  for (const file of walk(SRC_DIR)) {
    const rel = relative(ROOT_DIR, file).replaceAll('\\', '/');
    const source = readFileSync(file, 'utf8');
    const lines = source.split('\n');
    const matches = [
      ...source.matchAll(FIXED_FONT_SIZE_PATTERN),
      ...source.matchAll(FIXED_FONT_SHORTHAND_PATTERN),
    ].sort((left, right) => (left.index ?? 0) - (right.index ?? 0));

    for (const match of matches) {
      const index = match.index ?? 0;
      const line = lineNumberForIndex(source, index);
      const selector = selectorBeforeLine(lines, line - 1);
      const declaration = match[0].trim();
      const finding = { file: rel, line, selector, declaration };

      if (isAllowed(finding)) {
        continue;
      }

      findings.push(finding);
    }
  }

  return findings;
}

const findings = collectFindings();

if (findings.length === 0) {
  process.stdout.write('Font size governance: no non-allowlisted fixed font sizes found.\n');
} else {
  process.stdout.write('Font size governance findings:\n');
  for (const finding of findings) {
    process.stdout.write(`- ${finding.file}:${finding.line} ${finding.selector} -> ${finding.declaration}\n`);
  }
  process.stdout.write(
    '\nUse TDesign/Graft typography tokens instead, or add a selector-specific allowlist entry with a concrete reason.\n',
  );
  process.exitCode = 1;
}
