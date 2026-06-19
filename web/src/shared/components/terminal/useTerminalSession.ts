// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { computed, ref } from 'vue';

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
 * @param options - 包含 connector 用于打开会话，以及监听连接状态和消息事件的回调函数
 * @returns 会话管理对象，提供建立/断开连接、发送消息和监控连接状态的接口
 */
export function useTerminalSession(options: UseTerminalSessionOptions) {
  const socket = ref<WebSocket | null>(null);
  const state = ref<TerminalConnectionState>('idle');
  const lastError = ref<string>('');

  const isConnected = computed(() => state.value === 'connected');

  function setState(nextState: TerminalConnectionState) {
    state.value = nextState;
    options.onStateChange?.(nextState);
  }

  async function connect(initialSize: TerminalResizePayload) {
    disconnect('manual_disconnect');
    setState('connecting');
    lastError.value = '';

    try {
      const opened = await options.connector.open({
        cols: initialSize.cols,
        rows: initialSize.rows,
      });
      const nextSocket = opened.protocols?.length
        ? new WebSocket(opened.url, opened.protocols)
        : new WebSocket(opened.url);
      socket.value = nextSocket;

      nextSocket.onopen = () => {
        setState('connected');
        options.onOpen?.();
      };

      nextSocket.onmessage = (event) => {
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
        const error = new Error('Terminal transport error');
        lastError.value = error.message;
        setState('error');
        options.onTransportError?.(error);
      };

      nextSocket.onclose = () => {
        socket.value = null;
        if (state.value !== 'error') {
          setState('disconnected');
        }
        options.onClose?.('remote_close');
      };
    } catch (error) {
      const normalized = normalizeError(error, 'Failed to create terminal session');
      lastError.value = normalized.message;
      setState('error');
      options.onTransportError?.(normalized);
      throw normalized;
    }
  }

  function disconnect(reason: TerminalLifecycleCloseReason = 'manual_disconnect') {
    const current = socket.value;
    socket.value = null;
    if (current && current.readyState === WebSocket.OPEN) {
      current.close(1000, reason);
    } else if (current && current.readyState < WebSocket.CLOSING) {
      current.close();
    }
    if (state.value !== 'idle') {
      setState(reason === 'component_unmount' ? 'idle' : 'disconnected');
    }
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
    socket.value.send(JSON.stringify(message));
  }

  return {
    connect,
    disconnect,
    isConnected,
    lastError,
    sendInput,
    sendPing,
    sendResize,
    socket,
    state,
  };
}

/**
 * 将 WebSocket 消息解析为终端服务器消息。
 *
 * @returns 如果输入有效则返回解析的消息，否则返回 `null`
 */
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

/**
 * 将未知值规范化为 Error 实例。
 *
 * @returns 如果 error 是 Error 实例则返回原值，否则返回以 fallback 消息创建的新 Error 实例。
 */
function normalizeError(error: unknown, fallback: string) {
  if (error instanceof Error) {
    return error;
  }
  return new Error(fallback);
}
