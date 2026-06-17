// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export function formatRefreshCountdown(seconds: number | null | undefined): string {
  if (typeof seconds !== 'number' || !Number.isFinite(seconds) || seconds < 0) {
    return '--';
  }

  const normalizedSeconds = Math.floor(seconds);
  if (normalizedSeconds < 60) {
    return `${normalizedSeconds}s`;
  }

  if (normalizedSeconds < 3600) {
    const minutes = Math.floor(normalizedSeconds / 60);
    const remainingSeconds = normalizedSeconds % 60;
    return `${minutes}m ${padTimeUnit(remainingSeconds)}s`;
  }

  const hours = Math.floor(normalizedSeconds / 3600);
  const minutes = Math.floor((normalizedSeconds % 3600) / 60);
  return `${hours}h ${padTimeUnit(minutes)}m`;
}

function padTimeUnit(value: number) {
  return String(value).padStart(2, '0');
}
