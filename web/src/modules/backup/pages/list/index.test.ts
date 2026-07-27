import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import BackupListPage from './index.vue';

const apiMocks = vi.hoisted(() => ({
  getBackup: vi.fn(),
  listBackups: vi.fn(),
  submitBackup: vi.fn(),
}));

vi.mock('../../api/backup', () => apiMocks);
vi.mock('@/modules/task/task-observer', () => ({
  isTerminalTaskStatus: () => false,
  observeTask: () => ({ stop: vi.fn() }),
}));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
vi.mock('@/shared/observability', () => ({
  copyText: vi.fn(),
  formatBytes: (value: number) => `${value} B`,
  formatLocaleDateTime: (value: string) => value,
}));
vi.mock('vue-i18n', async () => ({
  ...(await vi.importActual<typeof import('vue-i18n')>('vue-i18n')),
  useI18n: () => ({
    locale: { value: 'zh-CN' },
    t: (key: string, values?: Record<string, unknown>) => `${key}${values ? JSON.stringify(values) : ''}`,
  }),
}));

const passthrough = (name: string) =>
  defineComponent({
    name,
    setup(_props, { slots }) {
      return () => h('div', { 'data-stub': name }, [slots.head?.(), slots.actions?.(), slots.default?.()]);
    },
  });

const TaskDetailDrawerStub = defineComponent({
  name: 'TaskDetailDrawer',
  props: {
    taskId: { type: Number, default: null },
    visible: { type: Boolean, default: false },
  },
  setup(props) {
    return () => h('div', { 'data-testid': 'task-detail-drawer', 'data-task-id': String(props.taskId ?? '') });
  },
});

const TButtonStub = defineComponent({
  name: 'TButton',
  emits: ['click'],
  setup(_props, { emit, slots }) {
    return () => h('button', { onClick: () => emit('click') }, [slots.icon?.(), slots.default?.()]);
  },
});

const TTableStub = defineComponent({
  name: 'TTable',
  props: { data: { type: Array, default: () => [] } },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        { 'data-testid': 'backup-table' },
        props.data.map((row) =>
          h('div', { 'data-row-id': String((row as { id: number }).id) }, [
            slots.contents?.({ row }),
            slots.actions?.({ row }),
          ]),
        ),
      );
  },
});

function mountPage() {
  return mount(BackupListPage, {
    global: {
      directives: { permission: () => undefined },
      stubs: {
        'management-page-header': passthrough('ManagementPageHeader'),
        'management-table-card': passthrough('ManagementTableCard'),
        'management-toolbar': passthrough('ManagementToolbar'),
        'task-detail-drawer': TaskDetailDrawerStub,
        't-alert': defineComponent({
          name: 'TAlert',
          props: {
            message: { type: String, default: '' },
            title: { type: String, default: '' },
          },
          setup(props) {
            return () => h('div', [props.title, props.message]);
          },
        }),
        't-button': TButtonStub,
        't-card': passthrough('TCard'),
        't-descriptions': passthrough('TDescriptions'),
        't-descriptions-item': passthrough('TDescriptionsItem'),
        't-dialog': passthrough('TDialog'),
        't-drawer': passthrough('TDrawer'),
        't-empty': passthrough('TEmpty'),
        't-form': passthrough('TForm'),
        't-form-item': passthrough('TFormItem'),
        't-loading': passthrough('TLoading'),
        't-option': passthrough('TOption'),
        't-select': passthrough('TSelect'),
        't-table': TTableStub,
        't-tag': passthrough('TTag'),
        't-tooltip': passthrough('TTooltip'),
      },
    },
  });
}

function detail(overrides: Record<string, unknown> = {}) {
  return {
    id: 2,
    task_id: 75,
    purpose: 'platform_manual',
    status: 'AVAILABLE',
    retain_until: '2026-07-28T07:17:00Z',
    created_at: '2026-07-27T07:17:00Z',
    config_snapshot: { size_bytes: 128, sha256: 'a'.repeat(64) },
    database_dump: { size_bytes: 256, sha256: 'b'.repeat(64) },
    restore_evidence: { status: 'NOT_VERIFIED' },
    ...overrides,
  };
}

describe('BackupListPage', () => {
  afterEach(() => vi.clearAllMocks());

  it('opens asset detail from the list without exposing a peer Task column', async () => {
    apiMocks.listBackups.mockResolvedValue({
      items: [
        {
          id: 2,
          purpose: 'platform_manual',
          status: 'AVAILABLE',
          retain_until: '2026-07-28T07:17:00Z',
          created_at: '2026-07-27T07:17:00Z',
        },
      ],
      total: 1,
    });
    apiMocks.getBackup.mockResolvedValue(detail());

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('backup.content.summary');
    expect(wrapper.text()).not.toContain('backup.list.columns.task');

    await wrapper.get('[data-row-id="2"] button').trigger('click');
    await flushPromises();

    expect(apiMocks.getBackup).toHaveBeenCalledWith(2);
    expect(wrapper.get('[data-testid="backup-detail-drawer"]').text()).toContain('SHA-256');
    expect(wrapper.text()).toContain('backup.detail.restore.notVerifiedTitle');
  });

  it('opens the related Task only from the asset detail', async () => {
    apiMocks.listBackups.mockResolvedValue({
      items: [
        {
          id: 2,
          purpose: 'platform_manual',
          status: 'AVAILABLE',
          retain_until: '2026-07-28T07:17:00Z',
          created_at: '2026-07-27T07:17:00Z',
        },
      ],
      total: 1,
    });
    apiMocks.getBackup.mockResolvedValue(detail());
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-row-id="2"] button').trigger('click');
    await flushPromises();
    const taskButton = wrapper.findAll('button').find((button) => button.text().includes('backup.detail.task.view'));
    expect(taskButton).toBeDefined();
    await taskButton?.trigger('click');

    expect(wrapper.get('[data-testid="task-detail-drawer"]').attributes('data-task-id')).toBe('75');
  });

  it('renders recorded restore evidence without treating it as a restore operation', async () => {
    apiMocks.listBackups.mockResolvedValue({
      items: [
        {
          id: 2,
          purpose: 'platform_manual',
          status: 'RESTORED',
          retain_until: '2026-07-28T07:17:00Z',
          created_at: '2026-07-27T07:17:00Z',
        },
      ],
      total: 1,
    });
    apiMocks.getBackup.mockResolvedValue(
      detail({
        restore_evidence: {
          status: 'RECORDED',
          result_code: 'manual_restore_verified',
          recorded_at: '2026-07-27T08:00:00Z',
        },
      }),
    );
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-row-id="2"] button').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('backup.detail.restore.recordedTitle');
    expect(wrapper.text()).not.toContain('backup.detail.restore.action');
  });

  it('shows the detail load error state', async () => {
    apiMocks.listBackups.mockResolvedValue({
      items: [
        {
          id: 2,
          purpose: 'platform_manual',
          status: 'AVAILABLE',
          retain_until: '2026-07-28T07:17:00Z',
          created_at: '2026-07-27T07:17:00Z',
        },
      ],
      total: 1,
    });
    apiMocks.getBackup.mockRejectedValue(new Error('network'));
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-row-id="2"] button').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('backup.list.loadFailed');
  });
});
