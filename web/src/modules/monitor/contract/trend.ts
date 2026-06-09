// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const MONITOR_TREND_RANGE = {
  TEN_MINUTES: '10m',
  THIRTY_MINUTES: '30m',
  ONE_HOUR: '1h',
} as const;

export type MonitorTrendRange = (typeof MONITOR_TREND_RANGE)[keyof typeof MONITOR_TREND_RANGE];
