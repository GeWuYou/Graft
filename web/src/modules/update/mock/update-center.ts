import type { UpdateCenterDataSource } from '../types/preview';
import type { CreateUpdateOperationRequest, UpdateOperation, UpdateStatus } from '../types/update';

export const updateCenterPreviewStatus: UpdateStatus = {
  current_version: '0.9.8-beta.2',
  channel: 'beta',
  update_policy: 'beta',
  policy_initialized: true,
  available_releases: [
    {
      version: '0.9.8-beta.3',
      channel: 'beta',
      notes: 'Verified Beta release.',
      published_at: '2026-07-24T08:00:00Z',
      manifest_url: 'https://example.test/releases/v0.9.8-beta.3/manifest.json',
      server_digest: 'sha256:preview-server',
      web_digest: 'sha256:preview-web',
    },
    {
      version: '0.9.8',
      channel: 'stable',
      notes: 'Verified stable release.',
      published_at: '2026-07-22T08:00:00Z',
      manifest_url: 'https://example.test/releases/v0.9.8/manifest.json',
      server_digest: 'sha256:preview-server-stable',
      web_digest: 'sha256:preview-web-stable',
    },
  ],
  latest: {
    version: '0.9.8-beta.3',
    channel: 'beta',
    notes:
      '## Fixed\n\n- Update Center keeps deployment paths out of release notes.\n- Compose root candidates are selected only when an upgrade starts.',
    published_at: '2026-07-24T08:00:00Z',
    manifest_url: 'https://example.test/releases/v0.9.8-beta.3/manifest.json',
    notes_url: 'https://example.test/releases/v0.9.8-beta.3/notes',
    checksums_url: 'https://example.test/releases/v0.9.8-beta.3/checksums.txt',
    upgrade_notes: 'A Compose root is required before the upgrade can be submitted.',
    server_digest: 'sha256:preview-server',
    web_digest: 'sha256:preview-web',
  },
  installation_profile: {
    declared_mode: 'compose',
    detected_mode: 'compose',
    capability: 'compose_upgrade_available',
    guidance: 'The Compose root was discovered from the running container.',
    compose_root_source: 'docker_discovered',
    compose_candidates: [
      {
        key: 'working-dir',
        host_path: '/home/gewuyou/apps/graft/beta',
        compose_files: ['/home/gewuyou/apps/graft/beta/compose.yaml'],
        project_name: 'graft-beta',
        confidence: 'high',
      },
    ],
  },
  checked_at: '2026-07-24T08:00:00Z',
  last_successful_at: '2026-07-24T08:00:00Z',
  cache_stale: false,
  check_error: '',
};

/** 创建独立的内存数据源，使每次开发预览都不读取或改写后端状态。 */
export function createUpdateCenterPreviewDataSource(): UpdateCenterDataSource {
  const operations: UpdateOperation[] = [];

  return {
    permissions: { check: true, manage: true },
    async getStatus() {
      return updateCenterPreviewStatus;
    },
    async checkForUpdates() {
      return { ...updateCenterPreviewStatus, checked_at: new Date().toISOString() };
    },
    async getOperations() {
      return operations;
    },
    async getFailureDiagnostic() {
      return null;
    },
    async createOperation(payload: CreateUpdateOperationRequest) {
      const now = new Date().toISOString();
      const operation: UpdateOperation = {
        operation_id: `preview-${operations.length + 1}`,
        source_version: updateCenterPreviewStatus.current_version,
        update_policy: payload.update_policy,
        target_version: payload.target_version,
        task_id: operations.length + 1,
        status: 'PLANNING',
        recovery_completed: false,
        created_at: now,
        started_at: now,
      };
      operations.unshift(operation);
      return operation;
    },
  };
}
