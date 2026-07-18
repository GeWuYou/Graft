import { describe, expect, it } from 'vitest';

import { dockerImageQueryKeys } from './docker-image-queries';

describe('docker image query keys', () => {
  it('separates server pagination and keyword requests', () => {
    expect(dockerImageQueryKeys.list({ pageSize: 20, offset: 40, keyword: 'graft' })).toEqual([
      'container',
      'images',
      { pageSize: 20, offset: 40, keyword: 'graft' },
    ]);
  });
});
