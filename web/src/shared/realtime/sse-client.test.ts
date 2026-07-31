import { afterEach, describe, expect, it, vi } from 'vitest';

import { openRealtimeTopicEventStream } from './sse-client';

describe('openRealtimeTopicEventStream', () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

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

  it('does not retry ticket errors for non-retryable status codes', async () => {
    vi.useFakeTimers();
    const issueTicket = vi.fn().mockRejectedValue({ status: 403 });
    const onStateChange = vi.fn();

    const controller = openRealtimeTopicEventStream({
      topic: 'platform.update.operations.update-1',
      issueTicket,
      onStateChange,
    });
    await vi.waitFor(() => expect(onStateChange).toHaveBeenCalledWith('error'));
    await vi.advanceTimersByTimeAsync(30_000);

    expect(issueTicket).toHaveBeenCalledTimes(1);
    controller.close();
  });

  it('backs off successive empty streams instead of resetting to a hot loop', async () => {
    vi.useFakeTimers();
    const issueTicket = vi.fn().mockResolvedValue({ sse_url: '/empty' });
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(() =>
        Promise.resolve(
          new Response(
            new ReadableStream({
              start(stream) {
                stream.close();
              },
            }),
            { status: 200 },
          ),
        ),
      ),
    );

    const controller = openRealtimeTopicEventStream({
      topic: 'platform.update.operations.update-1',
      issueTicket,
    });
    await vi.advanceTimersByTimeAsync(0);
    expect(issueTicket).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(1000);
    expect(issueTicket).toHaveBeenCalledTimes(2);
    await vi.advanceTimersByTimeAsync(1999);
    expect(issueTicket).toHaveBeenCalledTimes(2);
    await vi.advanceTimersByTimeAsync(1);
    expect(issueTicket).toHaveBeenCalledTimes(3);
    controller.close();
  });

  it('drops events without data, frames chunks, and flushes a final unterminated event', async () => {
    const encoder = new TextEncoder();
    const body = new ReadableStream<Uint8Array>({
      start(stream) {
        stream.enqueue(encoder.encode('event: message\n\n'));
        stream.enqueue(encoder.encode('event: message\ndata: {"data":{"status":"PULL'));
        stream.enqueue(encoder.encode('ING"}}\n\n'));
        stream.enqueue(encoder.encode('data: {"data":{"status":"DONE"}}'));
        stream.close();
      },
    });
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(body, { status: 200 })));
    const onMessage = vi.fn();
    const controller = openRealtimeTopicEventStream({
      topic: 'platform.update.operations.update-1',
      issueTicket: vi.fn().mockResolvedValue({ sse_url: '/stream' }),
      onMessage,
    });

    await vi.waitFor(() => expect(onMessage).toHaveBeenCalledTimes(2));
    expect(onMessage).toHaveBeenNthCalledWith(1, { status: 'PULLING' });
    expect(onMessage).toHaveBeenNthCalledWith(2, { status: 'DONE' });
    controller.close();
  });
});
