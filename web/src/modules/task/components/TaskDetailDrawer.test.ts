import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import enUS from '../locales/en-US.json';
import zhCN from '../locales/zh-CN.json';
import { resolveTaskFailureMessage } from '../shared/failure-copy';
import TaskDetailDrawer from './TaskDetailDrawer.vue';

const apiMocks = vi.hoisted(() => ({
  cancelTask: vi.fn(),
  getTask: vi.fn(),
  getTaskLogs: vi.fn(),
  retryTaskStage: vi.fn(),
}));
const realtimeMocks = vi.hoisted(() => ({ openRealtimeTopicSocket: vi.fn() }));

vi.mock('../api/task', () => apiMocks);
vi.mock('@/shared/realtime', () => realtimeMocks);

const ResourceDetailLayoutStub = defineComponent({
  name: 'ResourceDetailLayout',
  props: {
    backLabel: { type: String, required: true },
    presentation: { type: String, required: true },
    size: { type: String, required: true },
    title: { type: String, required: true },
    visible: { type: Boolean, required: true },
  },
  emits: ['update:visible'],
  setup(_props, { slots }) {
    return () => h('section', { 'data-testid': 'task-detail-surface' }, slots.default?.());
  },
});

const alertStub = defineComponent({
  name: 'TAlert',
  props: { title: { type: String, default: '' }, message: { type: String, default: '' } },
  template: '<div data-testid="failure-alert">{{ title }}: {{ message }}</div>',
});

const passthrough = (name: string) => defineComponent({ name, template: '<div><slot /></div>' });

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages: { 'zh-CN': zhCN, 'en-US': enUS },
});

const globalStubs = {
  plugins: [i18n],
  stubs: {
    ResourceDetailLayout: ResourceDetailLayoutStub,
    't-loading': { template: '<div><slot /></div>' },
    't-alert': alertStub,
    't-tag': passthrough('TTag'),
    't-descriptions': passthrough('TDescriptions'),
    't-descriptions-item': passthrough('TDescriptionsItem'),
    't-steps': passthrough('TSteps'),
    't-space': passthrough('TSpace'),
    't-button': passthrough('TButton'),
    'log-viewer': passthrough('LogViewer'),
    't-empty': passthrough('TEmpty'),
  },
};

describe('TaskDetailDrawer', () => {
  it('maps Runtime Agent failure codes to stable product copy without exposing provider text', () => {
    const translate = (key: string) => key;
    expect(resolveTaskFailureMessage('docker_runtime_unavailable', 'raw sdk error', translate)).toBe(
      'task.detail.failureCodes.runtimeUnavailable',
    );
    expect(resolveTaskFailureMessage('unrecognized_provider_code', 'raw sdk error', translate)).toBe(
      'task.detail.failureCodes.unknown',
    );
  });

  it('uses the shared responsive detail surface and forwards close requests', async () => {
    const wrapper = mount(TaskDetailDrawer, {
      props: { taskId: null, visible: true },
      global: {
        plugins: [i18n],
        stubs: {
          ResourceDetailLayout: ResourceDetailLayoutStub,
          't-loading': { template: '<div><slot /></div>' },
        },
      },
    });

    const surface = wrapper.getComponent(ResourceDetailLayoutStub);
    expect(surface.props()).toMatchObject({
      backLabel: '返回任务列表',
      presentation: 'overlay',
      size: 'large',
      title: '任务详情',
      visible: true,
    });

    surface.vm.$emit('update:visible', false);
    expect(wrapper.emitted('update:visible')).toEqual([[false]]);
    wrapper.unmount();
  });

  it('renders the sanitized diagnostic summary next to the stable failure code label', async () => {
    const summary = 'yaml: line 3: could not find expected';
    apiMocks.getTask.mockResolvedValue({
      id: 9,
      type: 'project.compose.redeploy',
      owner_type: 'application',
      owner_id: 'app-1',
      status: 'failed',
      current_stage_key: 'apply',
      created_at: '2026-11-20T00:00:00Z',
      duration_ms: 1500,
      failure_code: 'provider_operation_failed',
      failure_message: summary,
      failure_stage: 'compose_load',
      failure_class: 'infrastructure',
      capabilities: { cancel: false, retry: false, downloadLog: false },
      stages: [
        {
          id: 3,
          key: 'apply',
          sequence: 1,
          executor_type: 'project.compose.apply',
          execution_mode: 'agent',
          execution_version: 'docker/v1',
          status: 'failed',
          attempt: 1,
          max_attempts: 1,
          recovery_policy: 'manual_reconcile',
          failure_code: 'provider_operation_failed',
          failure_message: summary,
          failure_stage: 'compose_load',
          failure_class: 'infrastructure',
        },
      ],
    });
    apiMocks.getTaskLogs.mockResolvedValue({ items: [], total: 0, limit: 200 });
    realtimeMocks.openRealtimeTopicSocket.mockReturnValue({ close: vi.fn() });

    const wrapper = mount(TaskDetailDrawer, {
      props: { taskId: 9, visible: true },
      global: globalStubs,
    });
    await flushPromises();

    const alert = wrapper.get('[data-testid="failure-alert"]');
    const alertText = alert.text();
    const label = i18n.global.t('task.detail.failureCodes.providerOperationFailed');
    // 脱敏摘要必须真实渲染，与稳定失败码文案、阶段与分类归因并列为独立段落。
    expect(alertText).toContain(summary);
    expect(alertText).toContain(label);
    expect(alertText).toContain(i18n.global.t('task.detail.failureStages.compose_load'));
    expect(alertText).toContain(i18n.global.t('task.detail.failureClasses.infrastructure'));
    const segments = alertText.split('失败详情: ')[1].split(' · ');
    expect(segments).toContain(summary);
    expect(segments.length).toBe(4);
    expect(wrapper.text()).toContain(summary);
    wrapper.unmount();
  });

  it('withholds the raw failure message when the row carries no stage or class provenance', async () => {
    const ordinaryError = '/srv/graft/workspace/current working directory does not exist';
    apiMocks.getTask.mockResolvedValue({
      id: 10,
      type: 'project.cleanup',
      owner_type: 'application',
      owner_id: 'app-1',
      status: 'failed',
      current_stage_key: 'cleanup',
      created_at: '2026-11-20T00:00:00Z',
      duration_ms: 800,
      failure_code: 'unrecognized_provider_code',
      failure_message: ordinaryError,
      capabilities: { cancel: false, retry: false, downloadLog: false },
      stages: [
        {
          id: 4,
          key: 'cleanup',
          sequence: 1,
          executor_type: 'project.cleanup',
          execution_mode: 'server',
          execution_version: '',
          status: 'failed',
          attempt: 1,
          max_attempts: 1,
          recovery_policy: 'manual_reconcile',
          failure_code: 'unrecognized_provider_code',
          failure_message: ordinaryError,
        },
      ],
    });
    apiMocks.getTaskLogs.mockResolvedValue({ items: [], total: 0, limit: 200 });
    realtimeMocks.openRealtimeTopicSocket.mockReturnValue({ close: vi.fn() });

    const wrapper = mount(TaskDetailDrawer, {
      props: { taskId: 10, visible: true },
      global: globalStubs,
    });
    await flushPromises();

    const alert = wrapper.get('[data-testid="failure-alert"]');
    const alertText = alert.text();
    // 无 stage/class provenance 的行只渲染稳定码文案，failure_message 原文不得出现。
    expect(alertText).toContain(i18n.global.t('task.detail.failureCodes.unknown'));
    expect(alertText).not.toContain(ordinaryError);
    expect(wrapper.text()).not.toContain(ordinaryError);
    wrapper.unmount();
  });
});
