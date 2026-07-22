import { afterEach, describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { UPDATE_API_PATH } from '../contract/paths';
import { checkForUpdates, createUpdateOperation, getUpdateOperations, getUpdateStatus } from './update';

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

  it('uses the canonical operation collection for history and exact confirmation', async () => {
    const requestGet = vi.mocked(request.get);
    const requestPost = vi.mocked(request.post);
    requestGet.mockResolvedValueOnce([] as never);
    requestPost.mockResolvedValueOnce({ operation_id: 'update-1' } as never);

    await getUpdateOperations();
    await createUpdateOperation({ target_version: '1.1.0', confirmation: '1.1.0' });

    expect(requestGet).toHaveBeenCalledWith({ url: UPDATE_API_PATH.OPERATIONS, params: { limit: 20 } });
    expect(requestPost).toHaveBeenCalledWith({
      url: UPDATE_API_PATH.OPERATIONS,
      data: { target_version: '1.1.0', confirmation: '1.1.0' },
    });
  });
});
