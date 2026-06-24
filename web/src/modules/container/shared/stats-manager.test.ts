import { beforeEach, describe, expect, it } from 'vitest';

import type { ContainerDetailRecord, ContainerSummaryRecord } from '../types/container';
import {
  applyContainerRealtimeStats,
  resetContainerStatsManager,
  seedContainerDetail,
  seedContainerList,
  selectContainerDetailView,
  selectContainerListViews,
} from './stats-manager';

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
    resetContainerStatsManager();
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
});
