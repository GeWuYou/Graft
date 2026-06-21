import { describe, expect, it } from 'vitest';

import { formatBytes, formatNanosecondsAsDuration, formatPercent, toProgressPercent } from './resource-format';

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

  it('formats nanosecond CPU counters as readable milliseconds or seconds', () => {
    expect(formatNanosecondsAsDuration(40_401_700)).toBe('40.4 ms');
    expect(formatNanosecondsAsDuration(10_000_000)).toBe('10 ms');
    expect(formatNanosecondsAsDuration(50_468_370_000_000)).toBe('50,468.37 s');
    expect(formatNanosecondsAsDuration(undefined)).toBe('-');
  });

  it('formats duration numbers with an explicit locale', () => {
    expect(formatNanosecondsAsDuration(50_468_370_000_000, '-', 'de-DE')).toBe('50.468,37 s');
  });
});
