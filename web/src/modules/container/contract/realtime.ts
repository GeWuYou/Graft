export const CONTAINER_REALTIME_TOPIC = {
  DASHBOARD_SUMMARY: 'container.dashboard.summary',
  LOGS_PREFIX: 'container.logs:',
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
 * 生成容器日志的实时主题名称。
 *
 * @param containerId - 容器 ID
 * @returns 拼接日志主题前缀与 `containerId` 后得到的主题名称
 */
export function buildContainerLogsTopicName(containerId: string) {
  return `${CONTAINER_REALTIME_TOPIC.LOGS_PREFIX}${containerId}`;
}

/**
 * 从日志 realtime topic 中解析容器 ID。
 *
 * @param topic - 待解析的 realtime topic
 * @returns topic 对应的容器 ID；无效时返回空字符串
 */
export function parseContainerLogsTopicContainerId(topic: string) {
  const normalizedTopic = topic.trim();
  if (!normalizedTopic.startsWith(CONTAINER_REALTIME_TOPIC.LOGS_PREFIX)) {
    return '';
  }
  return normalizedTopic.slice(CONTAINER_REALTIME_TOPIC.LOGS_PREFIX.length).trim();
}

/**
 * 判断日志 realtime topic 是否与容器 ID 对应。
 *
 * @param topic - realtime topic
 * @param containerId - 容器 ID
 * @returns `true` if topic 与容器 ID 精确匹配；`false` otherwise
 */
export function isContainerLogsTopicForContainer(topic: string, containerId: string) {
  const normalizedContainerId = containerId.trim();
  return normalizedContainerId.length > 0 && parseContainerLogsTopicContainerId(topic) === normalizedContainerId;
}

/**
 * 获取容器仪表盘汇总的实时主题名称。
 *
 * @returns 容器仪表盘汇总的 canonical realtime 主题字符串
 */
export function getContainerDashboardSummaryTopicName() {
  return CONTAINER_REALTIME_TOPIC.DASHBOARD_SUMMARY;
}
