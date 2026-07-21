import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { nextTick, reactive } from 'vue';

import ApplicationSourceCreate from './source-create.vue';

const mocks = vi.hoisted(() => ({
  push: vi.fn(),
  replace: vi.fn(),
  getApplicationTemplateCatalog: vi.fn().mockResolvedValue({ items: [], has_more: false }),
  navigateToApplicationCreateSource: vi.fn(),
}));

const route = reactive({ query: { deployment: 'compose', runtime_target_id: '7' } as Record<string, string> });

vi.mock('vue-router', () => ({
  useRoute: () => route,
  useRouter: () => ({ push: mocks.push, replace: mocks.replace }),
}));
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ locale: { value: 'en-US' }, t: (key: string) => key }),
}));
vi.mock('tdesign-vue-next', () => ({ MessagePlugin: { error: vi.fn(), success: vi.fn(), warning: vi.fn() } }));
vi.mock('../../api/project', () => ({ getApplicationTemplateCatalog: mocks.getApplicationTemplateCatalog }));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
vi.mock('../../shared/navigation', () => ({
  navigateToApplicationCreateSource: mocks.navigateToApplicationCreateSource,
  refreshApplicationCreatePage: mocks.replace,
}));

describe('ApplicationSourceCreate', () => {
  beforeEach(() => {
    route.query = { deployment: 'compose', runtime_target_id: '7' };
    mocks.getApplicationTemplateCatalog.mockReset();
    mocks.getApplicationTemplateCatalog.mockResolvedValue({ items: [], has_more: false });
    mocks.navigateToApplicationCreateSource.mockReset();
    mocks.push.mockReset();
    mocks.replace.mockReset();
  });

  it('returns to source selection with deployment and runtime target context', async () => {
    const wrapper = mount(ApplicationSourceCreate, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': { template: '<header><slot name="actions" /></header>' },
          't-button': { inheritAttrs: false, template: '<button v-bind="$attrs"><slot /></button>' },
          't-space': { template: '<div><slot /></div>' },
          't-card': { template: '<section><slot name="header" /><slot /><slot name="footer" /></section>' },
          't-alert': true,
          't-empty': true,
          't-input': true,
          't-select': true,
          't-loading': { template: '<div><slot /></div>' },
          't-pagination': true,
          't-tag': true,
          't-descriptions': { template: '<div><slot /></div>' },
          't-descriptions-item': { template: '<div><slot /></div>' },
        },
      },
    });
    await flushPromises();

    await wrapper.get('button').trigger('click');

    expect(mocks.navigateToApplicationCreateSource).toHaveBeenCalledWith(expect.anything(), {
      deployment: 'compose',
      runtime_target_id: '7',
    });
  });

  it('keeps the newest catalog response when filter requests overlap', async () => {
    let resolveFirst!: (value: unknown) => void;
    let resolveSecond!: (value: unknown) => void;
    const firstRequest = new Promise((resolve) => {
      resolveFirst = resolve;
    });
    const secondRequest = new Promise((resolve) => {
      resolveSecond = resolve;
    });
    let requestCount = 0;
    mocks.getApplicationTemplateCatalog.mockReset().mockImplementation(() => {
      requestCount += 1;
      return requestCount === 1 ? firstRequest : secondRequest;
    });

    const wrapper = mount(ApplicationSourceCreate, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': { template: '<header><slot name="actions" /></header>' },
          't-button': { inheritAttrs: false, template: '<button v-bind="$attrs"><slot /></button>' },
          't-space': { template: '<div><slot /></div>' },
          't-card': { template: '<section><slot name="header" /><slot /><slot name="footer" /></section>' },
          't-alert': true,
          't-empty': true,
          't-input': true,
          't-select': true,
          't-loading': { template: '<div><slot /></div>' },
          't-pagination': true,
          't-tag': true,
          't-descriptions': { template: '<div><slot /></div>' },
          't-descriptions-item': { template: '<div><slot /></div>' },
        },
      },
    });

    await nextTick();
    expect(mocks.getApplicationTemplateCatalog).toHaveBeenCalledTimes(1);
    route.query = { deployment: 'compose', runtime_target_id: '7', q: 'new' };
    await nextTick();
    expect(mocks.getApplicationTemplateCatalog).toHaveBeenCalledTimes(3);
    resolveSecond({
      items: [
        {
          template_id: 'new-template',
          display_name: 'Newest template',
          description: '',
          category: 'other',
          deployment_adapter_kind: 'compose',
          updated_at: '2026-07-18T00:00:00Z',
          version: { template_version_id: 'version-2', version_number: 2 },
        },
      ],
      has_more: false,
    });
    await flushPromises();
    resolveFirst({
      items: [
        {
          template_id: 'old-template',
          display_name: 'Stale template',
          description: '',
          category: 'other',
          deployment_adapter_kind: 'compose',
          updated_at: '2026-07-18T00:00:00Z',
          version: { template_version_id: 'version-1', version_number: 1 },
        },
      ],
      has_more: false,
    });
    await flushPromises();

    expect(wrapper.text()).toContain('Newest template');
    expect(wrapper.text()).not.toContain('Stale template');
  });

  it('offers a recovery action when filters return no templates', async () => {
    mocks.getApplicationTemplateCatalog.mockResolvedValue({ items: [], has_more: false });
    route.query = {
      deployment: 'compose',
      runtime_target_id: '7',
      q: 'missing',
      category: 'cache',
      sort: 'name_asc',
      page: '2',
    };
    const wrapper = mount(ApplicationSourceCreate, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': { template: '<header><slot name="actions" /></header>' },
          't-button': { inheritAttrs: false, template: '<button v-bind="$attrs"><slot /></button>' },
          't-space': { template: '<div><slot /></div>' },
          't-card': { template: '<section><slot /></section>' },
          't-alert': true,
          't-empty': { template: '<div><slot name="action" /></div>' },
          't-input': true,
          't-select': true,
          't-loading': { template: '<div><slot /></div>' },
          't-pagination': true,
          't-tag': true,
          't-descriptions': { template: '<div><slot /></div>' },
          't-descriptions-item': { template: '<div><slot /></div>' },
        },
      },
    });
    await flushPromises();

    const clearButton = wrapper
      .findAll('button')
      .find((button) => button.text() === 'project.create.actions.clearFilters');
    expect(clearButton).toBeTruthy();
    await clearButton!.trigger('click');
    expect(mocks.replace).toHaveBeenCalledWith({
      query: expect.objectContaining({
        q: undefined,
        category: undefined,
        sort: undefined,
        page: undefined,
      }),
    });
  });
});
