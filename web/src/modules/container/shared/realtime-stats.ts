import type { ContainerDashboardSummaryResponse } from '../contract/dashboard-summary';
import {
  buildContainerStatsTopicName,
  CONTAINER_REALTIME_TOPIC,
  getContainerDashboardSummaryTopicName,
} from '../contract/realtime';
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

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return isObject(value) && !Array.isArray(value);
}

/**
 * 解析实时事件的数据部分。
 *
 * @param raw - 原始事件内容
 * @returns 解析后的 `data` 对象；解析失败或结构不符合时返回 `null`
 */
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

/**
 * 构建容器列表实时统计主题。
 *
 * @returns 容器列表实时统计主题名称。
 */
export function buildContainerListStatsTopic() {
  return CONTAINER_REALTIME_TOPIC.LIST_STATS;
}

/**
 * 构建容器仪表盘汇总实时主题。
 *
 * @returns 容器仪表盘汇总实时主题名称
 */
export function buildContainerDashboardSummaryTopic() {
  return getContainerDashboardSummaryTopicName();
}

export type ContainerListStatsRealtimeItem = {
  id: string;
  resource: ContainerResourceSummary;
};

/**
 * 解析容器实时统计载荷。
 *
 * @param raw - 待解析的原始载荷
 * @returns 解析成功时返回包含 `resource`，以及可选 `id` 的对象；格式不符合或解析失败时返回 `null`
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

/**
 * 解析容器实时统计列表载荷。
 *
 * 仅保留包含有效 `id` 和 `resource` 的条目，并返回解析后的条目数组。
 *
 * @param raw - 原始事件载荷
 * @returns 解析后的列表数据；解析失败时返回 `null`
 */
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

/**
 * 解析容器仪表盘汇总实时载荷。
 *
 * @param raw - 原始事件载荷
 * @returns 符合仪表盘汇总结构的数据；解析失败时返回 `null`
 */
export function parseContainerDashboardSummaryPayload(raw: unknown): ContainerDashboardSummaryResponse | null {
  const eventData = parseRealtimeEventData(raw);
  if (!eventData) {
    return null;
  }

  const summaryData = isDashboardSummaryPayloadShape(eventData)
    ? eventData
    : isObject(eventData.data) && isDashboardSummaryPayloadShape(eventData.data)
      ? eventData.data
      : null;

  return summaryData as ContainerDashboardSummaryResponse | null;
}

/**
 * 判断值是否符合容器仪表盘汇总载荷结构。
 *
 * @param value - 待检查的值
 * @returns `true` if `value` 符合容器仪表盘汇总结构，`false` otherwise.
 */
function isDashboardSummaryPayloadShape(value: unknown): value is ContainerDashboardSummaryResponse {
  return (
    isPlainObject(value) &&
    isPlainObject(value.overview) &&
    isPlainObject(value.hotspots) &&
    Array.isArray(value.anomalies) &&
    Array.isArray(value.hotspots.cpu_top) &&
    Array.isArray(value.hotspots.memory_top)
  );
}
