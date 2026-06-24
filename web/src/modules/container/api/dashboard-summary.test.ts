import { beforeEach, describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { CONTAINER_API_PATH } from '../contract/paths';
import { getContainerDashboardSummary, mapContainerDashboardSummary } from './dashboard-summary';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
  },
}));

describe('container dashboard summary api', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('reads the canonical dashboard summary path', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({
      overview: {
        running_containers: 10,
        abnormal_containers: 2,
        cpu_total_percent: 18.5,
      },
      hotspots: {
        cpu_top: [],
        memory_top: [],
      },
      anomalies: [],
    } as never);

    await getContainerDashboardSummary();

    expect(requestGet).toHaveBeenCalledWith({
      url: CONTAINER_API_PATH.DASHBOARD_SUMMARY,
    });
  });

  it('maps the canonical dashboard summary response into the dashboard view model', () => {
    expect(
      mapContainerDashboardSummary({
        overview: {
          running_containers: 8,
          abnormal_containers: 3,
          cpu_total_percent: 52.5,
          memory_total_usage_bytes: 268435456,
          memory_total_limit_bytes: 536870912,
          memory_total_percent: 61.2,
        },
        hotspots: {
          cpu_top: [
            {
              id: 'cpu-1',
              name: 'api',
              short_id: 'cpu-1',
              image: 'graft/api:latest',
              state: 'running',
              health: 'healthy',
              restart_count: 1,
              resource: {
                available: true,
                stats_available: true,
                cpu_percent: 48.1,
                memory_percent: 18.2,
                memory_usage_bytes: 1207959552,
                memory_limit_bytes: 2147483648,
                collected_at: '2026-06-24T12:00:00Z',
              },
            },
          ],
          memory_top: [
            {
              id: 'mem-1',
              name: 'worker',
              short_id: 'mem-1',
              image: 'graft/worker:latest',
              state: 'running',
              resource: {
                available: true,
                stats_available: true,
                memory_percent: 72.4,
                memory_usage_bytes: 268435456,
                memory_limit_bytes: 536870912,
                collected_at: '2026-06-24T12:01:00Z',
              },
            },
          ],
        },
        anomalies: [
          {
            id: 'bad-1',
            name: 'scheduler',
            short_id: 'bad-1',
            image: 'graft/scheduler:latest',
            state: 'restarting',
            resource: {
              available: true,
              stats_available: true,
              cpu_percent: 2.4,
              memory_percent: 12.3,
              collected_at: '2026-06-24T12:02:00Z',
            },
          },
        ],
      }),
    ).toEqual({
      overview: {
        abnormalContainers: 3,
        collectedAt: '2026-06-24T12:02:00Z',
        cpuTotalPercent: 52.5,
        memoryTotalLimitBytes: 536870912,
        memoryTotalPercent: 61.2,
        memoryTotalUsageBytes: 268435456,
        runningContainers: 8,
      },
      hotspots: {
        cpu: [
          {
            cpuPercent: 48.1,
            health: 'healthy',
            id: 'cpu-1',
            image: 'graft/api:latest',
            memoryLimitBytes: 2147483648,
            memoryPercent: 18.2,
            memoryUsageBytes: 1207959552,
            name: 'api',
            restartCount: 1,
            shortId: 'cpu-1',
            state: 'running',
            collectedAt: '2026-06-24T12:00:00Z',
          },
        ],
        memory: [
          {
            collectedAt: '2026-06-24T12:01:00Z',
            cpuPercent: null,
            health: null,
            id: 'mem-1',
            image: 'graft/worker:latest',
            memoryLimitBytes: 536870912,
            memoryPercent: 72.4,
            memoryUsageBytes: 268435456,
            name: 'worker',
            restartCount: null,
            shortId: 'mem-1',
            state: 'running',
          },
        ],
      },
      anomalies: [
        {
          collectedAt: '2026-06-24T12:02:00Z',
          cpuPercent: 2.4,
          health: null,
          id: 'bad-1',
          image: 'graft/scheduler:latest',
          memoryPercent: 12.3,
          memoryUsageBytes: null,
          memoryLimitBytes: null,
          name: 'scheduler',
          restartCount: null,
          shortId: 'bad-1',
          state: 'restarting',
        },
      ],
    });
  });
});
