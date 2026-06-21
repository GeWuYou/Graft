import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { nextTick } from 'vue';

import type { TerminalResizePayload, TerminalSessionConnector } from './terminal-types';
import { useTerminalSession } from './useTerminalSession';

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;

  readonly url: string;
  readonly protocols?: string[];
  readyState = MockWebSocket.CONNECTING;
  onopen: (() => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  onclose: (() => void) | null = null;
  close = vi.fn((code?: number, reason?: string) => {
    this.closeCode = code;
    this.closeReason = reason;
    this.readyState = MockWebSocket.CLOSING;
  });
  send = vi.fn();
  closeCode?: number;
  closeReason?: string;

  constructor(url: string, protocols?: string | string[]) {
    this.url = url;
    this.protocols = Array.isArray(protocols) ? protocols : protocols ? [protocols] : undefined;
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

function createConnector() {
  const open = vi.fn<TerminalSessionConnector['open']>().mockResolvedValue({
    url: 'ws://terminal.example/ws',
  });
  const connector: TerminalSessionConnector = {
    open,
  };
  return { connector, open };
}

describe('useTerminalSession', () => {
  const originalWebSocket = globalThis.WebSocket;

  function createSize(overrides?: Partial<TerminalResizePayload>): TerminalResizePayload {
    return {
      cols: 120,
      rows: 32,
      ...overrides,
    };
  }

  beforeEach(() => {
    MockWebSocket.instances = [];
    globalThis.WebSocket = Object.assign(MockWebSocket, {
      CONNECTING: MockWebSocket.CONNECTING,
      OPEN: MockWebSocket.OPEN,
      CLOSING: MockWebSocket.CLOSING,
      CLOSED: MockWebSocket.CLOSED,
    }) as unknown as typeof WebSocket;
  });

  afterEach(() => {
    if (originalWebSocket) {
      globalThis.WebSocket = originalWebSocket;
      return;
    }
    Reflect.deleteProperty(globalThis, 'WebSocket');
  });

  it('ignores stale socket close events after reconnect', async () => {
    const { connector } = createConnector();
    const onClose = vi.fn();
    const session = useTerminalSession({
      connector,
      onClose,
    });

    await session.connect(createSize());
    const firstSocket = MockWebSocket.instances[0];
    firstSocket.emitOpen();
    firstSocket.emitMessage(JSON.stringify({ type: 'status', state: 'connected' }));

    await session.connect(createSize({ cols: 140 }));
    const secondSocket = MockWebSocket.instances[1];
    secondSocket.emitOpen();
    secondSocket.emitMessage(JSON.stringify({ type: 'status', state: 'connected' }));

    firstSocket.emitClose();

    expect(session.state.value).toBe('connected');
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenLastCalledWith('manual_disconnect');
  });

  it('reports manual disconnect once without remote close overwrite', async () => {
    const { connector } = createConnector();
    const onClose = vi.fn();
    const session = useTerminalSession({
      connector,
      onClose,
    });

    await session.connect(createSize());
    const socket = MockWebSocket.instances[0];
    socket.emitOpen();

    session.disconnect('manual_disconnect');
    socket.emitClose();

    expect(onClose).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledWith('manual_disconnect');
    expect(session.state.value).toBe('disconnected');
  });

  it('cancels an in-flight connect when disconnected before connector resolves', async () => {
    const deferredOpen: { resolve?: (value: { url: string }) => void } = {};
    const connector: TerminalSessionConnector = {
      open: vi.fn().mockImplementation(
        () =>
          new Promise((resolve) => {
            deferredOpen.resolve = resolve;
          }),
      ),
    };
    const onClose = vi.fn();
    const onTransportError = vi.fn();
    const session = useTerminalSession({
      connector,
      onClose,
      onTransportError,
    });

    const connecting = session.connect(createSize());
    expect(session.state.value).toBe('connecting');

    session.disconnect('component_unmount');
    expect(session.state.value).toBe('idle');
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledWith('component_unmount');

    if (typeof deferredOpen.resolve === 'function') {
      deferredOpen.resolve({ url: 'ws://terminal.example/ws' });
    }
    await connecting;
    await nextTick();

    expect(MockWebSocket.instances).toHaveLength(0);
    expect(onTransportError).not.toHaveBeenCalled();
    expect(session.state.value).toBe('idle');
  });

  it('emits connect_error when connector open rejects', async () => {
    const connector: TerminalSessionConnector = {
      open: vi.fn().mockRejectedValue(new Error('connect failed')),
    };
    const onClose = vi.fn();
    const onTransportError = vi.fn();
    const session = useTerminalSession({
      connector,
      onClose,
      onTransportError,
    });

    await expect(session.connect(createSize())).rejects.toThrow('connect failed');

    expect(onClose).toHaveBeenCalledWith('connect_error');
    expect(onTransportError).toHaveBeenCalledTimes(1);
    expect(session.state.value).toBe('error');
  });

  it('ignores stale connector errors after a new connect starts', async () => {
    const deferredReject: { reject?: (reason?: unknown) => void } = {};
    const connector: TerminalSessionConnector = {
      open: vi
        .fn()
        .mockImplementationOnce(
          () =>
            new Promise((_, reject) => {
              deferredReject.reject = reject;
            }),
        )
        .mockResolvedValueOnce({
          url: 'ws://terminal.example/ws',
        }),
    };
    const onClose = vi.fn();
    const onTransportError = vi.fn();
    const session = useTerminalSession({
      connector,
      onClose,
      onTransportError,
    });

    const firstConnect = session.connect(createSize());
    const secondConnect = session.connect(createSize({ cols: 140 }));
    if (typeof deferredReject.reject === 'function') {
      deferredReject.reject(new Error('stale connect failed'));
    }

    await expect(firstConnect).resolves.toBeUndefined();
    await secondConnect;

    const socket = MockWebSocket.instances[0];
    socket.emitOpen();

    expect(onTransportError).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalledWith('connect_error');
    expect(session.state.value).toBe('connected');
  });

  it('emits session_error when the socket closes after a transport error', async () => {
    const { connector } = createConnector();
    const onClose = vi.fn();
    const session = useTerminalSession({
      connector,
      onClose,
    });

    await session.connect(createSize());
    const socket = MockWebSocket.instances[0];
    socket.emitOpen();
    socket.onerror?.();
    socket.emitClose();

    expect(onClose).toHaveBeenCalledWith('session_error');
    expect(session.state.value).toBe('error');
  });
});
