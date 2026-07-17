import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';

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
}));

vi.mock('../../api/project', () => mocks);
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { templateId: 'tpl_1' } }),
  useRouter: () => ({ push: mocks.push }),
}));
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
  default: { template: '<div />' },
}));

const WrapperStub = { template: '<div><slot /><slot name="actions" /><slot name="meta" /></div>' };

function mountPage() {
  return mount(ApplicationTemplateDetailIndex, {
    global: {
      stubs: {
        'management-page-content': WrapperStub,
        'management-page-header': WrapperStub,
        't-space': WrapperStub,
        't-loading': WrapperStub,
        't-alert': WrapperStub,
        't-card': WrapperStub,
        't-form': WrapperStub,
        't-form-item': WrapperStub,
        't-tag': WrapperStub,
        't-dialog': WrapperStub,
        't-button': { template: '<button @click="$emit(\'click\')"><slot /></button>' },
        't-input': { template: '<input :value="modelValue ?? value" />', props: ['modelValue', 'value'] },
        't-textarea': { template: '<textarea :value="modelValue" />', props: ['modelValue'] },
      },
    },
  });
}

describe('ApplicationTemplateDetailIndex', () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset());
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
  });
});
