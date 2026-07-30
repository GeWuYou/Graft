import type { UpdateCenterDataSource } from '../types/preview';
import type { CreateUpdateOperationRequest, UpdateOperation, UpdateStatus } from '../types/update';

export const updateCenterPreviewStatus: UpdateStatus = {
  current_version: '0.9.8-beta.2',
  channel: 'beta',
  image_tag: 'beta',
  deployment_strategy: 'beta_tracking',
  available_releases: [
    {
      version: '0.9.8-beta.3',
      channel: 'beta',
      notes: 'Verified Beta release.',
      published_at: '2026-07-24T08:00:00Z',
      manifest_url: 'https://example.test/releases/v0.9.8-beta.3/manifest.json',
      server_digest: 'sha256:preview-server',
      web_digest: 'sha256:preview-web',
      server_image: 'ghcr.io/example/graft-server',
      web_image: 'ghcr.io/example/graft-web',
      server_reference: 'ghcr.io/example/graft-server@sha256:preview-server',
      web_reference: 'ghcr.io/example/graft-web@sha256:preview-web',
      runner_image: 'ghcr.io/example/graft-compose-runner',
      runner_digest: 'sha256:preview-runner',
      runner_reference: 'ghcr.io/example/graft-compose-runner@sha256:preview-runner',
    },
    {
      version: '0.9.8',
      channel: 'stable',
      notes: 'Verified stable release.',
      published_at: '2026-07-22T08:00:00Z',
      manifest_url: 'https://example.test/releases/v0.9.8/manifest.json',
      server_digest: 'sha256:preview-server-stable',
      web_digest: 'sha256:preview-web-stable',
      server_image: 'ghcr.io/example/graft-server',
      web_image: 'ghcr.io/example/graft-web',
      server_reference: 'ghcr.io/example/graft-server@sha256:preview-server-stable',
      web_reference: 'ghcr.io/example/graft-web@sha256:preview-web-stable',
      runner_image: 'ghcr.io/example/graft-compose-runner',
      runner_digest: 'sha256:preview-runner',
      runner_reference: 'ghcr.io/example/graft-compose-runner@sha256:preview-runner',
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
    server_image: 'ghcr.io/example/graft-server',
    web_image: 'ghcr.io/example/graft-web',
    server_reference: 'ghcr.io/example/graft-server@sha256:preview-server',
    web_reference: 'ghcr.io/example/graft-web@sha256:preview-web',
    runner_image: 'ghcr.io/example/graft-compose-runner',
    runner_digest: 'sha256:preview-runner',
    runner_reference: 'ghcr.io/example/graft-compose-runner@sha256:preview-runner',
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
  readiness: {
    overall: 'upgrade_blocked',
    ready_count: 3,
    total_count: 5,
    next_action: {
      id: 'view_documentation',
      type: 'documentation',
      label_key: 'platformUpdate.readiness.actions.viewComposeMigration',
      target: '/docs/official-compose-migration',
    },
    checks: [
      {
        id: 'official_compose',
        order: 10,
        state: 'failed',
        severity: 'critical',
        blocking: true,
        title_key: 'platformUpdate.readiness.officialCompose.title',
        summary_key: 'platformUpdate.readiness.officialCompose.failed',
        detail_key: 'platformUpdate.readiness.officialCompose.detail',
        evidence: [
          {
            code: 'declared_deployment_mode',
            state: 'passed',
            label_key: 'platformUpdate.readiness.evidence.declaredDeploymentMode',
            value: 'compose',
            expected: 'compose',
          },
          {
            code: 'detected_deployment_mode',
            state: 'failed',
            label_key: 'platformUpdate.readiness.evidence.detectedDeploymentMode',
            value: 'custom-compose',
            expected: 'official-compose',
          },
        ],
        actions: [
          {
            id: 'view_documentation',
            type: 'documentation',
            label_key: 'platformUpdate.readiness.actions.viewComposeMigration',
            target: '/docs/official-compose-migration',
          },
        ],
      },
      {
        id: 'compose_project',
        order: 20,
        state: 'passed',
        severity: 'success',
        blocking: false,
        title_key: 'platformUpdate.readiness.composeProject.title',
        summary_key: 'platformUpdate.readiness.composeProject.passed',
        detail_key: 'platformUpdate.readiness.composeProject.detail',
        evidence: [],
        actions: [],
      },
      {
        id: 'image_strategy',
        order: 30,
        state: 'passed',
        severity: 'success',
        blocking: false,
        title_key: 'platformUpdate.readiness.imageStrategy.title',
        summary_key: 'platformUpdate.readiness.imageStrategy.passed',
        detail_key: 'platformUpdate.readiness.imageStrategy.detail',
        evidence: [],
        actions: [],
      },
      {
        id: 'release_availability',
        order: 40,
        state: 'warning',
        severity: 'info',
        blocking: false,
        title_key: 'platformUpdate.readiness.releaseAvailability.title',
        summary_key: 'platformUpdate.readiness.releaseAvailability.available',
        detail_key: 'platformUpdate.readiness.releaseAvailability.detail',
        evidence: [],
        actions: [
          {
            id: 'view_release',
            type: 'navigate',
            label_key: 'platformUpdate.readiness.actions.viewRelease',
            target: 'https://example.test/releases/v0.9.8-beta.3/notes',
          },
        ],
      },
      {
        id: 'update_manage_permission',
        order: 50,
        state: 'passed',
        severity: 'success',
        blocking: false,
        title_key: 'platformUpdate.readiness.updateManagePermission.title',
        summary_key: 'platformUpdate.readiness.updateManagePermission.passed',
        detail_key: 'platformUpdate.readiness.updateManagePermission.detail',
        evidence: [],
        actions: [],
      },
    ],
  },
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
        target_version: payload.target_version,
        deployment_strategy: updateCenterPreviewStatus.deployment_strategy,
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
