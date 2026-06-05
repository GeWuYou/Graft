import { readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT_DIR = fileURLToPath(new URL('..', import.meta.url));
const REPOSITORY_DIR = fileURLToPath(new URL('../..', import.meta.url));
const SRC_DIR = join(ROOT_DIR, 'src');
const SERVER_TITLE_KEY_DIRS = [join(REPOSITORY_DIR, 'server/internal'), join(REPOSITORY_DIR, 'server/modules')];

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

type LocaleCode = 'zh-CN' | 'en-US';

type LocaleCatalog = {
  file: string;
  locale: LocaleCode;
  messages: Map<string, string>;
};

type RuntimeReferenceSet = {
  exactKeys: Set<string>;
  requiredKeys: Set<string>;
  dynamicPatterns: RegExp[];
};

type ParsedString = {
  value: string;
  endIndex: number;
  hasInterpolation: boolean;
};

const EXTERNAL_BOOTSTRAP_KEY_ALLOWLIST = [
  // Menu title keys are also supplied by backend bootstrap metadata at runtime.
  /^menu\./,
  // Language labels are consumed by locale aggregation rather than page code.
  /^lang$/,
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
  return /^[a-z][a-z0-9]*(?:[.-][a-zA-Z0-9][\w-]*)+$/.test(value);
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

function localeFromFile(file: string): LocaleCode | null {
  const match = file.match(/(?:^|\/)(zh-CN|en-US)\.json$/);
  return match ? (match[1] as LocaleCode) : null;
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

function resolveSourceOwner(file: string): string {
  const moduleMatch = file.match(/^src\/modules\/([^/]+)\/locales\//);
  if (moduleMatch) {
    return `module:${moduleMatch[1]}`;
  }

  if (file.startsWith('src/locales/lang/')) {
    return 'root';
  }

  return 'unknown';
}

function moduleMenuPrefix(moduleName: string): string {
  return moduleName.replace(/-([a-z0-9])/g, (_, value: string) => value.toUpperCase());
}

function resolveKeyOwner(file: string, key: string): string {
  const sourceOwner = resolveSourceOwner(file);
  if (sourceOwner === 'root') {
    return 'root';
  }

  const moduleName = sourceOwner.match(/^module:(.+)$/)?.[1];
  if (!moduleName) {
    return sourceOwner;
  }

  const camelMenu = moduleMenuPrefix(moduleName);
  const snakeMenu = moduleName.replaceAll('-', '_');
  const modulePrefixes = [`${moduleName}.`, `${camelMenu}.`, `menu.${camelMenu}.`, `menu.${snakeMenu}.`];

  if (modulePrefixes.some((prefix) => key === prefix.slice(0, -1) || key.startsWith(prefix))) {
    return sourceOwner;
  }

  return `module:${moduleName}:foreign`;
}

function collectLocaleCatalogs(): LocaleCatalog[] {
  const catalogs: LocaleCatalog[] = [];

  for (const filePath of collectLocaleFiles(SRC_DIR)) {
    const file = relative(ROOT_DIR, filePath).replaceAll('\\', '/');
    const locale = localeFromFile(file);
    if (!locale) {
      continue;
    }

    catalogs.push({
      file,
      locale,
      messages: flattenLocaleStrings(JSON.parse(readFileSync(filePath, 'utf8'))),
    });
  }

  return catalogs.sort((left, right) => left.file.localeCompare(right.file));
}

function collectDuplicateKeyFindings(catalogs: LocaleCatalog[]): LocaleFinding[] {
  const keyDefinitions = new Map<string, LocaleCatalog[]>();
  const findings: LocaleFinding[] = [];

  for (const catalog of catalogs) {
    for (const key of catalog.messages.keys()) {
      const definitionKey = `${catalog.locale}:${key}`;
      const definitions = keyDefinitions.get(definitionKey) ?? [];
      definitions.push(catalog);
      keyDefinitions.set(definitionKey, definitions);
    }
  }

  for (const [definitionKey, definitions] of keyDefinitions) {
    if (definitions.length <= 1) {
      continue;
    }

    const [, key] = definitionKey.split(/:(.*)/s);
    findings.push({
      file: definitions.map((definition) => definition.file).join(', '),
      message: `duplicate locale key ${key} for ${definitions[0].locale}`,
    });
  }

  return findings;
}

function collectSplitOwnerFindings(catalogs: LocaleCatalog[]): LocaleFinding[] {
  const ownerDefinitions = new Map<string, Map<string, Set<string>>>();
  const findings: LocaleFinding[] = [];

  for (const catalog of catalogs) {
    for (const key of catalog.messages.keys()) {
      const keyOwner = resolveKeyOwner(catalog.file, key);
      const sourceOwners = ownerDefinitions.get(key) ?? new Map<string, Set<string>>();
      const files = sourceOwners.get(keyOwner) ?? new Set<string>();
      files.add(catalog.file);
      sourceOwners.set(keyOwner, files);
      ownerDefinitions.set(key, sourceOwners);
    }
  }

  for (const [key, owners] of ownerDefinitions) {
    const rootFiles = owners.get('root');
    if (!rootFiles) {
      continue;
    }

    const moduleOwners = [...owners.entries()].filter(([owner]) => owner.startsWith('module:'));
    if (moduleOwners.length === 0) {
      continue;
    }

    const moduleFiles = moduleOwners.flatMap(([, files]) => [...files]);
    findings.push({
      file: [...rootFiles, ...moduleFiles].sort().join(', '),
      message: `split locale ownership for ${key} between root and module catalogs`,
    });
  }

  return findings;
}

function shouldScanRuntimeFile(file: string): boolean {
  if (!shouldScanFile(file)) {
    return false;
  }

  const normalized = relative(ROOT_DIR, file).replaceAll('\\', '/');
  if (
    normalized.includes('/locales/') ||
    normalized.startsWith('src/locales/') ||
    normalized.includes('/mock/') ||
    normalized.includes('/mocks/') ||
    normalized.includes('/__mocks__/')
  ) {
    return false;
  }

  return true;
}

function isAllowedUnusedLocaleKey(key: string): boolean {
  return EXTERNAL_BOOTSTRAP_KEY_ALLOWLIST.some((pattern) => pattern.test(key));
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function buildTemplateKeyPattern(template: string): RegExp | null {
  if (!template.includes('${')) {
    return null;
  }

  const expressionPattern = /\$\{[^}]+\}/g;
  const staticParts = template.split(expressionPattern);
  if (!staticParts.some((part) => isLikelyI18nKey(part.replace(/\.$/, '')) || part.includes('.'))) {
    return null;
  }

  return new RegExp(`^${staticParts.map((part) => escapeRegExp(part)).join('.+')}$`);
}

function collectRuntimeReferenceSet(): RuntimeReferenceSet {
  const referenced = new Set<string>();
  const required = new Set<string>();
  const dynamicPatterns: RegExp[] = [];
  const messagePrefixes = new Set<string>();
  const messagePrefixSuffixes = new Set<string>();
  const literalPattern = /(['"`])([a-z][a-z0-9]*(?:[.-][a-zA-Z0-9][\w-]*)+)\1/g;
  const templateLiteralPattern = /`([^`]*\$\{[^`]+}[^`]*)`/g;
  const directTranslatePattern = /\b(?:t|i18n\.global\.t)\(\s*(['"`])([^'"`$]+)\1/g;
  const dynamicTranslatePattern = /\b(?:t|i18n\.global\.t)\(\s*`([^`]+)`/g;
  const keyFieldPattern = /\b(?:titleKey|title_key)\b\s*[:=]\s*(['"`])([^'"`$]+)\1/g;
  const messagePrefixPropPattern = /\bmessage-prefix\s*=\s*(['"])([^'"`$]+)\1/g;
  const messagePrefixTemplatePattern = /\$\{messagePrefix\}((?:\.[a-zA-Z0-9][\w-]*)+)/g;

  for (const filePath of walk(SRC_DIR)) {
    if (!shouldScanRuntimeFile(filePath)) {
      continue;
    }

    const source = preserveLineStructure(readFileSync(filePath, 'utf8'));
    for (const match of source.matchAll(directTranslatePattern)) {
      referenced.add(match[2]);
      required.add(match[2]);
    }

    for (const match of source.matchAll(dynamicTranslatePattern)) {
      const pattern = buildTemplateKeyPattern(match[1]);
      if (pattern) {
        dynamicPatterns.push(pattern);
      }
    }

    for (const match of source.matchAll(templateLiteralPattern)) {
      const pattern = buildTemplateKeyPattern(match[1]);
      if (pattern) {
        dynamicPatterns.push(pattern);
      }
    }

    for (const match of source.matchAll(keyFieldPattern)) {
      referenced.add(match[2]);
      required.add(match[2]);
    }

    for (const match of source.matchAll(messagePrefixPropPattern)) {
      messagePrefixes.add(match[2]);
    }

    for (const match of source.matchAll(messagePrefixTemplatePattern)) {
      messagePrefixSuffixes.add(match[1]);
    }

    for (const match of source.matchAll(literalPattern)) {
      const value = match[2];
      if (isLikelyI18nKey(value)) {
        referenced.add(value);
      }
    }
  }

  for (const prefix of messagePrefixes) {
    for (const suffix of messagePrefixSuffixes) {
      referenced.add(`${prefix}${suffix}`);
      required.add(`${prefix}${suffix}`);
    }
  }

  for (const key of collectServerMenuTitleKeys()) {
    referenced.add(key);
    required.add(key);
  }

  return { exactKeys: referenced, requiredKeys: required, dynamicPatterns };
}

function shouldScanServerTitleKeyFile(file: string): boolean {
  const normalized = relative(REPOSITORY_DIR, file).replaceAll('\\', '/');
  return (
    file.endsWith('.go') &&
    (normalized.startsWith('server/internal/') || normalized.startsWith('server/modules/')) &&
    !normalized.includes('/contract/openapi/generated/')
  );
}

function walkServerTitleKeyFiles(dir: string): string[] {
  const files: string[] = [];

  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const fullPath = join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walkServerTitleKeyFiles(fullPath));
      continue;
    }

    if (shouldScanServerTitleKeyFile(fullPath)) {
      files.push(fullPath);
    }
  }

  return files;
}

function collectServerMenuTitleKeys(): Set<string> {
  const keys = new Set<string>();
  const stringConstantPattern = /\b([A-Za-z_]\w*)(?:\s+[A-Za-z_]\w*)?\s*=\s*"([^"$]+)"/g;
  const titleKeyPattern =
    /\b(?:TitleKey|title_key)\s*:\s*(?:"([^"$]+)"|(?:(?:[A-Za-z_]\w*)\.)?([A-Za-z_]\w*)(?:\.String\(\))?)/g;

  for (const dir of SERVER_TITLE_KEY_DIRS) {
    for (const filePath of walkServerTitleKeyFiles(dir)) {
      const source = preserveLineStructure(readFileSync(filePath, 'utf8'));
      const stringConstants = new Map<string, string>();

      for (const match of source.matchAll(stringConstantPattern)) {
        stringConstants.set(match[1], match[2]);
      }

      for (const match of source.matchAll(titleKeyPattern)) {
        const literalKey = match[1];
        const constantKey = match[2] ? stringConstants.get(match[2]) : undefined;
        const key = literalKey ?? constantKey;
        if (key) {
          keys.add(key);
        }
      }
    }
  }

  return keys;
}

function isRuntimeReferenced(key: string, referenceSet: RuntimeReferenceSet): boolean {
  return referenceSet.exactKeys.has(key) || referenceSet.dynamicPatterns.some((pattern) => pattern.test(key));
}

function collectUnusedKeyFindings(catalogs: LocaleCatalog[]): LocaleFinding[] {
  const referenceSet = collectRuntimeReferenceSet();
  const keyDefinitions = new Map<string, Set<string>>();
  const findings: LocaleFinding[] = [];

  for (const catalog of catalogs) {
    for (const key of catalog.messages.keys()) {
      const files = keyDefinitions.get(key) ?? new Set<string>();
      files.add(catalog.file);
      keyDefinitions.set(key, files);
    }
  }

  for (const [key, files] of [...keyDefinitions.entries()].sort(([left], [right]) => left.localeCompare(right))) {
    if (isRuntimeReferenced(key, referenceSet) || isAllowedUnusedLocaleKey(key)) {
      continue;
    }

    findings.push({
      file: [...files].sort().join(', '),
      message: `unused locale key ${key}`,
    });
  }

  return findings;
}

function collectMissingReferenceFindings(catalogs: LocaleCatalog[]): LocaleFinding[] {
  const referenceSet = collectRuntimeReferenceSet();
  const definedKeys = new Set<string>();
  const findings: LocaleFinding[] = [];

  for (const catalog of catalogs) {
    for (const key of catalog.messages.keys()) {
      definedKeys.add(key);
    }
  }

  for (const key of [...referenceSet.requiredKeys].sort()) {
    if (!definedKeys.has(key)) {
      findings.push({
        file: 'src',
        message: `referenced locale key ${key} is missing from locale catalogs`,
      });
    }
  }

  return findings;
}

function isEnglishInitialCaseExempt(key: string): boolean {
  const conjunctionKey = ['common', 'conjunction'].join('.');
  return key === conjunctionKey || key.endsWith('.unit');
}

function startsWithLowercaseLetter(value: string): boolean {
  return /^[a-z]/.test(normalizeText(value));
}

function collectEnglishInitialCaseFindings(catalogs: LocaleCatalog[]): LocaleFinding[] {
  const findings: LocaleFinding[] = [];

  for (const catalog of catalogs) {
    if (catalog.locale !== 'en-US') {
      continue;
    }

    for (const [key, value] of catalog.messages) {
      if (isEnglishInitialCaseExempt(key) || !startsWithLowercaseLetter(value)) {
        continue;
      }

      findings.push({
        file: catalog.file,
        message: `English locale value for ${key} should start with an uppercase letter`,
      });
    }
  }

  return findings;
}

function collectLocaleFindings(): LocaleFinding[] {
  const catalogs = collectLocaleCatalogs();
  const groupedFiles = new Map<string, Partial<Record<LocaleCode, LocaleCatalog>>>();
  const findings: LocaleFinding[] = [];

  for (const catalog of catalogs) {
    const pairKey = localePairKey(catalog.file);
    const group = groupedFiles.get(pairKey) ?? {};
    group[catalog.locale] = catalog;
    groupedFiles.set(pairKey, group);
  }

  findings.push(...collectDuplicateKeyFindings(catalogs));
  findings.push(...collectSplitOwnerFindings(catalogs));
  findings.push(...collectMissingReferenceFindings(catalogs));
  findings.push(...collectUnusedKeyFindings(catalogs));
  findings.push(...collectEnglishInitialCaseFindings(catalogs));

  for (const [pairKey, group] of groupedFiles) {
    if (!group['zh-CN'] || !group['en-US']) {
      findings.push({
        file: pairKey,
        message: 'missing paired zh-CN/en-US locale file',
      });
      continue;
    }

    const zhFile = group['zh-CN'].file;
    const enFile = group['en-US'].file;
    const zhMessages = group['zh-CN'].messages;
    const enMessages = group['en-US'].messages;
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
