import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { ContainerDashboardSummary } from '../contract/dashboard-summary';
import type { ContainerDetailRecord, ContainerSummaryRecord } from '../types/container';
import {
  acquireContainerDashboardSummarySubscription,
  acquireContainerStatsSubscription,
  acquireContainerSummaryCollectionSubscription,
  applyContainerRealtimeStats,
  clearContainerDashboardSummary,
  clearContainerSummaryCollection,
  releaseContainerDashboardSummarySubscription,
  releaseContainerStatsSubscription,
  releaseContainerSummaryCollectionSubscription,
  resetContainerStatsManager,
  seedContainerDashboardSummary,
  seedContainerDetail,
  seedContainerList,
  selectContainerDashboardRealtimeState,
  selectContainerDashboardSummaryView,
  selectContainerDetailView,
  selectContainerListViews,
  selectContainerStatsChangeState,
  selectContainerStatsHistory,
  selectContainerStatsRealtimeState,
  selectContainerSummaryCollectionViews,
} from './stats-manager';

const realtimeMocks = vi.hoisted(() => ({
  controllers: [] as Array<{
    close: ReturnType<typeof vi.fn>;
    emitMessage: (payload: unknown) => void;
    reconnect: ReturnType<typeof vi.fn>;
  }>,
  openRealtimeTopicSocket: vi.fn(
    (options?: { onMessage?: (payload: unknown) => void; parseMessage?: (payload: unknown) => unknown }) => {
      const controller = {
        close: vi.fn(),
        emitMessage: (payload: unknown) => {
          const parsed = options?.parseMessage ? options.parseMessage(payload) : payload;
          if (parsed) {
            options?.onMessage?.(parsed);
          }
        },
        reconnect: vi.fn(),
      };
      realtimeMocks.controllers.push(controller);
      return controller;
    },
  ),
}));

vi.mock('@/shared/realtime', () => ({
  openRealtimeTopicSocket: realtimeMocks.openRealtimeTopicSocket,
}));

function createSummary(
  resourceOverrides?: Partial<NonNullable<ContainerSummaryRecord['resource']>>,
): ContainerSummaryRecord {
  return {
    id: 'container-1',
    short_id: 'container-1',
    name: 'graft-web',
    names: ['graft-web'],
    image: 'graft/web:latest',
    image_id: 'sha256:1',
    labels: {},
    ports: [],
    restart_policy: 'unless-stopped',
    runtime: 'docker',
    state: 'running',
    health: 'healthy',
    status: 'Up 10 minutes',
    created_at: '2026-06-14T01:00:00Z',
    started_at: '2026-06-14T01:05:00Z',
    networks: [],
    resource: {
      available: true,
      stats_available: true,
      cpu_percent: 21.8,
      memory_limit_bytes: 536870912,
      memory_percent: 50,
      memory_usage_bytes: 268435456,
      collected_at: '2026-06-14T01:09:00Z',
      ...resourceOverrides,
    },
    can_start: false,
    can_stop: true,
    can_restart: true,
    can_remove: true,
  };
}

function createDetail(
  resourceOverrides?: Partial<NonNullable<ContainerDetailRecord['resource']>>,
): ContainerDetailRecord {
  const summary = createSummary(resourceOverrides);
  return {
    ...summary,
    command: [],
    entrypoint: [],
    environment: [],
    environment_masked_copy_enabled: false,
    environment_policy: 'masked',
    healthcheck: {
      command: [],
      configured: false,
      status: 'none',
    },
    inspect_updated_at: '2026-06-14T01:10:00Z',
    mounts: [],
    names: [...(summary.names ?? [])],
    networks: [...(summary.networks ?? [])],
    ports: [...(summary.ports ?? [])],
    runtime_info: {
      endpoint: 'unix:///var/run/docker.sock',
      runtime: 'docker',
      status: 'enabled',
    },
  };
}

function createDashboardSummary(overrides?: Partial<ContainerDashboardSummary>): ContainerDashboardSummary {
  return {
    overview: {
      runningContainers: 3,
      abnormalContainers: 1,
      cpuTotalPercent: 32.5,
      memoryTotalUsageBytes: 2147483648,
      memoryTotalLimitBytes: 4294967296,
      memoryTotalPercent: 50,
      collectedAt: '2026-06-14T01:09:00Z',
      ...overrides?.overview,
    },
    hotspots: {
      cpu: [
        {
          id: 'container-1',
          name: 'graft-web',
          shortId: 'container-1',
          image: 'graft/web:latest',
          state: 'running',
          health: 'healthy',
          restartCount: 0,
          cpuPercent: 32.5,
          memoryPercent: 40.2,
          memoryUsageBytes: 268435456,
          memoryLimitBytes: 536870912,
          collectedAt: '2026-06-14T01:09:00Z',
        },
      ],
      memory: [],
      ...overrides?.hotspots,
    },
    anomalies: [
      {
        id: 'bad-1',
        name: 'graft-worker',
        shortId: 'bad-1',
        image: 'graft/worker:latest',
        state: 'restarting',
        status: 'Restarting',
        health: null,
        reasonCode: 'state.restarting',
        reasonLabel: 'Restarting',
        restartCount: null,
        cpuPercent: 2.1,
        memoryPercent: 12.4,
        memoryUsageBytes: null,
        memoryLimitBytes: null,
        collectedAt: '2026-06-14T01:09:00Z',
      },
    ],
    ...overrides,
  };
}

describe('container stats manager', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    resetContainerStatsManager();
    realtimeMocks.controllers = [];
    realtimeMocks.openRealtimeTopicSocket.mockClear();
  });

  afterEach(() => {
    resetContainerStatsManager();
    vi.useRealTimers();
  });

  it('exposes seeded list rows through managed selectors', () => {
    seedContainerList([createSummary()]);

    const rows = selectContainerListViews();

    expect(rows).toHaveLength(1);
    expect(rows[0]?.resource?.cpu_percent).toBe(21.8);
    expect(rows[0]?.resource?.collected_at).toBe('2026-06-14T01:09:00Z');
  });

  it('keeps dashboard and list projections isolated while sharing the same stats authority', () => {
    seedContainerList([createSummary()], 'container:list');
    seedContainerList(
      [
        createSummary({
          cpu_percent: 11.2,
          collected_at: '2026-06-14T01:08:00Z',
        }),
      ],
      'dashboard:container-overview',
    );

    applyContainerRealtimeStats('container-1', {
      ...createSummary().resource!,
      cpu_percent: 64.5,
      collected_at: '2026-06-14T01:12:00Z',
    });

    expect(selectContainerListViews()[0]?.resource?.cpu_percent).toBe(64.5);
    expect(selectContainerSummaryCollectionViews('dashboard:container-overview')[0]?.resource?.cpu_percent).toBe(64.5);

    clearContainerSummaryCollection('dashboard:container-overview');

    expect(selectContainerListViews()).toHaveLength(1);
    expect(selectContainerSummaryCollectionViews('dashboard:container-overview')).toHaveLength(0);
  });

  it('shares one list-level realtime subscription controller across multiple collections', () => {
    seedContainerList([createSummary()], 'container:list');
    seedContainerList([createSummary()], 'dashboard:container-overview');

    acquireContainerSummaryCollectionSubscription();
    acquireContainerSummaryCollectionSubscription();

    expect(realtimeMocks.openRealtimeTopicSocket).toHaveBeenCalledTimes(1);
  });

  it('exposes seeded dashboard summary through the shared manager selector', () => {
    seedContainerDashboardSummary(createDashboardSummary());

    expect(selectContainerDashboardSummaryView()).toEqual(createDashboardSummary());
  });

  it('does not let an older dashboard http seed override a fresher realtime snapshot', () => {
    seedContainerDashboardSummary(createDashboardSummary());
    seedContainerDashboardSummary(
      {
        ...createDashboardSummary({
          overview: {
            ...createDashboardSummary().overview,
            cpuTotalPercent: 61.3,
            collectedAt: '2026-06-14T01:11:00Z',
          },
        }),
      },
      'realtime',
    );

    seedContainerDashboardSummary(
      createDashboardSummary({
        overview: {
          ...createDashboardSummary().overview,
          cpuTotalPercent: 8.2,
          collectedAt: '2026-06-14T01:10:00Z',
        },
      }),
    );

    expect(selectContainerDashboardSummaryView()?.overview.cpuTotalPercent).toBe(61.3);
    expect(selectContainerDashboardSummaryView()?.overview.collectedAt).toBe('2026-06-14T01:11:00Z');
  });

  it('applies dashboard summary topic payloads through the shared summary authority', () => {
    seedContainerDashboardSummary(createDashboardSummary());
    acquireContainerDashboardSummarySubscription();

    const controller = realtimeMocks.controllers.at(-1)!;
    controller.emitMessage(
      JSON.stringify({
        data: {
          collected_at: '2026-06-14T01:12:00Z',
          overview: {
            running_containers: 5,
            abnormal_containers: 2,
            cpu_total_percent: 64.5,
            memory_total_usage_bytes: 3221225472,
            memory_total_limit_bytes: 4294967296,
            memory_total_percent: 75,
          },
          hotspots: {
            cpu_top: [],
            memory_top: [],
          },
          anomalies: [],
        },
      }),
    );

    expect(selectContainerDashboardSummaryView()?.overview.cpuTotalPercent).toBe(64.5);
    expect(selectContainerDashboardSummaryView()?.overview.collectedAt).toBe('2026-06-14T01:12:00Z');
  });

  it('accepts dashboard summary topic payloads wrapped by the server publish envelope', () => {
    seedContainerDashboardSummary(createDashboardSummary());
    acquireContainerDashboardSummarySubscription();

    const controller = realtimeMocks.controllers.at(-1)!;
    controller.emitMessage(
      JSON.stringify({
        data: {
          topic: 'container.dashboard.summary',
          collected_at: '2026-06-14T01:13:00Z',
          data: {
            collected_at: '2026-06-14T01:13:00Z',
            overview: {
              running_containers: 9,
              abnormal_containers: 1,
              cpu_total_percent: 71.4,
              memory_total_usage_bytes: 2147483648,
              memory_total_limit_bytes: 4294967296,
              memory_total_percent: 50,
            },
            hotspots: {
              cpu_top: [],
              memory_top: [],
            },
            anomalies: [],
          },
        },
      }),
    );

    expect(selectContainerDashboardSummaryView()?.overview.cpuTotalPercent).toBe(71.4);
    expect(selectContainerDashboardSummaryView()?.overview.collectedAt).toBe('2026-06-14T01:13:00Z');
  });

  it('rejects dashboard summary topic payloads when hotspots cpu_top or memory_top are missing', () => {
    seedContainerDashboardSummary(createDashboardSummary());
    acquireContainerDashboardSummarySubscription();

    const controller = realtimeMocks.controllers.at(-1)!;
    controller.emitMessage(
      JSON.stringify({
        data: {
          collected_at: '2026-06-14T01:14:00Z',
          overview: {
            running_containers: 7,
            abnormal_containers: 2,
            cpu_total_percent: 45.1,
            memory_total_usage_bytes: 2147483648,
            memory_total_limit_bytes: 4294967296,
            memory_total_percent: 50,
          },
          hotspots: {
            cpu_top: [],
          },
          anomalies: [],
        },
      }),
    );

    expect(selectContainerDashboardSummaryView()?.overview.cpuTotalPercent).toBe(32.5);
    expect(selectContainerDashboardSummaryView()?.overview.collectedAt).toBe('2026-06-14T01:09:00Z');
  });

  it('rejects dashboard summary topic payloads when the payload root is an array', () => {
    seedContainerDashboardSummary(createDashboardSummary());
    acquireContainerDashboardSummarySubscription();

    const controller = realtimeMocks.controllers.at(-1)!;
    controller.emitMessage(JSON.stringify({ data: [] }));

    expect(selectContainerDashboardSummaryView()?.overview.cpuTotalPercent).toBe(32.5);
    expect(selectContainerDashboardSummaryView()?.overview.collectedAt).toBe('2026-06-14T01:09:00Z');
  });

  it('shares one dashboard summary realtime controller across repeated acquires', () => {
    acquireContainerDashboardSummarySubscription();
    acquireContainerDashboardSummarySubscription();

    expect(realtimeMocks.openRealtimeTopicSocket).toHaveBeenCalledTimes(1);
    expect(selectContainerDashboardRealtimeState()).toBe('connecting');
  });

  it('keeps the dashboard summary realtime socket alive until the last release', () => {
    acquireContainerDashboardSummarySubscription();
    acquireContainerDashboardSummarySubscription();
    const controller = realtimeMocks.controllers.at(-1)!;

    releaseContainerDashboardSummarySubscription();
    expect(controller.close).not.toHaveBeenCalled();

    releaseContainerDashboardSummarySubscription();
    vi.runOnlyPendingTimers();

    expect(controller.close).toHaveBeenCalledTimes(1);
    expect(selectContainerDashboardRealtimeState()).toBe('idle');
  });

  it('clears dashboard summary state without touching list state', () => {
    seedContainerList([createSummary()]);
    seedContainerDashboardSummary(createDashboardSummary());

    clearContainerDashboardSummary();

    expect(selectContainerDashboardSummaryView()).toBeNull();
    expect(selectContainerListViews()).toHaveLength(1);
  });

  it('applies list topic payloads through the shared stats authority', () => {
    seedContainerList([createSummary()], 'container:list');
    acquireContainerSummaryCollectionSubscription();

    const controller = realtimeMocks.controllers.at(-1)!;
    controller.emitMessage(
      JSON.stringify({
        data: {
          items: [
            {
              id: 'container-1',
              resource: {
                ...createSummary().resource!,
                cpu_percent: 64.5,
                collected_at: '2026-06-14T01:12:00Z',
              },
            },
          ],
        },
      }),
    );

    expect(selectContainerListViews()[0]?.resource?.cpu_percent).toBe(64.5);
  });

  it('does not let an older http seed override a fresher realtime snapshot', () => {
    seedContainerDetail(createDetail());
    applyContainerRealtimeStats('container-1', {
      ...createDetail().resource!,
      cpu_percent: 88.8,
      collected_at: '2026-06-14T01:11:00Z',
    });

    seedContainerDetail(
      createDetail({
        cpu_percent: 7.5,
        collected_at: '2026-06-14T01:10:00Z',
      }),
    );

    const detail = selectContainerDetailView('container-1');

    expect(detail?.resource?.cpu_percent).toBe(88.8);
    expect(detail?.resource?.collected_at).toBe('2026-06-14T01:11:00Z');
  });

  it('keeps a bounded history ring buffer separate from latest stats state', () => {
    seedContainerDetail(createDetail());
    applyContainerRealtimeStats('container-1', {
      ...createDetail().resource!,
      cpu_percent: 30.5,
      collected_at: '2026-06-14T01:10:00Z',
    });
    applyContainerRealtimeStats('container-1', {
      ...createDetail().resource!,
      cpu_percent: 42.1,
      collected_at: '2026-06-14T01:11:00Z',
    });

    const history = selectContainerStatsHistory('container-1');
    const detail = selectContainerDetailView('container-1');

    expect(history).toHaveLength(3);
    expect(history.at(-1)?.resource.cpu_percent).toBe(42.1);
    expect(detail?.resource?.cpu_percent).toBe(42.1);
  });

  it('replaces the latest history snapshot when realtime data arrives with the same collected_at timestamp', () => {
    seedContainerDetail(createDetail());

    applyContainerRealtimeStats('container-1', {
      ...createDetail().resource!,
      cpu_percent: 66.6,
      collected_at: '2026-06-14T01:09:00Z',
    });

    const history = selectContainerStatsHistory('container-1');

    expect(history).toHaveLength(1);
    expect(history[0]?.resource.cpu_percent).toBe(66.6);
    expect(history[0]?.resource.collected_at).toBe('2026-06-14T01:09:00Z');
  });

  it('marks realtime change direction without treating http seed as a highlight trigger', () => {
    seedContainerDetail(createDetail());

    expect(selectContainerStatsChangeState('container-1')).toEqual({
      changedAt: null,
      cpu: 'none',
      memory: 'none',
    });

    applyContainerRealtimeStats('container-1', {
      ...createDetail().resource!,
      cpu_percent: 35.5,
      memory_percent: 47.5,
      collected_at: '2026-06-14T01:11:00Z',
    });

    expect(selectContainerStatsChangeState('container-1').cpu).toBe('up');
    expect(selectContainerStatsChangeState('container-1').memory).toBe('down');
  });

  it('expires the change highlight window after 800ms', () => {
    seedContainerDetail(createDetail());
    applyContainerRealtimeStats('container-1', {
      ...createDetail().resource!,
      cpu_percent: 35.5,
      collected_at: '2026-06-14T01:11:00Z',
    });

    expect(selectContainerStatsChangeState('container-1').cpu).toBe('up');

    vi.advanceTimersByTime(900);

    expect(selectContainerStatsChangeState('container-1')).toEqual({
      changedAt: null,
      cpu: 'none',
      memory: 'none',
    });
  });

  it('does not mutate reactive state when reading an expired change highlight', () => {
    seedContainerDetail(createDetail());
    applyContainerRealtimeStats('container-1', {
      ...createDetail().resource!,
      cpu_percent: 35.5,
      collected_at: '2026-06-14T01:11:00Z',
    });

    vi.advanceTimersByTime(900);

    expect(() => selectContainerStatsChangeState('container-1')).not.toThrow();
    expect(selectContainerStatsChangeState('container-1')).toEqual({
      changedAt: null,
      cpu: 'none',
      memory: 'none',
    });
  });

  it('shares one realtime subscription controller across multiple acquires of the same container id', () => {
    acquireContainerStatsSubscription('container-1');
    acquireContainerStatsSubscription('container-1');

    expect(realtimeMocks.openRealtimeTopicSocket).toHaveBeenCalledTimes(1);
    expect(selectContainerStatsRealtimeState('container-1')).toBe('connecting');
  });

  it('keeps the realtime socket alive until the last release', () => {
    acquireContainerStatsSubscription('container-1');
    acquireContainerStatsSubscription('container-1');
    const controller = realtimeMocks.controllers.at(-1)!;

    releaseContainerStatsSubscription('container-1');
    expect(controller.close).not.toHaveBeenCalled();

    releaseContainerStatsSubscription('container-1');
    vi.runOnlyPendingTimers();

    expect(controller.close).toHaveBeenCalledTimes(1);
    expect(selectContainerStatsRealtimeState('container-1')).toBe('idle');
  });

  it('shares one list-level realtime controller across multiple collection acquires', () => {
    seedContainerList([createSummary()], 'container:list');
    acquireContainerSummaryCollectionSubscription();
    acquireContainerSummaryCollectionSubscription();

    expect(realtimeMocks.openRealtimeTopicSocket).toHaveBeenCalledTimes(1);
  });

  it('keeps the list-level realtime socket alive until the last collection release', () => {
    seedContainerList([createSummary()], 'container:list');
    acquireContainerSummaryCollectionSubscription();
    acquireContainerSummaryCollectionSubscription();
    const controller = realtimeMocks.controllers.at(-1)!;

    releaseContainerSummaryCollectionSubscription();
    expect(controller.close).not.toHaveBeenCalled();

    releaseContainerSummaryCollectionSubscription();
    vi.runOnlyPendingTimers();

    expect(controller.close).toHaveBeenCalledTimes(1);
  });
});
