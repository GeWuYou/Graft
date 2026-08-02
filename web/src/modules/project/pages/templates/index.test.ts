import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { computed } from 'vue';

import ApplicationTemplateListIndex from './index.vue';

const mocks = vi.hoisted(() => ({
  getApplicationManagedTemplates: vi.fn(), getApplicationTemplateSavedViews: vi.fn(),
  postApplicationTemplateSavedView: vi.fn(), putApplicationTemplateSavedView: vi.fn(), deleteApplicationTemplateSavedView: vi.fn(),
  push: vi.fn(), replace: vi.fn(),
}));
const density = vi.hoisted(() => ({ value: 'compact' }));

vi.mock('../../api/project', () => ({
  getApplicationManagedTemplates: mocks.getApplicationManagedTemplates, getApplicationTemplateSavedViews: mocks.getApplicationTemplateSavedViews,
  postApplicationTemplateSavedView: mocks.postApplicationTemplateSavedView, putApplicationTemplateSavedView: mocks.putApplicationTemplateSavedView,
  deleteApplicationTemplateSavedView: mocks.deleteApplicationTemplateSavedView, deleteApplicationTemplate: vi.fn(), postApplicationTemplateArchive: vi.fn(),
  postApplicationTemplateClone: vi.fn(), postApplicationTemplatePublish: vi.fn(), postApplicationTemplateWithdraw: vi.fn(),
}));
vi.mock('vue-router', () => ({ useRoute: () => ({ query: {} }), useRouter: () => ({ push: mocks.push, replace: mocks.replace }), isNavigationFailure: () => false, NavigationFailureType: { duplicated: 16 } }));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: { value: 'zh-CN' }, t: (key: string) => key }) }));
vi.mock('@/shared/composables/useViewportResponsiveVariant', () => ({ useViewportResponsiveVariant: () => computed(() => ({ density: density.value })) }));
vi.mock('tdesign-vue-next/es/message', () => ({ MessagePlugin: { error: vi.fn(), success: vi.fn() } }));

const Shell = { template: '<div><slot /><slot name="actions" /><slot name="filters" /><slot name="table" /><slot name="detail" /><slot name="feedback-extra" /></div>' };
const template = { template_id: 'tpl_1', display_name: 'Draft', description: '', deployment_adapter_kind: 'compose', updated_at: '2026-07-18T00:00:00Z', version: { template_version_id: 'tplv_1', version_number: 1, status: 'draft', definition_schema_version: 1, definition: {} } };

function mountPage() {
  return mount(ApplicationTemplateListIndex, { global: { stubs: { 'advanced-query-list-page': Shell, 'advanced-query-filter-builder': Shell, 'advanced-query-paged-table': Shell, 'advanced-query-column-drawer': Shell, 'saved-query-view-control': Shell, 'management-statistics-bar': Shell, 't-button': { template: '<button @click="$emit(\'click\')"><slot /></button>' }, 't-tooltip': Shell, 't-dialog': Shell, 't-form': Shell, 't-form-item': Shell, 't-input': true, 't-tag': Shell, 'table-action-menu': Shell } } });
}

describe('ApplicationTemplateListIndex', () => {
  beforeEach(() => { mocks.getApplicationManagedTemplates.mockResolvedValue({ items: [template], total: 1, limit: 20, offset: 0 }); mocks.getApplicationTemplateSavedViews.mockResolvedValue([]); mocks.push.mockReset(); mocks.replace.mockReset(); density.value = 'compact'; });

  it('loads the server-filtered management catalog with pagination', async () => {
    mountPage(); await flushPromises();
    expect(mocks.getApplicationManagedTemplates).toHaveBeenCalledWith(expect.objectContaining({ limit: 20, offset: 0, sort: 'updated_at:desc' }));
  });

  it('loads private saved filters and keeps compact card presentation', async () => {
    const wrapper = mountPage(); await flushPromises();
    expect(mocks.getApplicationTemplateSavedViews).toHaveBeenCalled();
    expect(wrapper.text()).toContain('Draft');
  });

  it('opens the template creation workflow', async () => {
    const wrapper = mountPage(); await flushPromises();
    await wrapper.get('[aria-label="project.templates.create"]').trigger('click');
    expect(mocks.push).toHaveBeenCalledWith({ name: 'ApplicationTemplateCreateWizardIndex' });
  });
});
