import type { RealtimeSubscriptionResponse } from './api';
import { postRealtimeSubscription } from './api';

type RealtimeEventStreamState = 'idle' | 'connecting' | 'open' | 'closed' | 'error';

const NON_RETRYABLE_STATUS_CODES = new Set([400, 401, 403, 404]);
const RECONNECT_DELAYS_MS = [1000, 2000, 4000, 8000, 10000] as const;

type OpenRealtimeTopicEventStreamOptions<TMessage> = {
  topic: string;
  issueTicket?: (topic: string) => Promise<RealtimeSubscriptionResponse>;
  onMessage?: (message: TMessage) => void | Promise<void>;
  onStateChange?: (state: RealtimeEventStreamState) => void;
  onError?: (message: string) => void;
  parseMessage?: (raw: unknown) => TMessage | null;
};

export type RealtimeTopicEventStreamController = {
  close: () => void;
  reconnect: () => void;
};

function defaultParseMessage<TMessage>(data: unknown) {
  return data as TMessage;
}

function hasStatusCode(error: unknown): error is { status: number } {
  return Boolean(error && typeof error === 'object' && typeof (error as { status?: unknown }).status === 'number');
}

function isRetryableTicketError(error: unknown) {
  return !hasStatusCode(error) || !NON_RETRYABLE_STATUS_CODES.has(error.status);
}

/**
 * 打开统一实时 SSE 主题连接，负责签发单次票据、解析事件和断线重连。
 *
 * SSE 请求只使用服务端签发的短期 URL；Bearer token 仅用于签发票据的普通 HTTP 请求，绝不进入流地址。
 */
export function openRealtimeTopicEventStream<TMessage>(
  options: OpenRealtimeTopicEventStreamOptions<TMessage>,
): RealtimeTopicEventStreamController {
  let streamAbort: AbortController | null = null;
  let reconnectTimer: number | null = null;
  let reconnectAttempt = 0;
  let closed = false;
  let connectionID = 0;

  function clearReconnectTimer() {
    if (reconnectTimer !== null) window.clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }

  function emitState(state: RealtimeEventStreamState) {
    options.onStateChange?.(state);
  }

  function scheduleReconnect() {
    if (closed) return;
    const delay = RECONNECT_DELAYS_MS[Math.min(reconnectAttempt, RECONNECT_DELAYS_MS.length - 1)];
    reconnectAttempt += 1;
    clearReconnectTimer();
    reconnectTimer = window.setTimeout(() => void connect(), delay);
  }

  async function connect() {
    if (closed) return;
    clearReconnectTimer();
    const currentConnectionID = ++connectionID;
    emitState('connecting');
    try {
      const issued = await (options.issueTicket?.(options.topic) ?? postRealtimeSubscription({ topic: options.topic }));
      if (closed || currentConnectionID !== connectionID) return;
      const controller = new AbortController();
      streamAbort = controller;
      const response = await fetch(issued.sse_url, { credentials: 'include', signal: controller.signal });
      if (!response.ok || !response.body) throw new Error(`SSE request failed with status ${response.status}`);
      if (closed || currentConnectionID !== connectionID) return;
      reconnectAttempt = 0;
      emitState('open');
      await consumeEvents(response.body, async (raw) => {
        if (closed || currentConnectionID !== connectionID) return;
        const data = parseRealtimeEventData(raw);
        if (data === null) return;
        const parsed = (options.parseMessage ?? defaultParseMessage<TMessage>)(data);
        if (parsed !== null) await options.onMessage?.(parsed);
      });
      if (controller.signal.aborted || closed || currentConnectionID !== connectionID) return;
      emitState('closed');
      scheduleReconnect();
    } catch (error) {
      if (closed || currentConnectionID !== connectionID || isAbortError(error)) return;
      emitState('error');
      options.onError?.(error instanceof Error ? error.message : 'Failed to open realtime event stream');
      if (isRetryableTicketError(error)) scheduleReconnect();
    } finally {
      if (currentConnectionID === connectionID) streamAbort = null;
    }
  }

  function close() {
    closed = true;
    connectionID += 1;
    clearReconnectTimer();
    streamAbort?.abort();
    streamAbort = null;
    emitState('idle');
  }

  function reconnect() {
    closed = false;
    connectionID += 1;
    clearReconnectTimer();
    reconnectAttempt = 0;
    streamAbort?.abort();
    streamAbort = null;
    void connect();
  }

  void connect();
  return { close, reconnect };
}

function parseRealtimeEventData(raw: string): unknown | null {
  try {
    const event = JSON.parse(raw) as { data?: unknown };
    return Object.hasOwn(event, 'data') ? event.data : null;
  } catch {
    return null;
  }
}

async function consumeEvents(body: ReadableStream<Uint8Array>, onMessage: (raw: string) => void | Promise<void>) {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let pending = '';
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) return;
      pending += decoder.decode(value, { stream: true });
      const events = pending.split(/\r?\n\r?\n/);
      pending = events.pop() ?? '';
      for (const event of events) {
        const data = event
          .split(/\r?\n/)
          .filter((line) => line.startsWith('data:'))
          .map((line) => line.slice(5).trimStart())
          .join('\n');
        if (data) await onMessage(data);
      }
    }
  } finally {
    try {
      await reader.cancel();
    } finally {
      reader.releaseLock();
    }
  }
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError';
}
