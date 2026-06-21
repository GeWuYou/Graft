export const MONITOR_REFRESH_INTERVAL = {
  FIVE_SECONDS: 5,
  TEN_SECONDS: 10,
  THIRTY_SECONDS: 30,
  ONE_MINUTE: 60,
} as const;

export type MonitorRefreshInterval = (typeof MONITOR_REFRESH_INTERVAL)[keyof typeof MONITOR_REFRESH_INTERVAL];
