import { describe, expect, it, vi } from 'vitest';

const createHighlighter = vi.hoisted(() => vi.fn());

vi.mock('shiki', () => ({ createHighlighter }));

import { renderHighlightedCodeBlock } from './shiki';

describe('renderHighlightedCodeBlock', () => {
  it('retries highlighter creation after a rejected singleton promise', async () => {
    const highlighter = { codeToHtml: vi.fn(() => '<pre>ok</pre>') };
    createHighlighter.mockRejectedValueOnce(new Error('initialization failed')).mockResolvedValueOnce(highlighter);

    await expect(renderHighlightedCodeBlock({ code: 'echo ok', lang: 'shell', themeMode: 'light' })).rejects.toThrow(
      'initialization failed',
    );
    await expect(renderHighlightedCodeBlock({ code: 'echo ok', lang: 'shell', themeMode: 'light' })).resolves.toBe(
      '<pre>ok</pre>',
    );
    expect(createHighlighter).toHaveBeenCalledTimes(2);
  });
});
