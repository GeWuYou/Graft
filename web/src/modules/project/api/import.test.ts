import { beforeEach, describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { APPLICATION_API_PATH } from '../contract/paths';
import { getApplicationImportRuntimeCandidates } from './import';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('project import api', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('reads runtime import candidates through the canonical query path', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({
      items: [],
      total: 0,
      limit: 10,
      offset: 0,
      filter_counts: { all: 0, ready: 0, unavailable: 0 },
    } as never);

    await getApplicationImportRuntimeCandidates({
      keyword: 'demo',
      availability: 'unavailable',
      limit: 10,
      offset: 20,
    });

    expect(requestGet).toHaveBeenCalledWith({
      url: APPLICATION_API_PATH.IMPORT_RUNTIME_CANDIDATES,
      params: {
        keyword: 'demo',
        availability: 'unavailable',
        limit: 10,
        offset: 20,
      },
    });
  });
});
