// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { formatNanosecondsAsDuration, formatPercent } from '@/shared/observability';

import type { ContainerDetail } from '../../../types/container';

const EMPTY_TEXT = '—';

export type CpuDetailMetric = {
  emphasized?: boolean;
  hint?: string;
  key: string;
  label: string;
  muted?: boolean;
  value: string;
};

export type CpuDetailPresenterLabels = {
  cpuLimit: string;
  cpuPercent: string;
  kernelTime: string;
  systemCpuTime: string;
  throttlingCount: string;
  throttlingInactiveHint: string;
  throttlingSignalHint: string;
  throttlingTime: string;
  totalCpuTime: string;
  userTime: string;
};

type ContainerResourceSummary = NonNullable<ContainerDetail['resource']>;

export function buildCpuDetailMetrics(
  resource: ContainerResourceSummary | null | undefined,
  labels: CpuDetailPresenterLabels,
): CpuDetailMetric[] {
  const throttlingActive =
    isPositiveNumber(resource?.throttling_throttled_periods) || isPositiveNumber(resource?.throttling_throttled_time);
  const throttlingHint = throttlingActive ? labels.throttlingSignalHint : labels.throttlingInactiveHint;

  return [
    {
      key: 'cpu-percent',
      label: labels.cpuPercent,
      value: formatCpuPercent(resource?.cpu_percent),
    },
    {
      key: 'cpu-capacity',
      label: labels.cpuLimit,
      value: `${EMPTY_TEXT} / ${formatCpuCount(resource?.online_cpus)}`,
    },
    {
      key: 'total-cpu-time',
      label: labels.totalCpuTime,
      value: formatCpuDuration(resource?.total_cpu_usage),
    },
    {
      key: 'system-cpu-time',
      label: labels.systemCpuTime,
      value: formatCpuDuration(resource?.system_cpu_usage),
    },
    {
      key: 'user-cpu-time',
      label: labels.userTime,
      value: formatCpuDuration(resource?.cpu_usage_in_usermode),
    },
    {
      key: 'kernel-cpu-time',
      label: labels.kernelTime,
      value: formatCpuDuration(resource?.cpu_usage_in_kernelmode),
    },
    {
      emphasized: throttlingActive,
      hint: throttlingHint,
      key: 'throttling-count',
      label: labels.throttlingCount,
      muted: !throttlingActive,
      value: formatPlainNumber(resource?.throttling_throttled_periods),
    },
    {
      emphasized: throttlingActive,
      hint: throttlingHint,
      key: 'throttling-time',
      label: labels.throttlingTime,
      muted: !throttlingActive,
      value: formatCpuDuration(resource?.throttling_throttled_time),
    },
  ];
}

export function formatCpuDuration(value?: number | null) {
  return formatNanosecondsAsDuration(value, EMPTY_TEXT);
}

export function formatCpuSystemTime(value?: number | null) {
  return formatCpuDuration(value);
}

export function formatCpuCountText(value?: number | null) {
  return formatCpuCount(value);
}

function formatCpuCount(value?: number | null) {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return EMPTY_TEXT;
  }
  return `${formatPlainNumber(value)} CPU`;
}

function formatCpuPercent(value?: number | null) {
  return formatPercent(value, EMPTY_TEXT);
}

function formatPlainNumber(value?: number | null) {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return EMPTY_TEXT;
  }
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 0,
  }).format(value);
}

function isPositiveNumber(value?: number | null) {
  return typeof value === 'number' && Number.isFinite(value) && value > 0;
}
