import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';

import RuntimeTargetDetailPage from './index.vue';

const apiMocks = vi.hoisted(() => ({
  getRuntimeTarget: vi.fn(),
  getRuntimeTargetAssignments: vi.fn(),
  refreshRuntimeTarget: vi.fn(),
  replaceRuntimeTargetAssignments: vi.fn(),
}));
const routeState = vi.hoisted(() => ({ params: { id: '7' } }));
const permissionState = vi.hoisted(() => ({ manage: false }));
const messageMocks = vi.hoisted(() => ({ success: vi.fn() }));

vi.mock('../../api/runtime-target', () => apiMocks);
vi.mock('vue-router', () => ({ useRoute: () => routeState }));
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key }),
}));
vi.mock('@/locales/useLocale', () => ({ useLocale: () => ({ locale: 'en-US' }) }));
vi.mock('@/shared/observability', () => ({
  formatBytes: (value: number) => `${value} B`,
  formatLocaleDateTime: (value: string, locale: string) => `${locale}:${value}`,
}));
vi.mock('@/store/modules/permission', () => ({
  getPermissionStore: () => ({ hasPermission: () => permissionState.manage }),
}));
vi.mock('@/utils/request', () => ({
  isApiRequestError: (error: unknown) => Boolean(error && typeof error === 'object' && 'status' in error),
}));
vi.mock('tdesign-vue-next/es/message', () => ({ MessagePlugin: messageMocks }));

const passthrough = (name: string) => defineComponent({ name, template: '<div><slot /><slot name="actions" /></div>' });
const statisticStub = defineComponent({
  name: 'TStatistic',
  props: {
    title: { type: String, default: '' },
    value: { type: [String, Number], default: '' },
    extra: { type: String, default: '' },
  },
  template: '<div>{{ title }} {{ value }} {{ extra }}</div>',
});
const buttonStub = defineComponent({
  name: 'TButton',
  emits: ['click'],
  template: '<button @click="$emit(\'click\')"><slot /></button>',
});
const alertStub = defineComponent({
  name: 'TAlert',
  props: { message: { type: String, default: '' } },
  template: '<div>{{ message }}<slot /></div>',
});
const tableStub = defineComponent({
  name: 'TTable',
  props: { data: { type: Array, default: () => [] } },
  template: `<div data-testid="assignment-table">
    <div v-for="row in data" :key="row.user_id" :data-testid="'assignment-row-' + row.user_id">
      <slot name="user" :row="row" />
      <slot name="username" :row="row" />
      <slot name="accountStatus" :row="row" />
      <slot name="authorizationStatus" :row="row" />
      <slot name="authorizedAt" :row="row" />
      <slot name="operation" :row="row" />
    </div>
    <slot v-if="!data.length" name="empty" />
  </div>`,
});

function targetDetail() {
  return {
    id: 7,
    agent: {
      agent_id: 'agent-local',
      generation: 1,
      version: '1.0.0',
      status: 'ready',
      diagnostic_code: 'none',
      capabilities: [
        { name: 'compose_execution', status: 'ready', version: 'v1', diagnostic_code: 'none' },
        { name: 'container_execution', status: 'ready', version: 'v1', diagnostic_code: 'none' },
      ],
    },
    displayName: 'Local Docker',
    runtime: { provider: 'docker', type: 'container_runtime', version: '27.0', apiVersion: '1.46' },
    connection: { endpoint: 'unix:///var/run/docker.sock', kind: 'unix_socket' },
    health: { status: 'healthy', lastCheckedAt: '2026-07-13T00:00:00Z', diagnostic: '' },
    resources: {
      workloads: { available: true, total: 6, active: 5, unavailableReason: '' },
      cpu: { available: true, usagePercent: 10, usedBytes: 0, totalBytes: 0, unavailableReason: '' },
      memory: { available: true, usagePercent: 20, usedBytes: 20, totalBytes: 100, unavailableReason: '' },
      storage: { available: true, usagePercent: 30, usedBytes: 30, totalBytes: 100, unavailableReason: '' },
    },
    providerDetails: {
      provider: 'docker',
      docker: {
        images: { available: true, total: 11, unavailableReason: '' },
        volumes: { available: true, total: 12, unavailableReason: '' },
        networks: { available: true, total: 13, unavailableReason: '' },
      },
    },
  };
}

function baseStubs() {
  return {
    'management-page-content': passthrough('ManagementPageContent'),
    'management-page-header': passthrough('ManagementPageHeader'),
    't-alert': alertStub,
    't-card': passthrough('TCard'),
    't-descriptions': passthrough('TDescriptions'),
    't-descriptions-item': passthrough('TDescriptionsItem'),
    't-tag': passthrough('TTag'),
    't-statistic': statisticStub,
    't-empty': passthrough('TEmpty'),
    't-button': buttonStub,
  };
}

function assignmentStubs() {
  return {
    't-table': tableStub,
    't-pagination': passthrough('TPagination'),
    'runtime-target-assignment-dialog': passthrough('RuntimeTargetAssignmentDialog'),
  };
}

describe('RuntimeTargetDetailPage', () => {
  beforeEach(() => {
    permissionState.manage = false;
    routeState.params.id = '7';
    apiMocks.getRuntimeTargetAssignments.mockResolvedValue({ items: [], revision: 1 });
    apiMocks.replaceRuntimeTargetAssignments.mockReset();
    messageMocks.success.mockReset();
  });
  it('renders Docker provider details separately from neutral runtime resources', async () => {
    apiMocks.getRuntimeTarget.mockResolvedValue({
      id: 7,
      agent: targetDetail().agent,
      displayName: 'Local Docker',
      runtime: { provider: 'docker', type: 'container_runtime', version: '27.0', apiVersion: '1.46' },
      connection: { endpoint: 'unix:///var/run/docker.sock', kind: 'unix_socket' },
      health: { status: 'healthy', lastCheckedAt: '2026-07-13T00:00:00Z', diagnostic: '' },
      resources: {
        workloads: { available: true, total: 6, active: 5, unavailableReason: '' },
        cpu: { available: true, usagePercent: 10, usedBytes: 0, totalBytes: 0, unavailableReason: '' },
        memory: { available: true, usagePercent: 20, usedBytes: 20, totalBytes: 100, unavailableReason: '' },
        storage: { available: true, usagePercent: 30, usedBytes: 30, totalBytes: 100, unavailableReason: '' },
      },
      providerDetails: {
        provider: 'docker',
        docker: {
          images: { available: true, total: 11, unavailableReason: '' },
          volumes: { available: true, total: 12, unavailableReason: '' },
          networks: { available: false, total: 0, unavailableReason: 'network metrics unavailable' },
        },
      },
    });

    const wrapper = mount(RuntimeTargetDetailPage, {
      global: {
        stubs: baseStubs(),
      },
    });
    await flushPromises();

    expect(apiMocks.getRuntimeTarget).toHaveBeenCalledWith(7);
    expect(wrapper.text()).toContain('11');
    expect(wrapper.text()).toContain('12');
    expect(wrapper.text()).toContain('network metrics unavailable');
    expect(wrapper.text()).toContain('en-US:2026-07-13T00:00:00Z');
  });

  it('shows unavailable reasons instead of zero for unavailable metrics', async () => {
    apiMocks.getRuntimeTarget.mockResolvedValue({
      id: 7,
      agent: targetDetail().agent,
      displayName: 'Local Docker',
      runtime: { provider: 'docker', type: 'container_runtime', version: '27.0', apiVersion: '1.46' },
      connection: { endpoint: 'unix:///var/run/docker.sock', kind: 'unix_socket' },
      health: { status: 'healthy', lastCheckedAt: null, diagnostic: '' },
      resources: {
        workloads: { available: false, total: 0, active: 0, unavailableReason: 'workloads unavailable' },
        cpu: { available: false, usagePercent: 0, usedBytes: 0, totalBytes: 0, unavailableReason: 'cpu unavailable' },
        memory: { available: true, usagePercent: 20, usedBytes: 20, totalBytes: 100, unavailableReason: '' },
        storage: { available: true, usagePercent: 30, usedBytes: 30, totalBytes: 100, unavailableReason: '' },
      },
      providerDetails: {
        provider: 'docker',
        docker: {
          images: { available: true, total: 1, unavailableReason: '' },
          volumes: { available: true, total: 2, unavailableReason: '' },
          networks: { available: true, total: 3, unavailableReason: '' },
        },
      },
    });

    const wrapper = mount(RuntimeTargetDetailPage, {
      global: {
        stubs: baseStubs(),
      },
    });
    await flushPromises();

    expect(wrapper.text()).toContain('workloads unavailable');
    expect(wrapper.text()).toContain('cpu unavailable');
  });

  it('does not refresh when the route id is not a positive integer', async () => {
    routeState.params.id = 'not-an-id';
    const wrapper = mount(RuntimeTargetDetailPage, {
      global: {
        stubs: baseStubs(),
      },
    });
    await flushPromises();

    await wrapper.get('button').trigger('click');

    expect(apiMocks.refreshRuntimeTarget).not.toHaveBeenCalled();
    expect(wrapper.text()).toContain('runtimeTarget.detail.refreshError');
    routeState.params.id = '7';
  });

  it('keeps a revoked assignment on the current page and allows restoring it immediately', async () => {
    permissionState.manage = true;
    apiMocks.getRuntimeTarget.mockResolvedValue(targetDetail());
    const assignment = {
      target_id: 7,
      user_id: 8,
      username: 'alice',
      display: 'Alice',
      status: 'enabled',
      created_at: '2026-08-19T00:00:00Z',
      created_by: 1,
    };
    apiMocks.getRuntimeTargetAssignments.mockResolvedValue({ items: [assignment], revision: 3 });
    apiMocks.replaceRuntimeTargetAssignments
      .mockResolvedValueOnce({ items: [], revision: 4 })
      .mockResolvedValueOnce({ items: [assignment], revision: 5 });

    const wrapper = mount(RuntimeTargetDetailPage, {
      global: {
        stubs: { ...baseStubs(), ...assignmentStubs() },
      },
    });
    await flushPromises();

    expect(wrapper.get('[data-testid="assignment-row-8"]').text()).toContain('Alice');
    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'runtimeTarget.detail.revokeAuthorization')
      ?.trigger('click');
    await flushPromises();

    expect(apiMocks.replaceRuntimeTargetAssignments).toHaveBeenNthCalledWith(1, 7, [], 3);
    expect(wrapper.get('[data-testid="assignment-row-8"]').text()).toContain(
      'runtimeTarget.detail.authorizationRevoked',
    );

    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'runtimeTarget.detail.restoreAuthorization')
      ?.trigger('click');
    await flushPromises();

    expect(apiMocks.replaceRuntimeTargetAssignments).toHaveBeenNthCalledWith(2, 7, [8], 4);
    expect(wrapper.get('[data-testid="assignment-row-8"]').text()).toContain(
      'runtimeTarget.detail.authorizationActive',
    );
  });

  it('keeps an assignment visible when its user account summary is unavailable', async () => {
    permissionState.manage = true;
    apiMocks.getRuntimeTarget.mockResolvedValue(targetDetail());
    apiMocks.getRuntimeTargetAssignments.mockResolvedValue({
      items: [{ target_id: 7, user_id: 99, created_at: '2026-08-19T00:00:00Z', created_by: 1 }],
      revision: 2,
    });

    const wrapper = mount(RuntimeTargetDetailPage, {
      global: {
        stubs: { ...baseStubs(), ...assignmentStubs() },
      },
    });
    await flushPromises();

    const row = wrapper.get('[data-testid="assignment-row-99"]');
    expect(row.text()).toContain('runtimeTarget.detail.missingUser');
    expect(row.text()).toContain('runtimeTarget.detail.accountUnavailable');
    expect(row.text()).toContain('runtimeTarget.detail.revokeAuthorization');
  });

  it('does not show a revoked state when the immediate request fails', async () => {
    permissionState.manage = true;
    apiMocks.getRuntimeTarget.mockResolvedValue(targetDetail());
    const assignment = {
      target_id: 7,
      user_id: 8,
      username: 'alice',
      display: 'Alice',
      status: 'enabled',
      created_at: '2026-08-19T00:00:00Z',
      created_by: 1,
    };
    apiMocks.getRuntimeTargetAssignments.mockResolvedValue({ items: [assignment], revision: 3 });
    apiMocks.replaceRuntimeTargetAssignments.mockRejectedValue(new Error('network failed'));

    const wrapper = mount(RuntimeTargetDetailPage, {
      global: {
        stubs: { ...baseStubs(), ...assignmentStubs() },
      },
    });
    await flushPromises();

    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'runtimeTarget.detail.revokeAuthorization')
      ?.trigger('click');
    await flushPromises();

    const row = wrapper.get('[data-testid="assignment-row-8"]');
    expect(row.text()).toContain('runtimeTarget.detail.authorizationActive');
    expect(wrapper.text()).toContain('runtimeTarget.detail.authorizationChangeError');
  });
});
