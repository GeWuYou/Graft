// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import MarkdownIt from 'markdown-it';
import type Token from 'markdown-it/lib/token.mjs';

const MARKDOWN_TABLE_SEPARATOR = /^\s*\|?(?:\s*:?-{3,}:?\s*\|)+\s*$/u;
const markdown = new MarkdownIt({ html: false, linkify: false, typographer: false });

export function markdownToPlainTextSummary(source: string | null | undefined, maxLength = 160) {
  const normalized = markdown
    .parse(source ?? '', {})
    .flatMap((token) => tokenToSummaryText(token))
    .join(' ')
    .replace(/\s+/gu, ' ')
    .trim();

  if (normalized.length <= maxLength) {
    return normalized;
  }

  return `${normalized.slice(0, Math.max(0, maxLength - 3)).trimEnd()}...`;
}

function tokenToSummaryText(token: Token) {
  if (token.type === 'fence' || token.type === 'code_block') {
    return [];
  }

  if (token.type === 'inline') {
    return inlineTokenToText(token);
  }

  if (token.type === 'td_open' || token.type === 'th_open') {
    return [' '];
  }

  if (token.type === 'tr_close') {
    return [' '];
  }

  if (MARKDOWN_TABLE_SEPARATOR.test(token.content)) {
    return [];
  }

  return [];
}

function inlineTokenToText(token: Token) {
  if (!token.children?.length) {
    return [token.content];
  }

  return token.children.flatMap((child) => {
    if (child.type === 'image') {
      return [child.content || child.attrGet('alt') || ''];
    }

    if (child.type === 'text' || child.type === 'code_inline') {
      return [child.content];
    }

    return [];
  });
}
