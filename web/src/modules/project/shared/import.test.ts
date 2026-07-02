import { describe, expect, it } from 'vitest';

import { normalizeProjectImportInspectResponse } from './import';

describe('project import normalization helpers', () => {
  it('preserves structured runtime network and volume resources during inspect normalization', () => {
    const result = normalizeProjectImportInspectResponse({
      inspection_id: 'inspect-structured',
      candidate_key: 'runtime:structured',
      resolved_working_directory: '/srv/structured',
      canonical_project_name: 'structured',
      canonical_project_name_source: 'computed',
      display_name_suggested: 'Structured',
      compose_files: [],
      env_files: [],
      services: ['web'],
      networks: [
        {
          name: 'frontend',
          driver: 'bridge',
          scope: 'local',
          internal: false,
          containers: ['web-1'],
          container_count: 1,
          services: ['web'],
          service_count: 1,
        },
      ],
      volumes: [
        {
          name: 'project-data',
          driver: 'local',
          anonymous: false,
          mount_target: '/var/lib/data',
          mounted_by: ['web'],
          containers: ['web-1'],
          container_count: 1,
        },
      ],
      runtime_members: [],
      warnings: [],
      conflicts: [],
      validation_status: 'ready',
      config_hash: 'hash-structured',
    });

    expect(result?.networks).toEqual([
      {
        name: 'frontend',
        driver: 'bridge',
        scope: 'local',
        internal: false,
        containers: ['web-1'],
        container_count: 1,
        services: ['web'],
        service_count: 1,
      },
    ]);
    expect(result?.volumes).toEqual([
      {
        name: 'project-data',
        driver: 'local',
        anonymous: false,
        mount_target: '/var/lib/data',
        mounted_by: ['web'],
        containers: ['web-1'],
        container_count: 1,
      },
    ]);
  });

  it('drops malformed inspect file entries instead of accepting any object with a display path', () => {
    const result = normalizeProjectImportInspectResponse({
      inspection_id: 'inspect-files',
      candidate_key: 'runtime:files',
      resolved_working_directory: '/srv/files',
      canonical_project_name: 'files',
      canonical_project_name_source: 'computed',
      display_name_suggested: 'Files',
      compose_files: [
        {
          kind: 'compose',
          role: 'primary',
          absolute_path: '/srv/files/compose.yaml',
          display_path: 'compose.yaml',
          order_index: 0,
          exists_on_last_refresh: true,
        },
        {
          display_path: 'broken.yaml',
        },
      ] as never,
      env_files: [
        {
          kind: 'env',
          role: 'detected',
          absolute_path: '/srv/files/.env',
          display_path: '.env',
          order_index: 0,
          exists_on_last_refresh: true,
          last_observed_hash: 'hash-env',
        },
        {
          display_path: '.env.malformed',
        },
      ] as never,
      services: [],
      networks: [],
      volumes: [],
      runtime_members: [],
      warnings: [],
      conflicts: [],
      validation_status: 'ready',
      config_hash: 'hash-files',
    });

    expect(result?.compose_files).toEqual([
      {
        kind: 'compose',
        role: 'primary',
        absolute_path: '/srv/files/compose.yaml',
        display_path: 'compose.yaml',
        order_index: 0,
        exists_on_last_refresh: true,
      },
    ]);
    expect(result?.env_files).toEqual([
      {
        kind: 'env',
        role: 'detected',
        absolute_path: '/srv/files/.env',
        display_path: '.env',
        order_index: 0,
        exists_on_last_refresh: true,
        last_observed_hash: 'hash-env',
      },
    ]);
  });
});
