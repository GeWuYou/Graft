import { isTechnicalString, normalizeText, positionForIndex } from '../text-utils';
import type { I18nGovernanceRule, RuleViolation, SourceFile } from '../types';

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

function isOpeningTag(tagText: string): boolean {
  return !/^<\//.test(tagText) && !/\/>$/.test(tagText);
}

function tagName(tagText: string): string | null {
  return tagText.match(/^<\/?\s*([A-Za-z][\w:-]*)/)?.[1]?.toLowerCase() ?? null;
}

function isAriaHiddenTrueTag(tagText: string): boolean {
  return /aria-hidden\s*=\s*(?:"true"|'true'|true)/.test(tagText);
}

function isInsideAriaHiddenAncestor(source: string, rangeStart: number, textStart: number): boolean {
  const stack: Array<{ name: string; ariaHidden: boolean }> = [];
  let index = rangeStart;

  while (index < textStart) {
    const tagStart = source.indexOf('<', index);
    if (tagStart === -1 || tagStart >= textStart) break;
    if (source.startsWith('<!--', tagStart)) {
      const commentEnd = source.indexOf('-->', tagStart + 4);
      index = commentEnd === -1 ? textStart : commentEnd + 3;
      continue;
    }

    const tagEnd = findTagEnd(source, tagStart);
    if (tagEnd === -1 || tagEnd >= textStart) break;

    const tagText = source.slice(tagStart, tagEnd + 1);
    const name = tagName(tagText);
    if (!name) {
      index = tagEnd + 1;
      continue;
    }

    if (/^<\//.test(tagText)) {
      const lastIndex = stack.findLastIndex((tag) => tag.name === name);
      if (lastIndex >= 0) stack.splice(lastIndex);
    } else if (isOpeningTag(tagText)) {
      stack.push({ name, ariaHidden: isAriaHiddenTrueTag(tagText) });
    }

    index = tagEnd + 1;
  }

  return stack.some((tag) => tag.ariaHidden);
}

function addTemplateTextViolation(violations: RuleViolation[], file: SourceFile, index: number, value: string) {
  const text = normalizeText(value);
  if (text.length === 0 || isTechnicalString(text)) return;

  const position = positionForIndex(file.lineStarts, index);
  violations.push({
    ruleId: 'no-hardcoded-template-text',
    severity: 'error',
    filePath: file.relativePath,
    line: position.line,
    column: position.column,
    message: 'Hard-coded template text',
    excerpt: text,
    suggestion: "Move visible copy into locale catalogs and render it with t('...').",
  });
}

function collectTemplateTextViolations(file: SourceFile): RuleViolation[] {
  if (file.kind !== 'vue') return [];

  const violations: RuleViolation[] = [];
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

      if (isInsideAriaHiddenAncestor(file.source, block.start, textStart)) continue;
      addTemplateTextViolation(violations, file, textStart, file.source.slice(textStart, index));
    }
  }

  return violations;
}

export const noHardcodedTemplateTextRule: I18nGovernanceRule = {
  id: 'no-hardcoded-template-text',
  description: 'Blocks raw visible text nodes inside Vue templates.',
  defaultSeverity: 'error',
  appliesTo: ['vue'],
  check(context) {
    return context.sourceFiles.flatMap((file) => collectTemplateTextViolations(file));
  },
};
