import { readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT_DIR = fileURLToPath(new URL('..', import.meta.url));
const SRC_DIR = join(ROOT_DIR, 'src');

const SCANNED_EXTENSIONS = new Set(['.vue', '.ts', '.tsx']);
const EXCLUDED_DIRS = new Set(['node_modules', 'dist', 'coverage', 'mock', '__mocks__', 'ai-libs']);
const UI_COPY_FIELDS = new Set([
  'label',
  'title',
  'description',
  'placeholder',
  'content',
  'header',
  'emptyText',
  'text',
  'message',
]);
const KNOWN_NON_I18N_NAMES = new Set([
  'Axios',
  'Bun',
  'Casbin',
  'Ent',
  'Gin',
  'Go',
  'Graft',
  'HarmonyOS Sans',
  'Inter',
  'Pinia',
  'PostgreSQL',
  'Redis',
  'Source Han Sans',
  'TDesign',
  'TDesign Original',
  'Tencent Cloud',
  'TypeScript',
  'UnoCSS',
  'Vite',
  'Vue',
  'Zap',
]);

const TECHNICAL_UNITS = new Set(['ms', 'px', 'em', 'rem', 'vh', 'vw']);

type Finding = {
  file: string;
  line: number;
  text: string;
};

type LocaleFinding = {
  file: string;
  message: string;
};

type ParsedString = {
  value: string;
  endIndex: number;
  hasInterpolation: boolean;
};

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
  const extension = file.endsWith('.vue') ? '.vue' : file.endsWith('.tsx') ? '.tsx' : file.endsWith('.ts') ? '.ts' : '';

  if (!SCANNED_EXTENSIONS.has(extension)) {
    return false;
  }

  if (/\.d\.ts$/.test(normalized) || /\.test\.(?:ts|tsx)$/.test(normalized)) {
    return false;
  }

  if (normalized.startsWith('src/contracts/openapi/generated/')) {
    return false;
  }

  if (normalized.endsWith('.json')) {
    return false;
  }

  return true;
}

function normalizeText(value: string): string {
  return value
    .replace(/&nbsp;/gi, ' ')
    .replace(/&amp;/gi, '&')
    .replace(/&lt;/gi, '<')
    .replace(/&gt;/gi, '>')
    .replace(/\s+/g, ' ')
    .trim();
}

function hasVisibleLetters(value: string): boolean {
  return /[A-Za-z\u3400-\u9FFF]/.test(value);
}

function isLikelyI18nKey(value: string): boolean {
  return /^[a-z][a-z0-9]*(?:[.-][a-zA-Z0-9][\w-]*){2,}$/.test(value);
}

function isTechnicalString(value: string): boolean {
  const text = normalizeText(value);

  if (!hasVisibleLetters(text)) {
    return true;
  }

  if (KNOWN_NON_I18N_NAMES.has(text)) {
    return true;
  }

  if (TECHNICAL_UNITS.has(text)) {
    return true;
  }

  if (isLikelyI18nKey(text)) {
    return true;
  }

  if (/^(?:https?:|wss?:|data:|mailto:|\/|\.\/|\.\.\/|@\/)/.test(text)) {
    return true;
  }

  if (/^#[0-9a-fA-F]{3,8}$/.test(text) || /^var\(--[\w-]+\)$/.test(text) || /^--[\w-]+$/.test(text)) {
    return true;
  }

  if (/^(?:\d+(?:\.\d+)?(?:px|r?em|%|vh|vw|s|ms)?|auto|none|block|inline|flex|grid)$/.test(text)) {
    return true;
  }

  if (/(?:sans-serif|serif|monospace|Arial|Helvetica|PingFang|Microsoft YaHei|font-family)/i.test(text)) {
    return true;
  }

  if (/^[A-Z0-9_./:-]+$/.test(text)) {
    return true;
  }

  if (/^[a-z0-9]+(?:[-_:/][a-z0-9]+)+$/.test(text)) {
    return true;
  }

  if (/^[A-Za-z]+Icon$/.test(text) || /^[A-Za-z]+(?:Outlined|Filled)$/.test(text)) {
    return true;
  }

  return false;
}

function preserveLineStructure(source: string): string {
  let output = '';
  let index = 0;
  let quote: '"' | "'" | '`' | null = null;

  while (index < source.length) {
    const char = source[index];
    const next = source[index + 1];

    if (quote) {
      output += char;
      if (char === '\\') {
        output += next ?? '';
        index += 2;
        continue;
      }
      if (char === quote) {
        quote = null;
      }
      index += 1;
      continue;
    }

    if (char === '"' || char === "'" || char === '`') {
      quote = char;
      output += char;
      index += 1;
      continue;
    }

    if (char === '/' && next === '/') {
      output += '  ';
      index += 2;
      while (index < source.length && source[index] !== '\n') {
        output += ' ';
        index += 1;
      }
      continue;
    }

    if (char === '/' && next === '*') {
      output += '  ';
      index += 2;
      while (index < source.length) {
        if (source[index] === '*' && source[index + 1] === '/') {
          output += '  ';
          index += 2;
          break;
        }
        output += source[index] === '\n' ? '\n' : ' ';
        index += 1;
      }
      continue;
    }

    output += char;
    index += 1;
  }

  return output;
}

function parseStringLiteral(source: string, quoteIndex: number): ParsedString | null {
  const quote = source[quoteIndex];
  if (quote !== '"' && quote !== "'" && quote !== '`') {
    return null;
  }

  let value = '';
  let hasInterpolation = false;
  let index = quoteIndex + 1;

  while (index < source.length) {
    const char = source[index];

    if (char === '\\') {
      value += source[index + 1] ?? '';
      index += 2;
      continue;
    }

    if (quote === '`' && char === '$' && source[index + 1] === '{') {
      hasInterpolation = true;
    }

    if (char === quote) {
      return { value, endIndex: index + 1, hasInterpolation };
    }

    value += char;
    index += 1;
  }

  return null;
}

function buildLineIndex(source: string): number[] {
  const lines = [0];

  for (let index = 0; index < source.length; index += 1) {
    if (source[index] === '\n') {
      lines.push(index + 1);
    }
  }

  return lines;
}

function lineForIndex(lineIndex: number[], index: number): number {
  let low = 0;
  let high = lineIndex.length - 1;

  while (low <= high) {
    const mid = Math.floor((low + high) / 2);
    if (lineIndex[mid] <= index) {
      low = mid + 1;
    } else {
      high = mid - 1;
    }
  }

  return high + 1;
}

function addFinding(findings: Finding[], file: string, line: number, text: string, fieldName?: string) {
  const normalized = normalizeText(text);
  if (normalized.length === 0 || normalized === fieldName || isTechnicalString(normalized)) {
    return;
  }

  findings.push({ file, line, text: normalized });
}

function findTemplateBlocks(source: string): Array<{ start: number; end: number }> {
  const blocks: Array<{ start: number; end: number }> = [];
  let searchIndex = 0;

  while (searchIndex < source.length) {
    const openMatch = source.slice(searchIndex).match(/<template(?:\s[^>]*)?>/i);
    if (!openMatch || openMatch.index === undefined) {
      break;
    }

    const openingTagStart = searchIndex + openMatch.index;
    const contentStart = openingTagStart + openMatch[0].length;
    let depth = 1;
    let index = contentStart;

    while (index < source.length) {
      const tagStart = source.indexOf('<', index);
      if (tagStart === -1) {
        break;
      }

      const tagEnd = findTagEnd(source, tagStart);
      if (tagEnd === -1) {
        break;
      }

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

    if (depth !== 0) {
      break;
    }
  }

  return blocks;
}

function findTagEnd(source: string, tagStart: number): number {
  let quote: '"' | "'" | null = null;

  for (let index = tagStart + 1; index < source.length; index += 1) {
    const char = source[index];
    if (quote) {
      if (char === quote && source[index - 1] !== '\\') {
        quote = null;
      }
      continue;
    }

    if (char === '"' || char === "'") {
      quote = char;
      continue;
    }

    if (char === '>') {
      return index;
    }
  }

  return -1;
}

function collectTemplateTextFindings(source: string, file: string, lineIndex: number[]): Finding[] {
  const findings: Finding[] = [];

  for (const block of findTemplateBlocks(source)) {
    let index = block.start;
    while (index < block.end) {
      if (source.startsWith('<!--', index)) {
        const commentEnd = source.indexOf('-->', index + 4);
        index = commentEnd === -1 ? block.end : commentEnd + 3;
        continue;
      }

      if (source[index] === '<') {
        const tagEnd = findTagEnd(source, index);
        index = tagEnd === -1 ? block.end : tagEnd + 1;
        continue;
      }

      if (source.startsWith('{{', index)) {
        const interpolationEnd = source.indexOf('}}', index + 2);
        index = interpolationEnd === -1 ? block.end : interpolationEnd + 2;
        continue;
      }

      const textStart = index;
      while (index < block.end && source[index] !== '<' && !source.startsWith('{{', index)) {
        index += 1;
      }

      const rawText = source.slice(textStart, index);
      const previousTagStart = source.lastIndexOf('<', textStart);
      const previousTagEnd = previousTagStart === -1 ? -1 : findTagEnd(source, previousTagStart);
      const containingTag =
        previousTagStart !== -1 && previousTagEnd !== -1 && previousTagEnd < textStart
          ? source.slice(previousTagStart, previousTagEnd + 1)
          : '';
      if (/aria-hidden\s*=\s*(?:"true"|'true'|true)/.test(containingTag)) {
        continue;
      }

      addFinding(findings, file, lineForIndex(lineIndex, textStart), rawText);
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

function collectTemplateAttributeFindings(source: string, file: string, lineIndex: number[]): Finding[] {
  const findings: Finding[] = [];

  for (const block of findTemplateBlocks(source)) {
    let index = block.start;
    while (index < block.end) {
      const tagStart = source.indexOf('<', index);
      if (tagStart === -1 || tagStart >= block.end) {
        break;
      }

      if (source.startsWith('<!--', tagStart)) {
        const commentEnd = source.indexOf('-->', tagStart + 4);
        index = commentEnd === -1 ? block.end : commentEnd + 3;
        continue;
      }

      const tagEnd = findTagEnd(source, tagStart);
      if (tagEnd === -1) {
        break;
      }

      if (source[tagStart + 1] === '/') {
        index = tagEnd + 1;
        continue;
      }

      let cursor = tagStart + 1;
      while (cursor < tagEnd && !/[\s/>]/.test(source[cursor])) {
        cursor += 1;
      }

      while (cursor < tagEnd) {
        while (cursor < tagEnd && /\s/.test(source[cursor])) {
          cursor += 1;
        }

        if (cursor >= tagEnd || source[cursor] === '/') {
          cursor += 1;
          continue;
        }

        const attrNameStart = cursor;
        while (cursor < tagEnd && !/[\s=>]/.test(source[cursor])) {
          cursor += 1;
        }
        const attrName = source.slice(attrNameStart, cursor);

        while (cursor < tagEnd && /\s/.test(source[cursor])) {
          cursor += 1;
        }

        if (source[cursor] !== '=') {
          continue;
        }
        cursor += 1;

        while (cursor < tagEnd && /\s/.test(source[cursor])) {
          cursor += 1;
        }

        const quote = source[cursor];
        if (quote !== '"' && quote !== "'") {
          while (cursor < tagEnd && !/\s/.test(source[cursor])) {
            cursor += 1;
          }
          continue;
        }

        const valueStart = cursor + 1;
        cursor = valueStart;
        while (cursor < tagEnd && source[cursor] !== quote) {
          cursor += source[cursor] === '\\' ? 2 : 1;
        }
        const value = source.slice(valueStart, cursor);
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

        addFinding(findings, file, lineForIndex(lineIndex, valueStart), value, fieldName);
      }

      index = tagEnd + 1;
    }
  }

  return findings;
}

function collectUiFieldFindings(source: string, file: string, lineIndex: number[]): Finding[] {
  const findings: Finding[] = [];
  const strippedSource = preserveLineStructure(source);
  const fieldPattern =
    /(^|[,{(]\s*)(['"]?)(label|title|description|placeholder|content|header|emptyText|text|message)\2\s*:\s*(['"`])/gm;

  for (const match of strippedSource.matchAll(fieldPattern)) {
    const fieldName = match[3];
    if (!UI_COPY_FIELDS.has(fieldName)) {
      continue;
    }

    const quoteIndex = (match.index ?? 0) + match[0].length - 1;
    const parsed = parseStringLiteral(strippedSource, quoteIndex);
    if (!parsed || parsed.hasInterpolation) {
      continue;
    }

    addFinding(findings, file, lineForIndex(lineIndex, quoteIndex), parsed.value, fieldName);
  }

  return findings;
}

function collectPluginStringFindings(source: string, file: string, lineIndex: number[]): Finding[] {
  const findings: Finding[] = [];
  const strippedSource = preserveLineStructure(source);
  const pluginPattern = /\b(?:MessagePlugin|NotificationPlugin|DialogPlugin)(?:\.\w+)?\s*\(\s*(['"`])/g;

  for (const match of strippedSource.matchAll(pluginPattern)) {
    const quoteIndex = (match.index ?? 0) + match[0].length - 1;
    const parsed = parseStringLiteral(strippedSource, quoteIndex);
    if (!parsed || parsed.hasInterpolation) {
      continue;
    }

    addFinding(findings, file, lineForIndex(lineIndex, quoteIndex), parsed.value);
  }

  return findings;
}

function collectFindings(): Finding[] {
  const findings: Finding[] = [];

  for (const filePath of walk(SRC_DIR)) {
    const file = relative(ROOT_DIR, filePath).replaceAll('\\', '/');
    const source = readFileSync(filePath, 'utf8');
    const lineIndex = buildLineIndex(source);

    if (file.endsWith('.vue')) {
      findings.push(...collectTemplateTextFindings(source, file, lineIndex));
      findings.push(...collectTemplateAttributeFindings(source, file, lineIndex));
    }

    findings.push(...collectUiFieldFindings(source, file, lineIndex));
    findings.push(...collectPluginStringFindings(source, file, lineIndex));
  }

  return dedupeFindings(findings).sort((left, right) => {
    if (left.file !== right.file) {
      return left.file.localeCompare(right.file);
    }
    if (left.line !== right.line) {
      return left.line - right.line;
    }
    return left.text.localeCompare(right.text);
  });
}

function dedupeFindings(findings: Finding[]): Finding[] {
  const seen = new Set<string>();
  const deduped: Finding[] = [];

  for (const finding of findings) {
    const key = `${finding.file}:${finding.line}:${finding.text}`;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    deduped.push(finding);
  }

  return deduped;
}

function isLocaleFile(file: string): boolean {
  return /(?:^|\/)(?:zh-CN|en-US)\.json$/.test(file);
}

function localePairKey(file: string): string {
  return file.replace(/(?:zh-CN|en-US)\.json$/, '{locale}.json');
}

function localeFromFile(file: string): 'zh-CN' | 'en-US' | null {
  const match = file.match(/(?:^|\/)(zh-CN|en-US)\.json$/);
  return match ? (match[1] as 'zh-CN' | 'en-US') : null;
}

function collectLocaleFiles(dir: string): string[] {
  const files: string[] = [];

  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (EXCLUDED_DIRS.has(entry.name)) {
      continue;
    }

    const fullPath = join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...collectLocaleFiles(fullPath));
      continue;
    }

    const file = relative(ROOT_DIR, fullPath).replaceAll('\\', '/');
    if (isLocaleFile(file)) {
      files.push(fullPath);
    }
  }

  return files;
}

function flattenLocaleStrings(value: unknown, prefix = '', output = new Map<string, string>()): Map<string, string> {
  if (typeof value === 'string') {
    output.set(prefix, value);
    return output;
  }

  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return output;
  }

  for (const [key, child] of Object.entries(value)) {
    flattenLocaleStrings(child, prefix ? `${prefix}.${key}` : key, output);
  }

  return output;
}

function collectLocaleFindings(): LocaleFinding[] {
  const groupedFiles = new Map<string, Partial<Record<'zh-CN' | 'en-US', string>>>();
  const findings: LocaleFinding[] = [];

  for (const filePath of collectLocaleFiles(SRC_DIR)) {
    const file = relative(ROOT_DIR, filePath).replaceAll('\\', '/');
    const locale = localeFromFile(file);
    if (!locale) {
      continue;
    }

    const pairKey = localePairKey(file);
    const group = groupedFiles.get(pairKey) ?? {};
    group[locale] = filePath;
    groupedFiles.set(pairKey, group);
  }

  for (const [pairKey, group] of groupedFiles) {
    if (!group['zh-CN'] || !group['en-US']) {
      findings.push({
        file: pairKey,
        message: 'missing paired zh-CN/en-US locale file',
      });
      continue;
    }

    const zhFile = relative(ROOT_DIR, group['zh-CN']).replaceAll('\\', '/');
    const enFile = relative(ROOT_DIR, group['en-US']).replaceAll('\\', '/');
    const zhMessages = flattenLocaleStrings(JSON.parse(readFileSync(group['zh-CN'], 'utf8')));
    const enMessages = flattenLocaleStrings(JSON.parse(readFileSync(group['en-US'], 'utf8')));
    const zhKeys = new Set(zhMessages.keys());
    const enKeys = new Set(enMessages.keys());

    for (const key of [...zhKeys].sort()) {
      if (!enKeys.has(key)) {
        findings.push({ file: enFile, message: `missing locale key ${key}` });
      }
    }

    for (const key of [...enKeys].sort()) {
      if (!zhKeys.has(key)) {
        findings.push({ file: zhFile, message: `missing locale key ${key}` });
      }
    }

    for (const [key, value] of [...zhMessages.entries(), ...enMessages.entries()]) {
      if (normalizeText(value) === key) {
        findings.push({ file: pairKey, message: `locale key ${key} resolves to itself` });
      }
    }
  }

  return findings.sort((left, right) => {
    if (left.file !== right.file) {
      return left.file.localeCompare(right.file);
    }
    return left.message.localeCompare(right.message);
  });
}

const findings = collectFindings();
const localeFindings = collectLocaleFindings();

if (findings.length > 0) {
  process.stdout.write('Found hard-coded UI text:\n');
  for (const finding of findings) {
    process.stdout.write(`- ${finding.file}:${finding.line} ${finding.text}\n`);
  }
}

if (localeFindings.length > 0) {
  process.stdout.write('Found locale governance issues:\n');
  for (const finding of localeFindings) {
    process.stdout.write(`- ${finding.file} ${finding.message}\n`);
  }
}

if (findings.length > 0 || localeFindings.length > 0) {
  process.exitCode = 1;
} else {
  process.stdout.write('No hard-coded UI text or locale governance issues found.\n');
}
