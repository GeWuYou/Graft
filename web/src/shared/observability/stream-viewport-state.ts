export type StreamViewportState =
  | 'idle'
  | 'connecting'
  | 'streaming'
  | 'paused'
  | 'reconnecting'
  | 'disconnected'
  | 'error'
  | 'empty';

export type StreamViewportStateResolverInput = Readonly<{
  hasContent?: boolean | null;
  hasStarted?: boolean | null;
  isConnecting?: boolean | null;
  isStreaming?: boolean | null;
  isPaused?: boolean | null;
  isReconnecting?: boolean | null;
  isDisconnected?: boolean | null;
  error?: unknown;
}>;

export type NormalizedStreamViewportStateResolverInput = Readonly<{
  hasContent: boolean;
  hasStarted: boolean;
  isConnecting: boolean;
  isStreaming: boolean;
  isPaused: boolean;
  isReconnecting: boolean;
  isDisconnected: boolean;
  hasError: boolean;
  errorMessage: string;
}>;

const STREAM_VIEWPORT_BUSY_STATES = new Set<StreamViewportState>(['connecting', 'streaming', 'reconnecting']);
const STREAM_VIEWPORT_CURSOR_STATES = new Set<StreamViewportState>([
  'connecting',
  'streaming',
  'paused',
  'reconnecting',
]);

export function normalizeStreamViewportStateResolverInput(
  input: StreamViewportStateResolverInput = {},
): NormalizedStreamViewportStateResolverInput {
  const errorMessage = resolveStreamViewportErrorMessage(input.error);

  return {
    hasContent: Boolean(input.hasContent),
    hasStarted: Boolean(input.hasStarted),
    isConnecting: Boolean(input.isConnecting),
    isStreaming: Boolean(input.isStreaming),
    isPaused: Boolean(input.isPaused),
    isReconnecting: Boolean(input.isReconnecting),
    isDisconnected: Boolean(input.isDisconnected),
    hasError: hasStreamViewportError(input.error),
    errorMessage,
  };
}

export function resolveStreamViewportState(input: StreamViewportStateResolverInput = {}): StreamViewportState {
  const normalized = normalizeStreamViewportStateResolverInput(input);

  if (normalized.hasError) return 'error';
  if (normalized.isReconnecting) return 'reconnecting';
  if (normalized.isConnecting) return 'connecting';
  if (normalized.isPaused) return 'paused';
  if (normalized.isStreaming) return 'streaming';
  if (normalized.isDisconnected) return 'disconnected';
  if (normalized.hasStarted && !normalized.hasContent) return 'empty';
  return 'idle';
}

export function isStreamViewportBusyState(state: StreamViewportState) {
  return STREAM_VIEWPORT_BUSY_STATES.has(state);
}

export function isStreamViewportCursorState(state: StreamViewportState) {
  return STREAM_VIEWPORT_CURSOR_STATES.has(state);
}

function hasStreamViewportError(error: unknown) {
  if (error === null || error === undefined) {
    return false;
  }
  if (typeof error === 'string') {
    return error.trim().length > 0;
  }
  return true;
}

function resolveStreamViewportErrorMessage(error: unknown) {
  if (typeof error === 'string') {
    return error.trim();
  }
  if (error instanceof Error) {
    return error.message;
  }
  return '';
}
