import { describe, expect, it } from 'vitest';
import { reactive } from 'vue';

import { dockerImageQueryKeys } from './docker-image-queries';

describe('docker image query keys', () => {
  it('separates server pagination and keyword requests', () => {
    expect(dockerImageQueryKeys.list({ pageSize: 20, offset: 40, keyword: 'graft' })).toEqual([
      'container',
      'images',
      { pageSize: 20, offset: 40, keyword: 'graft' },
    ]);
  });

  it('snapshots reactive query fields instead of retaining the mutable object', () => {
    const query = reactive({ pageSize: 20, offset: 0, keyword: '', unused: true });
    const key = dockerImageQueryKeys.list(query);

    query.offset = 20;

    expect(key[2]).toEqual({ pageSize: 20, offset: 0, keyword: '', unused: true });
    expect(key[2]).not.toBe(query);
    expect(dockerImageQueryKeys.list(query)[2].offset).toBe(20);
  });
});
