import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useProjectImportFlow } from './useProjectImportFlow';

const mocks = vi.hoisted(() => ({
  postProjectImportExecute: vi.fn(),
  postProjectImportInspect: vi.fn(),
}));

vi.mock('../api/import', () => ({
  postProjectImportExecute: mocks.postProjectImportExecute,
  postProjectImportInspect: mocks.postProjectImportInspect,
}));

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

describe('useProjectImportFlow', () => {
  beforeEach(() => {
    mocks.postProjectImportExecute.mockReset();
    mocks.postProjectImportInspect.mockReset();
  });

  it('inspects a selected directory and hydrates editable fields from inspect output', async () => {
    mocks.postProjectImportInspect.mockResolvedValue({
      inspection_id: 'inspect-1',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'apps/demo' },
      resolved_working_directory: '/srv/apps/demo',
      canonical_project_name: 'demo',
      display_name_suggested: 'Demo Service',
      compose_files: [{ display_path: 'compose.yaml' }],
      env_files: [{ display_path: '.env' }],
      services: ['web', 'worker'],
      network_names: ['default'],
      volume_names: ['data'],
      warnings: [],
      conflicts: [],
      validation_status: 'ready',
      config_hash: 'abc',
    });

    const flow = useProjectImportFlow((key: string) => key);
    await flow.selectDirectory({ provider: 'local', root_id: 'managed-root', path: 'apps/demo' });

    expect(mocks.postProjectImportInspect).toHaveBeenCalledWith({
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'apps/demo' },
    });
    expect(flow.inspectResult.value?.inspection_id).toBe('inspect-1');
    expect(flow.displayName.value).toBe('Demo Service');
    expect(flow.canImport.value).toBe(true);
  });

  it('submits import using inspection authority and editable overrides only', async () => {
    mocks.postProjectImportInspect.mockResolvedValue({
      inspection_id: 'inspect-2',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: '' },
      resolved_working_directory: '/srv',
      canonical_project_name: 'srv',
      display_name_suggested: 'Srv',
      compose_files: [],
      env_files: [],
      services: [],
      network_names: [],
      volume_names: [],
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
    await flow.selectDirectory({ provider: 'local', root_id: 'managed-root', path: '' });
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
    mocks.postProjectImportInspect.mockResolvedValue({
      inspection_id: 'inspect-3',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'conflict' },
      resolved_working_directory: '/srv/conflict',
      canonical_project_name: 'conflict',
      compose_files: [],
      env_files: [],
      services: [],
      network_names: [],
      volume_names: [],
      warnings: [],
      conflicts: ['Canonical project name already exists'],
    });

    const flow = useProjectImportFlow((key: string) => key);
    await flow.selectDirectory({ provider: 'local', root_id: 'managed-root', path: 'conflict' });

    expect(flow.canImport.value).toBe(false);
  });
});
