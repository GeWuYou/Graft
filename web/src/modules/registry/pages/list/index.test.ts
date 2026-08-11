import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

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
        't-input': passthrough('TInput'),
        't-input-number': passthrough('TInputNumber'),
        't-popconfirm': passthrough('TPopconfirm'),
        't-space': passthrough('TSpace'),
        't-switch': passthrough('TSwitch'),
        't-table': passthrough('TTable'),
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
});
