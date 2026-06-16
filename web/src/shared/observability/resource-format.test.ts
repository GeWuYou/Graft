// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { formatBytes, formatPercent, toProgressPercent } from './resource-format';

describe('resource-format', () => {
  it('formats bytes as MiB and GiB', () => {
    expect(formatBytes(9.3 * 1024 * 1024)).toBe('9.3 MiB');
    expect(formatBytes(32002.7 * 1024 * 1024)).toBe('31.25 GiB');
  });

  it('formats percentages and clamps progress values', () => {
    expect(formatPercent(21.83)).toBe('21.8%');
    expect(formatPercent(undefined)).toBe('-');
    expect(toProgressPercent(120)).toBe(100);
    expect(toProgressPercent(-1)).toBe(0);
  });
});
