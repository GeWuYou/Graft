import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, reactive } from 'vue';

import RegistryDetailPage from './index.vue';

const pageSource = readFileSync(join(dirname(fileURLToPath(import.meta.url)), 'index.vue'), 'utf8');

const apiMocks = vi.hoisted(() => ({
  getRegistry: vi.fn(),
  getRegistryRepositories: vi.fn(),
  updateRegistry: vi.fn(),
}));
const messageMocks = vi.hoisted(() => ({ error: vi.fn(), success: vi.fn() }));
const routerMocks = vi.hoisted(() => ({ push: vi.fn(), replace: vi.fn().mockResolvedValue(undefined) }));
const tabsRouterStoreMocks = vi.hoisted(() => ({ updateActiveTabTitle: vi.fn() }));
const routeState = reactive({
  name: 'RegistryConnectionDetailIndex',
  params: { connectionRef: 'registry-a' },
  path: '/infrastructure/registries/registry-a',
  query: {} as Record<string, unknown>,
});
const formMocks = vi.hoisted(() => ({ validate: vi.fn() }));

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => routerMocks,
}));
vi.mock('@/store/modules/tabs-router', () => ({
  useTabsRouterStore: () => tabsRouterStoreMocks,
}));
vi.mock('@/utils/route/title', () => ({
  buildDetailTitleWithFallback: (_titleKey: string, name: string) => ({
    'en-US': `Image Registry Detail - ${name}`,
    'zh-CN': `镜像仓库详情 - ${name}`,
  }),
}));
vi.mock('../../api/registry', () => ({
  createRegistryRepository: vi.fn(),
  deleteRegistryRepository: vi.fn(),
  getRegistry: apiMocks.getRegistry,
  getRegistryRepositories: apiMocks.getRegistryRepositories,
  getRegistryRepositoryAssignmentCandidates: vi.fn(),
  getRegistryRepositoryAssignments: vi.fn(),
  replaceRegistryRepositoryAssignments: vi.fn(),
  addRegistryRepositoryAssignments: vi.fn(),
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
vi.mock('@/shared/components/selection', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/shared/components/selection')>();
  return {
    ...actual,
    PagedMultiSelect: defineComponent({
      setup:
        (_props, { slots }) =>
        () =>
          h('div', slots.default?.()),
    }),
  };
});

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

  it('updates the active detail tab title with the loaded Connection name', async () => {
    mountPage();
    await flushPromises();

    expect(tabsRouterStoreMocks.updateActiveTabTitle).toHaveBeenCalledWith(
      'RegistryConnectionDetailIndex',
      routeState,
      {
        'en-US': 'Image Registry Detail - Registry A',
        'zh-CN': '镜像仓库详情 - Registry A',
      },
    );
  });

  it('saves the editable fields, refreshes detail data, and restores the normal URL', async () => {
    apiMocks.getRegistry
      .mockResolvedValueOnce(connection())
      .mockResolvedValueOnce(connection({ display_name: 'Updated Registry' }));
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
    expect(tabsRouterStoreMocks.updateActiveTabTitle).toHaveBeenLastCalledWith(
      'RegistryConnectionDetailIndex',
      routeState,
      {
        'en-US': 'Image Registry Detail - Updated Registry',
        'zh-CN': '镜像仓库详情 - Updated Registry',
      },
    );
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

describe('RegistryDetailPage assignment safety', () => {
  it('uses the candidate selector rather than a manually entered batch of user IDs', () => {
    expect(pageSource).toContain('<paged-multi-select');
    expect(pageSource).not.toContain('batchUserIds');
    expect(pageSource).not.toContain('grantBatchAssignments');
  });

  it('uses atomic additive batch submission and final-set replacement semantics', () => {
    expect(pageSource).toContain('addRegistryRepositoryAssignments(connectionRef.value');
    expect(pageSource).toContain('replaceRegistryRepositoryAssignments(connectionRef.value');
    expect(pageSource).not.toContain('ASSIGNMENT_MUTATION_CONCURRENCY');
  });
});
