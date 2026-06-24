import type { ContainerSummaryRecord } from '../types/container';
import { getContainers } from './container';

const DASHBOARD_CONTAINER_OVERVIEW_LIMIT = 5;

export async function getDashboardContainerStatsSeed() {
  const response = await getContainers({
    limit: DASHBOARD_CONTAINER_OVERVIEW_LIMIT,
    offset: 0,
  });

  return {
    items: response.items as ContainerSummaryRecord[],
    summary: response.summary,
  };
}
