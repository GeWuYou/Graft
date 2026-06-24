import {
  acquireContainerStatsSubscription,
  clearContainerSummaryCollection,
  releaseContainerStatsSubscription,
  seedContainerList,
  selectContainerSummaryCollectionViews,
} from '../shared/stats-manager';
import type { ContainerSummaryRecord } from '../types/container';

const CONTAINER_DASHBOARD_COLLECTION_KEY = 'dashboard:container-overview';

export function seedDashboardContainerStats(items: ContainerSummaryRecord[]) {
  seedContainerList(items, CONTAINER_DASHBOARD_COLLECTION_KEY);
}

export function clearDashboardContainerStats() {
  clearContainerSummaryCollection(CONTAINER_DASHBOARD_COLLECTION_KEY);
}

export function selectDashboardContainerStatsViews() {
  return selectContainerSummaryCollectionViews(CONTAINER_DASHBOARD_COLLECTION_KEY);
}

export function acquireDashboardContainerStats(containerId: string) {
  acquireContainerStatsSubscription(containerId);
}

export function releaseDashboardContainerStats(containerId: string) {
  releaseContainerStatsSubscription(containerId);
}
