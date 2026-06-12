// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

const MARKDOWN_TABLE_SEPARATOR = /^\s*\|?(?:\s*:?-{3,}:?\s*\|)+\s*$/u;

export function markdownToPlainTextSummary(source: string | null | undefined, maxLength = 160) {
  const normalized = (source ?? '')
    .replace(/```[\s\S]*?```/gu, (block) =>
      block
        .replace(/```[a-z0-9_-]*\n?/iu, '')
        .replace(/```$/u, '')
        .trim(),
    )
    .replace(/`([^`]+)`/gu, '$1')
    .replace(/!\[([^\]]*)\]\([^)]+\)/gu, '$1')
    .replace(/\[([^\]]+)\]\([^)]+\)/gu, '$1')
    .replace(/^\s{0,3}#{1,6}\s+/gmu, '')
    .replace(/^\s{0,3}>\s?/gmu, '')
    .replace(/^\s*[-*+]\s+/gmu, '')
    .replace(/^\s*\d+[.)]\s+/gmu, '')
    .replace(/[*_~]+/gu, '')
    .split('\n')
    .filter((line) => !MARKDOWN_TABLE_SEPARATOR.test(line))
    .join(' ')
    .replace(/\|/gu, ' ')
    .replace(/\s+/gu, ' ')
    .trim();

  if (normalized.length <= maxLength) {
    return normalized;
  }

  return `${normalized.slice(0, Math.max(0, maxLength - 3)).trimEnd()}...`;
}
