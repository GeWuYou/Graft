import { buildContainerStatsTopicName, CONTAINER_REALTIME_TOPIC } from '../contract/realtime';
import type { ContainerResourceSummary } from '../types/container';

/**
 * 判断值是否为对象。
 *
 * @param value - 待检查的值
 * @returns `true` 如果值为真且类型为 `object`，`false` 否则
 */
function isObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === 'object');
}

function parseRealtimeEventData(raw: unknown) {
  if (typeof raw !== 'string') {
    return null;
  }

  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!isObject(parsed)) {
      return null;
    }
    return isObject(parsed.data) ? parsed.data : null;
  } catch {
    return null;
  }
}

/**
 * 生成容器实时统计的主题名称。
 *
 * @param containerId - 容器标识
 * @returns 由 `containerId` 生成的主题名称
 */
export function buildContainerStatsTopic(containerId: string) {
  return buildContainerStatsTopicName(containerId);
}

export function buildContainerListStatsTopic() {
  return CONTAINER_REALTIME_TOPIC.LIST_STATS;
}

export type ContainerListStatsRealtimeItem = {
  id: string;
  resource: ContainerResourceSummary;
};

/**
 * 解析容器实时统计载荷。
 *
 * @param raw - 待解析的原始载荷
 * @returns 解析成功时返回包含 `id` 和 `resource` 的对象；格式不符合或解析失败时返回 `null`
 */
export function parseContainerStatsPayload(raw: unknown) {
  const eventData = parseRealtimeEventData(raw);
  if (!eventData) {
    return null;
  }
  try {
    const resource = isObject(eventData.resource) ? (eventData.resource as ContainerResourceSummary) : null;
    if (!resource) {
      return null;
    }
    const id = typeof eventData.id === 'string' ? eventData.id : undefined;

    return {
      id,
      resource,
    };
  } catch {
    return null;
  }
}

export function parseContainerListStatsPayload(raw: unknown): { items: ContainerListStatsRealtimeItem[] } | null {
  const eventData = parseRealtimeEventData(raw);
  if (!eventData || !Array.isArray(eventData.items)) {
    return null;
  }

  try {
    const items = eventData.items
      .map((item) => {
        if (!isObject(item) || typeof item.id !== 'string' || !isObject(item.resource)) {
          return null;
        }
        return {
          id: item.id,
          resource: item.resource as ContainerResourceSummary,
        };
      })
      .filter((item): item is ContainerListStatsRealtimeItem => item !== null);

    return { items };
  } catch {
    return null;
  }
}
