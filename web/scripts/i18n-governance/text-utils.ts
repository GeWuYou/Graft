import { KNOWN_NON_I18N_NAMES, TECHNICAL_UNITS } from './config';

export type ParsedString = {
  value: string;
  endIndex: number;
  hasInterpolation: boolean;
};

export function normalizeText(value: string): string {
  return value
    .replace(/&(nbsp|lt|gt|amp);/gi, (_, entity: string) => {
      const normalizedEntity = entity.toLowerCase();
      if (normalizedEntity === 'nbsp') return ' ';
      if (normalizedEntity === 'lt') return '<';
      if (normalizedEntity === 'gt') return '>';
      return '&';
    })
    .replace(/\s+/g, ' ')
    .trim();
}

export function hasVisibleLetters(value: string): boolean {
  return /[A-Za-z\u3400-\u9FFF]/.test(value);
}

export function hasCjk(value: string): boolean {
  return /[\u3400-\u9FFF]/.test(value);
}

export function isLikelyI18nKey(value: string): boolean {
  if (!value || !isAsciiLowercase(value[0])) return false;

  let segmentStart = 0;
  let hasSeparator = false;

  for (let index = 0; index < value.length; index += 1) {
    const char = value[index];

    if (char === '.' || char === '-') {
      if (index === 0 || index === value.length - 1 || !isAsciiAlphanumeric(value[index + 1])) return false;
      hasSeparator = true;
      segmentStart = index + 1;
      continue;
    }

    if (index === segmentStart) {
      if (segmentStart === 0 ? !isAsciiAlphanumeric(char) : !isAsciiAlphanumeric(char)) return false;
      continue;
    }

    if (segmentStart === 0 ? !isAsciiAlphanumeric(char) : !isAsciiWord(char)) return false;
  }

  return hasSeparator;
}

function isAsciiLowercase(char: string): boolean {
  const code = char.charCodeAt(0);
  return code >= 97 && code <= 122;
}

function isAsciiAlphanumeric(char: string): boolean {
  const code = char.charCodeAt(0);
  return (code >= 48 && code <= 57) || (code >= 65 && code <= 90) || (code >= 97 && code <= 122);
}

function isAsciiWord(char: string): boolean {
  return isAsciiAlphanumeric(char) || char === '_';
}

export function isTechnicalString(value: string): boolean {
  const text = normalizeText(value);

  if (!hasVisibleLetters(text)) return true;
  if (KNOWN_NON_I18N_NAMES.has(text)) return true;
  if (TECHNICAL_UNITS.has(text)) return true;
  if (isLikelyI18nKey(text)) return true;
  if (/^(?:GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)$/.test(text)) return true;
  if (/^(?:https?:|wss?:|data:|mailto:|\/|\.\/|\.\.\/|@\/)/.test(text)) return true;
  if (/^#[0-9a-fA-F]{3,8}$/.test(text) || /^var\(--[\w-]+\)$/.test(text) || /^--[\w-]+$/.test(text)) return true;
  if (/^(?:\d+(?:\.\d+)?(?:px|r?em|%|vh|vw|s|ms)?|auto|none|block|inline|flex|grid)$/.test(text)) return true;
  if (/(?:sans-serif|serif|monospace|Arial|Helvetica|PingFang|Microsoft YaHei|font-family)/i.test(text)) return true;
  if (/^[A-Z0-9_./:-]+$/.test(text)) return true;
  if (/^[a-z0-9]+(?:[-_:/][a-z0-9]+)+$/.test(text)) return true;
  if (/^[A-Za-z]+Icon$/.test(text) || /^[A-Za-z]+(?:Outlined|Filled)$/.test(text)) return true;

  return false;
}

export function preserveLineStructure(source: string): string {
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
      if (char === quote) quote = null;
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

export function parseStringLiteral(source: string, quoteIndex: number): ParsedString | null {
  const quote = source[quoteIndex];
  if (quote !== '"' && quote !== "'" && quote !== '`') return null;

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
    if (quote === '`' && char === '$' && source[index + 1] === '{') hasInterpolation = true;
    if (char === quote) return { value, endIndex: index + 1, hasInterpolation };
    value += char;
    index += 1;
  }

  return null;
}

export function buildLineIndex(source: string): number[] {
  const lines = [0];
  for (let index = 0; index < source.length; index += 1) {
    if (source[index] === '\n') lines.push(index + 1);
  }
  return lines;
}

export function positionForIndex(lineStarts: number[], index: number): { line: number; column: number } {
  let low = 0;
  let high = lineStarts.length - 1;
  while (low <= high) {
    const mid = Math.floor((low + high) / 2);
    if (lineStarts[mid] <= index) low = mid + 1;
    else high = mid - 1;
  }
  const lineStart = lineStarts[high] ?? 0;
  return { line: high + 1, column: index - lineStart + 1 };
}
