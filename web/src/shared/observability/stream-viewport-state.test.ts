import { describe, expect, it } from 'vitest';

import {
  isStreamViewportBusyState,
  isStreamViewportCursorState,
  normalizeStreamViewportStateResolverInput,
  resolveStreamViewportState,
} from './stream-viewport-state';

describe('stream-viewport-state', () => {
  it('normalizes nullable resolver input into booleans and error metadata', () => {
    const normalized = normalizeStreamViewportStateResolverInput({
      hasContent: true,
      hasStarted: null,
      isConnecting: undefined,
      isStreaming: false,
      isPaused: null,
      isReconnecting: true,
      error: new Error('stream transport failed'),
    });

    expect(normalized.hasContent).toBe(true);
    expect(normalized.hasStarted).toBe(false);
    expect(normalized.isConnecting).toBe(false);
    expect(normalized.isStreaming).toBe(false);
    expect(normalized.isPaused).toBe(false);
    expect(normalized.isReconnecting).toBe(true);
    expect(normalized.hasError).toBe(true);
    expect(normalized.errorMessage).toBe('stream transport failed');
  });

  it('resolves idle before the viewport lifecycle starts and empty after a started no-content pass', () => {
    expect(resolveStreamViewportState()).toBe('idle');
    expect(resolveStreamViewportState({ hasStarted: true, hasContent: false })).toBe('empty');
  });

  it('resolves transport and lifecycle states in deterministic priority order', () => {
    expect(resolveStreamViewportState({ hasStarted: true, isConnecting: true })).toBe('connecting');
    expect(resolveStreamViewportState({ hasStarted: true, isConnecting: true, isReconnecting: true })).toBe(
      'reconnecting',
    );
    expect(resolveStreamViewportState({ hasStarted: true, isStreaming: true })).toBe('streaming');
    expect(resolveStreamViewportState({ hasStarted: true, isStreaming: true, isPaused: true })).toBe('paused');
    expect(resolveStreamViewportState({ hasStarted: true, isDisconnected: true })).toBe('disconnected');
    expect(resolveStreamViewportState({ hasStarted: true, isStreaming: true, error: 'transport exploded' })).toBe(
      'error',
    );
  });

  it('exposes busy and cursor hints for live console-like states', () => {
    expect(isStreamViewportBusyState('connecting')).toBe(true);
    expect(isStreamViewportBusyState('streaming')).toBe(true);
    expect(isStreamViewportBusyState('reconnecting')).toBe(true);
    expect(isStreamViewportBusyState('paused')).toBe(false);
    expect(isStreamViewportCursorState('paused')).toBe(true);
    expect(isStreamViewportCursorState('error')).toBe(false);
  });
});
