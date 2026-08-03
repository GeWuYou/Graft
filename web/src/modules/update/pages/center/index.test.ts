import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { usePermissionStore } from '@/store';

import { createUpdateOperation, getUpdateFailureDiagnostic, getUpdateOperationDiagnostic } from '../../api/update';
import DiagnosticDrawer from '../../components/DiagnosticDrawer.vue';
import { UPDATE_OPERATION_FAILURE_CODE } from '../../contract/failure-codes';
import { UPDATE_PERMISSION_CODE } from '../../contract/permissions';
import { useUpdateDiscoveryStore } from '../../store/discovery';
import type { UpdateCenterDataSource } from '../../types/preview';
import type { UpdateReadinessAction, UpdateStatus } from '../../types/update';
import UpdateCenter from './index.vue';

const apiMocks = vi.hoisted(() => ({
  createUpdateOperation: vi.fn(),
  getUpdateOperation: vi.fn(),
  getUpdateFailureDiagnostic: vi.fn(),
  getUpdateOperationDiagnostic: vi.fn(),
  getUpdateOperations: vi.fn(),
  subscribeToUpdateOperation: vi.fn(() => ({ close: vi.fn(), reconnect: vi.fn() })),
}));

const updateStartFailure = (code: string, traceId = 'request-update-42') =>
  Object.assign(new Error('internal implementation detail'), {
    code,
    isApiRequestError: true as const,
    status: 500,
    traceId,
  });

vi.mock('../../api/update', () => apiMocks);
vi.mock('vue-router', () => ({ useRoute: () => ({ query: {} }), useRouter: () => ({ push: vi.fn() }) }));
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({
    locale: { value: 'en-US' },
    t: (key: string, params?: Record<string, unknown>) =>
      params?.requestId
        ? `${key}:${String(params.requestId)}`
        : key === 'update.center.history.messages.update_completed'
          ? 'Update completed'
          : key,
    te: (key: string) => key === 'update.center.history.messages.update_completed',
  }),
}));
vi.mock('@/shared/observability', () => ({ formatLocaleDateTime: (value: string) => value }));
vi.mock('@/shared/components/markdown', () => ({
  MarkdownViewer: defineComponent({
    props: { source: { type: String, default: '' } },
    setup: (props) => () => h('article', props.source),
  }),
}));

const passthrough = defineComponent({ template: '<section><slot /></section>' });
const cardStub = defineComponent({
  props: { title: { type: String, default: '' } },
  template: '<section><header>{{ title }}</header><slot /></section>',
});
const alertStub = defineComponent({
  props: { message: { type: String, default: '' } },
  template: '<section><slot />{{ message }}</section>',
});
const dialogStub = defineComponent({
  props: { visible: Boolean, confirmBtn: { type: Object, default: () => ({}) } },
  emits: ['confirm', 'update:visible'],
  template:
    '<section v-if="visible" data-testid="update-confirmation-dialog"><slot /><button data-testid="update-confirmation-submit" :disabled="confirmBtn.disabled" @click="$emit(\'confirm\')">confirm</button></section>',
});
const radioGroupStub = defineComponent({
  props: { modelValue: { type: String, default: '' } },
  emits: ['update:modelValue'],
  template: '<section data-testid="candidate-options"><slot /></section>',
});
const radioStub = defineComponent({
  props: { value: { type: String, default: '' } },
  template: '<button type="button" :data-candidate="value"><slot /></button>',
});
const status = (candidates: Array<Record<string, unknown>>) =>
  ({
    current_version: '1.0.0',
    channel: 'stable',
    image_tag: 'latest',
    deployment_strategy: 'stable_tracking',
    available_releases: [{ version: '1.1.0', channel: 'stable', published_at: '2026-07-24T00:00:00Z' }],
    latest: {
      version: '1.1.0',
      channel: 'stable',
      notes: 'Release notes only',
      published_at: '2026-07-24T00:00:00Z',
      manifest_url: 'https://example.test/manifest',
      server_digest: 'server',
      web_digest: 'web',
    },
    installation_profile: {
      declared_mode: 'compose',
      detected_mode: 'compose',
      capability: 'compose_upgrade_available',
      guidance: '',
      compose_root_source: 'docker_discovered',
      compose_root_confirmation_required: candidates.filter(({ confidence }) => confidence === 'high').length !== 1,
      compose_candidates: candidates,
    },
    cache_stale: false,
    check_error: '',
  }) as UpdateStatus;

function mountCenter(dataSource?: UpdateCenterDataSource) {
  return mount(UpdateCenter, {
    props: { dataSource },
    global: {
      stubs: {
        't-alert': alertStub,
        't-button': defineComponent({
          props: { disabled: Boolean },
          emits: ['click'],
          template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
        }),
        't-card': cardStub,
        't-collapse': passthrough,
        't-collapse-panel': defineComponent({ template: '<section><slot name="header" /><slot /></section>' }),
        't-dialog': dialogStub,
        't-link': passthrough,
        't-loading': passthrough,
        't-radio': radioStub,
        't-radio-group': radioGroupStub,
        't-select': defineComponent({
          props: { modelValue: { type: String, default: '' } },
          emits: ['update:modelValue'],
          template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)" />',
        }),
        't-table': defineComponent({
          props: { data: { type: Array, default: () => [] } },
          template: '<section><slot name="message" :row="data[0]" /></section>',
        }),
        't-tag': passthrough,
        ManagementEmptyState: passthrough,
      },
    },
  });
}

describe('UpdateCenter', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    usePermissionStore().setBootstrapSnapshot({
      permissions: [UPDATE_PERMISSION_CODE.READ, UPDATE_PERMISSION_CODE.MANAGE],
    } as never);
    useUpdateDiscoveryStore().replaceSnapshot(status([]));
    apiMocks.getUpdateOperations.mockResolvedValue([]);
    apiMocks.getUpdateOperation.mockResolvedValue({
      operation_id: 'operation-1',
      runner_id: 'runner-1',
      phase: 'READY',
      progress: 0,
      message: '',
    });
    apiMocks.getUpdateFailureDiagnostic.mockResolvedValue(null);
    apiMocks.getUpdateOperationDiagnostic.mockResolvedValue(null);
    apiMocks.createUpdateOperation.mockResolvedValue({ operation_id: 'operation-1', runner_id: 'runner-1' });
    vi.clearAllMocks();
  });

  it('keeps Docker candidate paths out of release notes until upgrade confirmation', async () => {
    useUpdateDiscoveryStore().replaceSnapshot(
      status([{ key: 'high', host_path: '/srv/graft', compose_files: ['/srv/graft/compose.yml'], confidence: 'high' }]),
    );
    const wrapper = mountCenter();
    await flushPromises();

    expect(wrapper.text()).not.toContain('/srv/graft');

    await wrapper.get('[data-testid="update-center-upgrade"]').trigger('click');
    expect(wrapper.get('[data-testid="update-confirmation-dialog"]').text()).toContain('/srv/graft');
  });

  it('shows the unique high-confidence candidate without returning its opaque key', async () => {
    useUpdateDiscoveryStore().replaceSnapshot(
      status([
        { key: 'high', host_path: '/srv/graft', compose_files: ['/srv/graft/compose.yml'], confidence: 'high' },
        { key: 'mount', host_path: '/data', compose_files: [], confidence: 'medium' },
      ]),
    );
    const wrapper = mountCenter();
    await flushPromises();

    await wrapper.get('[data-testid="update-center-upgrade"]').trigger('click');
    await wrapper.get('[data-testid="update-confirmation-submit"]').trigger('click');
    await flushPromises();

    expect(createUpdateOperation).toHaveBeenCalledWith({
      target_version: '1.1.0',
    });
  });

  it('requires an explicit selection when multiple high-confidence candidates are present', async () => {
    useUpdateDiscoveryStore().replaceSnapshot(
      status([
        { key: 'first', host_path: '/srv/first', compose_files: ['/srv/first/compose.yml'], confidence: 'high' },
        { key: 'second', host_path: '/srv/second', compose_files: ['/srv/second/compose.yml'], confidence: 'high' },
      ]),
    );
    const wrapper = mountCenter();
    await flushPromises();

    await wrapper.get('[data-testid="update-center-upgrade"]').trigger('click');
    expect(wrapper.get('[data-testid="update-confirmation-submit"]').attributes('disabled')).toBeDefined();
    expect(wrapper.text()).toContain('update.center.composeRoot.selectionRequired');
  });

  it('derives the tracking strategy from the image tag without offering a browser-owned setup flow', async () => {
    useUpdateDiscoveryStore().replaceSnapshot({
      ...status([{ key: 'high', host_path: '/srv/graft', compose_files: [], confidence: 'high' }]),
      image_tag: 'beta',
      deployment_strategy: 'beta_tracking',
    } as never);
    const wrapper = mountCenter();
    await flushPromises();

    expect(wrapper.text()).toContain('update.center.strategy.options.beta_tracking.title');
    expect(wrapper.find('[data-testid="update-center-configure-policy"]').exists()).toBe(false);
    expect(wrapper.text()).toContain('update.center.readiness.title');
    expect(createUpdateOperation).not.toHaveBeenCalled();

    await wrapper.get('[data-testid="update-center-upgrade"]').trigger('click');
    expect(wrapper.find('select').exists()).toBe(false);
  });

  it('reports no eligible release without claiming that the installation is invalid', async () => {
    useUpdateDiscoveryStore().replaceSnapshot({
      ...status([]),
      image_tag: 'beta',
      deployment_strategy: 'beta_tracking',
      latest: undefined,
    } as UpdateStatus);
    const wrapper = mountCenter();
    await flushPromises();

    expect(wrapper.text()).toContain('update.center.overall.status_unknown.title');
    expect(wrapper.text()).not.toContain('update.center.release.executionUnavailable');
  });

  it('opens a fixed-tag flow with only newer releases from the matching channel', async () => {
    useUpdateDiscoveryStore().replaceSnapshot({
      ...status([{ key: 'high', host_path: '/srv/graft', compose_files: [], confidence: 'high' }]),
      image_tag: 'v1.0.0',
      deployment_strategy: 'pinned_stable',
      latest: undefined,
      available_releases: [
        {
          version: '0.9.9',
          channel: 'stable',
          notes: 'Older release',
          published_at: '2026-07-25T00:00:00Z',
          manifest_url: 'https://example.test/older-manifest',
          server_digest: 'server',
          web_digest: 'web',
        },
        {
          version: '1.3.0-beta.1',
          channel: 'beta',
          notes: 'Other channel release',
          published_at: '2026-07-25T00:00:00Z',
          manifest_url: 'https://example.test/beta-manifest',
          server_digest: 'server',
          web_digest: 'web',
        },
        {
          version: '1.1.0',
          channel: 'stable',
          notes: 'Earlier fixed release notes',
          published_at: '2026-07-25T00:00:00Z',
          manifest_url: 'https://example.test/earlier-fixed-manifest',
          server_digest: 'server',
          web_digest: 'web',
        },
        {
          version: '1.2.0',
          channel: 'stable',
          notes: 'Fixed release notes',
          published_at: '2026-07-25T00:00:00Z',
          manifest_url: 'https://example.test/fixed-manifest',
          server_digest: 'server',
          web_digest: 'web',
          server_image: 'ghcr.io/example/graft-server',
          web_image: 'ghcr.io/example/graft-web',
          server_reference: 'ghcr.io/example/graft-server@sha256:server',
          web_reference: 'ghcr.io/example/graft-web@sha256:web',
          runner_image: 'ghcr.io/example/graft-compose-runner',
          runner_digest: 'runner',
          runner_reference: 'ghcr.io/example/graft-compose-runner@sha256:runner',
        },
      ],
    } as UpdateStatus);
    const wrapper = mountCenter();
    await flushPromises();

    await wrapper.get('[data-testid="update-center-upgrade"]').trigger('click');
    expect(wrapper.find('[data-testid="update-confirmation-dialog"]').exists()).toBe(true);
    expect(wrapper.find('select').exists()).toBe(true);
    await wrapper.get('[data-testid="update-confirmation-submit"]').trigger('click');
    await flushPromises();

    expect(createUpdateOperation).toHaveBeenCalledWith({ target_version: '1.2.0' });
  });

  it('shows a pinned Beta candidate in the summary when tracking latest is absent', async () => {
    useUpdateDiscoveryStore().replaceSnapshot({
      ...status([{ key: 'high', host_path: '/srv/graft', compose_files: [], confidence: 'high' }]),
      current_version: '0.11.0-beta.27',
      channel: 'beta',
      image_tag: 'v0.11.0-beta.27',
      deployment_strategy: 'pinned_beta',
      latest: undefined,
      available_releases: [
        {
          version: '0.11.0-beta.28',
          channel: 'beta',
          notes: 'Pinned Beta release notes',
          published_at: '2026-07-31T00:00:00Z',
          manifest_url: 'https://example.test/beta-manifest',
          server_digest: 'server',
          web_digest: 'web',
        },
      ],
    } as UpdateStatus);
    const wrapper = mountCenter();
    await flushPromises();

    expect(wrapper.text()).toContain('0.11.0-beta.28');
    expect(wrapper.text()).not.toContain('update.center.latest.upToDate');
  });

  it('renders a safe failure reason and request ID for a rejected upgrade submission', async () => {
    useUpdateDiscoveryStore().replaceSnapshot(
      status([{ key: 'high', host_path: '/srv/graft', compose_files: ['/srv/graft/compose.yml'], confidence: 'high' }]),
    );
    apiMocks.createUpdateOperation.mockRejectedValueOnce(
      updateStartFailure(UPDATE_OPERATION_FAILURE_CODE.COMPOSE_PREFLIGHT_FAILED),
    );
    const wrapper = mountCenter();
    await flushPromises();

    await wrapper.get('[data-testid="update-center-upgrade"]').trigger('click');
    await wrapper.get('[data-testid="update-confirmation-submit"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('update.center.confirmation.failure.composePreflightFailed');
    expect(wrapper.get('[data-testid="update-operation-request-id"]').text()).toContain('request-update-42');
    expect(wrapper.text()).not.toContain('internal implementation detail');
  });

  it('maps invalid deployment image tags to their dedicated safe message', async () => {
    useUpdateDiscoveryStore().replaceSnapshot(
      status([{ key: 'high', host_path: '/srv/graft', compose_files: ['/srv/graft/compose.yml'], confidence: 'high' }]),
    );
    apiMocks.createUpdateOperation.mockRejectedValueOnce(
      updateStartFailure(UPDATE_OPERATION_FAILURE_CODE.IMAGE_TAG_INVALID),
    );
    const wrapper = mountCenter();
    await flushPromises();

    await wrapper.get('[data-testid="update-center-upgrade"]').trigger('click');
    await wrapper.get('[data-testid="update-confirmation-submit"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('update.center.confirmation.failure.imageTagInvalid');
  });

  it('loads and renders the protected sanitized diagnostic for a failed update start', async () => {
    useUpdateDiscoveryStore().replaceSnapshot(
      status([{ key: 'high', host_path: '/srv/graft', compose_files: ['/srv/graft/compose.yml'], confidence: 'high' }]),
    );
    apiMocks.createUpdateOperation.mockRejectedValueOnce(
      updateStartFailure(UPDATE_OPERATION_FAILURE_CODE.OPERATION_START_FAILED),
    );
    apiMocks.getUpdateFailureDiagnostic.mockResolvedValueOnce({
      request_id: 'request-update-42',
      target_version: '1.1.0',
      failure_code: UPDATE_OPERATION_FAILURE_CODE.OPERATION_START_FAILED,
      failure_stage: 'runner_launch',
      summary: 'platform update rollout start failed',
      detail: 'docker launch failed: [REDACTED]',
      occurred_at: '2026-07-27T10:00:00Z',
    });
    const wrapper = mountCenter();
    await flushPromises();

    await wrapper.get('[data-testid="update-center-upgrade"]').trigger('click');
    await wrapper.get('[data-testid="update-confirmation-submit"]').trigger('click');
    await flushPromises();

    expect(getUpdateFailureDiagnostic).toHaveBeenCalledWith('request-update-42');
    expect(wrapper.get('[data-testid="update-operation-diagnostic"]').text()).toContain(
      'docker launch failed: [REDACTED]',
    );
    expect(wrapper.get('[data-testid="update-operation-diagnostic"]').text()).toContain(
      'update.center.confirmation.diagnosticStages.unknown',
    );
    expect(wrapper.text()).toContain('update.center.confirmation.diagnosticTitle');
  });

  it('localizes known terminal backup diagnostic stages', async () => {
    useUpdateDiscoveryStore().replaceSnapshot(
      status([{ key: 'high', host_path: '/srv/graft', compose_files: ['/srv/graft/compose.yml'], confidence: 'high' }]),
    );
    apiMocks.createUpdateOperation.mockRejectedValueOnce(
      updateStartFailure(UPDATE_OPERATION_FAILURE_CODE.OPERATION_START_FAILED),
    );
    apiMocks.getUpdateFailureDiagnostic.mockResolvedValueOnce({
      request_id: 'request-update-42',
      target_version: '1.1.0',
      failure_code: UPDATE_OPERATION_FAILURE_CODE.RUNNER_TERMINAL_FAILED,
      failure_stage: 'env_snapshot',
      summary: 'platform update runner reported a terminal failure',
      detail: 'deployment environment snapshot was denied by deployment filesystem permissions',
      occurred_at: '2026-07-27T10:00:00Z',
    });
    const wrapper = mountCenter();
    await flushPromises();

    await wrapper.get('[data-testid="update-center-upgrade"]').trigger('click');
    await wrapper.get('[data-testid="update-confirmation-submit"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="update-operation-diagnostic"]').text()).toContain(
      'update.center.confirmation.diagnosticStages.backupConfigSnapshot',
    );
    expect(wrapper.get('[data-testid="update-operation-diagnostic"]').text()).not.toContain('env_snapshot');
  });

  it('uses the generic failure text and hides the request ID for an unknown or network error', async () => {
    useUpdateDiscoveryStore().replaceSnapshot(
      status([{ key: 'high', host_path: '/srv/graft', compose_files: ['/srv/graft/compose.yml'], confidence: 'high' }]),
    );
    apiMocks.createUpdateOperation.mockRejectedValueOnce(updateStartFailure('UNKNOWN_CODE'));
    const wrapper = mountCenter();
    await flushPromises();

    await wrapper.get('[data-testid="update-center-upgrade"]').trigger('click');
    await wrapper.get('[data-testid="update-confirmation-submit"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('update.center.confirmation.failure.generic');
    expect(wrapper.find('[data-testid="update-operation-request-id"]').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('internal implementation detail');
  });

  it('uses the injected data source for status, refresh, history, and upgrade submission', async () => {
    const dataSource: UpdateCenterDataSource = {
      permissions: { check: true, manage: true },
      getStatus: vi
        .fn()
        .mockResolvedValue(
          status([{ key: 'preview', host_path: '/srv/graft', compose_files: [], confidence: 'high' }]),
        ),
      checkForUpdates: vi
        .fn()
        .mockResolvedValue(
          status([{ key: 'preview', host_path: '/srv/graft', compose_files: [], confidence: 'high' }]),
        ),
      getOperations: vi.fn().mockResolvedValue([]),
      getFailureDiagnostic: vi.fn().mockResolvedValue(null),
      createOperation: vi.fn().mockResolvedValue({ operation_id: 'preview-operation', runner_id: 'preview-runner' }),
    };
    const wrapper = mountCenter(dataSource);
    await flushPromises();

    expect(dataSource.getStatus).toHaveBeenCalledOnce();
    expect(dataSource.getOperations).toHaveBeenCalledOnce();

    await wrapper.get('button').trigger('click');
    await flushPromises();
    expect(dataSource.checkForUpdates).toHaveBeenCalledOnce();

    await wrapper.get('[data-testid="update-center-upgrade"]').trigger('click');
    await wrapper.get('[data-testid="update-confirmation-submit"]').trigger('click');
    await flushPromises();

    expect(dataSource.createOperation).toHaveBeenCalledWith({
      target_version: '1.1.0',
    });
    expect(dataSource.getOperations).toHaveBeenCalledTimes(3);
  });

  it('uses the injected data source for diagnostics after an injected update submission fails', async () => {
    const diagnostic = {
      request_id: 'request-update-42',
      target_version: '1.1.0',
      failure_code: UPDATE_OPERATION_FAILURE_CODE.OPERATION_START_FAILED,
      failure_stage: 'runner_launch',
      summary: 'platform update rollout start failed',
      detail: 'preview diagnostic',
      occurred_at: '2026-07-27T10:00:00Z',
    };
    const dataSource: UpdateCenterDataSource = {
      permissions: { check: true, manage: true },
      getStatus: vi
        .fn()
        .mockResolvedValue(
          status([{ key: 'preview', host_path: '/srv/graft', compose_files: [], confidence: 'high' }]),
        ),
      checkForUpdates: vi.fn(),
      getOperations: vi.fn().mockResolvedValue([]),
      getFailureDiagnostic: vi.fn().mockResolvedValue(diagnostic),
      createOperation: vi
        .fn()
        .mockRejectedValue(updateStartFailure(UPDATE_OPERATION_FAILURE_CODE.OPERATION_START_FAILED)),
    };
    const wrapper = mountCenter(dataSource);
    await flushPromises();

    await wrapper.get('[data-testid="update-center-upgrade"]').trigger('click');
    await wrapper.get('[data-testid="update-confirmation-submit"]').trigger('click');
    await flushPromises();

    expect(dataSource.getFailureDiagnostic).toHaveBeenCalledWith('request-update-42');
    expect(getUpdateFailureDiagnostic).not.toHaveBeenCalled();
    expect(wrapper.get('[data-testid="update-operation-diagnostic"]').text()).toContain('preview diagnostic');
  });

  it('does not expose live operation tracking from an injected preview data source', async () => {
    const dataSource: UpdateCenterDataSource = {
      permissions: { check: true, manage: true },
      getStatus: vi.fn().mockResolvedValue(status([])),
      checkForUpdates: vi.fn(),
      getOperations: vi.fn().mockResolvedValue([
        {
          operation_id: 'preview-operation',
          runner_id: 'preview-runner',
          phase: 'FAILED',
          progress: 100,
          message: 'preview failure',
        },
      ]),
      getFailureDiagnostic: vi.fn().mockResolvedValue(null),
      createOperation: vi.fn(),
    };
    const wrapper = mountCenter(dataSource);
    await flushPromises();

    expect(wrapper.text()).not.toContain('update.center.history.viewCause');
  });

  it('loads a failed history cause directly without opening live progress tracking', async () => {
    apiMocks.getUpdateOperations.mockResolvedValueOnce([
      {
        operation_id: 'failed-operation',
        runner_id: 'failed-runner',
        phase: 'FAILED',
        progress: 100,
        message: 'update_failed',
      },
    ]);
    apiMocks.getUpdateOperationDiagnostic.mockResolvedValueOnce({
      operation_id: 'failed-operation',
      request_id: 'request-update-42',
      target_version: '1.1.0',
      failure_code: UPDATE_OPERATION_FAILURE_CODE.RUNNER_TERMINAL_FAILED,
      failure_stage: 'runner_launch',
      summary: 'runner failed',
      detail: 'sanitized history detail',
      occurred_at: '2026-07-27T10:00:00Z',
    });
    const wrapper = mountCenter();
    await flushPromises();

    const causeButton = wrapper.findAll('button').find((button) => button.text() === 'update.center.history.viewCause');
    expect(causeButton).toBeDefined();
    await causeButton!.trigger('click');
    await flushPromises();

    expect(getUpdateOperationDiagnostic).toHaveBeenCalledWith('failed-operation');
    expect(apiMocks.getUpdateOperation).not.toHaveBeenCalled();
    expect(apiMocks.subscribeToUpdateOperation).not.toHaveBeenCalled();
    expect(wrapper.get('[data-testid="history-operation-diagnostic"]').text()).toContain('sanitized history detail');
  });

  it('localizes known runner history messages instead of rendering their internal codes', async () => {
    const dataSource: UpdateCenterDataSource = {
      permissions: { check: true, manage: true },
      getStatus: vi.fn().mockResolvedValue(status([])),
      checkForUpdates: vi.fn(),
      getOperations: vi.fn().mockResolvedValue([
        {
          operation_id: 'completed-operation',
          runner_id: 'completed-runner',
          phase: 'SUCCESS',
          progress: 100,
          message: 'update_completed',
        },
      ]),
      getFailureDiagnostic: vi.fn().mockResolvedValue(null),
      createOperation: vi.fn(),
    };
    const wrapper = mountCenter(dataSource);
    await flushPromises();

    expect(wrapper.text()).toContain('Update completed');
    expect(wrapper.text()).not.toContain('update_completed');
  });

  it('keeps server readiness checks scannable and opens diagnostics on demand', async () => {
    const source = status([]);
    source.readiness = {
      overall: 'upgrade_blocked',
      ready_count: 1,
      total_count: 2,
      checks: [
        {
          id: 'later',
          order: 20,
          state: 'passed',
          severity: 'success',
          blocking: false,
          title_key: 'platformUpdate.readiness.imageStrategy.title',
          summary_key: 'platformUpdate.readiness.imageStrategy.passed',
          evidence: [],
          actions: [],
        },
        {
          id: 'first',
          order: 10,
          state: 'failed',
          severity: 'critical',
          blocking: true,
          title_key: 'platformUpdate.readiness.officialCompose.title',
          summary_key: 'platformUpdate.readiness.officialCompose.failed',
          detail_key: 'platformUpdate.readiness.officialCompose.detail',
          evidence: [],
          actions: [],
        },
      ],
    };
    const dataSource: UpdateCenterDataSource = {
      permissions: { check: true, manage: true },
      getStatus: vi.fn().mockResolvedValue(source),
      checkForUpdates: vi.fn(),
      getOperations: vi.fn().mockResolvedValue([]),
      getFailureDiagnostic: vi.fn(),
      createOperation: vi.fn(),
    };
    const wrapper = mountCenter(dataSource);
    await flushPromises();

    const content = wrapper.text();
    expect(content.indexOf('platformUpdate.readiness.officialCompose.title')).toBeLessThan(
      content.indexOf('platformUpdate.readiness.imageStrategy.title'),
    );
    expect(content).toContain('update.center.current.title');
    expect(content).toContain('update.center.strategy.title');
    expect(content).toContain('update.center.latest.title');
    expect(content).toContain('update.center.overall.upgrade_blocked.title');
    expect(wrapper.get('[data-testid="update-readiness-first"]').text()).not.toContain(
      'platformUpdate.readiness.officialCompose.detail',
    );

    await wrapper.get('[data-testid="update-readiness-detail-first"]').trigger('click');

    const diagnosticDrawer = wrapper.getComponent(DiagnosticDrawer);
    expect(diagnosticDrawer.props('visible')).toBe(true);
    expect(diagnosticDrawer.props('check')).toMatchObject({ id: 'first' });
  });

  it.each([
    {
      name: 'recheck action type',
      action: { id: 'start_upgrade', type: 'recheck' },
    },
    {
      name: 'check_updates action id',
      action: { id: 'check_updates', type: 'command', target: 'ignored' },
    },
  ] satisfies Array<{ name: string; action: Omit<UpdateReadinessAction, 'label_key'> }>)(
    'closes a diagnostic after a successful $name so it cannot show stale readiness',
    async ({ action }) => {
      const source = status([]);
      source.readiness = {
        overall: 'upgrade_blocked',
        ready_count: 0,
        total_count: 1,
        next_action: { ...action, label_key: 'update.center.check' },
        checks: [
          {
            id: 'compose',
            order: 10,
            state: 'failed',
            severity: 'critical',
            blocking: true,
            title_key: 'platformUpdate.readiness.officialCompose.title',
            summary_key: 'platformUpdate.readiness.officialCompose.failed',
            evidence: [],
            actions: [{ ...action, label_key: 'update.center.check' }],
          },
        ],
      };
      const refreshedSource = {
        ...source,
        readiness: {
          ...source.readiness,
          overall: 'upgrade_ready' as const,
          ready_count: 1,
          checks: [
            {
              ...source.readiness.checks[0],
              state: 'passed' as const,
              severity: 'success' as const,
              blocking: false,
            },
          ],
        },
      };
      const dataSource: UpdateCenterDataSource = {
        permissions: { check: true, manage: true },
        getStatus: vi.fn().mockResolvedValue(source),
        checkForUpdates: vi.fn().mockResolvedValue(refreshedSource),
        getOperations: vi.fn().mockResolvedValue([]),
        getFailureDiagnostic: vi.fn(),
        createOperation: vi.fn(),
      };
      const wrapper = mountCenter(dataSource);
      await flushPromises();

      expect(wrapper.findAll('button').filter((button) => button.text() === 'update.center.check')).toHaveLength(0);

      await wrapper.get('[data-testid="update-readiness-detail-compose"]').trigger('click');
      const diagnosticDrawer = wrapper.getComponent(DiagnosticDrawer);
      diagnosticDrawer.vm.$emit('action', source.readiness.checks[0].actions[0]);
      await flushPromises();

      expect(dataSource.checkForUpdates).toHaveBeenCalledOnce();
      expect(diagnosticDrawer.props('visible')).toBe(false);
      expect(diagnosticDrawer.props('check')).toBeNull();
    },
  );
});
