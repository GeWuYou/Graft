import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';
import { createI18n } from 'vue-i18n';

import type { TaskSummary } from '../types/task';
import TaskHistoryTable from './TaskHistoryTable.vue';

const getTasksMock = vi.hoisted(() => vi.fn());

vi.mock('../api/task', () => ({ getTasks: getTasksMock }));
vi.mock('@/shared/localized-api-error', () => ({ resolveLocalizedErrorMessage: () => 'Task history load failed.' }));
vi.mock('@/shared/observability', () => ({ formatLocaleDateTime: (value: string) => `formatted:${value}` }));

class ResizeObserverMock {
  static instances: ResizeObserverMock[] = [];

  callback: ResizeObserverCallback;
  disconnect = vi.fn();
  observe = vi.fn();

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
    ResizeObserverMock.instances.push(this);
  }

  emit(width: number) {
    this.callback([{ contentRect: { height: 0, width } } as ResizeObserverEntry], this as unknown as ResizeObserver);
  }
}

const TableStub = defineComponent({
  name: 'TTableStub',
  props: {
    columns: { type: Array, default: () => [] },
    data: { type: Array, default: () => [] },
  },
  emits: ['row-click'],
  setup(props, { emit }) {
    return () =>
      h('table', { 'data-testid': 'task-history-table' }, [
        h(
          'thead',
          (props.columns as Array<{ colKey: string }>).map((column) => h('th', column.colKey)),
        ),
        h(
          'tbody',
          (props.data as TaskSummary[]).map((row) =>
            h('tr', { onClick: () => emit('row-click', { row }) }, [h('td', String(row.id))]),
          ),
        ),
      ]);
  },
});

const ButtonStub = defineComponent({
  name: 'TButtonStub',
  inheritAttrs: false,
  setup(_props, { attrs, slots }) {
    return () => h('button', attrs, slots.default?.());
  },
});

const TagStub = defineComponent({
  name: 'TTagStub',
  setup(_props, { slots }) {
    return () => h('span', { 'data-testid': 'task-status-tag' }, slots.default?.());
  },
});

const LoadingStub = defineComponent({
  name: 'TLoadingStub',
  props: { loading: { type: Boolean, default: false } },
  setup(props, { slots }) {
    return () =>
      h('div', { 'data-loading': String(props.loading), 'data-testid': 'task-history-loading' }, slots.default?.());
  },
});

const EmptyStub = defineComponent({
  name: 'TEmptyStub',
  props: { title: { type: String, default: '' } },
  setup(props) {
    return () => h('div', { 'data-testid': 'task-history-empty' }, props.title);
  },
});

const i18n = createI18n({
  legacy: false,
  locale: 'en-US',
  messages: {
    'en-US': {
      task: {
        actions: { refresh: 'Refresh', view: 'View' },
        history: {
          columns: {
            createdAt: 'Created At',
            operation: 'Operation',
            stage: 'Current Stage',
            status: 'Status',
            type: 'Type',
          },
          description: 'Review task execution history.',
          empty: 'No task records are available.',
          loadFailed: 'Task history could not be loaded.',
          title: 'Task History',
        },
        status: { success: 'Succeeded' },
      },
    },
  },
});

function taskSummary(): TaskSummary {
  return {
    created_at: '2026-07-18T23:06:22Z',
    current_stage_key: 'up',
    id: 42,
    owner_id: 'app-1',
    owner_type: 'application',
    status: 'success',
    type: 'application.compose.redeploy',
  } as TaskSummary;
}

function mountTaskHistory() {
  return mount(TaskHistoryTable, {
    props: {
      ownerId: 'app-1',
      ownerType: 'application',
      resolveTaskType: (type: string) => `Resolved ${type}`,
    },
    global: {
      plugins: [i18n],
      stubs: {
        't-button': ButtonStub,
        't-empty': EmptyStub,
        't-loading': LoadingStub,
        't-table': TableStub,
        't-tag': TagStub,
      },
    },
  });
}

async function setResponsiveWidth(width: number) {
  ResizeObserverMock.instances[0]?.emit(width);
  await nextTick();
}

describe('TaskHistoryTable', () => {
  beforeEach(() => {
    ResizeObserverMock.instances = [];
    vi.stubGlobal('ResizeObserver', ResizeObserverMock);
    getTasksMock.mockResolvedValue({ items: [taskSummary()] });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it('shows complete task summaries as cards below the compact threshold and opens details from the explicit action', async () => {
    const wrapper = mountTaskHistory();
    await flushPromises();
    await setResponsiveWidth(375);

    const card = wrapper.get('.task-history__card');
    expect(wrapper.find('[data-testid="task-history-table"]').exists()).toBe(false);
    expect(card.text()).toContain('Resolved application.compose.redeploy');
    expect(card.text()).toContain('Status');
    expect(card.text()).toContain('Succeeded');
    expect(card.text()).toContain('Current Stage');
    expect(card.text()).toContain('up');
    expect(card.text()).toContain('formatted:2026-07-18T23:06:22Z');

    await card.get('button').trigger('click');

    expect(card.attributes('role')).toBeUndefined();
    expect(wrapper.emitted('open')).toEqual([[taskSummary()]]);
  });

  it('keeps the desktop table at and above 768px with the existing columns and row action', async () => {
    const wrapper = mountTaskHistory();
    await flushPromises();
    await setResponsiveWidth(768);

    const table = wrapper.get('[data-testid="task-history-table"]');
    expect(wrapper.find('.task-history__card').exists()).toBe(false);
    expect(table.findAll('th').map((column) => column.text())).toEqual([
      'type',
      'status',
      'current_stage_key',
      'created_at',
      'operation',
    ]);

    await table.get('tr').trigger('click');
    expect(wrapper.emitted('open')).toEqual([[taskSummary()]]);
  });

  it('keeps the existing localized load failure feedback', async () => {
    getTasksMock.mockRejectedValueOnce(new Error('unavailable'));

    const wrapper = mountTaskHistory();
    await flushPromises();

    expect(wrapper.get('.task-history__error').text()).toBe('Task history load failed.');
  });

  it('keeps compact loading and empty feedback visible without the desktop table', async () => {
    getTasksMock.mockResolvedValueOnce({ items: [] });

    const wrapper = mountTaskHistory();
    await flushPromises();
    await setResponsiveWidth(375);

    expect(wrapper.find('[data-testid="task-history-table"]').exists()).toBe(false);
    expect(wrapper.get('[data-testid="task-history-loading"]').attributes('data-loading')).toBe('false');
    expect(wrapper.get('[data-testid="task-history-empty"]').text()).toBe('No task records are available.');
  });
});
