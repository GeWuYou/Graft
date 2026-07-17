import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import ApplicationTemplateDetailIndex from './index.vue';

const mocks = vi.hoisted(() => ({
  getApplicationTemplate: vi.fn(),
  postApplicationTemplateArchive: vi.fn(),
  postApplicationTemplateClone: vi.fn(),
  postApplicationTemplatePublish: vi.fn(),
  postApplicationTemplateWithdraw: vi.fn(),
  putApplicationTemplate: vi.fn(),
  deleteApplicationTemplate: vi.fn(),
  push: vi.fn(),
  replace: vi.fn(),
  route: {
    params: { templateId: 'tpl_1' },
    path: '/applications/templates/tpl_1',
    query: {} as Record<string, unknown>,
  },
  tabsRouterStore: {
    activeTabKey: '/applications/templates/tpl_1',
    discardTabRouter: vi.fn(),
    tabRouters: [
      {
        tabKey: '/applications/templates/tpl_1',
        path: '/applications/templates/tpl_1',
      },
    ],
  },
}));

vi.mock('../../api/project', () => mocks);
vi.mock('vue-router', () => ({
  useRoute: () => mocks.route,
  useRouter: () => ({ push: mocks.push, replace: mocks.replace }),
}));
vi.mock('@/store', () => ({ useTabsRouterStore: () => mocks.tabsRouterStore }));
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key }),
}));
vi.mock('tdesign-vue-next/es/message', () => ({ MessagePlugin: { error: vi.fn(), success: vi.fn() } }));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
vi.mock('../../components/ProjectCreateWorkspaceEditor.vue', () => ({
  default: { template: '<div>{{ files?.[0]?.path }}</div>', props: ['files'] },
}));
vi.mock('../../components/ProjectLifecycleConfigurationReview.vue', () => ({
  default: { props: ['disabled'], template: '<div data-testid="lifecycle-review" :data-disabled="disabled" />' },
}));

const WrapperStub = { template: '<div><slot /><slot name="actions" /><slot name="meta" /></div>' };
const TTabsStub = defineComponent({
  props: { value: { type: String, default: 'overview' } },
  emits: ['update:value'],
  setup(_props, { slots }) {
    return () => h('div', { class: 't-tabs-stub' }, slots.default?.());
  },
});
const TTabPanelStub = defineComponent({
  props: {
    destroyOnHide: { type: Boolean, default: true },
    label: { type: String, default: '' },
    value: { type: String, default: '' },
  },
  setup(_props, { slots }) {
    return () => h('section', { class: 't-tab-panel-stub' }, slots.default?.());
  },
});

function mountPage() {
  return mount(ApplicationTemplateDetailIndex, {
    global: {
      stubs: {
        'management-page-content': WrapperStub,
        'management-page-header': WrapperStub,
        't-space': WrapperStub,
        't-tabs': TTabsStub,
        't-tab-panel': TTabPanelStub,
        't-loading': WrapperStub,
        't-alert': WrapperStub,
        't-card': WrapperStub,
        't-form': WrapperStub,
        't-form-item': WrapperStub,
        't-tag': WrapperStub,
        't-dialog': {
          template: '<div v-if="visible"><slot /><button data-testid="dialog-confirm" @click="$emit(\'confirm\')">confirm</button></div>',
          props: ['visible'],
        },
        't-button': { template: '<button @click="$emit(\'click\')"><slot /></button>' },
        't-input': { template: '<input :value="modelValue ?? value" />', props: ['modelValue', 'value'] },
        't-textarea': { template: '<textarea :value="modelValue" />', props: ['modelValue'] },
      },
    },
  });
}

describe('ApplicationTemplateDetailIndex', () => {
  beforeEach(() => {
    [
      mocks.getApplicationTemplate,
      mocks.postApplicationTemplateArchive,
      mocks.postApplicationTemplateClone,
      mocks.postApplicationTemplatePublish,
      mocks.postApplicationTemplateWithdraw,
      mocks.putApplicationTemplate,
      mocks.deleteApplicationTemplate,
      mocks.push,
      mocks.replace,
      mocks.tabsRouterStore.discardTabRouter,
    ].forEach((mock) => mock.mockReset());
    mocks.tabsRouterStore.activeTabKey = '/applications/templates/tpl_1';
    mocks.tabsRouterStore.tabRouters = [
      { tabKey: '/applications/templates/tpl_1', path: '/applications/templates/tpl_1' },
    ];
    mocks.route.query = {};
  });

  it('hydrates the unwrapped template detail response into editable fields', async () => {
    mocks.getApplicationTemplate.mockResolvedValue({
      template_id: 'tpl_1',
      display_name: 'Compose template',
      description: 'Reusable compose application',
      deployment_adapter_kind: 'compose',
      version: {
        template_version_id: 'tplv_1',
        version_number: 2,
        status: 'draft',
        definition_schema_version: 1,
        definition: {
          compose_file_path: 'stack.yml',
          workspace_entries: [{ path: 'stack.yml', node_type: 'file', content: 'services: {}' }],
          lifecycle_configuration: {},
        },
      },
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(mocks.getApplicationTemplate).toHaveBeenCalledWith('tpl_1');
    expect(wrapper.find('input').element.value).toBe('Compose template');
    expect(wrapper.text()).toContain('stack.yml');
    expect(wrapper.findAllComponents(TTabPanelStub)).toHaveLength(3);
    expect(wrapper.findAllComponents(TTabPanelStub).every((panel) => panel.props('destroyOnHide') === false)).toBe(true);
  });

  it('initializes the active detail tab from the route query and syncs tab changes', async () => {
    mocks.route.query = { tab: 'lifecycle' };
    mocks.getApplicationTemplate.mockResolvedValue({
      template_id: 'tpl_1',
      display_name: 'Compose template',
      description: '',
      deployment_adapter_kind: 'compose',
      version: { template_version_id: 'tplv_1', version_number: 1, status: 'draft', definition_schema_version: 1, definition: { compose_file_path: 'compose.yaml', workspace_entries: [] } },
    });

    const wrapper = mountPage();
    await flushPromises();

    const tabs = wrapper.getComponent(TTabsStub);
    expect(tabs.props('value')).toBe('lifecycle');
    await tabs.vm.$emit('update:value', 'workspace');
    await flushPromises();

    expect(mocks.replace).toHaveBeenCalledWith({ query: { tab: 'workspace' } });
  });

  it('falls back to the overview tab for an unknown detail tab query', async () => {
    mocks.route.query = { tab: 'unknown' };
    mocks.getApplicationTemplate.mockResolvedValue({
      template_id: 'tpl_1',
      display_name: 'Compose template',
      description: '',
      deployment_adapter_kind: 'compose',
      version: { template_version_id: 'tplv_1', version_number: 1, status: 'draft', definition_schema_version: 1, definition: { compose_file_path: 'compose.yaml', workspace_entries: [] } },
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.getComponent(TTabsStub).props('value')).toBe('overview');
  });

  it('disables lifecycle inputs for a published template version', async () => {
    mocks.route.query = { tab: 'lifecycle' };
    mocks.getApplicationTemplate.mockResolvedValue({
      template_id: 'tpl_1',
      display_name: 'Compose template',
      description: '',
      deployment_adapter_kind: 'compose',
      version: { template_version_id: 'tplv_1', version_number: 1, status: 'published', definition_schema_version: 1, definition: { compose_file_path: 'compose.yaml', workspace_entries: [] } },
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.get('[data-testid="lifecycle-review"]').attributes('data-disabled')).toBe('true');
  });

  it('discards a deleted template tab and replaces it with the template list', async () => {
    mocks.getApplicationTemplate.mockResolvedValue({
      template_id: 'tpl_1',
      display_name: 'Compose template',
      description: '',
      deployment_adapter_kind: 'compose',
      version: { template_version_id: 'tplv_1', version_number: 1, status: 'draft', definition_schema_version: 1, definition: { compose_file_path: 'compose.yaml', workspace_entries: [] } },
    });
    mocks.deleteApplicationTemplate.mockResolvedValue(undefined);

    const wrapper = mountPage();
    await flushPromises();
    await wrapper.findAll('button').find((button) => button.text() === 'project.templates.delete')?.trigger('click');
    await wrapper.get('[data-testid="dialog-confirm"]').trigger('click');
    await flushPromises();

    expect(mocks.tabsRouterStore.discardTabRouter).toHaveBeenCalledWith(
      expect.objectContaining({ tabKey: '/applications/templates/tpl_1' }),
    );
    expect(mocks.replace).toHaveBeenCalledWith({ name: 'ApplicationTemplatesIndex' });
  });

  it('replaces a deleted template URL with the template list', async () => {
    const error = Object.assign(new Error('not found'), { isApiRequestError: true, status: 404 });
    mocks.getApplicationTemplate.mockRejectedValue(error);

    mountPage();
    await flushPromises();

    expect(mocks.tabsRouterStore.discardTabRouter).toHaveBeenCalledWith(
      expect.objectContaining({ tabKey: '/applications/templates/tpl_1' }),
    );
    expect(mocks.replace).toHaveBeenCalledWith({ name: 'ApplicationTemplatesIndex' });
  });
});
