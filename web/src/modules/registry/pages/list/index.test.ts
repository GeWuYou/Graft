import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import RegistryListPage from './index.vue';

const apiMocks = vi.hoisted(() => ({
  getRegistries: vi.fn(),
}));

vi.mock('../../api/registry', () => ({
  createRegistry: vi.fn(),
  createRegistryRepository: vi.fn(),
  deleteRegistry: vi.fn(),
  deleteRegistryRepository: vi.fn(),
  getRegistries: apiMocks.getRegistries,
  getRegistryRepositories: vi.fn(),
  getRegistryRepositoryAssignments: vi.fn(),
  grantRegistryRepositoryAssignment: vi.fn(),
  revokeRegistryRepositoryAssignment: vi.fn(),
  updateRegistry: vi.fn(),
  updateRegistryRepository: vi.fn(),
  verifyRegistry: vi.fn(),
}));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
vi.mock('@/shared/observability', () => ({ formatLocaleDateTime: () => '' }));
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ locale: { value: 'zh-CN' }, t: (key: string) => key }),
}));

const passthrough = (name: string) =>
  defineComponent({
    name,
    setup(_props, { slots }) {
      return () => h('div', slots.default?.());
    },
  });

const buttonStub = defineComponent({
  name: 'TButton',
  emits: ['click'],
  setup(_props, { emit, slots }) {
    return () => h('button', { type: 'button', onClick: () => emit('click') }, slots.default?.());
  },
});

const drawerStub = defineComponent({
  name: 'TDrawer',
  props: { visible: { type: Boolean, default: false } },
  setup(props, { slots }) {
    return () => h('aside', { 'data-visible': String(props.visible) }, slots.default?.());
  },
});

const inputStub = defineComponent({
  name: 'TInput',
  props: { modelValue: { type: String, default: '' } },
  emits: ['clear', 'enter', 'update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('input', {
        'data-testid': 'registry-search',
        value: props.modelValue,
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
      });
  },
});

const tableStub = defineComponent({
  name: 'TTable',
  props: { columns: { type: Array, default: () => [] }, data: { type: Array, default: () => [] }, loading: Boolean },
  setup(props, { slots }) {
    return () =>
      h('div', [
        ...(props.data as Array<Record<string, unknown>>).flatMap((row) => [
          slots.status?.({ row }),
          slots.verified?.({ row }),
          slots.created_at?.({ row }),
        ]),
        slots.default?.(),
      ]);
  },
});

const paginationStub = defineComponent({
  name: 'TPagination',
  props: ['current', 'pageSize', 'total'],
  emits: ['change', 'update:current', 'update:pageSize'],
  setup() {
    return () => h('div');
  },
});

function mountPage() {
  return mount(RegistryListPage, {
    global: {
      stubs: {
        't-alert': passthrough('TAlert'),
        't-button': buttonStub,
        't-drawer': drawerStub,
        't-empty': passthrough('TEmpty'),
        't-form': passthrough('TForm'),
        't-form-item': passthrough('TFormItem'),
        't-input': inputStub,
        't-input-number': passthrough('TInputNumber'),
        't-popconfirm': passthrough('TPopconfirm'),
        't-space': passthrough('TSpace'),
        't-switch': passthrough('TSwitch'),
        't-pagination': paginationStub,
        't-table': tableStub,
        't-tag': passthrough('TTag'),
        't-textarea': passthrough('TTextarea'),
      },
    },
  });
}

describe('RegistryListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiMocks.getRegistries.mockResolvedValue({ items: [], total: 0 });
  });

  it('opens the Connection create drawer from the management action', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.findAll('aside').at(0)?.attributes('data-visible')).toBe('false');
    await wrapper.get('button').trigger('click');

    expect(wrapper.findAll('aside').at(0)?.attributes('data-visible')).toBe('true');
  });

  it('resets pagination when clearing search and queries the first page', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const pagination = wrapper.findComponent(paginationStub);
    pagination.vm.$emit('change', { current: 3, pageSize: 10 });
    await flushPromises();
    wrapper.findComponent(inputStub).vm.$emit('clear');
    await flushPromises();

    expect(apiMocks.getRegistries).toHaveBeenLastCalledWith({ limit: 10, offset: 0, search: undefined });
  });

  it('keeps the newest page response when an older request resolves later', async () => {
    let resolveFirst: ((value: { items: Array<{ connection_ref: string }>; total: number }) => void) | undefined;
    let resolveSecond: ((value: { items: Array<{ connection_ref: string }>; total: number }) => void) | undefined;
    apiMocks.getRegistries.mockReset();
    apiMocks.getRegistries
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveFirst = resolve;
          }),
      )
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveSecond = resolve;
          }),
      );

    const wrapper = mountPage();
    await nextTick();
    wrapper.findComponent(inputStub).vm.$emit('enter');
    await nextTick();
    resolveSecond?.({ items: [{ connection_ref: 'newer' }], total: 1 });
    await flushPromises();
    resolveFirst?.({ items: [{ connection_ref: 'older' }], total: 1 });
    await flushPromises();

    expect(wrapper.findComponent(tableStub).props('data')).toEqual([{ connection_ref: 'newer' }]);
  });
});
