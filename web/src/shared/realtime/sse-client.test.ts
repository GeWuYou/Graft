import { afterEach, describe, expect, it, vi } from 'vitest';

import { openRealtimeTopicEventStream } from './sse-client';

describe('openRealtimeTopicEventStream', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('uses a server-issued ticket URL for SSE without forwarding bearer headers', async () => {
    const encoder = new TextEncoder();
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(
          encoder.encode(
            'event: message\ndata: {"topic":"platform.update.operations.update-1","data":{"status":"PULLING"},"occurred_at":"2026-07-31T00:00:00Z"}\n\n',
          ),
        );
      },
    });
    const fetchMock = vi.fn().mockResolvedValue(new Response(body, { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);
    const issueTicket = vi.fn().mockResolvedValue({
      topic: 'platform.update.operations.update-1',
      ticket: 'opaque-ticket',
      websocket_url: '/ws?topic=platform.update.operations.update-1&ticket=opaque-ticket',
      sse_url: '/sse?topic=platform.update.operations.update-1&ticket=opaque-ticket',
      expires_at: '2026-07-31T00:00:00Z',
    });
    const onMessage = vi.fn();

    const controller = openRealtimeTopicEventStream({
      topic: 'platform.update.operations.update-1',
      issueTicket,
      onMessage,
    });
    await vi.waitFor(() => expect(onMessage).toHaveBeenCalledWith({ status: 'PULLING' }));
    controller.close();

    expect(issueTicket).toHaveBeenCalledWith('platform.update.operations.update-1');
    expect(fetchMock).toHaveBeenCalledWith(
      '/sse?topic=platform.update.operations.update-1&ticket=opaque-ticket',
      expect.objectContaining({ credentials: 'include', signal: expect.any(AbortSignal) }),
    );
    expect(fetchMock.mock.calls[0]?.[1]?.headers).toBeUndefined();
  });
});
