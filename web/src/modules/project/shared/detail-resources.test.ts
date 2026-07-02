import { describe, expect, it } from 'vitest';

import type { ProjectServiceItem } from '../types/project';
import {
  buildProjectNetworkResourceRows,
  buildProjectVolumeResourceRows,
  paginateProjectResourceRows,
  parseDeclaredVolumeMount,
} from './detail-resources';

function createService(
  overrides: Partial<ProjectServiceItem> & Pick<ProjectServiceItem, 'service_name'>,
): ProjectServiceItem {
  const { service_name, ...rest } = overrides;
  return {
    build_context: null,
    container_members: [],
    declared_networks: [],
    declared_ports: [],
    declared_volumes: [],
    image: null,
    running_count: 0,
    service_name,
    stopped_count: 0,
    ...rest,
  };
}

describe('detail-resources', () => {
  it('parses named and anonymous compose volume mounts while skipping bind mounts', () => {
    expect(parseDeclaredVolumeMount('postgres-data:/var/lib/postgresql/data')).toEqual({
      anonymous: false,
      mountTarget: '/var/lib/postgresql/data',
      name: 'postgres-data',
    });

    expect(parseDeclaredVolumeMount('/var/lib/postgresql/data')).toEqual({
      anonymous: true,
      mountTarget: '/var/lib/postgresql/data',
      name: '/var/lib/postgresql/data',
    });

    expect(parseDeclaredVolumeMount('type=volume,target=/cache')).toEqual({
      anonymous: true,
      mountTarget: '/cache',
      name: '/cache',
    });

    expect(parseDeclaredVolumeMount('./data:/cache')).toBeNull();
    expect(parseDeclaredVolumeMount('type=bind,source=./data,target=/cache')).toBeNull();
  });

  it('aggregates network rows from declared service networks', () => {
    const rows = buildProjectNetworkResourceRows([
      createService({
        service_name: 'api',
        container_members: [
          { container_id: '1', container_name: 'api-1', state: 'running' },
          { container_id: '2', container_name: 'api-2', state: 'running' },
        ],
        declared_networks: ['backend', 'frontend'],
      }),
      createService({
        service_name: 'web',
        container_members: [{ container_id: '3', container_name: 'web-1', state: 'running' }],
        declared_networks: ['frontend'],
      }),
    ]);

    expect(rows).toEqual([
      {
        containerCount: 2,
        containers: ['api-1', 'api-2'],
        driver: '-',
        id: 'backend',
        internal: null,
        name: 'backend',
        scope: '-',
        serviceCount: 1,
        services: ['api'],
      },
      {
        containerCount: 3,
        containers: ['api-1', 'api-2', 'web-1'],
        driver: '-',
        id: 'frontend',
        internal: null,
        name: 'frontend',
        scope: '-',
        serviceCount: 2,
        services: ['api', 'web'],
      },
    ]);
  });

  it('aggregates named and anonymous volume rows from declared service volumes', () => {
    const rows = buildProjectVolumeResourceRows([
      createService({
        service_name: 'db',
        container_members: [{ container_id: '1', container_name: 'db-1', state: 'running' }],
        declared_volumes: ['postgres-data:/var/lib/postgresql/data', './tmp:/tmp', '/var/lib/app/cache'],
      }),
      createService({
        service_name: 'worker',
        container_members: [{ container_id: '2', container_name: 'worker-1', state: 'running' }],
        declared_volumes: ['postgres-data:/var/lib/postgresql/data', 'type=volume,target=/var/lib/app/cache'],
      }),
    ]);

    expect(rows).toEqual([
      {
        anonymous: true,
        containerCount: 2,
        containers: ['db-1', 'worker-1'],
        driver: '-',
        id: '/var/lib/app/cache',
        mountTarget: '/var/lib/app/cache',
        mountedBy: ['db', 'worker'],
        name: '/var/lib/app/cache',
      },
      {
        anonymous: false,
        containerCount: 2,
        containers: ['db-1', 'worker-1'],
        driver: '-',
        id: 'postgres-data',
        mountTarget: '/var/lib/postgresql/data',
        mountedBy: ['db', 'worker'],
        name: 'postgres-data',
      },
    ]);
  });

  it('paginates local rows deterministically', () => {
    expect(paginateProjectResourceRows([1, 2, 3, 4, 5], 2, 2)).toEqual([3, 4]);
    expect(paginateProjectResourceRows([1, 2, 3], 1, 50)).toEqual([1, 2, 3]);
  });
});
