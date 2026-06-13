// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { markdownToPlainTextSummary } from './summary';

describe('markdownToPlainTextSummary', () => {
  it('summarizes fenced code without leaking fence markers', () => {
    expect(markdownToPlainTextSummary('```ts\nconst answer = 42;\n```\nAfter')).toBe('After');
  });

  it('uses MarkdownIt inline parsing for links, code, images, and emphasis', () => {
    expect(markdownToPlainTextSummary('![Alt text](/x.png) [Docs](https://example.test) and `code` **bold**')).toBe(
      'Alt text Docs and code bold',
    );
  });
});
