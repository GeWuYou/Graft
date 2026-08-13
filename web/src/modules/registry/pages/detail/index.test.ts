import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, reactive } from 'vue';

import RegistryDetailPage from './index.vue';

const apiMocks = vi.hoisted(() => ({
  getRegistry: vi.fn(),
  getRegistryRepositories: vi.fn(),
  updateRegistry: vi.fn(),
}));
const messageMocks = vi.hoisted(() => ({ error: vi.fn(), success: vi.fn() }));
const routerMocks = vi.hoisted(() => ({ push: vi.fn(), replace: vi.fn().mockResolvedValue(undefined) }));
const routeState = reactive({
  params: { connectionRef: 'registry-a' },
  query: {} as Record<string, unknown>,
});
const formMocks = vi.hoisted(() => ({ validate: vi.fn() }));

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => routerMocks,
}));
vi.mock('../../api/registry', () => ({
  createRegistryRepository: vi.fn(),
  deleteRegistryRepository: vi.fn(),
  getRegistry: apiMocks.getRegistry,
  getRegistryRepositories: apiMocks.getRegistryRepositories,
  getRegistryRepositoryAssignmentCandidates: vi.fn(),
  getRegistryRepositoryAssignments: vi.fn(),
  grantRegistryRepositoryAssignment: vi.fn(),
  revokeRegistryRepositoryAssignment: vi.fn(),
  updateRegistry: apiMocks.updateRegistry,
  updateRegistryRepository: vi.fn(),
}));
vi.mock('tdesign-vue-next/es/message', () => ({ MessagePlugin: messageMocks }));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: { value: 'zh-CN' }, t: (key: string) => key }) }));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
vi.mock('@/shared/observability', () => ({ formatLocaleDateTime: () => '' }));
vi.mock('@/shared/components/management', () => ({
  ManagementBatchBar: defineComponent({
    setup:
      (_props, { slots }) =>
      () =>
        h('div', slots.default?.()),
  }),
  ManagementPageContent: defineComponent({
    setup:
      (_props, { slots }) =>
      () =>
        h('div', slots.default?.()),
  }),
  ManagementPageHeader: defineComponent({
    setup:
      (_props, { slots }) =>
      () =>
        h('div', slots.default?.()),
  }),
}));
vi.mock('@/shared/components/management/ManagementPagedTable.vue', () => ({
  default: defineComponent({
    name: 'ManagementPagedTable',
    setup:
      (_props, { slots }) =>
      () =>
        h('div', slots.default?.()),
  }),
}));
vi.mock('@/shared/components/responsive/ResponsiveDialog.vue', () => ({
  default: defineComponent({
    name: 'ResponsiveDialog',
    props: { title: { default: '', type: String }, visible: Boolean },
    emits: ['update:visible'],
    setup(props, { slots }) {
      return () =>
        h('aside', { 'data-header': props.title, 'data-visible': String(props.visible) }, [
          slots.default?.(),
          slots.footer?.(),
        ]);
    },
  }),
}));
vi.mock('@/shared/components/selection', () => ({
  PagedMultiSelect: defineComponent({
    setup:
      (_props, { slots }) =>
      () =>
        h('div', slots.default?.()),
  }),
}));

const connectionFormStub = defineComponent({
  name: 'RegistryConnectionForm',
  props: { editing: Boolean, modelValue: { type: Object, required: true } },
  emits: ['update:modelValue'],
  setup(_props, { expose }) {
    expose({ validate: formMocks.validate });
    return () => h('div', { 'data-testid': 'connection-form' });
  },
});
const drawerStub = defineComponent({
  name: 'TDrawer',
  props: { confirmLoading: Boolean, header: { default: '', type: String }, visible: Boolean },
  emits: ['cancel', 'close', 'confirm'],
  setup(props, { slots }) {
    return () =>
      h('aside', { 'data-header': props.header, 'data-visible': String(props.visible) }, [slots.default?.()]);
  },
});
const buttonStub = defineComponent({
  name: 'TButton',
  emits: ['click'],
  setup(_props, { emit, slots }) {
    return () => h('button', { type: 'button', onClick: () => emit('click') }, slots.default?.());
  },
});
const passthrough = (name: string) =>
  defineComponent({
    name,
    setup:
      (_props, { slots }) =>
      () =>
        h('div', slots.default?.()),
  });

function connection(overrides: Record<string, unknown> = {}) {
  return {
    availability: true,
    connection_ref: 'registry-a',
    created_at: '2026-08-13T00:00:00Z',
    description: 'initial description',
    display_name: 'Registry A',
    enabled: true,
    endpoint: 'https://registry.example.com',
    insecure: false,
    provider: 'generic_oci',
    updated_at: '2026-08-13T00:00:00Z',
    verification_status: 'verified',
    credential_configured: true,
    ...overrides,
  };
}

function mountPage() {
  return mount(RegistryDetailPage, {
    global: {
      stubs: {
        'registry-connection-form': connectionFormStub,
        't-alert': passthrough('TAlert'),
        't-button': buttonStub,
        't-card': passthrough('TCard'),
        't-descriptions': passthrough('TDescriptions'),
        't-descriptions-item': passthrough('TDescriptionsItem'),
        't-drawer': drawerStub,
        't-empty': passthrough('TEmpty'),
        't-form': passthrough('TForm'),
        't-form-item': passthrough('TFormItem'),
        't-input': passthrough('TInput'),
        't-input-number': passthrough('TInputNumber'),
        't-loading': passthrough('TLoading'),
        't-popconfirm': passthrough('TPopconfirm'),
        't-space': passthrough('TSpace'),
        't-switch': passthrough('TSwitch'),
        't-table': passthrough('TTable'),
        't-tag': passthrough('TTag'),
      },
    },
  });
}

function connectionDialog(wrapper: ReturnType<typeof mountPage>) {
  return wrapper
    .findAllComponents({ name: 'ResponsiveDialog' })
    .find((dialog) => dialog.props('title') === 'registry.route.detail.editConnection')!;
}

describe('RegistryDetailPage connection editing', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routeState.params.connectionRef = 'registry-a';
    routeState.query = { mode: 'edit', source: 'list' };
    apiMocks.getRegistry.mockResolvedValue(connection());
    apiMocks.getRegistryRepositories.mockResolvedValue({ items: [], total: 0 });
    apiMocks.updateRegistry.mockResolvedValue(connection());
    formMocks.validate.mockResolvedValue(true);
  });

  it('opens from the canonical edit URL and remains open after a refresh', async () => {
    const first = mountPage();
    await flushPromises();
    expect(connectionDialog(first).props('visible')).toBe(true);
    expect(first.findComponent(connectionFormStub).props('modelValue')).toMatchObject({
      connection_ref: 'registry-a',
      display_name: 'Registry A',
      endpoint: 'https://registry.example.com',
    });
    first.unmount();

    const refreshed = mountPage();
    await flushPromises();
    expect(connectionDialog(refreshed).props('visible')).toBe(true);
  });

  it('saves the editable fields, refreshes detail data, and restores the normal URL', async () => {
    const wrapper = mountPage();
    await flushPromises();
    wrapper.findComponent(connectionFormStub).vm.$emit('update:modelValue', {
      connection_ref: 'registry-a',
      display_name: 'Updated Registry',
      endpoint: 'https://updated.example.com',
      enabled: false,
      insecure: true,
      description: '',
    });
    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'registry.list.save')
      ?.trigger('click');
    await flushPromises();

    expect(apiMocks.updateRegistry).toHaveBeenCalledWith('registry-a', {
      display_name: 'Updated Registry',
      endpoint: 'https://updated.example.com',
      enabled: false,
      insecure: true,
      description: null,
    });
    expect(apiMocks.getRegistry).toHaveBeenCalledTimes(2);
    expect(apiMocks.getRegistryRepositories).toHaveBeenCalledTimes(2);
    expect(messageMocks.success).toHaveBeenCalledWith('registry.route.detail.connectionSaveSuccess');
    expect(routerMocks.replace).toHaveBeenCalledWith({
      path: '/infrastructure/registries/registry-a',
      query: { source: 'list', mode: undefined },
    });
  });

  it('does not submit when the Connection form is invalid', async () => {
    formMocks.validate.mockResolvedValueOnce({ display_name: [{ result: false }] });
    const wrapper = mountPage();
    await flushPromises();
    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'registry.list.save')
      ?.trigger('click');

    expect(apiMocks.updateRegistry).not.toHaveBeenCalled();
    expect(routerMocks.replace).not.toHaveBeenCalled();
  });

  it('keeps the edit URL and draft available when saving fails', async () => {
    apiMocks.updateRegistry.mockRejectedValue(new Error('failed'));
    const wrapper = mountPage();
    await flushPromises();
    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'registry.list.save')
      ?.trigger('click');
    await flushPromises();

    expect(connectionDialog(wrapper).props('visible')).toBe(true);
    expect(routerMocks.replace).not.toHaveBeenCalled();
    expect(messageMocks.error).toHaveBeenCalledWith('registry.route.detail.connectionSaveFailed');
  });

  it('removes edit mode when the drawer closes', async () => {
    routeState.params.connectionRef = 'registry:acceptance-ghcr';
    const wrapper = mountPage();
    await flushPromises();
    await connectionDialog(wrapper).vm.$emit('update:visible', false);
    await flushPromises();

    expect(routerMocks.replace).toHaveBeenCalledWith({
      path: '/infrastructure/registries/registry%3Aacceptance-ghcr',
      query: { source: 'list', mode: undefined },
    });
    expect(apiMocks.getRegistry).toHaveBeenCalledTimes(1);
    expect(apiMocks.getRegistryRepositories).toHaveBeenCalledTimes(1);
  });

  it('deduplicates repeated close events while the canonical route is being replaced', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await connectionDialog(wrapper).vm.$emit('update:visible', false);
    await connectionDialog(wrapper).vm.$emit('update:visible', false);
    await flushPromises();

    expect(routerMocks.replace).toHaveBeenCalledTimes(1);
  });
});
