export const CONTAINER_REALTIME_TOPIC = {
  DASHBOARD_SUMMARY: 'container.dashboard.summary',
  LIST_STATS: 'container.stats.list',
  STATS_PREFIX: 'container.stats:',
} as const;

export type ContainerRealtimeTopicPrefix = (typeof CONTAINER_REALTIME_TOPIC)[keyof typeof CONTAINER_REALTIME_TOPIC];

/**
 * 生成容器实时统计主题名称。
 *
 * @param containerId - 容器标识
 * @returns 拼接 `STATS_PREFIX` 与 `containerId` 后得到的主题名称
 */
export function buildContainerStatsTopicName(containerId: string) {
  return `${CONTAINER_REALTIME_TOPIC.STATS_PREFIX}${containerId}`;
}

/**
 * 获取容器仪表盘汇总的实时主题名称。
 *
 * @returns 容器仪表盘汇总的 canonical realtime 主题字符串
 */
export function getContainerDashboardSummaryTopicName() {
  return CONTAINER_REALTIME_TOPIC.DASHBOARD_SUMMARY;
}
