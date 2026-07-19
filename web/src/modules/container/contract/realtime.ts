import { CONTAINER_REALTIME_TOPIC } from '@/contracts/generated/modules/container';
import type { components } from '@/contracts/openapi/generated/schema';
import { isRealtimePayloadObject, parseRealtimeEnvelopeData } from '@/shared/realtime';

export function buildContainerStatsTopicName(containerId: string) {
  return `${CONTAINER_REALTIME_TOPIC.STATS_PREFIX}${containerId}`;
}

export function buildContainerLogsTopicName(containerId: string) {
  return `${CONTAINER_REALTIME_TOPIC.LOGS_PREFIX}${containerId}`;
}

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
 * @returns 当主题与容器 ID 精确匹配时返回 `true`，否则返回 `false`
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
 * @returns 当主题对应指定容器时返回 `true`，否则返回 `false`
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
 * 解析容器事件实时载荷。
 *
 * @param raw - 原始 JSON 字符串
 * @returns 解析并通过字段校验后的载荷；解析失败或校验失败时返回 `null`
 */
export function parseContainerEventsPayload(raw: unknown): ContainerEventsRealtimePayload | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data)) {
    return null;
  }
  if (
    typeof data.topic !== 'string' ||
    typeof data.resource_id !== 'string' ||
    !isRealtimePayloadObject(data.context) ||
    typeof data.context.runtime !== 'string' ||
    !isRealtimePayloadObject(data.record) ||
    typeof data.record.seq !== 'number' ||
    !isRealtimePayloadObject(data.record.event)
  ) {
    return null;
  }
  return data as ContainerEventsRealtimePayload;
}

/**
 * 获取容器仪表盘汇总的实时主题名称。
 *
 * @returns 容器仪表盘汇总的 canonical realtime 主题字符串
 */
export function getContainerDashboardSummaryTopicName() {
  return CONTAINER_REALTIME_TOPIC.DASHBOARD_SUMMARY;
}
