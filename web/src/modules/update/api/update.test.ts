import { afterEach, describe, expect, it, vi } from 'vitest';

import { openRealtimeTopicEventStream } from '@/shared/realtime/sse-client';
import { request } from '@/utils/request';

import { UPDATE_API_PATH } from '../contract/paths';
import { buildUpdateOperationTopicName } from '../contract/realtime';
import {
  checkForUpdates,
  createUpdateOperation,
  getUpdateOperation,
  getUpdateOperationDiagnostic,
  getUpdateOperations,
  getUpdateStatus,
  subscribeToUpdateOperation,
} from './update';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));
vi.mock('@/shared/realtime/sse-client', () => ({
  openRealtimeTopicEventStream: vi.fn(() => ({ close: vi.fn(), reconnect: vi.fn() })),
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

  it('uses the canonical operation collection for history and controlled upgrade submission', async () => {
    const requestGet = vi.mocked(request.get);
    const requestPost = vi.mocked(request.post);
    requestGet.mockResolvedValueOnce([] as never);
    requestPost.mockResolvedValueOnce({ operation_id: 'update-1' } as never);

    await getUpdateOperations();
    await createUpdateOperation({
      target_version: '1.1.0',
      compose_candidate_key: 'candidate-1',
    });

    expect(requestGet).toHaveBeenCalledWith({ url: UPDATE_API_PATH.OPERATIONS, params: { limit: 20 } });
    expect(requestPost).toHaveBeenCalledWith({
      url: UPDATE_API_PATH.OPERATIONS,
      data: { target_version: '1.1.0', compose_candidate_key: 'candidate-1' },
    });
  });

  it('reads operation progress and its controlled diagnostic through operation-scoped endpoints', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValue({ operation_id: 'update-1' } as never);

    await getUpdateOperation('update-1');
    await getUpdateOperationDiagnostic('update-1');

    expect(requestGet).toHaveBeenNthCalledWith(1, { url: '/api/platform/updates/operations/update-1' });
    expect(requestGet).toHaveBeenNthCalledWith(2, {
      url: '/api/platform/updates/operations/update-1/diagnostic',
    });
  });

  it('subscribes through the canonical realtime topic rather than an operation polling endpoint', () => {
    const onOperation = vi.fn();

    subscribeToUpdateOperation('update-1', { onOperation });

    expect(openRealtimeTopicEventStream).toHaveBeenCalledWith(
      expect.objectContaining({ topic: 'platform.update.operations.update-1', onMessage: onOperation }),
    );
  });

  it('builds the operation topic from the module realtime contract', () => {
    expect(buildUpdateOperationTopicName('update-1')).toBe('platform.update.operations.update-1');
  });
});
