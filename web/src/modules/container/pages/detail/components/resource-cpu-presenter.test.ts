// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { buildCpuDetailMetrics, formatCpuDuration, formatCpuSystemTime } from './resource-cpu-presenter';

const labels = {
  cpuLimit: 'CPU 限制 / 在线 CPU',
  cpuPercent: '当前使用率',
  kernelTime: '内核态时间',
  systemCpuTime: '系统 CPU 时间',
  throttlingCount: 'Throttling 次数',
  throttlingInactiveHint: '未发生 CPU throttling',
  throttlingSignalHint: '存在 CPU throttling，可能受限',
  throttlingTime: 'Throttling 时间',
  totalCpuTime: '累计 CPU 时间',
  userTime: '用户态时间',
};

describe('resource-cpu-presenter', () => {
  it('formats nanosecond CPU usage counters as readable durations', () => {
    expect(formatCpuDuration(40_401_700)).toBe('40.4 ms');
    expect(formatCpuDuration(10_000_000)).toBe('10 ms');
  });

  it('formats large system CPU counters as seconds', () => {
    expect(formatCpuSystemTime(50_468_370_000_000)).toBe('50,468.37 s');
  });

  it('weakens throttling metrics when throttling is absent', () => {
    const metrics = buildCpuDetailMetrics(
      {
        available: true,
        cpu_percent: 0,
        online_cpus: 28,
        stats_available: true,
        throttling_throttled_periods: 0,
        throttling_throttled_time: 0,
      },
      labels,
    );

    expect(metrics).toHaveLength(8);
    expect(metrics.find((metric) => metric.key === 'cpu-capacity')?.value).toBe('— / 28 CPU');
    expect(metrics.find((metric) => metric.key === 'total-cpu-time')?.hint).toBeUndefined();
    expect(metrics.find((metric) => metric.key === 'system-cpu-time')?.hint).toBeUndefined();
    expect(metrics.find((metric) => metric.key === 'throttling-count')).toMatchObject({
      emphasized: false,
      hint: '未发生 CPU throttling',
      muted: true,
      value: '0',
    });
    expect(metrics.find((metric) => metric.key === 'throttling-time')).toMatchObject({
      emphasized: false,
      hint: '未发生 CPU throttling',
      muted: true,
      value: '0 ms',
    });
  });

  it('emphasizes throttling metrics when throttling is present', () => {
    const metrics = buildCpuDetailMetrics(
      {
        available: true,
        stats_available: true,
        throttling_throttled_periods: 3,
        throttling_throttled_time: 12_000_000,
      },
      labels,
    );

    expect(metrics.find((metric) => metric.key === 'throttling-count')).toMatchObject({
      emphasized: true,
      hint: '存在 CPU throttling，可能受限',
      muted: false,
      value: '3',
    });
    expect(metrics.find((metric) => metric.key === 'throttling-time')).toMatchObject({
      emphasized: true,
      hint: '存在 CPU throttling，可能受限',
      muted: false,
      value: '12 ms',
    });
  });

  it('uses an explicit dash for missing CPU fields', () => {
    const metrics = buildCpuDetailMetrics({ available: true, stats_available: true }, labels);

    expect(metrics.map((metric) => metric.value)).toEqual(['—', '— / —', '—', '—', '—', '—', '—', '—']);
  });
});
