import { describe, expect, it } from 'vitest';

import { clearQueryCache, queryClient } from './client';

describe('queryClient', () => {
  it('uses bounded defaults for server data and clears session-scoped snapshots', () => {
    queryClient.setQueryData(['test', 'session'], { value: true });

    expect(queryClient.getQueryData(['test', 'session'])).toEqual({ value: true });
    expect(queryClient.getDefaultOptions().queries?.staleTime).toBe(30_000);
    expect(queryClient.getDefaultOptions().queries?.gcTime).toBe(300_000);

    clearQueryCache();

    expect(queryClient.getQueryData(['test', 'session'])).toBeUndefined();
  });
});
