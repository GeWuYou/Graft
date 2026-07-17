import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { nextTick, reactive } from 'vue';

import ApplicationTemplateCatalogDetailIndex from './catalog-detail.vue';

const mocks = vi.hoisted(() => ({
  getPublishedApplicationTemplate: vi.fn(),
  push: vi.fn(),
}));

const route = reactive({
  params: { templateId: 'template-old' },
  query: { deployment: 'compose', runtime_target_id: '7' },
});

vi.mock('vue-router', () => ({
  useRoute: () => route,
  useRouter: () => ({ push: mocks.push }),
}));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock('../../api/project', () => ({ getPublishedApplicationTemplate: mocks.getPublishedApplicationTemplate }));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

describe('ApplicationTemplateCatalogDetailIndex', () => {
  it('keeps the newest published template when route requests overlap', async () => {
    let resolveOld!: (value: unknown) => void;
    let resolveNew!: (value: unknown) => void;
    mocks.getPublishedApplicationTemplate
      .mockReset()
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveOld = resolve;
          }),
      )
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveNew = resolve;
          }),
      );

    const wrapper = mount(ApplicationTemplateCatalogDetailIndex, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': {
            props: ['description'],
            template: '<header>{{ description }}<slot name="actions" /></header>',
          },
          't-space': { template: '<div><slot /></div>' },
          't-button': { template: '<button><slot /></button>' },
          't-tag': { template: '<span><slot /></span>' },
          't-loading': { template: '<div><slot /></div>' },
          't-alert': true,
          't-card': { template: '<section><slot name="header" /><slot /></section>' },
          't-table': { template: '<div><slot name="empty" /></div>' },
          't-empty': { template: '<div><slot /></div>' },
          'markdown-viewer': { props: ['source'], template: '<p>{{ source }}</p>' },
        },
      },
    });

    await nextTick();
    route.params.templateId = 'template-new';
    await nextTick();
    resolveNew(template('template-new', 'Newest template'));
    await flushPromises();
    resolveOld(template('template-old', 'Stale template'));
    await flushPromises();

    expect(wrapper.text()).toContain('Newest template');
    expect(wrapper.text()).not.toContain('Stale template');
  });
});

function template(templateId: string, description: string) {
  return {
    template_id: templateId,
    display_name: templateId,
    description,
    category: 'other',
    deployment_adapter_kind: 'compose',
    updated_at: '2026-07-18T00:00:00Z',
    version: {
      template_version_id: `${templateId}-version`,
      version_number: 1,
      definition: {
        catalog_documentation: { readme_markdown: '', variables: [] },
        workspace_entries: [],
      },
    },
  };
}
