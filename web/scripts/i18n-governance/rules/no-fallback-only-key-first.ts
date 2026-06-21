import {
  isTechnicalString,
  normalizeText,
  parseStringLiteral,
  positionForIndex,
  preserveLineStructure,
} from '../text-utils';
import type { I18nGovernanceRule, RuleViolation, SourceFile } from '../types';

type Pair = {
  fallback: string;
  keys: string[];
};

const GO_KEY_FIRST_PAIRS: Pair[] = [
  { fallback: 'Title', keys: ['TitleKey'] },
  { fallback: 'Description', keys: ['DescriptionKey'] },
  { fallback: 'Display', keys: ['DisplayKey'] },
  { fallback: 'Name', keys: ['DisplayKey'] },
  { fallback: 'Message', keys: ['MessageKey'] },
  { fallback: 'Label', keys: ['LabelKey'] },
];

function shouldScanServerKeyFirstFile(filePath: string) {
  if (filePath.includes('/ent/') || filePath.includes('/storeent/') || filePath.includes('/migrations/')) return false;
  return /(?:registry|definition|registration|dashboard|config|scheduler|notification|menu|permission|module|retention)/.test(
    filePath,
  );
}

function hasKeyNearby(source: string, start: number, pair: Pair) {
  const window = source.slice(Math.max(0, start - 500), Math.min(source.length, start + 500));
  return pair.keys.some((key) => {
    const keyPattern = new RegExp(`(?:"${key}"|\\b${key})\\s*:\\s*(?:"[^"]+"|[A-Za-z_][\\w.()]*)`);
    return keyPattern.test(window);
  });
}

function pairForLowercaseFallback(field: string): Pair {
  const keyMap: Record<string, string[]> = {
    description: ['descriptionKey', 'description_key'],
    display: ['displayKey', 'display_key'],
    message: ['messageKey', 'message_key'],
    title: ['titleKey', 'title_key'],
  };

  return { fallback: field, keys: keyMap[field] ?? ['displayKey', 'display_key'] };
}

function addFallbackOnlyViolation(
  violations: RuleViolation[],
  file: SourceFile,
  index: number,
  field: string,
  keyField: string,
  value: string,
  strict: boolean,
) {
  const text = normalizeText(value);
  if (!text || isTechnicalString(text)) return;
  const position = positionForIndex(file.lineStarts, index);
  const preferredKeyField = keyField.split('|')[0] ?? keyField;
  violations.push({
    ruleId: 'no-fallback-only-key-first',
    severity: strict ? 'error' : 'warning',
    filePath: file.relativePath,
    line: position.line,
    column: position.column,
    message: `${field} fallback is present without ${keyField}`,
    excerpt: text,
    suggestion: `Add ${preferredKeyField} and zh-CN/en-US locale catalog entries; keep fallback as compatibility only.`,
  });
}

function collectGoFallbackOnly(file: SourceFile, strict: boolean): RuleViolation[] {
  const violations: RuleViolation[] = [];
  const source = preserveLineStructure(file.source);

  for (const pair of GO_KEY_FIRST_PAIRS) {
    const pattern = new RegExp(`\\b${pair.fallback}\\s*:\\s*(['"\`])`, 'g');
    for (const match of source.matchAll(pattern)) {
      const quoteIndex = (match.index ?? 0) + match[0].length - 1;
      const parsed = parseStringLiteral(source, quoteIndex);
      if (!parsed || hasKeyNearby(source, quoteIndex, pair)) continue;
      addFallbackOnlyViolation(violations, file, quoteIndex, pair.fallback, pair.keys.join('|'), parsed.value, strict);
    }
  }

  for (const match of source.matchAll(/\b(?:title|description|message|display)\s*:\s*(['"`])/g)) {
    const quoteIndex = (match.index ?? 0) + match[0].length - 1;
    const parsed = parseStringLiteral(source, quoteIndex);
    if (!parsed) continue;
    const field = match[0].split(':')[0]?.trim() ?? 'fallback';
    const pair = pairForLowercaseFallback(field);
    if (hasKeyNearby(source, quoteIndex, pair)) continue;
    addFallbackOnlyViolation(violations, file, quoteIndex, pair.fallback, pair.keys.join('|'), parsed.value, strict);
  }

  return violations;
}

export const noFallbackOnlyKeyFirstRule: I18nGovernanceRule = {
  id: 'no-fallback-only-key-first',
  description:
    'Reports fallback-only copy in server key-first registries while allowing key + fallback pairs; strict mode turns warnings into blockers.',
  defaultSeverity: 'warning',
  appliesTo: ['go'],
  check(context) {
    return context.serverFiles
      .filter((file) => shouldScanServerKeyFirstFile(file.relativePath))
      .flatMap((file) => collectGoFallbackOnly(file, context.strictKeyFirst));
  },
};
