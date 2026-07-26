import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { computed, nextTick } from 'vue';

import { TableActionMenu } from '@/shared/components/management';

import ApplicationTemplateListIndex from './index.vue';

const mocks = vi.hoisted(() => ({
  getApplicationManagedTemplates: vi.fn(),
  deleteApplicationTemplate: vi.fn(),
  postApplicationTemplateArchive: vi.fn(),
  postApplicationTemplateClone: vi.fn(),
  postApplicationTemplatePublish: vi.fn(),
  postApplicationTemplateWithdraw: vi.fn(),
  push: vi.fn(),
}));
const responsiveVariantMocks = vi.hoisted(() => ({ density: 'compact' }));

vi.mock('../../api/project', () => ({
  getApplicationManagedTemplates: mocks.getApplicationManagedTemplates,
  deleteApplicationTemplate: mocks.deleteApplicationTemplate,
  postApplicationTemplateArchive: mocks.postApplicationTemplateArchive,
  postApplicationTemplateClone: mocks.postApplicationTemplateClone,
  postApplicationTemplatePublish: mocks.postApplicationTemplatePublish,
  postApplicationTemplateWithdraw: mocks.postApplicationTemplateWithdraw,
}));
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }));
vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>();
  return {
    ...actual,
    useI18n: () => ({ locale: { value: 'zh-CN' }, t: (key: string) => key }),
  };
});
vi.mock('@/shared/composables', () => ({
  useResponsiveVariant: () => computed(() => ({ density: responsiveVariantMocks.density })),
}));
vi.mock('tdesign-vue-next/es/message', () => ({ MessagePlugin: { error: vi.fn() } }));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

const WrapperStub = { template: '<div><slot /><slot name="meta" /><slot name="actions" /></div>' };

function mountPage() {
  return mount(ApplicationTemplateListIndex, {
    global: {
      stubs: {
        'management-page-content': WrapperStub,
        'management-page-header': WrapperStub,
        't-space': WrapperStub,
        't-card': WrapperStub,
        't-alert': WrapperStub,
        't-table': {
          template: '<div><slot /><slot name="operation" :row="row" /><slot name="empty" /></div>',
          props: ['data'],
          computed: {
            row() {
              return this.data?.[0] ?? { archived_at: null, version: { status: 'draft' } };
            },
          },
        },
        't-tag': WrapperStub,
        't-dialog': {
          template: '<div v-if="visible"><slot /><button data-testid="dialog-confirm" @click="$emit(\'confirm\')">confirm</button></div>',
          props: ['visible'],
        },
        't-form': WrapperStub,
        't-form-item': WrapperStub,
        't-empty': { template: '<div>{{ title }}</div>', props: ['title'] },
        't-button': { template: '<button @click="$emit(\'click\')"><slot /></button>' },
        't-input': {
          template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
          props: ['modelValue'],
        },
        't-select': { template: '<select />' },
      },
    },
  });
}

describe('ApplicationTemplateListIndex', () => {
  beforeEach(() => {
    mocks.getApplicationManagedTemplates.mockReset();
    mocks.deleteApplicationTemplate.mockReset();
    mocks.postApplicationTemplateArchive.mockReset();
    mocks.postApplicationTemplateClone.mockReset();
    mocks.postApplicationTemplatePublish.mockReset();
    mocks.postApplicationTemplateWithdraw.mockReset();
    mocks.push.mockReset();
    responsiveVariantMocks.density = 'compact';
  });

  it('loads the management catalog instead of the published creator catalog', async () => {
    mocks.getApplicationManagedTemplates.mockResolvedValue({
      items: [
        {
          template_id: 'tpl_1',
          display_name: 'Draft',
          description: '',
          deployment_adapter_kind: 'compose',
          updated_at: '2026-07-18T00:00:00Z',
          version: {
            template_version_id: 'tplv_1',
            version_number: 1,
            status: 'draft',
            definition_schema_version: 1,
            definition: {},
          },
        },
      ],
    });

    mountPage();
    await flushPromises();

    expect(mocks.getApplicationManagedTemplates).toHaveBeenCalledTimes(1);
  });

  it('renders cards without a table below the spacious content threshold', async () => {
    mocks.getApplicationManagedTemplates.mockResolvedValue({
      items: [
        {
          template_id: 'tpl_1',
          display_name: 'Uptime Kuma',
          description: '',
          deployment_adapter_kind: 'compose',
          updated_at: '2026-07-18T00:00:00Z',
          version: {
            template_version_id: 'tplv_1',
            version_number: 1,
            status: 'published',
            definition_schema_version: 1,
            definition: {},
          },
        },
      ],
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.get('[data-testid="template-card-items"]').text()).toContain('Uptime Kuma');
    expect(wrapper.find('.application-template-list__table').exists()).toBe(false);
  });

  it('keeps the desktop table at the spacious content threshold', async () => {
    responsiveVariantMocks.density = 'spacious';
    mocks.getApplicationManagedTemplates.mockResolvedValue({ items: [] });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('.application-template-list__table').exists()).toBe(true);
    expect(wrapper.find('[data-testid="template-card-items"]').exists()).toBe(false);
  });

  it('shows card-shaped skeletons while the management catalog is loading', async () => {
    mocks.getApplicationManagedTemplates.mockReturnValue(new Promise(() => undefined));

    const wrapper = mountPage();
    await nextTick();

    expect(wrapper.find('[data-testid="template-card-skeletons"]').exists()).toBe(true);
  });

  it('shows draft actions without the removed legacy import entry', async () => {
    mocks.getApplicationManagedTemplates.mockResolvedValue({
      items: [
        {
          template_id: 'tpl_1',
          display_name: 'Draft',
          description: '',
          deployment_adapter_kind: 'compose',
          updated_at: '2026-07-18T00:00:00Z',
          version: {
            template_version_id: 'tplv_1',
            version_number: 1,
            status: 'draft',
            definition_schema_version: 1,
            definition: {},
          },
        },
      ],
    });

    const wrapper = mountPage();
    await flushPromises();

    const actions = wrapper.findComponent(TableActionMenu).props('actions') as Array<{ value: string }>;
    expect(actions.map((action) => action.value)).toEqual(['detail', 'publish', 'clone', 'archive', 'delete']);
    expect(wrapper.text()).not.toContain('project.templates.importLegacy');
  });

  it('opens the template creation workflow without creating a draft', async () => {
    mocks.getApplicationManagedTemplates.mockResolvedValue({ items: [] });

    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[aria-label="project.templates.create"]').trigger('click');

    expect(mocks.push).toHaveBeenCalledWith({ name: 'ApplicationTemplateCreateWizardIndex' });
  });

  it('shows withdraw for published templates and keeps clone/delete for archived templates', async () => {
    mocks.getApplicationManagedTemplates.mockResolvedValue({
      items: [
        {
          template_id: 'tpl_published',
          display_name: 'Published',
          description: '',
          deployment_adapter_kind: 'compose',
          updated_at: '2026-07-18T00:00:00Z',
          version: {
            template_version_id: 'tplv_published',
            version_number: 1,
            status: 'published',
            definition_schema_version: 1,
            definition: {},
          },
        },
      ],
    });

    const publishedWrapper = mountPage();
    await flushPromises();

    const publishedActions = publishedWrapper.findComponent(TableActionMenu).props('actions') as Array<{ value: string }>;
    expect(publishedActions.map((action) => action.value)).toEqual(['detail', 'clone', 'withdraw', 'archive', 'delete']);

    mocks.getApplicationManagedTemplates.mockResolvedValue({
      items: [
        {
          template_id: 'tpl_archived',
          display_name: 'Archived',
          description: '',
          archived_at: '2026-07-17T00:00:00Z',
          deployment_adapter_kind: 'compose',
          updated_at: '2026-07-18T00:00:00Z',
          version: {
            template_version_id: 'tplv_archived',
            version_number: 1,
            status: 'published',
            definition_schema_version: 1,
            definition: {},
          },
        },
      ],
    });

    const archivedWrapper = mountPage();
    await flushPromises();

    const archivedActions = archivedWrapper.findComponent(TableActionMenu).props('actions') as Array<{
      value: string;
    }>;
    expect(archivedActions.map((action) => action.value)).toEqual(['detail', 'clone', 'delete']);
  });
});
