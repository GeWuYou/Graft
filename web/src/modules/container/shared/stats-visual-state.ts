import type { ContainerStatsChangeDirection, ContainerStatsChangeState } from './stats-manager';

export function metricProgressStatus(direction: ContainerStatsChangeDirection): 'success' | 'warning' | undefined {
  if (direction === 'up') {
    return 'warning';
  }
  if (direction === 'down') {
    return 'success';
  }
  return undefined;
}

export function metricChangedClass(change: ContainerStatsChangeState, metric: 'cpu' | 'memory') {
  return {
    'container-metric-change--down': change[metric] === 'down',
    'container-metric-change--up': change[metric] === 'up',
  };
}
