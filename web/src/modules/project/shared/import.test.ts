import { describe, expect, it } from 'vitest';

import { hasBlockingImportConflicts, normalizeApplicationImportInspectResponse } from './import';

const lifecycleConfiguration = {
  strategy_kind: 'standard' as const,
  profiles: [],
  down_before_redeploy: true,
  pull_before_redeploy: false,
  build_before_up: false,
  force_recreate: false,
  remove_orphans: true,
  wait_after_up: false,
  wait_timeout_seconds: 120,
  renew_anon_volumes: false,
  prune_images_after_redeploy: false,
};

describe('project import normalization helpers', () => {
  it('preserves structured runtime network and volume resources during inspect normalization', () => {
    const result = normalizeApplicationImportInspectResponse({
      inspection_id: 'inspect-structured',
      expires_at: '2026-07-11T08:05:00Z',
      candidate_key: 'runtime:structured',
      resolved_workspace_path: '/srv/structured',
      compose_project_name: 'structured',
      compose_project_name_source: 'computed',
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
      lifecycle_configuration: lifecycleConfiguration,
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
    const result = normalizeApplicationImportInspectResponse({
      inspection_id: 'inspect-files',
      expires_at: '2026-07-11T08:05:00Z',
      candidate_key: 'runtime:files',
      resolved_workspace_path: '/srv/files',
      compose_project_name: 'files',
      compose_project_name_source: 'computed',
      display_name_suggested: 'Files',
      compose_files: [
        {
          kind: 'compose',
          role: 'primary',
          absolute_path: '/srv/files/compose.yaml',
          display_path: 'compose.yaml',
          order_index: 0,
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
      lifecycle_configuration: lifecycleConfiguration,
    });

    expect(result?.compose_files).toEqual([
      {
        kind: 'compose',
        role: 'primary',
        absolute_path: '/srv/files/compose.yaml',
        display_path: 'compose.yaml',
        order_index: 0,
      },
    ]);
    expect(result?.env_files).toEqual([
      {
        kind: 'env',
        role: 'detected',
        absolute_path: '/srv/files/.env',
        display_path: '.env',
        order_index: 0,
        last_observed_hash: 'hash-env',
      },
    ]);
  });

  it('treats an explicit conflict status as blocking even without conflict details', () => {
    expect(
      hasBlockingImportConflicts({
        inspection_id: 'inspect-conflict',
        expires_at: '2026-07-11T08:05:00Z',
        candidate_key: 'runtime:conflict',
        resolved_workspace_path: '/srv/conflict',
        compose_project_name: 'conflict',
        compose_project_name_source: 'computed',
        display_name_suggested: 'Conflict',
        compose_files: [],
        env_files: [],
        services: [],
        networks: [],
        volumes: [],
        runtime_members: [],
        warnings: [],
        conflicts: [],
        validation_status: 'conflict',
        config_hash: 'hash-conflict',
        lifecycle_configuration: lifecycleConfiguration,
      }),
    ).toBe(true);
  });
});
