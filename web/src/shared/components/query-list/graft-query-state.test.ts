import { describe, expect, it, vi } from 'vitest';

import {
  graftQueryStorageKey,
  readGraftQueryState,
  resolveGraftQueryState,
  writeGraftQueryState,
} from './graft-query-state';

const defaults = { keyword: '', filters: {}, page: 1, pageSize: 20 };

describe('graft query state', () => {
  it('uses the documented URL to defaults restore order', () => {
    expect(
      resolveGraftQueryState({
        defaultState: defaults,
        recentState: { ...defaults, keyword: 'recent' },
        defaultViewState: { ...defaults, keyword: 'default' },
        urlState: { ...defaults, keyword: 'url', page: 3 },
      }).source,
    ).toBe('url');
    expect(
      resolveGraftQueryState({
        defaultState: defaults,
        recentState: { ...defaults, keyword: 'recent' },
        defaultViewState: { ...defaults, keyword: 'default' },
      }).state.keyword,
    ).toBe('default');
  });

  it('persists page size but not the current page and ignores malformed data', () => {
    const setItem = vi.fn();
    vi.stubGlobal('window', { localStorage: { getItem: vi.fn(() => '{bad'), setItem } });
    expect(readGraftQueryState('container.images')).toBeUndefined();
    writeGraftQueryState('container.images', { keyword: 'redis', filters: { unused: true }, page: 4, pageSize: 50 });
    expect(setItem).toHaveBeenCalledWith(graftQueryStorageKey('container.images'), expect.stringContaining('"page":1'));
    vi.unstubAllGlobals();
  });
});
