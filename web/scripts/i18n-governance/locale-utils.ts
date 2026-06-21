import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';

import { EXCLUDED_DIRS, ROOT_DIR } from './config';
import { isLikelyI18nKey, parseStringLiteral, positionForIndex, preserveLineStructure } from './text-utils';
import type { RuleViolation, ScanContext, SourceFile } from './types';

export type LocaleCode = 'zh-CN' | 'en-US';

export interface LocaleCatalog {
  absolutePath: string;
  file: string;
  locale: LocaleCode;
  messages: Map<string, string>;
  source: string;
  lineStarts: number[];
}

export interface RuntimeReferenceSet {
  exactKeys: Set<string>;
  requiredKeys: Set<string>;
  dynamicPatterns: TemplateKeyMatcher[];
}

export type TemplateKeyMatcher = (key: string) => boolean;

type DuplicateLocaleKey = {
  file: string;
  line: number;
  key: string;
};

const EXTERNAL_BOOTSTRAP_KEY_ALLOWLIST = [/^menu\./, /^lang$/];

export function isLocaleFile(file: string): boolean {
  return /(?:^|\/)(?:zh-CN|en-US)\.json$/.test(file);
}

export function localePairKey(file: string): string {
  return file.replace(/(?:zh-CN|en-US)\.json$/, '{locale}.json');
}

function localeFromFile(file: string): LocaleCode | null {
  const match = file.match(/(?:^|\/)(zh-CN|en-US)\.json$/);
  return match ? (match[1] as LocaleCode) : null;
}

function collectLocaleFiles(dir: string): string[] {
  if (!existsSync(dir)) return [];

  const files: string[] = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (EXCLUDED_DIRS.has(entry.name)) continue;

    const fullPath = join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...collectLocaleFiles(fullPath));
      continue;
    }

    const file = relative(ROOT_DIR, fullPath).replaceAll('\\', '/');
    if (isLocaleFile(file)) files.push(fullPath);
  }

  return files;
}

export function flattenLocaleStrings(
  value: unknown,
  prefix = '',
  output = new Map<string, string>(),
): Map<string, string> {
  if (typeof value === 'string') {
    output.set(prefix, value);
    return output;
  }

  if (!value || typeof value !== 'object' || Array.isArray(value)) return output;

  for (const [key, child] of Object.entries(value)) {
    flattenLocaleStrings(child, prefix ? `${prefix}.${key}` : key, output);
  }

  return output;
}

export function collectLocaleCatalogs(context: ScanContext): LocaleCatalog[] {
  const catalogs: LocaleCatalog[] = [];

  for (const filePath of collectLocaleFiles(context.srcDir)) {
    const file = relative(context.rootDir, filePath).replaceAll('\\', '/');
    const locale = localeFromFile(file);
    if (!locale) continue;

    const source = readFileSync(filePath, 'utf8');
    catalogs.push({
      absolutePath: filePath,
      file,
      locale,
      messages: flattenLocaleStrings(JSON.parse(source)),
      source,
      lineStarts: buildLineIndex(source),
    });
  }

  return catalogs.sort((left, right) => left.file.localeCompare(right.file));
}

function buildLineIndex(source: string): number[] {
  const lines = [0];
  for (let index = 0; index < source.length; index += 1) {
    if (source[index] === '\n') lines.push(index + 1);
  }
  return lines;
}

export function resolveSourceOwner(file: string): string {
  const moduleMatch = file.match(/^src\/modules\/([^/]+)\/locales\//);
  if (moduleMatch) return `module:${moduleMatch[1]}`;
  if (file.startsWith('src/locales/lang/')) return 'root';
  return 'unknown';
}

function moduleMenuPrefix(moduleName: string): string {
  return moduleName.replace(/-([a-z0-9])/g, (_, value: string) => value.toUpperCase());
}

export function resolveKeyOwner(file: string, key: string): string {
  const sourceOwner = resolveSourceOwner(file);
  if (sourceOwner === 'root') return 'root';

  const moduleName = sourceOwner.match(/^module:(.+)$/)?.[1];
  if (!moduleName) return sourceOwner;

  const camelMenu = moduleMenuPrefix(moduleName);
  const snakeMenu = moduleName.replaceAll('-', '_');
  const modulePrefixes = [`${moduleName}.`, `${camelMenu}.`, `menu.${camelMenu}.`, `menu.${snakeMenu}.`];

  if (modulePrefixes.some((prefix) => key === prefix.slice(0, -1) || key.startsWith(prefix))) {
    return sourceOwner;
  }

  return `module:${moduleName}:foreign`;
}

function shouldScanRuntimeFile(file: SourceFile): boolean {
  const normalized = file.relativePath;
  return (
    !normalized.includes('/locales/') &&
    !normalized.startsWith('src/locales/') &&
    !normalized.includes('/mock/') &&
    !normalized.includes('/mocks/') &&
    !normalized.includes('/__mocks__/')
  );
}

export function isAllowedUnusedLocaleKey(key: string): boolean {
  return EXTERNAL_BOOTSTRAP_KEY_ALLOWLIST.some((pattern) => pattern.test(key));
}

function buildTemplateKeyMatcher(template: string): TemplateKeyMatcher | null {
  if (!template.includes('${')) return null;

  const expressionPattern = /\$\{[^}]+}/g;
  const staticParts = template.split(expressionPattern);
  if (!staticParts.some((part) => isLikelyI18nKey(part.replace(/\.$/, '')) || part.includes('.'))) return null;

  return (key: string) => matchesTemplateStaticParts(key, staticParts);
}

function matchesTemplateStaticParts(key: string, staticParts: string[]): boolean {
  const firstPart = staticParts[0] ?? '';
  const lastPart = staticParts[staticParts.length - 1] ?? '';

  let lowerBound = 0;
  if (firstPart) {
    if (!key.startsWith(firstPart)) return false;
    lowerBound = firstPart.length;
  }

  const finalStart = lastPart ? key.lastIndexOf(lastPart) : key.length;
  if (lastPart && finalStart + lastPart.length !== key.length) return false;

  for (let index = 1; index < staticParts.length - 1; index += 1) {
    lowerBound += 1;
    const part = staticParts[index];
    if (!part) continue;
    const partIndex = key.indexOf(part, lowerBound);
    if (partIndex < 0 || partIndex + part.length > finalStart) return false;
    lowerBound = partIndex + part.length;
  }

  lowerBound += 1;
  return lastPart ? finalStart >= lowerBound : key.length >= lowerBound;
}

function collectStaticStringLiterals(source: string): string[] {
  const literals: string[] = [];
  let index = 0;

  while (index < source.length) {
    const char = source[index];
    if (char !== '"' && char !== "'" && char !== '`') {
      index += 1;
      continue;
    }

    const parsed = parseStringLiteral(source, index);
    if (!parsed) {
      index += 1;
      continue;
    }
    if (!parsed.hasInterpolation) literals.push(parsed.value);
    index = parsed.endIndex;
  }

  return literals;
}

export function collectRuntimeReferenceSet(context: ScanContext): RuntimeReferenceSet {
  const referenced = new Set<string>();
  const required = new Set<string>();
  const dynamicPatterns: TemplateKeyMatcher[] = [];
  const messagePrefixes = new Set<string>();
  const messagePrefixSuffixes = new Set<string>();
  const templateLiteralPattern = /`([^`]*\$\{[^`]+}[^`]*)`/g;
  const directTranslatePattern = /\b(?:t|i18n\.global\.t)\(\s*(['"`])([^'"`$]+)\1/g;
  const dynamicTranslatePattern = /\b(?:t|i18n\.global\.t)\(\s*`([^`]+)`/g;
  const keyFieldPattern =
    /\b(?:titleKey|title_key|descriptionKey|description_key|messageKey|message_key)\b\s*[:=]\s*(['"`])([^'"`$]+)\1/g;
  const messagePrefixPropPattern = /\bmessage-prefix\s*=\s*(['"])([^'"`$]+)\1/g;
  const messagePrefixTemplatePattern = /\$\{messagePrefix\}((?:\.[a-zA-Z0-9][\w-]*)+)/g;

  for (const file of context.sourceFiles.filter(shouldScanRuntimeFile)) {
    const source = preserveLineStructure(file.source);
    for (const match of source.matchAll(directTranslatePattern)) {
      referenced.add(match[2]);
      required.add(match[2]);
    }

    for (const match of source.matchAll(dynamicTranslatePattern)) {
      const matcher = buildTemplateKeyMatcher(match[1]);
      if (matcher) dynamicPatterns.push(matcher);
    }

    for (const match of source.matchAll(templateLiteralPattern)) {
      const matcher = buildTemplateKeyMatcher(match[1]);
      if (matcher) dynamicPatterns.push(matcher);
    }

    for (const match of source.matchAll(keyFieldPattern)) {
      referenced.add(match[2]);
      required.add(match[2]);
    }

    for (const match of source.matchAll(messagePrefixPropPattern)) messagePrefixes.add(match[2]);
    for (const match of source.matchAll(messagePrefixTemplatePattern)) messagePrefixSuffixes.add(match[1]);

    for (const value of collectStaticStringLiterals(source)) {
      if (isLikelyI18nKey(value)) referenced.add(value);
    }
  }

  for (const prefix of messagePrefixes) {
    for (const suffix of messagePrefixSuffixes) {
      referenced.add(`${prefix}${suffix}`);
      required.add(`${prefix}${suffix}`);
    }
  }

  for (const key of collectServerI18nKeys(context)) {
    referenced.add(key);
    required.add(key);
  }

  return { exactKeys: referenced, requiredKeys: required, dynamicPatterns };
}

function collectServerI18nKeys(context: ScanContext): Set<string> {
  const keys = new Set<string>();
  const stringConstantPattern = /^\s*(?:const|var)?\s*([A-Za-z_]\w*)(?:\s+[A-Za-z_]\w*)?\s*=\s*"([^"$]+)"/gm;
  const dynamicKeyFunctionPattern =
    /func\s+([A-Za-z_]\w*)\(\s*[A-Za-z_]\w*\s+string\s*\)\s+string\s*{\s*return\s*"([^"$]*)"\s*\+\s*[A-Za-z_]\w*\s*\+\s*"([^"$]*)"\s*}/g;
  const configDefinitionCallPattern = /\b[A-Za-z_]\w*Definition\(\s*(?:"([^"$]+)"|([A-Za-z_]\w*))/g;
  const serverKeyFieldPattern =
    /\b(?:DomainKey|GroupKey|GroupDescriptionKey|TitleKey|DisplayKey|DescriptionKey|LabelKey|EmptyKey|MessageKey|DisplayMessageKey|DescriptionMessageKey|domainKey|groupKey|groupDescriptionKey|titleKey|displayKey|descriptionKey|labelKey|emptyKey|messageKey|domain_key|group_key|group_description_key|title_key|display_key|description_key|label_key|empty_key|message_key)\s*:\s*(?:"([^"$]+)"|(?:(?:[A-Za-z_]\w*)\.)?([A-Za-z_]\w*)(?:\.String\(\))?)/g;
  const serverQuotedKeyFieldPattern =
    /["'](?:domainKey|groupKey|groupDescriptionKey|titleKey|displayKey|descriptionKey|labelKey|emptyKey|messageKey|unitKey|placeholderKey|domain_key|group_key|group_description_key|title_key|display_key|description_key|label_key|empty_key|message_key|unit_key|placeholder_key)["']\s*:\s*(?:"([^"$]+)"|(?:(?:[A-Za-z_]\w*)\.)?([A-Za-z_]\w*)(?:\.String\(\))?)/g;
  const serverEscapedQuotedKeyFieldPattern =
    /\\"(?:domainKey|groupKey|groupDescriptionKey|titleKey|displayKey|descriptionKey|labelKey|emptyKey|messageKey|unitKey|placeholderKey|domain_key|group_key|group_description_key|title_key|display_key|description_key|label_key|empty_key|message_key|unit_key|placeholder_key)\\"\s*:\s*\\"([^"\\$]+)\\"/g;
  const serverSQLAliasKeyPattern =
    /['"]([^'"$]+)['"]\s+(?:AS\s+)?(?:title_key|display_key|description_key|label_key|empty_key|message_key)\b/gi;
  const dynamicKeyCallPattern = /\b([A-Za-z_]\w*)\(\s*([A-Za-z_]\w*)\s*\)/g;

  for (const file of context.serverFiles) {
    const source = preserveLineStructure(file.source);
    const stringConstants = new Map<string, string>();
    const dynamicKeyFunctions = new Map<string, { prefix: string; suffix: string }>();
    const configDefinitionKeys = new Set<string>();

    for (const match of source.matchAll(stringConstantPattern)) {
      stringConstants.set(match[1], match[2]);
      if (isLikelyServerI18nConstant(match[1], match[2])) addServerI18nKey(keys, match[2]);
    }

    for (const match of source.matchAll(dynamicKeyFunctionPattern)) {
      if (match[2].startsWith('systemConfig.') && isSystemConfigDynamicKeyFunction(match[1], match[2], match[3])) {
        dynamicKeyFunctions.set(match[1], { prefix: match[2], suffix: match[3] });
      }
    }

    for (const match of source.matchAll(configDefinitionCallPattern)) {
      const configKey = match[1] ?? (match[2] ? stringConstants.get(match[2]) : undefined);
      if (configKey && isLikelyConfigDefinitionKey(configKey)) configDefinitionKeys.add(configKey);
    }

    for (const match of source.matchAll(serverKeyFieldPattern)) {
      addServerI18nKey(keys, match[1] ?? (match[2] ? stringConstants.get(match[2]) : undefined));
    }

    for (const match of source.matchAll(serverQuotedKeyFieldPattern)) {
      addServerI18nKey(keys, match[1] ?? (match[2] ? stringConstants.get(match[2]) : undefined));
    }

    for (const match of source.matchAll(serverEscapedQuotedKeyFieldPattern)) addServerI18nKey(keys, match[1]);
    for (const match of source.matchAll(serverSQLAliasKeyPattern)) addServerI18nKey(keys, match[1]);

    for (const configKey of configDefinitionKeys) {
      for (const template of dynamicKeyFunctions.values())
        addServerI18nKey(keys, `${template.prefix}${configKey}${template.suffix}`);
    }

    for (const match of source.matchAll(dynamicKeyCallPattern)) {
      const template = dynamicKeyFunctions.get(match[1]);
      const argumentValue = stringConstants.get(match[2]);
      if (template && argumentValue) addServerI18nKey(keys, `${template.prefix}${argumentValue}${template.suffix}`);
    }
  }

  return keys;
}

function isSystemConfigDynamicKeyFunction(functionName: string, prefix: string, suffix: string) {
  return (
    prefix.startsWith('systemConfig.') &&
    suffix.startsWith('.') &&
    /(?:Config)?(?:Title|Description)Key$/.test(functionName)
  );
}

function isLikelyConfigDefinitionKey(value: string) {
  return /^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)+$/.test(value) && !value.startsWith('systemConfig.');
}

function isLikelyServerI18nConstant(name: string, value: string) {
  return (
    value.startsWith('systemConfig.') &&
    /(?:DomainKey|GroupKey|GroupDescKey|GroupDescriptionKey|TitleKey|DisplayKey|DescriptionKey|DescKey|LabelKey|EmptyKey|MessageKey|UnitKey|PlaceholderKey)$/.test(
      name,
    )
  );
}

function addServerI18nKey(keys: Set<string>, rawKey: string | undefined) {
  const key = rawKey?.trim();
  if (!key || key.endsWith('.') || !isLikelyI18nKey(key)) return;
  keys.add(key);
}

export function isRuntimeReferenced(key: string, referenceSet: RuntimeReferenceSet): boolean {
  return referenceSet.exactKeys.has(key) || referenceSet.dynamicPatterns.some((matches) => matches(key));
}

export function localeViolation(
  ruleId: string,
  severity: RuleViolation['severity'],
  filePath: string,
  message: string,
  suggestion: string,
  line = 1,
): RuleViolation {
  return {
    ruleId,
    severity,
    filePath,
    line,
    message,
    suggestion,
  };
}

export function collectExactDuplicateKeys(catalog: LocaleCatalog): DuplicateLocaleKey[] {
  const seen = new Map<string, number>();
  const duplicates: DuplicateLocaleKey[] = [];
  const pathStack: string[][] = [];
  let nextContainerKey: string | null = null;
  let index = 0;

  while (index < catalog.source.length) {
    const char = catalog.source[index];

    if (char === '{') {
      const parentPath = pathStack[pathStack.length - 1] ?? [];
      pathStack.push(nextContainerKey ? [...parentPath, nextContainerKey] : parentPath);
      nextContainerKey = null;
      index += 1;
      continue;
    }

    if (char === '}') {
      pathStack.pop();
      nextContainerKey = null;
      index += 1;
      continue;
    }

    if (char !== '"') {
      index += 1;
      continue;
    }

    const parsed = parseStringLiteral(catalog.source, index);
    if (!parsed) {
      index += 1;
      continue;
    }

    const after = catalog.source.slice(parsed.endIndex).match(/^\s*:/);
    if (!after) {
      index = parsed.endIndex;
      continue;
    }

    const path = [...(pathStack[pathStack.length - 1] ?? []), parsed.value].join('.');
    const firstIndex = seen.get(path);
    if (firstIndex !== undefined) {
      duplicates.push({
        file: catalog.file,
        line: positionForIndex(catalog.lineStarts, index).line,
        key: path,
      });
    } else {
      seen.set(path, index);
    }
    nextContainerKey = parsed.value;
    index = parsed.endIndex;
  }

  return duplicates;
}
