import { computed, ref, shallowRef } from 'vue';

import {
  isRealtimePlatformAvailable,
  registerRealtimeAvailabilityController,
} from '@/shared/realtime/platform-availability';

import type {
  TerminalClientMessage,
  TerminalConnectionState,
  TerminalLifecycleCloseReason,
  TerminalResizePayload,
  TerminalServerMessage,
  TerminalSessionConnector,
} from './terminal-types';

type UseTerminalSessionOptions = {
  connector: TerminalSessionConnector;
  onMessage?: (message: TerminalServerMessage) => void;
  onOpen?: () => void;
  onClose?: (reason: TerminalLifecycleCloseReason) => void;
  onStateChange?: (state: TerminalConnectionState) => void;
  onTransportError?: (error: Error) => void;
};

/**
 * 创建并管理终端 WebSocket 会话。
 *
 * 连接建立使用递增 ID 隔离旧请求；断开或重连后，迟到的 ticket、socket 事件和错误不得回写当前会话。
 * `onTransportError` 只接收传输层错误，业务消息错误仍通过 `onMessage` 交给调用方处理。
 */
export function useTerminalSession(options: UseTerminalSessionOptions) {
  const socket = shallowRef<WebSocket | null>(null);
  const state = ref<TerminalConnectionState>('idle');
  const lastError = ref<string>('');
  let activeConnectionId = 0;
  let activeClose: ((reason: TerminalLifecycleCloseReason) => void) | null = null;
  let resumeSize: TerminalResizePayload | null = null;
  let suspendedByPlatform = false;

  const isConnected = computed(() => state.value === 'connected');

  function setState(nextState: TerminalConnectionState) {
    state.value = nextState;
    options.onStateChange?.(nextState);
  }

  function isActiveSocket(nextSocket: WebSocket, connectionId: number) {
    return connectionId === activeConnectionId && socket.value === nextSocket;
  }

  async function connect(initialSize: TerminalResizePayload) {
    resumeSize = initialSize;
    if (!isRealtimePlatformAvailable()) {
      setState('disconnected');
      return;
    }
    disconnect('manual_disconnect');
    setState('connecting');
    lastError.value = '';
    const connectionId = ++activeConnectionId;

    try {
      const opened = await options.connector.open({
        cols: initialSize.cols,
        rows: initialSize.rows,
      });
      if (connectionId !== activeConnectionId) {
        return;
      }
      const nextSocket = opened.protocols?.length
        ? new WebSocket(opened.url, opened.protocols)
        : new WebSocket(opened.url);
      socket.value = nextSocket;
      let didClose = false;
      let closeReason: TerminalLifecycleCloseReason = 'remote_close';

      const finalizeClose = (reason: TerminalLifecycleCloseReason) => {
        if (didClose) {
          return;
        }
        didClose = true;
        nextSocket.onopen = null;
        nextSocket.onmessage = null;
        nextSocket.onerror = null;
        nextSocket.onclose = null;
        if (socket.value === nextSocket) {
          socket.value = null;
        }
        if (activeClose === finalizeClose) {
          activeClose = null;
        }
        if (connectionId === activeConnectionId) {
          if (reason === 'component_unmount') {
            setState('idle');
          } else if (state.value !== 'error') {
            setState('disconnected');
          }
        }
        options.onClose?.(reason);
      };
      activeClose = finalizeClose;

      nextSocket.onopen = () => {
        if (!isActiveSocket(nextSocket, connectionId)) {
          return;
        }
        setState('connected');
        options.onOpen?.();
      };

      nextSocket.onmessage = (event) => {
        if (!isActiveSocket(nextSocket, connectionId)) {
          return;
        }
        const payload = parseServerMessage(event.data);
        if (!payload) {
          return;
        }
        if (payload.type === 'status') {
          setState(payload.state === 'connected' ? 'connected' : 'disconnected');
        }
        if (payload.type === 'error') {
          lastError.value = payload.message;
        }
        options.onMessage?.(payload);
      };

      nextSocket.onerror = () => {
        if (!isActiveSocket(nextSocket, connectionId)) {
          return;
        }
        const error = new Error('Terminal transport error');
        closeReason = 'session_error';
        lastError.value = error.message;
        setState('error');
        options.onTransportError?.(error);
      };

      nextSocket.onclose = () => {
        if (!isActiveSocket(nextSocket, connectionId)) {
          return;
        }
        finalizeClose(closeReason === 'remote_close' && state.value === 'error' ? 'session_error' : closeReason);
      };
    } catch (error) {
      if (connectionId !== activeConnectionId) {
        return;
      }
      const normalized = normalizeError(error, 'Failed to create terminal session');
      lastError.value = normalized.message;
      setState('error');
      options.onTransportError?.(normalized);
      options.onClose?.('connect_error');
      throw normalized;
    }
  }

  function disconnect(reason: TerminalLifecycleCloseReason = 'manual_disconnect') {
    if (!suspendedByPlatform && reason !== 'component_unmount') {
      resumeSize = null;
    }
    if (state.value === 'connecting') {
      activeConnectionId += 1;
    }
    const current = socket.value;
    socket.value = null;
    if (current && current.readyState === WebSocket.OPEN) {
      current.onopen = null;
      current.onmessage = null;
      current.onerror = null;
      current.onclose = null;
      try {
        current.close(1000, reason);
      } catch (error) {
        const normalized = normalizeError(error, 'Failed to close terminal session');
        lastError.value = normalized.message;
        options.onTransportError?.(normalized);
      }
    } else if (current && current.readyState < WebSocket.CLOSING) {
      current.onopen = null;
      current.onmessage = null;
      current.onerror = null;
      current.onclose = null;
      try {
        current.close();
      } catch (error) {
        const normalized = normalizeError(error, 'Failed to close terminal session');
        lastError.value = normalized.message;
        options.onTransportError?.(normalized);
      }
    }
    if (current) {
      activeClose?.(reason);
      return;
    }
    if (state.value === 'idle') {
      return;
    }
    setState(reason === 'component_unmount' ? 'idle' : 'disconnected');
    options.onClose?.(reason);
  }

  function sendInput(data: string) {
    sendMessage({ type: 'input', data });
  }

  function sendResize(payload: TerminalResizePayload) {
    sendMessage({ type: 'resize', cols: payload.cols, rows: payload.rows });
  }

  function sendPing() {
    sendMessage({ type: 'ping' });
  }

  function sendMessage(message: TerminalClientMessage) {
    if (!socket.value || socket.value.readyState !== WebSocket.OPEN) {
      return;
    }
    try {
      socket.value.send(JSON.stringify(message));
    } catch (error) {
      const normalized = normalizeError(error, 'Failed to send terminal message');
      lastError.value = normalized.message;
      setState('error');
      options.onTransportError?.(normalized);
    }
  }

  const unregisterAvailability = registerRealtimeAvailabilityController({
    close: () => {
      suspendedByPlatform = true;
      disconnect('manual_disconnect');
    },
    reconnect: () => {
      const size = resumeSize;
      suspendedByPlatform = false;
      if (size) void connect(size);
    },
  });

  function dispose() {
    unregisterAvailability();
    resumeSize = null;
    disconnect('component_unmount');
  }

  return {
    connect,
    disconnect,
    dispose,
    isConnected,
    lastError,
    sendInput,
    sendPing,
    sendResize,
    socket,
    state,
  };
}

function parseServerMessage(raw: unknown): TerminalServerMessage | null {
  if (typeof raw !== 'string') {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as TerminalServerMessage;
    if (!parsed || typeof parsed !== 'object' || typeof parsed.type !== 'string') {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

function normalizeError(error: unknown, fallback: string) {
  if (error instanceof Error) {
    return error;
  }
  return new Error(fallback);
}
