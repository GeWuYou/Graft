import { request } from '@/utils/request';

import type {
  ContainerDashboardAnomalyItem,
  ContainerDashboardHotspotItem,
  ContainerDashboardSummary,
  ContainerDashboardSummaryResponse,
} from '../contract/dashboard-summary';
import { CONTAINER_API_PATH } from '../contract/paths';

export function getContainerDashboardSummary() {
  return request
    .get<ContainerDashboardSummaryResponse>({
      url: CONTAINER_API_PATH.DASHBOARD_SUMMARY,
    })
    .then(mapContainerDashboardSummary);
}

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

function mapContainerDashboardHotspotItem(
  payload: ContainerDashboardSummaryResponse['hotspots']['cpu_top'][number],
): ContainerDashboardHotspotItem {
  return mapContainerDashboardItemBase(payload);
}

function mapContainerDashboardAnomalyItem(
  payload: ContainerDashboardSummaryResponse['anomalies'][number],
): ContainerDashboardAnomalyItem {
  return mapContainerDashboardItemBase(payload);
}

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

function collectSummaryTimestamp(payload: ContainerDashboardSummaryResponse) {
  const timestamps = [...payload.hotspots.cpu_top, ...payload.hotspots.memory_top, ...payload.anomalies]
    .map((item) => item.resource.collected_at ?? '')
    .filter((value) => value.length > 0)
    .sort();

  return timestamps.at(-1) ?? null;
}
