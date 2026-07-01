export type StreamViewportState =
  'idle' | 'connecting' | 'streaming' | 'paused' | 'reconnecting' | 'disconnected' | 'error' | 'empty';

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

/**
 * 将流视口状态解析入参归一化为布尔标志和错误信息。
 *
 * @param input - 待归一化的解析入参
 * @returns 归一化后的状态对象，包含布尔化标志、错误标记和错误消息
 */
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

/**
 * 根据解析输入确定当前视口状态。
 *
 * @param input - 视口状态解析输入
 * @returns 解析得到的视口状态值
 */
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

/**
 * 判断视口状态是否属于忙碌状态。
 *
 * @param state - 要检查的视口状态
 * @returns `true` if `state` 属于忙碌状态，`false` otherwise.
 */
export function isStreamViewportBusyState(state: StreamViewportState) {
  return STREAM_VIEWPORT_BUSY_STATES.has(state);
}

/**
 * 判断流视口状态是否属于光标状态。
 *
 * @param state - 要检查的状态
 * @returns `true` 如果该状态属于光标状态集合，`false` 否则
 */
export function isStreamViewportCursorState(state: StreamViewportState) {
  return STREAM_VIEWPORT_CURSOR_STATES.has(state);
}

/**
 * 判断输入是否应视为错误。
 *
 * @param error - 待检查的错误值
 * @returns `true` 如果输入包含有效错误内容，`false` 否则
 */
function hasStreamViewportError(error: unknown) {
  if (error === null || error === undefined) {
    return false;
  }
  if (typeof error === 'string') {
    return error.trim().length > 0;
  }
  return true;
}

/**
 * 解析视口错误消息。
 *
 * @param error - 错误输入值
 * @returns 经过处理的错误消息；字符串会去除首尾空白，`Error` 会返回其 `message`，其他值返回空字符串
 */
function resolveStreamViewportErrorMessage(error: unknown) {
  if (typeof error === 'string') {
    return error.trim();
  }
  if (error instanceof Error) {
    return error.message;
  }
  return '';
}
