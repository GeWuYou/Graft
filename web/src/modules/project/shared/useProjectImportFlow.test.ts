import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { ApplicationImportRuntimeCandidate } from '../types/import';
import { useApplicationImportFlow } from './useProjectImportFlow';

const mocks = vi.hoisted(() => ({
  postApplicationImportExecute: vi.fn(),
  postApplicationImportRuntimeInspect: vi.fn(),
}));

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

vi.mock('../api/import', () => ({
  postApplicationImportExecute: mocks.postApplicationImportExecute,
  postApplicationImportRuntimeInspect: mocks.postApplicationImportRuntimeInspect,
}));

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

describe('useApplicationImportFlow', () => {
  beforeEach(() => {
    mocks.postApplicationImportExecute.mockReset();
    mocks.postApplicationImportRuntimeInspect.mockReset();
  });

  function buildRuntimeCandidate(
    overrides: Partial<ApplicationImportRuntimeCandidate> = {},
  ): ApplicationImportRuntimeCandidate {
    return {
      candidate_key: 'runtime:demo',
      compose_project_name: 'demo',
      status: 'ready',
      status_reason_codes: [],
      importable: true,
      runtime_type: 'docker',
      runtime_version: '28.3.3',
      workspace_path: '/srv/apps/demo',
      workspace_path_source: 'runtime_label',
      config_files: ['/srv/apps/demo/compose.yaml'],
      service_names: ['web', 'worker'],
      container_counts: {
        running: 1,
        stopped: 1,
        transitioning: 0,
        issue: 0,
        total: 2,
      },
      warnings: [],
      ...overrides,
    };
  }

  it('inspects a ready candidate and hydrates editable fields from inspect output', async () => {
    mocks.postApplicationImportRuntimeInspect.mockResolvedValue({
      inspection_id: 'inspect-1',
      candidate_key: 'runtime:demo',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'apps/demo' },
      resolved_workspace_path: '/srv/apps/demo',
      compose_project_name: 'demo',
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
      lifecycle_configuration: {
        strategy_kind: 'standard',
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
      },
    });

    const flow = useApplicationImportFlow((key: string) => key);
    await flow.inspectCandidate(buildRuntimeCandidate());

    expect(mocks.postApplicationImportRuntimeInspect).toHaveBeenCalledWith({
      candidate_key: 'runtime:demo',
    });
    expect(flow.inspectResult.value?.inspection_id).toBe('inspect-1');
    expect(flow.displayName.value).toBe('Demo Service');
    expect(flow.canImport.value).toBe(true);
  });

  it('submits import using inspection authority and editable overrides only', async () => {
    mocks.postApplicationImportRuntimeInspect.mockResolvedValue({
      inspection_id: 'inspect-2',
      candidate_key: 'runtime:srv',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: '' },
      resolved_workspace_path: '/srv',
      compose_project_name: 'srv',
      display_name_suggested: 'Srv',
      compose_files: [],
      env_files: [],
      services: [],
      networks: [],
      volumes: [],
      warnings: [],
      conflicts: [],
      lifecycle_configuration: {
        strategy_kind: 'standard',
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
        managed_service_names: [],
      },
    });
    mocks.postApplicationImportExecute.mockResolvedValue({
      project: {
        id: 1,
        display_name: 'Srv Override',
      },
    });

    const flow = useApplicationImportFlow((key: string) => key);
    await flow.inspectCandidate(
      buildRuntimeCandidate({
        candidate_key: 'runtime:srv',
        compose_project_name: 'srv',
        config_files: ['/srv/compose.yaml'],
        service_names: [],
        container_counts: { running: 0, stopped: 0, transitioning: 0, issue: 0, total: 0 },
        workspace_path: '/srv',
      }),
    );
    flow.displayName.value = 'Srv Override';
    flow.composeProjectNameOverride.value = 'srv-override';
    expect(flow.prepareLifecycleConfiguration()).toBe(true);

    await flow.submitImport();

    expect(mocks.postApplicationImportExecute).toHaveBeenCalledWith({
      inspection_id: 'inspect-2',
      display_name: 'Srv Override',
      compose_project_name_override: 'srv-override',
      lifecycle_configuration: {
        strategy_kind: 'standard',
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
        managed_service_names: [],
      },
    });
  });

  it('preserves an edited lifecycle draft when route sync prepares it again', async () => {
    mocks.postApplicationImportRuntimeInspect.mockResolvedValue({
      inspection_id: 'inspect-preserved',
      candidate_key: 'runtime:demo',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'apps/demo' },
      resolved_workspace_path: '/srv/apps/demo',
      compose_project_name: 'demo',
      compose_files: [],
      env_files: [],
      services: [],
      networks: [],
      volumes: [],
      warnings: [],
      conflicts: [],
      lifecycle_configuration: lifecycleConfiguration,
    });

    const flow = useApplicationImportFlow((key: string) => key);
    await flow.inspectCandidate(buildRuntimeCandidate());
    expect(flow.prepareLifecycleConfiguration()).toBe(true);
    flow.lifecycleDraft.value!.wait_after_up = true;

    expect(flow.prepareLifecycleConfiguration()).toBe(true);
    expect(flow.lifecycleDraft.value?.wait_after_up).toBe(true);
  });

  it('preserves editable import fields across a successful refresh', async () => {
    mocks.postApplicationImportRuntimeInspect.mockResolvedValue({
      inspection_id: 'inspect-refresh',
      candidate_key: 'runtime:demo',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'apps/demo' },
      resolved_workspace_path: '/srv/apps/demo',
      compose_project_name: 'demo',
      display_name_suggested: 'Demo Suggested',
      compose_files: [],
      env_files: [],
      services: [],
      networks: [],
      volumes: [],
      warnings: [],
      conflicts: [],
      lifecycle_configuration: lifecycleConfiguration,
    });

    const flow = useApplicationImportFlow((key: string) => key);
    await flow.inspectCandidate(buildRuntimeCandidate());
    flow.displayName.value = 'Edited Demo';
    flow.composeProjectNameOverride.value = 'edited-demo';
    expect(flow.prepareLifecycleConfiguration()).toBe(true);
    flow.lifecycleDraft.value!.profiles = ['production'];
    flow.lifecycleDraft.value!.wait_after_up = true;
    await expect(flow.refreshInspect()).resolves.toBe('applied');

    expect(flow.canImport.value).toBe(true);
    expect(flow.displayName.value).toBe('Edited Demo');
    expect(flow.composeProjectNameOverride.value).toBe('edited-demo');
    expect(flow.lifecycleDraft.value).toMatchObject({
      profiles: ['production'],
      wait_after_up: true,
    });
  });

  it('blocks import until an invalidated inspection session refreshes successfully', async () => {
    mocks.postApplicationImportRuntimeInspect.mockResolvedValue({
      inspection_id: 'inspect-invalidated',
      candidate_key: 'runtime:demo',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'apps/demo' },
      resolved_workspace_path: '/srv/apps/demo',
      compose_project_name: 'demo',
      compose_files: [],
      env_files: [],
      services: [],
      networks: [],
      volumes: [],
      warnings: [],
      conflicts: [],
      lifecycle_configuration: lifecycleConfiguration,
    });

    const flow = useApplicationImportFlow((key: string) => key);
    await flow.inspectCandidate(buildRuntimeCandidate());
    expect(flow.canImport.value).toBe(true);

    flow.invalidateInspectionSession();
    expect(flow.canImport.value).toBe(false);

    await expect(flow.refreshInspect()).resolves.toBe('applied');
    expect(flow.canImport.value).toBe(true);
  });

  it('blocks import when inspect returns conflicts', async () => {
    mocks.postApplicationImportRuntimeInspect.mockResolvedValue({
      inspection_id: 'inspect-3',
      candidate_key: 'runtime:conflict',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'conflict' },
      resolved_workspace_path: '/srv/conflict',
      compose_project_name: 'conflict',
      compose_files: [],
      env_files: [],
      services: [],
      networks: [],
      volumes: [],
      warnings: [],
      conflicts: ['Canonical project name already exists'],
      lifecycle_configuration: lifecycleConfiguration,
    });

    const flow = useApplicationImportFlow((key: string) => key);
    await flow.inspectCandidate(
      buildRuntimeCandidate({
        candidate_key: 'runtime:conflict',
        compose_project_name: 'conflict',
        config_files: ['/srv/conflict/compose.yaml'],
        service_names: [],
        container_counts: { running: 0, stopped: 0, transitioning: 0, issue: 0, total: 0 },
        workspace_path: '/srv/conflict',
      }),
    );

    expect(flow.canImport.value).toBe(false);
  });

  it('normalizes nullable inspect arrays before exposing preview state', async () => {
    mocks.postApplicationImportRuntimeInspect.mockResolvedValue({
      inspection_id: 'inspect-nullable',
      candidate_key: 'runtime:nullable',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'nullable' },
      resolved_workspace_path: '/srv/nullable',
      compose_project_name: 'nullable',
      display_name_suggested: 'Nullable Service',
      compose_files: null,
      env_files: null,
      services: null,
      networks: null,
      volumes: null,
      runtime_members: null,
      warnings: null,
      conflicts: null,
      validation_status: 'ready',
      config_hash: 'nullable-hash',
      lifecycle_configuration: lifecycleConfiguration,
    });

    const flow = useApplicationImportFlow((key: string) => key);
    await flow.inspectCandidate(
      buildRuntimeCandidate({
        candidate_key: 'runtime:nullable',
        compose_project_name: 'nullable',
        config_files: ['/srv/nullable/compose.yaml'],
        service_names: [],
        container_counts: { running: 0, stopped: 0, transitioning: 0, issue: 0, total: 0 },
        workspace_path: '/srv/nullable',
      }),
    );

    expect(flow.inspectResult.value?.compose_files).toEqual([]);
    expect(flow.inspectResult.value?.env_files).toEqual([]);
    expect(flow.inspectResult.value?.services).toEqual([]);
    expect(flow.inspectResult.value?.networks).toEqual([]);
    expect(flow.inspectResult.value?.volumes).toEqual([]);
    expect(flow.inspectResult.value?.runtime_members).toEqual([]);
    expect(flow.inspectResult.value?.warnings).toEqual([]);
    expect(flow.inspectResult.value?.conflicts).toEqual([]);
    expect(flow.displayName.value).toBe('Nullable Service');
    expect(flow.canImport.value).toBe(true);
  });

  it('ignores stale inspect responses when a newer candidate inspect finishes later', async () => {
    let resolveFirst: (value: Record<string, unknown>) => void = () => {};
    let resolveSecond: (value: Record<string, unknown>) => void = () => {};

    mocks.postApplicationImportRuntimeInspect
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

    const flow = useApplicationImportFlow((key: string) => key);
    const firstSelection = flow.inspectCandidate(
      buildRuntimeCandidate({
        candidate_key: 'runtime:first',
        compose_project_name: 'first',
        config_files: ['/srv/apps/first/compose.yaml'],
        service_names: [],
        container_counts: { running: 0, stopped: 0, transitioning: 0, issue: 0, total: 0 },
        workspace_path: '/srv/apps/first',
      }),
    );
    const secondSelection = flow.inspectCandidate(
      buildRuntimeCandidate({
        candidate_key: 'runtime:second',
        compose_project_name: 'second',
        config_files: ['/srv/apps/second/compose.yaml'],
        service_names: [],
        container_counts: { running: 0, stopped: 0, transitioning: 0, issue: 0, total: 0 },
        workspace_path: '/srv/apps/second',
      }),
    );

    resolveSecond({
      inspection_id: 'inspect-second',
      candidate_key: 'runtime:second',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'apps/second' },
      resolved_workspace_path: '/srv/apps/second',
      compose_project_name: 'second',
      display_name_suggested: 'Second Service',
      compose_files: [],
      env_files: [],
      services: [],
      networks: [],
      volumes: [],
      warnings: [],
      conflicts: [],
      lifecycle_configuration: lifecycleConfiguration,
    });

    await expect(secondSelection).resolves.toBe('applied');
    expect(flow.inspectResult.value?.inspection_id).toBe('inspect-second');
    expect(flow.displayName.value).toBe('Second Service');
    expect(flow.selectedCandidateKey.value).toBe('runtime:second');

    resolveFirst({
      inspection_id: 'inspect-first',
      candidate_key: 'runtime:first',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'apps/first' },
      resolved_workspace_path: '/srv/apps/first',
      compose_project_name: 'first',
      display_name_suggested: 'First Service',
      compose_files: [],
      env_files: [],
      services: [],
      networks: [],
      volumes: [],
      warnings: [],
      conflicts: [],
      lifecycle_configuration: lifecycleConfiguration,
    });

    await expect(firstSelection).resolves.toBe('stale');
    expect(flow.inspectResult.value?.inspection_id).toBe('inspect-second');
    expect(flow.displayName.value).toBe('Second Service');
    expect(flow.selectedCandidateKey.value).toBe('runtime:second');
  });

  it('invalidates in-flight inspect responses when the flow is reset', async () => {
    let resolveInspect: (value: Record<string, unknown>) => void = () => {};

    mocks.postApplicationImportRuntimeInspect.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveInspect = resolve;
        }),
    );

    const flow = useApplicationImportFlow((key: string) => key);
    const pendingInspect = flow.inspectCandidate(
      buildRuntimeCandidate({
        candidate_key: 'runtime:reset-me',
        compose_project_name: 'reset-me',
        config_files: ['/srv/apps/reset-me/compose.yaml'],
        service_names: [],
        container_counts: { running: 0, stopped: 0, transitioning: 0, issue: 0, total: 0 },
        workspace_path: '/srv/apps/reset-me',
      }),
    );

    flow.reset();
    resolveInspect({
      inspection_id: 'inspect-reset',
      candidate_key: 'runtime:reset-me',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'apps/reset-me' },
      resolved_workspace_path: '/srv/apps/reset-me',
      compose_project_name: 'reset-me',
      display_name_suggested: 'Reset Me',
      compose_files: [],
      env_files: [],
      services: [],
      networks: [],
      volumes: [],
      warnings: [],
      conflicts: [],
    });

    await expect(pendingInspect).resolves.toBe('stale');
    expect(flow.inspectLoading.value).toBe(false);
    expect(flow.inspectResult.value).toBeNull();
    expect(flow.selectedCandidateKey.value).toBe('');
  });
});
