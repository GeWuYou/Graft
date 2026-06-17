// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { formatRefreshCountdown } from './countdown';

describe('formatRefreshCountdown', () => {
  it.each([
    [4, '4s'],
    [59, '59s'],
    [60, '1m 00s'],
    [65, '1m 05s'],
    [600, '10m 00s'],
    [3599, '59m 59s'],
    [3600, '1h 00m'],
    [3720, '1h 02m'],
    [null, '--'],
    [undefined, '--'],
    [-1, '--'],
  ])('formats %s as %s', (input, expected) => {
    expect(formatRefreshCountdown(input)).toBe(expected);
  });
});
