import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { usePermissionStore } from '@/store';

import { createUpdateOperation, getUpdateFailureDiagnostic } from '../../api/update';
import { UPDATE_OPERATION_FAILURE_CODE } from '../../contract/failure-codes';
import { UPDATE_PERMISSION_CODE } from '../../contract/permissions';
import { useUpdateDiscoveryStore } from '../../store/discovery';
import type { UpdateCenterDataSource } from '../../types/preview';
import type { UpdateStatus } from '../../types/update';
import UpdateCenter from './index.vue';

const apiMocks = vi.hoisted(() => ({
  createUpdateOperation: vi.fn(),
  getUpdateFailureDiagnostic: vi.fn(),
  getUpdateOperations: vi.fn(),
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
      params?.requestId ? `${key}:${String(params.requestId)}` : key,
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
    update_mode: 'stable_tracking',
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
        't-card': passthrough,
        't-collapse': passthrough,
        't-collapse-panel': passthrough,
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
        't-table': passthrough,
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
    apiMocks.getUpdateFailureDiagnostic.mockResolvedValue(null);
    apiMocks.createUpdateOperation.mockResolvedValue({ operation_id: 'operation-1' });
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
      update_mode: 'beta_tracking',
    } as never);
    const wrapper = mountCenter();
    await flushPromises();

    expect(wrapper.text()).toContain('update.center.strategy.options.beta_tracking.title');
    expect(wrapper.find('[data-testid="update-center-configure-policy"]').exists()).toBe(false);
    expect(wrapper.text()).toContain('update.center.readiness.strategy');
    expect(createUpdateOperation).not.toHaveBeenCalled();

    await wrapper.get('[data-testid="update-center-upgrade"]').trigger('click');
    expect(wrapper.find('select').exists()).toBe(false);
  });

  it('reports no eligible release without claiming that the installation is invalid', async () => {
    useUpdateDiscoveryStore().replaceSnapshot({
      ...status([]),
      image_tag: 'beta',
      update_mode: 'beta_tracking',
      latest: undefined,
    } as UpdateStatus);
    const wrapper = mountCenter();
    await flushPromises();

    expect(wrapper.text()).toContain('update.center.latest.upToDate');
    expect(wrapper.text()).not.toContain('update.center.release.executionUnavailable');
  });

  it('opens a fixed-tag flow with only newer releases from the matching channel', async () => {
    useUpdateDiscoveryStore().replaceSnapshot({
      ...status([{ key: 'high', host_path: '/srv/graft', compose_files: [], confidence: 'high' }]),
      image_tag: 'v1.0.0',
      update_mode: 'pinned_stable',
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
    expect(wrapper.text()).toContain('update.center.confirmation.diagnosticTitle');
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
      createOperation: vi.fn().mockResolvedValue({ operation_id: 'preview-operation' }),
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
          status: 'FAILED',
          failure_diagnostic_available: true,
        },
      ]),
      getFailureDiagnostic: vi.fn().mockResolvedValue(null),
      createOperation: vi.fn(),
    };
    const wrapper = mountCenter(dataSource);
    await flushPromises();

    expect(wrapper.text()).not.toContain('update.center.history.viewCause');
  });
});
