import { beforeEach, describe, expect, it } from 'vitest';

import { patchGlobalLoggerContext } from '@/utils/logger';

import { formatHintedMessage, resolveErrorMessageWithCorrelation } from './correlation';

function t(key: string, params?: Record<string, unknown>) {
  if (key === 'audit.correlation.hintRequestOnly') {
    return `Request ID: ${String(params?.requestId ?? '')}`;
  }
  if (key === 'audit.correlation.hintTraceOnly') {
    return `Trace ID: ${String(params?.traceId ?? '')}`;
  }
  if (key === 'audit.correlation.hintRequestAndTrace') {
    return `Request ID: ${String(params?.requestId ?? '')}; Trace ID: ${String(params?.traceId ?? '')}`;
  }
  return key;
}

describe('correlation message helpers', () => {
  beforeEach(() => {
    patchGlobalLoggerContext({
      requestId: '',
      traceId: '',
    });
  });

  it('does not append correlation info to success messages', () => {
    patchGlobalLoggerContext({
      requestId: 'req-success',
      traceId: 'req-success',
    });

    expect(formatHintedMessage('Saved successfully')).toBe('Saved successfully');
  });

  it('does not append correlation info to 4xx api errors', () => {
    const error = Object.assign(new Error('Invalid request'), {
      status: 400,
      code: 'COMMON_INVALID_ARGUMENT',
      traceId: 'trace-400',
      messageKey: undefined,
      locale: 'en-US',
      responseData: undefined,
      isApiRequestError: true as const,
    });

    expect(resolveErrorMessageWithCorrelation(t, error, 'Fallback')).toBe('Invalid request');
  });

  it('appends correlation info to 5xx api errors', () => {
    const error = Object.assign(new Error('Server exploded'), {
      status: 500,
      code: 'COMMON_INTERNAL_ERROR',
      traceId: 'trace-500',
      messageKey: undefined,
      locale: 'en-US',
      responseData: undefined,
      isApiRequestError: true as const,
    });

    expect(resolveErrorMessageWithCorrelation(t, error, 'Fallback')).toBe('Server exploded Trace ID: trace-500');
  });
});
