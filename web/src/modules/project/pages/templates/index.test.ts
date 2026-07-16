import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import ApplicationTemplateListIndex from './index.vue';

const mocks = vi.hoisted(() => ({
  getApplicationManagedTemplates: vi.fn(),
  postApplicationTemplate: vi.fn(),
  postApplicationTemplateLegacyImport: vi.fn(),
  push: vi.fn(),
}));

vi.mock('../../api/project', () => ({
  getApplicationManagedTemplates: mocks.getApplicationManagedTemplates,
  postApplicationTemplate: mocks.postApplicationTemplate,
  postApplicationTemplateLegacyImport: mocks.postApplicationTemplateLegacyImport,
}));
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock('tdesign-vue-next/es/message', () => ({ MessagePlugin: { error: vi.fn() } }));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

const WrapperStub = { template: '<div><slot /><slot name="actions" /></div>' };

function mountPage() {
  return mount(ApplicationTemplateListIndex, {
    global: {
      stubs: {
        'management-page-content': WrapperStub,
        'management-page-header': WrapperStub,
        't-space': WrapperStub,
        't-card': WrapperStub,
        't-alert': WrapperStub,
        't-table': { template: '<div><slot /><slot name="empty" /></div>' },
        't-tag': WrapperStub,
        't-dialog': WrapperStub,
        't-form': WrapperStub,
        't-form-item': WrapperStub,
        't-empty': { template: '<div>{{ title }}</div>', props: ['title'] },
        't-button': { template: '<button @click="$emit(\'click\')"><slot /></button>' },
        't-input': { template: '<input />' },
        't-select': { template: '<select />' },
      },
    },
  });
}

describe('ApplicationTemplateListIndex', () => {
  beforeEach(() => {
    mocks.getApplicationManagedTemplates.mockReset();
    mocks.postApplicationTemplate.mockReset();
    mocks.push.mockReset();
  });

  it('loads the management catalog instead of the published creator catalog', async () => {
    mocks.getApplicationManagedTemplates.mockResolvedValue({
      items: [
        {
          template_id: 'tpl_1',
          display_name: 'Draft',
          description: '',
          deployment_adapter_kind: 'compose',
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
});
