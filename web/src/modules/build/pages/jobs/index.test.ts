import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import BuildJobsPage from './index.vue';

const mocks = vi.hoisted(() => ({ getBuildJob: vi.fn(), getBuildJobs: vi.fn(), push: vi.fn() }));

vi.mock('../../api/build', () => ({ getBuildJob: mocks.getBuildJob, getBuildJobs: mocks.getBuildJobs }));
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: { value: 'en-US' }, t: (key: string) => key }) }));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
vi.mock('@/shared/observability', () => ({ formatLocaleDateTime: (value: string) => value }));
vi.mock('@/shared/components/management', () => ({
  ManagementPageHeader: defineComponent({ setup: () => () => null }),
  ManagementToolbar: defineComponent({
    setup:
      (_props, { slots }) =>
      () =>
        h('div', [slots.filters?.(), slots.actions?.()]),
  }),
}));
vi.mock('@/shared/components/management/ManagementPagedTable.vue', () => ({
  default: defineComponent({
    props: { rows: { type: Array, default: () => [] } },
    setup(props, { slots }) {
      return () =>
        h(
          'div',
          (props.rows as Array<{ build_id: string }>).map((row) => h('div', [row.build_id, slots.actions?.({ row })])),
        );
    },
  }),
}));
vi.mock('tdesign-icons-vue-next', () => ({
  AddIcon: defineComponent({ setup: () => () => null }),
  RefreshIcon: defineComponent({ setup: () => () => null }),
}));

const WrapperStub = defineComponent({
  setup(_props, { slots }) {
    return () => h('div', slots.default?.());
  },
});
const ButtonStub = defineComponent({
  props: { type: { type: String, default: 'button' } },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h('button', { type: props.type, onClick: (event: MouseEvent) => emit('click', event) }, slots.default?.());
  },
});

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((nextResolve) => {
    resolve = nextResolve;
  });
  return { promise, resolve };
}

function mountPage() {
  return mount(BuildJobsPage, {
    global: {
      stubs: {
        't-alert': WrapperStub,
        't-button': ButtonStub,
        't-descriptions': WrapperStub,
        't-descriptions-item': WrapperStub,
        't-drawer': WrapperStub,
        't-empty': WrapperStub,
        't-form': WrapperStub,
        't-form-item': WrapperStub,
        't-input': WrapperStub,
        't-input-number': WrapperStub,
        't-loading': WrapperStub,
        't-space': WrapperStub,
        't-tag': WrapperStub,
      },
    },
  });
}

describe('BuildJobsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getBuildJobs.mockResolvedValue({ items: [], total: 0 });
  });

  it('keeps the newest list response when requests resolve out of order', async () => {
    const first = deferred<{ items: Array<{ build_id: string }>; total: number }>();
    const second = deferred<{ items: Array<{ build_id: string }>; total: number }>();
    mocks.getBuildJobs.mockReset().mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise);
    const wrapper = mountPage();
    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'build.jobs.refresh')
      ?.trigger('click');
    expect(mocks.getBuildJobs).toHaveBeenCalledTimes(2);
    second.resolve({ items: [{ build_id: 'new' }], total: 1 });
    await flushPromises();
    first.resolve({ items: [{ build_id: 'old' }], total: 1 });
    await flushPromises();
    expect(wrapper.text()).toContain('new');
    expect(wrapper.text()).not.toContain('old');
  });

  it('keeps the newest detail response when requests resolve out of order', async () => {
    mocks.getBuildJobs.mockResolvedValue({ items: [{ build_id: 'first' }, { build_id: 'second' }], total: 2 });
    const first = deferred<{ build_id: string }>();
    const second = deferred<{ build_id: string }>();
    mocks.getBuildJob.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise);
    const wrapper = mountPage();
    await flushPromises();
    const detailButtons = wrapper.findAll('button').filter((button) => button.text() === 'build.jobs.detail.title');
    await detailButtons[0]?.trigger('click');
    await detailButtons[1]?.trigger('click');
    second.resolve({ build_id: 'second' });
    await flushPromises();
    first.resolve({ build_id: 'first' });
    await flushPromises();
    expect(wrapper.text()).toContain('second');
  });
});
