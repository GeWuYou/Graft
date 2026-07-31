import { beforeEach, describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { postRealtimeSubscription, REALTIME_API_PATH } from './api';

vi.mock('@/utils/request', () => ({
  request: {
    post: vi.fn(),
  },
}));

describe('shared realtime api', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('issues realtime subscription tickets through the canonical subscription path', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce({
      topic: 'container.stats:container-1',
      ticket: 'opaque-ticket',
      websocket_url: '/ws?topic=container.stats%3Acontainer-1&ticket=opaque-ticket',
      sse_url: '/sse?topic=container.stats%3Acontainer-1&ticket=opaque-ticket',
      expires_at: '2026-06-24T08:00:30Z',
    } as never);

    await postRealtimeSubscription({ topic: 'container.stats:container-1' });

    expect(requestPost).toHaveBeenCalledWith({
      url: REALTIME_API_PATH.SUBSCRIPTIONS,
      data: { topic: 'container.stats:container-1' },
    });
  });
});
