import { afterEach, describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { UPDATE_API_PATH } from '../contract/paths';
import { checkForUpdates, getUpdateStatus } from './update';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('platform update api', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('reads the current update discovery status from the canonical endpoint', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ current_version: '0.9.1' } as never);

    await getUpdateStatus();

    expect(requestGet).toHaveBeenCalledWith({ url: UPDATE_API_PATH.STATUS });
  });

  it('requests a discovery refresh without submitting an update payload', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce({ current_version: '0.9.1' } as never);

    await checkForUpdates();

    expect(requestPost).toHaveBeenCalledWith({ url: UPDATE_API_PATH.CHECK });
  });
});
