// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

const BYTES_PER_MIB = 1024 * 1024;
const BYTES_PER_GIB = 1024 * BYTES_PER_MIB;
const NANOSECONDS_PER_MILLISECOND = 1_000_000;
const NANOSECONDS_PER_SECOND = 1_000_000_000;

export function formatBytes(value?: number | null, emptyText = '-') {
  const normalizedValue = finiteNumberOrNull(value);
  if (normalizedValue === null) {
    return emptyText;
  }

  const absValue = Math.abs(normalizedValue);
  if (absValue >= BYTES_PER_GIB) {
    return `${(normalizedValue / BYTES_PER_GIB).toFixed(2)} GiB`;
  }

  const mib = normalizedValue / BYTES_PER_MIB;
  return `${mib.toFixed(absValue >= BYTES_PER_MIB ? 1 : 2)} MiB`;
}

export function formatPercent(value?: number | null, emptyText = '-') {
  const normalizedValue = finiteNumberOrNull(value);
  if (normalizedValue === null) {
    return emptyText;
  }

  return `${normalizedValue.toFixed(1)}%`;
}

export function formatNanosecondsAsDuration(value?: number | null, emptyText = '-') {
  const normalizedValue = finiteNumberOrNull(value);
  if (normalizedValue === null) {
    return emptyText;
  }

  const absValue = Math.abs(normalizedValue);
  if (absValue >= NANOSECONDS_PER_SECOND) {
    return `${formatNumber(normalizedValue / NANOSECONDS_PER_SECOND, 2)} s`;
  }

  return `${formatNumber(normalizedValue / NANOSECONDS_PER_MILLISECOND, 2)} ms`;
}

export function toProgressPercent(value?: number | null) {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return 0;
  }

  return Math.min(100, Math.max(0, value));
}

function formatNumber(value: number, maximumFractionDigits: number) {
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits,
    minimumFractionDigits: 0,
  }).format(value);
}

function finiteNumberOrNull(value?: number | null) {
  return value === null || value === undefined || !Number.isFinite(value) ? null : value;
}
