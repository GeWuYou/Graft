import type { components } from '@/contracts/openapi/generated/schema';

export const CONTAINER_REALTIME_TOPIC = {
  DASHBOARD_SUMMARY: 'container.dashboard.summary',
  EVENTS_PREFIX: 'container.events:',
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
 * 生成容器事件实时主题名称。
 *
 * @param containerId - 容器标识
 * @returns 由事件主题前缀和 `containerId` 拼接得到的主题名称
 */
export function buildContainerEventsTopicName(containerId: string) {
  return `${CONTAINER_REALTIME_TOPIC.EVENTS_PREFIX}${containerId}`;
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
 * 判断日志实时主题是否对应指定容器。
 *
 * @param topic - realtime topic
 * @param containerId - 容器 ID
 * @returns `true` if topic 与容器 ID 精确匹配，`false` otherwise
 */
export function isContainerLogsTopicForContainer(topic: string, containerId: string) {
  const normalizedContainerId = containerId.trim();
  return normalizedContainerId.length > 0 && parseContainerLogsTopicContainerId(topic) === normalizedContainerId;
}

/**
 * 判断事件实时主题是否对应指定容器。
 *
 * @param topic - 实时主题名称
 * @param containerId - 容器标识
 * @returns `true` if the topic 对应该容器，`false` otherwise.
 */
export function isContainerEventsTopicForContainer(topic: string, containerId: string) {
  const normalizedContainerId = containerId.trim();
  if (!normalizedContainerId.length) {
    return false;
  }

  const normalizedTopic = topic.trim();
  if (!normalizedTopic.startsWith(CONTAINER_REALTIME_TOPIC.EVENTS_PREFIX)) {
    return false;
  }

  return normalizedTopic.slice(CONTAINER_REALTIME_TOPIC.EVENTS_PREFIX.length).trim() === normalizedContainerId;
}

type ContainerRuntimeEventRecord = components['schemas']['ContainerRuntimeEventRecord'];
type ContainerRuntimeEventStreamContext = components['schemas']['ContainerRuntimeEventStreamContext'];

export type ContainerEventsRealtimePayload = {
  topic: string;
  resource_id: string;
  context: ContainerRuntimeEventStreamContext;
  record: ContainerRuntimeEventRecord;
};

/**
 * 判断值是否为对象。
 *
 * @param value - 待检查的值
 * @returns `true` if `value` 为真值且类型为 `object`，`false` otherwise.
 */
function isObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === 'object');
}

/**
 * 解析容器事件实时载荷。
 *
 * @param raw - 原始 JSON 字符串
 * @returns 解析并通过字段校验后的载荷；解析失败或校验失败时返回 `null`
 */
export function parseContainerEventsPayload(raw: unknown): ContainerEventsRealtimePayload | null {
  if (typeof raw !== 'string') {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as { data?: unknown };
    if (!isObject(parsed) || !isObject(parsed.data)) {
      return null;
    }
    const data = parsed.data;
    if (
      typeof data.topic !== 'string' ||
      typeof data.resource_id !== 'string' ||
      !isObject(data.context) ||
      typeof data.context.runtime !== 'string' ||
      !isObject(data.record) ||
      typeof data.record.seq !== 'number' ||
      !isObject(data.record.event)
    ) {
      return null;
    }
    return data as ContainerEventsRealtimePayload;
  } catch {
    return null;
  }
}

/**
 * 获取容器仪表盘汇总的实时主题名称。
 *
 * @returns 容器仪表盘汇总的 canonical realtime 主题字符串
 */
export function getContainerDashboardSummaryTopicName() {
  return CONTAINER_REALTIME_TOPIC.DASHBOARD_SUMMARY;
}
