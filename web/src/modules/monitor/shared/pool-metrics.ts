import type { ServerStatusConnectionPool } from '../types/server-status';

export type PoolUsageStatus = 'healthy' | 'warning' | 'danger' | 'unknown';

export function formatDependencyPoolUsage(pool: ServerStatusConnectionPool, noDataText: string) {
  return `${formatPoolCount(pool.in_use_connections, noDataText)} / ${formatPoolCount(pool.capacity, noDataText)}`;
}

export function formatPoolCount(value: number | null | undefined, noDataText: string) {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return noDataText;
  }

  return String(Math.max(0, Math.round(value)));
}

export function poolUsagePercent(pool: ServerStatusConnectionPool) {
  const inUse = Number(pool.in_use_connections);
  const capacity = Number(pool.capacity);
  if (!Number.isFinite(inUse) || !Number.isFinite(capacity) || capacity <= 0) {
    return null;
  }

  return Math.min(Math.max((Math.max(inUse, 0) / capacity) * 100, 0), 100);
}

export function poolUsageStatus(percent: number | null): PoolUsageStatus {
  if (percent === null || Number.isNaN(percent)) {
    return 'unknown';
  }
  if (percent >= 90) {
    return 'danger';
  }
  if (percent >= 70) {
    return 'warning';
  }

  return 'healthy';
}
