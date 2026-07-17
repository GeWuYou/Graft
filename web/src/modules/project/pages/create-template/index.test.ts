import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import ApplicationTemplateCreateWizardIndex from './index.vue';

const mocks = vi.hoisted(() => ({
  postApplicationTemplate: vi.fn(),
  push: vi.fn(),
}));

vi.mock('../../api/project', () => ({ postApplicationTemplate: mocks.postApplicationTemplate }));
vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: { error: vi.fn(), success: vi.fn(), warning: vi.fn() },
}));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
vi.mock('../../shared/page-context', () => ({
  useApplicationPageContext: () => ({ router: { push: mocks.push }, t: (key: string) => key }),
}));
vi.mock('../../components/ProjectCreateWorkspaceEditor.vue', () => ({
  default: { template: '<div data-testid="workspace-editor">workspace</div>', props: ['files'] },
}));
vi.mock('../../components/ProjectLifecycleConfigurationReview.vue', () => ({
  default: { template: '<div data-testid="lifecycle-review">lifecycle</div>', props: ['draft'] },
}));

const WrapperStub = { template: '<div><slot /><slot name="actions" /></div>' };
const FormStub = defineComponent({
  emits: ['submit'],
  setup(_, { emit, slots, expose }) {
    expose({ validate: async () => true });
    return () => h('form', { onSubmit: (event: Event) => (event.preventDefault(), emit('submit')) }, slots.default?.());
  },
});

function mountPage() {
  return mount(ApplicationTemplateCreateWizardIndex, {
    global: {
      stubs: {
        'management-page-content': WrapperStub,
        'management-page-header': WrapperStub,
        't-steps': { template: '<div />', props: ['current', 'options'] },
        't-card': WrapperStub,
        't-form': FormStub,
        't-form-item': WrapperStub,
        't-select': { template: '<select />', props: ['modelValue', 'options'] },
        't-input': {
          template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
          props: ['modelValue'],
        },
        't-button': {
          template: '<button :type="type" @click="$emit(\'click\')"><slot /></button>',
          props: ['disabled', 'loading', 'type'],
        },
        't-dialog': {
          template: '<div v-if="visible"><slot /><button @click="$emit(\'confirm\')">confirm</button></div>',
          props: ['visible'],
        },
      },
    },
  });
}

describe('ApplicationTemplateCreateWizardIndex', () => {
  beforeEach(() => {
    mocks.postApplicationTemplate.mockReset();
    mocks.push.mockReset();
  });

  it('keeps the template local until the lifecycle step completes', async () => {
    mocks.postApplicationTemplate.mockResolvedValue({ template_id: 'tpl_new' });
    const wrapper = mountPage();

    await wrapper.get('input').setValue('Nginx baseline');
    await wrapper.get('form').trigger('submit');
    await flushPromises();
    expect(wrapper.find('[data-testid="workspace-editor"]').exists()).toBe(true);
    expect(mocks.postApplicationTemplate).not.toHaveBeenCalled();

    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'project.templates.next')
      ?.trigger('click');
    await flushPromises();
    expect(wrapper.find('[data-testid="lifecycle-review"]').exists()).toBe(true);

    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'project.templates.createComplete')
      ?.trigger('click');
    await flushPromises();

    expect(mocks.postApplicationTemplate).toHaveBeenCalledWith(
      expect.objectContaining({
        display_name: 'Nginx baseline',
        deployment_adapter_kind: 'compose',
        definition: expect.objectContaining({
          compose_file_path: 'compose.yaml',
          workspace_entries: [
            { path: '.env', node_type: 'file', content: 'APP_IMAGE=nginx:alpine\nAPP_PORT=8080\n' },
            {
              path: 'compose.yaml',
              node_type: 'file',
              content: 'services:\n  app:\n    image: ${APP_IMAGE}\n    ports:\n      - "${APP_PORT}:80"\n',
            },
          ],
        }),
      }),
    );
    expect(mocks.push).toHaveBeenCalledWith({
      name: 'ApplicationTemplateDetailIndex',
      params: { templateId: 'tpl_new' },
    });
  });
});
