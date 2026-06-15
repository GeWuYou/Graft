// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

const BYTES_PER_MIB = 1024 * 1024;
const BYTES_PER_GIB = 1024 * BYTES_PER_MIB;

export function formatBytes(value?: number | null, emptyText = '-') {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return emptyText;
  }

  const absValue = Math.abs(value);
  if (absValue >= BYTES_PER_GIB) {
    return `${(value / BYTES_PER_GIB).toFixed(2)} GiB`;
  }

  const mib = value / BYTES_PER_MIB;
  return `${mib.toFixed(absValue >= BYTES_PER_MIB ? 1 : 2)} MiB`;
}

export function formatPercent(value?: number | null, emptyText = '-') {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return emptyText;
  }

  return `${value.toFixed(1)}%`;
}

export function toProgressPercent(value?: number | null) {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return 0;
  }

  return Math.min(100, Math.max(0, value));
}
