import type { RealtimeSubscriptionResponse } from './api';
import { postRealtimeSubscription } from './api';
import { toRealtimeWebSocketUrl } from './url';

type RealtimeSocketState = 'idle' | 'connecting' | 'open' | 'closed' | 'error';

const NON_RETRYABLE_STATUS_CODES = new Set([400, 401, 403, 404]);
const RECONNECT_DELAYS_MS = [1000, 2000, 4000, 8000, 16000] as const;
const MAX_RECONNECT_ATTEMPTS = RECONNECT_DELAYS_MS.length;
const MAX_RECONNECT_ERROR_MESSAGE = 'Realtime reconnect stopped after maximum retry attempts';

type OpenRealtimeTopicSocketOptions<TMessage> = {
  topic: string;
  issueTicket?: (topic: string) => Promise<RealtimeSubscriptionResponse>;
  onMessage?: (message: TMessage) => void;
  onStateChange?: (state: RealtimeSocketState) => void;
  onError?: (message: string) => void;
  parseMessage?: (raw: unknown) => TMessage | null;
};

export type RealtimeTopicSocketController = {
  close: () => void;
  reconnect: () => void;
};

function defaultParseMessage<TMessage>(raw: unknown) {
  if (typeof raw !== 'string') {
    return null;
  }
  try {
    return JSON.parse(raw) as TMessage;
  } catch {
    return null;
  }
}

function hasStatusCode(error: unknown): error is { status: number } {
  return Boolean(error && typeof error === 'object' && typeof (error as { status?: unknown }).status === 'number');
}

function isRetryableTicketError(error: unknown) {
  return !hasStatusCode(error) || !NON_RETRYABLE_STATUS_CODES.has(error.status);
}

export function openRealtimeTopicSocket<TMessage>(
  options: OpenRealtimeTopicSocketOptions<TMessage>,
): RealtimeTopicSocketController {
  let socket: WebSocket | null = null;
  let reconnectTimer: number | null = null;
  let reconnectAttempts = 0;
  let closed = false;
  let connectionId = 0;

  function clearReconnectTimer() {
    if (reconnectTimer !== null) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
  }

  function emitState(nextState: RealtimeSocketState) {
    options.onStateChange?.(nextState);
  }

  function resetReconnectAttempts() {
    reconnectAttempts = 0;
  }

  function scheduleReconnect(terminalErrorMessage?: string) {
    clearReconnectTimer();
    if (closed) {
      return false;
    }
    if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      if (terminalErrorMessage) {
        options.onError?.(terminalErrorMessage);
      }
      return false;
    }

    const delay = RECONNECT_DELAYS_MS[Math.min(reconnectAttempts, RECONNECT_DELAYS_MS.length - 1)];
    reconnectAttempts += 1;
    reconnectTimer = window.setTimeout(() => {
      void connect();
    }, delay);
    return true;
  }

  async function connect() {
    clearReconnectTimer();
    if (closed) {
      return;
    }

    const currentConnectionId = ++connectionId;
    emitState('connecting');
    try {
      const issued = await (options.issueTicket?.(options.topic) ?? postRealtimeSubscription({ topic: options.topic }));
      if (closed || currentConnectionId !== connectionId) {
        return;
      }
      const nextSocket = new WebSocket(toRealtimeWebSocketUrl(issued.websocket_url));
      socket = nextSocket;

      nextSocket.onopen = () => {
        if (socket !== nextSocket || closed || currentConnectionId !== connectionId) {
          return;
        }
        resetReconnectAttempts();
        emitState('open');
      };

      nextSocket.onmessage = (event) => {
        if (socket !== nextSocket || closed || currentConnectionId !== connectionId) {
          return;
        }
        const parser = options.parseMessage ?? defaultParseMessage<TMessage>;
        const parsed = parser(event.data);
        if (!parsed) {
          return;
        }
        options.onMessage?.(parsed);
      };

      nextSocket.onerror = () => {
        if (socket !== nextSocket || closed || currentConnectionId !== connectionId) {
          return;
        }
        emitState('error');
        options.onError?.('WebSocket transport error');
      };

      nextSocket.onclose = () => {
        if (socket !== nextSocket || currentConnectionId !== connectionId) {
          return;
        }
        socket = null;
        if (closed) {
          emitState('idle');
          return;
        }
        emitState('closed');
        scheduleReconnect(MAX_RECONNECT_ERROR_MESSAGE);
      };
    } catch (error) {
      if (closed || currentConnectionId !== connectionId) {
        return;
      }
      emitState('error');
      options.onError?.(error instanceof Error ? error.message : 'Failed to issue realtime subscription ticket');
      if (isRetryableTicketError(error)) {
        scheduleReconnect();
      }
    }
  }

  function close() {
    closed = true;
    clearReconnectTimer();
    if (socket) {
      socket.onopen = null;
      socket.onmessage = null;
      socket.onerror = null;
      socket.onclose = null;
      if (socket.readyState < WebSocket.CLOSING) {
        socket.close();
      }
    }
    socket = null;
    emitState('idle');
  }

  function reconnect() {
    closed = false;
    connectionId += 1;
    clearReconnectTimer();
    resetReconnectAttempts();
    if (socket && socket.readyState < WebSocket.CLOSING) {
      socket.close();
    }
    socket = null;
    void connect();
  }

  void connect();

  return {
    close,
    reconnect,
  };
}
