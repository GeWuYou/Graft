import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { usePermissionStore } from '@/store';

import { createUpdateOperation } from '../../api/update';
import { UPDATE_OPERATION_FAILURE_CODE } from '../../contract/failure-codes';
import { UPDATE_PERMISSION_CODE } from '../../contract/permissions';
import { useUpdateDiscoveryStore } from '../../store/discovery';
import type { UpdateCenterDataSource } from '../../types/preview';
import UpdateCenter from './index.vue';

const apiMocks = vi.hoisted(() => ({ createUpdateOperation: vi.fn(), getUpdateOperations: vi.fn() }));

const updateStartFailure = (code: string, traceId = 'request-update-42') =>
  Object.assign(new Error('internal implementation detail'), {
    code,
    isApiRequestError: true as const,
    status: 500,
    traceId,
  });

vi.mock('../../api/update', () => apiMocks);
vi.mock('vue-router', () => ({ useRoute: () => ({ query: {} }) }));
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
      compose_candidates: candidates,
    },
    cache_stale: false,
    check_error: '',
  }) as never;

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

  it('uses the unique high-confidence candidate when submitting the dialog confirmation', async () => {
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
      compose_candidate_key: 'high',
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

  it('uses the generic failure text and hides the request ID for an unknown or network error', async () => {
    useUpdateDiscoveryStore().replaceSnapshot(
      status([{ key: 'high', host_path: '/srv/graft', compose_files: ['/srv/graft/compose.yml'], confidence: 'high' }]),
    );
    apiMocks.createUpdateOperation.mockRejectedValueOnce(new Error('network unavailable'));
    const wrapper = mountCenter();
    await flushPromises();

    await wrapper.get('[data-testid="update-center-upgrade"]').trigger('click');
    await wrapper.get('[data-testid="update-confirmation-submit"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('update.center.confirmation.failure.generic');
    expect(wrapper.find('[data-testid="update-operation-request-id"]').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('network unavailable');
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
      compose_candidate_key: 'preview',
    });
    expect(dataSource.getOperations).toHaveBeenCalledTimes(3);
  });
});
