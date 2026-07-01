import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { ProjectImportRuntimeCandidate } from '../types/import';
import { useProjectImportFlow } from './useProjectImportFlow';

const mocks = vi.hoisted(() => ({
  postProjectImportExecute: vi.fn(),
  postProjectImportRuntimeInspect: vi.fn(),
}));

vi.mock('../api/import', () => ({
  postProjectImportExecute: mocks.postProjectImportExecute,
  postProjectImportRuntimeInspect: mocks.postProjectImportRuntimeInspect,
}));

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

describe('useProjectImportFlow', () => {
  beforeEach(() => {
    mocks.postProjectImportExecute.mockReset();
    mocks.postProjectImportRuntimeInspect.mockReset();
  });

  function buildRuntimeCandidate(
    overrides: Partial<ProjectImportRuntimeCandidate> = {},
  ): ProjectImportRuntimeCandidate {
    return {
      candidate_key: 'runtime:demo',
      canonical_project_name: 'demo',
      status: 'ready',
      status_reason_codes: [],
      importable: true,
      runtime_type: 'docker',
      runtime_version: '28.3.3',
      working_directory: '/srv/apps/demo',
      working_directory_source: 'runtime_label',
      config_files: ['/srv/apps/demo/compose.yaml'],
      service_names: ['web', 'worker'],
      container_counts: {
        running: 1,
        stopped: 1,
        total: 2,
      },
      warnings: [],
      ...overrides,
    };
  }

  it('inspects a ready candidate and hydrates editable fields from inspect output', async () => {
    mocks.postProjectImportRuntimeInspect.mockResolvedValue({
      inspection_id: 'inspect-1',
      candidate_key: 'runtime:demo',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'apps/demo' },
      resolved_working_directory: '/srv/apps/demo',
      canonical_project_name: 'demo',
      display_name_suggested: 'Demo Service',
      compose_files: [{ display_path: 'compose.yaml' }],
      env_files: [{ display_path: '.env' }],
      services: ['web', 'worker'],
      networks: ['default'],
      volumes: ['data'],
      warnings: [],
      conflicts: [],
      validation_status: 'ready',
      config_hash: 'abc',
    });

    const flow = useProjectImportFlow((key: string) => key);
    await flow.inspectCandidate(buildRuntimeCandidate());

    expect(mocks.postProjectImportRuntimeInspect).toHaveBeenCalledWith({
      candidate_key: 'runtime:demo',
    });
    expect(flow.inspectResult.value?.inspection_id).toBe('inspect-1');
    expect(flow.displayName.value).toBe('Demo Service');
    expect(flow.canImport.value).toBe(true);
  });

  it('submits import using inspection authority and editable overrides only', async () => {
    mocks.postProjectImportRuntimeInspect.mockResolvedValue({
      inspection_id: 'inspect-2',
      candidate_key: 'runtime:srv',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: '' },
      resolved_working_directory: '/srv',
      canonical_project_name: 'srv',
      display_name_suggested: 'Srv',
      compose_files: [],
      env_files: [],
      services: [],
      networks: [],
      volumes: [],
      warnings: [],
      conflicts: [],
    });
    mocks.postProjectImportExecute.mockResolvedValue({
      project: {
        id: 1,
        display_name: 'Srv Override',
      },
    });

    const flow = useProjectImportFlow((key: string) => key);
    await flow.inspectCandidate(
      buildRuntimeCandidate({
        candidate_key: 'runtime:srv',
        canonical_project_name: 'srv',
        config_files: ['/srv/compose.yaml'],
        service_names: [],
        container_counts: { running: 0, stopped: 0, total: 0 },
        working_directory: '/srv',
      }),
    );
    flow.displayName.value = 'Srv Override';
    flow.canonicalProjectNameOverride.value = 'srv-override';

    await flow.submitImport();

    expect(mocks.postProjectImportExecute).toHaveBeenCalledWith({
      inspection_id: 'inspect-2',
      display_name: 'Srv Override',
      canonical_project_name_override: 'srv-override',
    });
  });

  it('blocks import when inspect returns conflicts', async () => {
    mocks.postProjectImportRuntimeInspect.mockResolvedValue({
      inspection_id: 'inspect-3',
      candidate_key: 'runtime:conflict',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'conflict' },
      resolved_working_directory: '/srv/conflict',
      canonical_project_name: 'conflict',
      compose_files: [],
      env_files: [],
      services: [],
      networks: [],
      volumes: [],
      warnings: [],
      conflicts: ['Canonical project name already exists'],
    });

    const flow = useProjectImportFlow((key: string) => key);
    await flow.inspectCandidate(
      buildRuntimeCandidate({
        candidate_key: 'runtime:conflict',
        canonical_project_name: 'conflict',
        config_files: ['/srv/conflict/compose.yaml'],
        service_names: [],
        container_counts: { running: 0, stopped: 0, total: 0 },
        working_directory: '/srv/conflict',
      }),
    );

    expect(flow.canImport.value).toBe(false);
  });

  it('ignores stale inspect responses when a newer candidate inspect finishes later', async () => {
    let resolveFirst: (value: Record<string, unknown>) => void = () => {};
    let resolveSecond: (value: Record<string, unknown>) => void = () => {};

    mocks.postProjectImportRuntimeInspect
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveFirst = resolve;
          }),
      )
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveSecond = resolve;
          }),
      );

    const flow = useProjectImportFlow((key: string) => key);
    const firstSelection = flow.inspectCandidate(
      buildRuntimeCandidate({
        candidate_key: 'runtime:first',
        canonical_project_name: 'first',
        config_files: ['/srv/apps/first/compose.yaml'],
        service_names: [],
        container_counts: { running: 0, stopped: 0, total: 0 },
        working_directory: '/srv/apps/first',
      }),
    );
    const secondSelection = flow.inspectCandidate(
      buildRuntimeCandidate({
        candidate_key: 'runtime:second',
        canonical_project_name: 'second',
        config_files: ['/srv/apps/second/compose.yaml'],
        service_names: [],
        container_counts: { running: 0, stopped: 0, total: 0 },
        working_directory: '/srv/apps/second',
      }),
    );

    resolveSecond({
      inspection_id: 'inspect-second',
      candidate_key: 'runtime:second',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'apps/second' },
      resolved_working_directory: '/srv/apps/second',
      canonical_project_name: 'second',
      display_name_suggested: 'Second Service',
      compose_files: [],
      env_files: [],
      services: [],
      networks: [],
      volumes: [],
      warnings: [],
      conflicts: [],
    });

    await expect(secondSelection).resolves.toBe('applied');
    expect(flow.inspectResult.value?.inspection_id).toBe('inspect-second');
    expect(flow.displayName.value).toBe('Second Service');
    expect(flow.selectedCandidateKey.value).toBe('runtime:second');

    resolveFirst({
      inspection_id: 'inspect-first',
      candidate_key: 'runtime:first',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'apps/first' },
      resolved_working_directory: '/srv/apps/first',
      canonical_project_name: 'first',
      display_name_suggested: 'First Service',
      compose_files: [],
      env_files: [],
      services: [],
      networks: [],
      volumes: [],
      warnings: [],
      conflicts: [],
    });

    await expect(firstSelection).resolves.toBe('stale');
    expect(flow.inspectResult.value?.inspection_id).toBe('inspect-second');
    expect(flow.displayName.value).toBe('Second Service');
    expect(flow.selectedCandidateKey.value).toBe('runtime:second');
  });
});
