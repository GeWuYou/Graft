import type { ContainerStatsChangeDirection, ContainerStatsChangeState } from './stats-manager';

/**
 * 将指标变化方向映射为进度状态。
 *
 * @param direction - 指标变化方向
 * @returns 当方向为 `up` 时返回 `warning`，当方向为 `down` 时返回 `success`，否则返回 `undefined`
 */
export function metricProgressStatus(direction: ContainerStatsChangeDirection): 'success' | 'warning' | undefined {
  if (direction === 'up') {
    return 'warning';
  }
  if (direction === 'down') {
    return 'success';
  }
  return undefined;
}

/**
 * 生成指定指标的变化状态类名映射。
 *
 * @param change - 容器统计变化状态
 * @param metric - 要检查的指标名称
 * @returns 包含 `container-metric-change--down` 和 `container-metric-change--up` 标记的对象
 */
export function metricChangedClass(change: ContainerStatsChangeState, metric: 'cpu' | 'memory') {
  return {
    'container-metric-change--down': change[metric] === 'down',
    'container-metric-change--up': change[metric] === 'up',
  };
}
