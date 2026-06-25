import { request } from '@/utils/request';

import type {
  ContainerDashboardAnomalyItem,
  ContainerDashboardHotspotItem,
  ContainerDashboardSummary,
  ContainerDashboardSummaryResponse,
} from '../contract/dashboard-summary';
import { CONTAINER_API_PATH } from '../contract/paths';

/**
 * 获取容器仪表盘汇总数据。
 *
 * @returns 容器仪表盘汇总信息。
 */
export function getContainerDashboardSummary() {
  return request
    .get<ContainerDashboardSummaryResponse>({
      url: CONTAINER_API_PATH.DASHBOARD_SUMMARY,
    })
    .then(mapContainerDashboardSummary);
}

/**
 * 将容器仪表盘汇总接口响应映射为前端数据结构。
 *
 * @param payload - 容器仪表盘汇总接口的原始响应数据
 * @returns 归一化后的容器仪表盘汇总数据
 */
export function mapContainerDashboardSummary(payload: ContainerDashboardSummaryResponse): ContainerDashboardSummary {
  return {
    overview: {
      runningContainers: payload.overview.running_containers,
      abnormalContainers: payload.overview.abnormal_containers,
      cpuTotalPercent: payload.overview.cpu_total_percent,
      memoryTotalUsageBytes: payload.overview.memory_total_usage_bytes ?? null,
      memoryTotalLimitBytes: payload.overview.memory_total_limit_bytes ?? null,
      memoryTotalPercent: payload.overview.memory_total_percent ?? null,
      collectedAt: collectSummaryTimestamp(payload),
    },
    hotspots: {
      cpu: payload.hotspots.cpu_top.map(mapContainerDashboardHotspotItem),
      memory: payload.hotspots.memory_top.map(mapContainerDashboardHotspotItem),
    },
    anomalies: payload.anomalies.map(mapContainerDashboardAnomalyItem),
  };
}

/**
 * 将容器仪表盘热点条目映射为统一结构。
 *
 * @returns 规范化后的热点条目
 */
function mapContainerDashboardHotspotItem(
  payload: ContainerDashboardSummaryResponse['hotspots']['cpu_top'][number],
): ContainerDashboardHotspotItem {
  return mapContainerDashboardItemBase(payload);
}

/**
 * 映射容器仪表盘异常条目。
 *
 * @param payload - 异常列表中的单个原始条目
 * @returns 规范化后的异常条目
 */
function mapContainerDashboardAnomalyItem(
  payload: ContainerDashboardSummaryResponse['anomalies'][number],
): ContainerDashboardAnomalyItem {
  return {
    ...mapContainerDashboardItemBase(payload),
    reasonCode: readOptionalString(payload, 'reason_code'),
    reasonLabel: readOptionalString(payload, 'reason_label'),
    status: readOptionalString(payload, 'status'),
  };
}

/**
 * 将容器仪表盘条目标准化为统一结构。
 *
 * @param payload - 热点条目或异常条目。
 * @returns 标准化后的条目对象。
 */
function mapContainerDashboardItemBase(
  payload:
    | ContainerDashboardSummaryResponse['hotspots']['cpu_top'][number]
    | ContainerDashboardSummaryResponse['anomalies'][number],
) {
  return {
    id: payload.id,
    name: payload.name,
    shortId: payload.short_id,
    image: payload.image,
    state: payload.state,
    health: payload.health ?? null,
    restartCount: payload.restart_count ?? null,
    cpuPercent: payload.resource.cpu_percent ?? null,
    memoryPercent: payload.resource.memory_percent ?? null,
    memoryUsageBytes: payload.resource.memory_usage_bytes ?? null,
    memoryLimitBytes: payload.resource.memory_limit_bytes ?? null,
    collectedAt: payload.resource.collected_at ?? null,
  };
}

function readOptionalString(payload: object, key: string) {
  const value = (payload as Record<string, unknown>)[key];
  return typeof value === 'string' && value.trim().length > 0 ? value : null;
}

/**
 * 提取汇总数据中最新的采集时间戳。
 *
 * @param payload - 容器仪表盘汇总接口返回值
 * @returns 最新的 `resource.collected_at` 时间戳；无可用时间戳时返回 `null`
 */
function collectSummaryTimestamp(payload: ContainerDashboardSummaryResponse) {
  const summaryCollectedAt = readOptionalString(payload as object, 'collected_at');
  if (summaryCollectedAt) {
    return summaryCollectedAt;
  }

  const timestamps = [...payload.hotspots.cpu_top, ...payload.hotspots.memory_top, ...payload.anomalies]
    .map((item) => item.resource.collected_at ?? '')
    .filter((value) => value.length > 0)
    .sort();

  return timestamps.at(-1) ?? null;
}
