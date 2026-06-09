// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { appLogCorrelationText, appLogFieldsCount, appLogOperationText, appLogSeverityTheme } from './presentation';

const t = ((key: string) =>
  ({
    'appLog.values.noOperation': 'No operation',
    'appLog.values.noCorrelation': 'No correlation',
  })[key] || key) as never;

describe('app-log presentation helpers', () => {
  it('maps severity to TDesign tag themes', () => {
    expect(appLogSeverityTheme('error')).toBe('danger');
    expect(appLogSeverityTheme('warn')).toBe('warning');
    expect(appLogSeverityTheme('debug')).toBe('default');
    expect(appLogSeverityTheme('info')).toBe('primary');
  });

  it('formats fallback operation and correlation text', () => {
    expect(appLogOperationText({ operation: '' } as never, t)).toBe('No operation');
    expect(appLogCorrelationText({ request_id: '', trace_id: 'trace-1' } as never, t)).toBe('trace-1');
    expect(appLogCorrelationText({ request_id: '', trace_id: '' } as never, t)).toBe('No correlation');
  });

  it('counts bounded fields', () => {
    expect(appLogFieldsCount({ fields: { module: 'user', latency: '12ms' } } as never)).toBe(2);
  });
});
