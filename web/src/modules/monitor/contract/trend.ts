export const MONITOR_TREND_RANGE = {
  TEN_MINUTES: '10m',
  THIRTY_MINUTES: '30m',
  ONE_HOUR: '1h',
} as const;

export type MonitorTrendRange = (typeof MONITOR_TREND_RANGE)[keyof typeof MONITOR_TREND_RANGE];
