// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { detectLogLevel, getLogLevelTone, tokenizeLogLine } from './log-highlight';

describe('log-highlight', () => {
  it('detects key-value log levels', () => {
    expect(detectLogLevel('time=2026-06-15 level=INFO msg="started"')).toBe('INFO');
    expect(detectLogLevel('time=2026-06-15 level=WARN msg="slow"')).toBe('WARN');
    expect(detectLogLevel('time=2026-06-15 level=ERROR msg="failed"')).toBe('ERROR');
  });

  it('maps log levels to display tones', () => {
    expect(getLogLevelTone('ERROR')).toBe('danger');
    expect(getLogLevelTone('WARN')).toBe('warning');
    expect(getLogLevelTone('INFO')).toBe('info');
  });

  it('tokenizes fields and search keywords', () => {
    const tokens = tokenizeLogLine('level=INFO msg="server started"', 'server');

    expect(tokens.some((token) => token.type === 'field-key' && token.text === 'level')).toBe(true);
    expect(tokens.some((token) => token.type === 'level' && token.level === 'INFO')).toBe(true);
    expect(tokens.some((token) => token.type === 'keyword' && token.text === 'server')).toBe(true);
  });
});
