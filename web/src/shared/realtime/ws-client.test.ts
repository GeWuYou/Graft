import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { openRealtimeTopicSocket } from './ws-client';

function createApiRequestError(status: number, message: string) {
  const error = new Error(message) as Error & { status: number; isApiRequestError: true };
  error.name = 'ApiRequestError';
  error.status = status;
  error.isApiRequestError = true;
  return error;
}

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;

  readonly url: string;
  readyState = MockWebSocket.CONNECTING;
  onopen: (() => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  onclose: (() => void) | null = null;
  close = vi.fn(() => {
    this.readyState = MockWebSocket.CLOSING;
  });

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  emitOpen() {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.();
  }

  emitMessage(data: unknown) {
    this.onmessage?.({ data } as MessageEvent);
  }

  emitClose() {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.();
  }
}

describe('openRealtimeTopicSocket', () => {
  const originalWebSocket = globalThis.WebSocket;

  beforeEach(() => {
    MockWebSocket.instances = [];
    vi.useFakeTimers();
    globalThis.WebSocket = Object.assign(MockWebSocket, {
      CONNECTING: MockWebSocket.CONNECTING,
      OPEN: MockWebSocket.OPEN,
      CLOSING: MockWebSocket.CLOSING,
      CLOSED: MockWebSocket.CLOSED,
    }) as unknown as typeof WebSocket;
  });

  afterEach(() => {
    vi.useRealTimers();
    if (originalWebSocket) {
      globalThis.WebSocket = originalWebSocket;
    } else {
      Reflect.deleteProperty(globalThis, 'WebSocket');
    }
  });

  it('issues a ticket before opening the unified websocket url', async () => {
    const issueTicket = vi.fn().mockResolvedValue({
      topic: 'container.stats:container-1',
      ticket: 'opaque-ticket',
      websocket_url: '/ws?topic=container.stats%3Acontainer-1&ticket=opaque-ticket',
      expires_at: '2026-06-24T08:00:30Z',
    });
    const onMessage = vi.fn();

    openRealtimeTopicSocket({
      topic: 'container.stats:container-1',
      issueTicket,
      onMessage,
      parseMessage: (raw) => (typeof raw === 'string' ? (JSON.parse(raw) as { value: number }) : null),
    });

    await vi.runAllTicks();

    expect(issueTicket).toHaveBeenCalledWith('container.stats:container-1');
    expect(MockWebSocket.instances).toHaveLength(1);
    expect(MockWebSocket.instances[0]?.url).toContain('/ws?topic=container.stats%3Acontainer-1&ticket=opaque-ticket');

    MockWebSocket.instances[0]?.emitOpen();
    MockWebSocket.instances[0]?.emitMessage(JSON.stringify({ value: 42 }));

    expect(onMessage).toHaveBeenCalledWith({ value: 42 });
  });

  it('delivers valid falsy parsed messages', async () => {
    const issueTicket = vi.fn().mockResolvedValue({
      topic: 'container.stats:container-1',
      ticket: 'opaque-ticket',
      websocket_url: '/ws?topic=container.stats%3Acontainer-1&ticket=opaque-ticket',
      expires_at: '2026-06-24T08:00:30Z',
    });
    const onMessage = vi.fn();

    openRealtimeTopicSocket({
      topic: 'container.stats:container-1',
      issueTicket,
      onMessage,
      parseMessage: () => 0,
    });

    await vi.runAllTicks();

    MockWebSocket.instances[0]?.emitOpen();
    MockWebSocket.instances[0]?.emitMessage('ignored');

    expect(onMessage).toHaveBeenCalledWith(0);
  });

  it('backs off reconnect attempts after socket close and stops after the retry limit', async () => {
    const issueTicket = vi.fn().mockResolvedValue({
      topic: 'container.stats:container-1',
      ticket: 'opaque-ticket',
      websocket_url: '/ws?topic=container.stats%3Acontainer-1&ticket=opaque-ticket',
      expires_at: '2026-06-24T08:00:30Z',
    });
    const onError = vi.fn();

    openRealtimeTopicSocket({
      topic: 'container.stats:container-1',
      issueTicket,
      onError,
    });

    await vi.runAllTicks();
    expect(issueTicket).toHaveBeenCalledTimes(1);

    const delays = [1000, 2000, 4000, 8000, 16000];
    for (let index = 0; index < delays.length; index += 1) {
      MockWebSocket.instances.at(-1)?.emitClose();
      await vi.advanceTimersByTimeAsync(delays[index]! - 1);
      expect(issueTicket).toHaveBeenCalledTimes(index + 1);
      await vi.advanceTimersByTimeAsync(1);
      await vi.runAllTicks();
      expect(issueTicket).toHaveBeenCalledTimes(index + 2);
    }

    MockWebSocket.instances.at(-1)?.emitClose();
    await vi.runAllTicks();

    expect(issueTicket).toHaveBeenCalledTimes(6);
    expect(onError).toHaveBeenLastCalledWith('Realtime reconnect stopped after maximum retry attempts');
  });

  it('does not retry non-retryable ticket issuance failures', async () => {
    const issueTicket = vi.fn().mockRejectedValue(createApiRequestError(401, 'Unauthorized'));
    const onError = vi.fn();

    openRealtimeTopicSocket({
      topic: 'container.stats:container-1',
      issueTicket,
      onError,
    });

    await vi.runAllTicks();
    await vi.advanceTimersByTimeAsync(30000);

    expect(issueTicket).toHaveBeenCalledTimes(1);
    expect(onError).toHaveBeenCalledWith('Unauthorized');
  });

  it('retries ticket issuance failures with backoff until a later attempt succeeds', async () => {
    const issueTicket = vi
      .fn()
      .mockRejectedValueOnce(new Error('Temporary failure'))
      .mockRejectedValueOnce(new Error('Temporary failure'))
      .mockResolvedValue({
        topic: 'container.stats:container-1',
        ticket: 'opaque-ticket',
        websocket_url: '/ws?topic=container.stats%3Acontainer-1&ticket=opaque-ticket',
        expires_at: '2026-06-24T08:00:30Z',
      });
    const onStateChange = vi.fn();

    openRealtimeTopicSocket({
      topic: 'container.stats:container-1',
      issueTicket,
      onStateChange,
    });

    await vi.runAllTicks();
    expect(issueTicket).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(1000);
    await vi.runAllTicks();
    expect(issueTicket).toHaveBeenCalledTimes(2);

    await vi.advanceTimersByTimeAsync(2000);
    await vi.runAllTicks();
    expect(issueTicket).toHaveBeenCalledTimes(3);

    MockWebSocket.instances[0]?.emitOpen();
    expect(onStateChange).toHaveBeenLastCalledWith('open');
  });

  it('resets retry backoff after a successful connection and manual reconnect', async () => {
    const issueTicket = vi.fn().mockResolvedValue({
      topic: 'container.stats:container-1',
      ticket: 'opaque-ticket',
      websocket_url: '/ws?topic=container.stats%3Acontainer-1&ticket=opaque-ticket',
      expires_at: '2026-06-24T08:00:30Z',
    });

    const controller = openRealtimeTopicSocket({
      topic: 'container.stats:container-1',
      issueTicket,
    });

    await vi.runAllTicks();
    MockWebSocket.instances[0]?.emitOpen();
    MockWebSocket.instances[0]?.emitClose();

    await vi.advanceTimersByTimeAsync(1000);
    await vi.runAllTicks();
    expect(issueTicket).toHaveBeenCalledTimes(2);

    controller.reconnect();
    await vi.runAllTicks();
    expect(issueTicket).toHaveBeenCalledTimes(3);

    MockWebSocket.instances[2]?.emitClose();
    await vi.advanceTimersByTimeAsync(999);
    expect(issueTicket).toHaveBeenCalledTimes(3);
    await vi.advanceTimersByTimeAsync(1);
    await vi.runAllTicks();
    expect(issueTicket).toHaveBeenCalledTimes(4);
  });
});
