import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { ContainerDetailRecord, ContainerSummaryRecord } from '../types/container';
import {
  acquireContainerStatsSubscription,
  applyContainerRealtimeStats,
  releaseContainerStatsSubscription,
  resetContainerStatsManager,
  seedContainerDetail,
  seedContainerList,
  selectContainerDetailView,
  selectContainerListViews,
  selectContainerStatsRealtimeState,
} from './stats-manager';

const realtimeMocks = vi.hoisted(() => ({
  controllers: [] as Array<{
    close: ReturnType<typeof vi.fn>;
    reconnect: ReturnType<typeof vi.fn>;
  }>,
  openRealtimeTopicSocket: vi.fn(() => {
    const controller = {
      close: vi.fn(),
      reconnect: vi.fn(),
    };
    realtimeMocks.controllers.push(controller);
    return controller;
  }),
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

describe('container stats manager', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    resetContainerStatsManager();
    realtimeMocks.controllers = [];
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
});
