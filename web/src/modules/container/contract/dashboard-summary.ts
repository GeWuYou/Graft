import type { paths } from '@/contracts/openapi/generated/schema';

import { CONTAINER_API_PATH } from './paths';

type ContainerDashboardSummaryPath = (typeof CONTAINER_API_PATH)['DASHBOARD_SUMMARY'];
type GetContainerDashboardSummaryOperation = paths[ContainerDashboardSummaryPath]['get'];
type GetContainerDashboardSummaryEnvelope =
  GetContainerDashboardSummaryOperation['responses'][200]['content']['application/json'];

export type ContainerDashboardSummaryResponse = NonNullable<GetContainerDashboardSummaryEnvelope['data']>;

export type ContainerDashboardOverview = {
  abnormalContainers: number;
  collectedAt: string | null;
  cpuTotalPercent: number;
  memoryTotalLimitBytes: number | null;
  memoryTotalPercent: number | null;
  memoryTotalUsageBytes: number | null;
  runningContainers: number;
};

export type ContainerDashboardResourceMetric = {
  collectedAt: string | null;
  cpuPercent: number | null;
  memoryLimitBytes: number | null;
  memoryPercent: number | null;
  memoryUsageBytes: number | null;
};

export type ContainerDashboardHotspotItem = ContainerDashboardResourceMetric & {
  health: string | null;
  id: string;
  image: string;
  name: string;
  restartCount: number | null;
  shortId: string;
  state: string;
};

export type ContainerDashboardAnomalyItem = ContainerDashboardResourceMetric & {
  health: string | null;
  id: string;
  image: string;
  name: string;
  restartCount: number | null;
  shortId: string;
  state: string;
};

export type ContainerDashboardSummary = {
  anomalies: ContainerDashboardAnomalyItem[];
  hotspots: {
    cpu: ContainerDashboardHotspotItem[];
    memory: ContainerDashboardHotspotItem[];
  };
  overview: ContainerDashboardOverview;
};
