import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import ProjectCreateIndex from './index.vue';

const mocks = vi.hoisted(() => ({
  getProjectManagedRoot: vi.fn(),
  postProjectCreateValidate: vi.fn(),
}));

vi.mock('../../api/project', () => ({
  getProjectManagedRoot: mocks.getProjectManagedRoot,
  postProjectCreate: vi.fn(),
  postProjectCreateValidate: mocks.postProjectCreateValidate,
  postProjectDeploy: vi.fn(),
}));

vi.mock('../../shared/page-context', () => ({
  useProjectPageContext: () => ({
    router: { push: vi.fn(), resolve: vi.fn() },
    tabsRouterStore: { appendTabRouterList: vi.fn() },
    t: (key: string) => key,
  }),
}));

vi.mock('@/store', () => ({
  usePermissionStore: () => ({ hasPermission: () => true }),
}));

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

vi.mock('../../components/ProjectCreateWorkspaceEditor.vue', () => ({
  default: defineComponent({
    name: 'ProjectCreateWorkspaceEditorStub',
    setup() {
      return () => h('div');
    },
  }),
}));

vi.mock('../../components/ProjectLifecycleConfigurationReview.vue', () => ({
  default: defineComponent({
    name: 'ProjectLifecycleConfigurationReviewStub',
    setup() {
      return () => h('div');
    },
  }),
}));

const WrapperStub = defineComponent({
  name: 'WrapperStub',
  setup(_props, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const TAlertStub = defineComponent({
  name: 'TAlertStub',
  props: { message: { type: String, default: '' } },
  setup(props) {
    return () => h('div', props.message);
  },
});

const TFormStub = defineComponent({
  name: 'TFormStub',
  emits: ['submit'],
  setup(_props, { emit, expose, slots }) {
    expose({ validate: vi.fn(async () => true) });
    return () =>
      h(
        'form',
        {
          onSubmit: (event: Event) => {
            event.preventDefault();
            emit('submit');
          },
        },
        slots.default?.(),
      );
  },
});

const TButtonStub = defineComponent({
  name: 'TButtonStub',
  props: { type: { type: String, default: 'button' } },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () => h('button', { type: props.type, onClick: () => emit('click') }, slots.default?.());
  },
});

function mountPage() {
  return mount(ProjectCreateIndex, {
    global: {
      stubs: {
        'management-page-content': WrapperStub,
        'management-page-header': WrapperStub,
        'project-create-workspace-editor': WrapperStub,
        'project-lifecycle-configuration-review': WrapperStub,
        't-alert': TAlertStub,
        't-button': TButtonStub,
        't-card': WrapperStub,
        't-checkbox': WrapperStub,
        't-descriptions': WrapperStub,
        't-descriptions-item': WrapperStub,
        't-form': TFormStub,
        't-form-item': WrapperStub,
        't-input': WrapperStub,
        't-space': WrapperStub,
        't-steps': WrapperStub,
        't-tag': WrapperStub,
      },
    },
  });
}

describe('ProjectCreateIndex', () => {
  beforeEach(() => {
    mocks.getProjectManagedRoot.mockResolvedValue({
      configured_root_directory: '/srv/graft',
      create_permission: 'project:create',
      status: 'ready',
      supports_managed_create: true,
    });
    mocks.postProjectCreateValidate.mockResolvedValue({
      warnings: ['compose file uses a deprecated attribute'],
      working_directory: '/srv/graft/demo-project',
    });
  });

  it('validates the lifecycle draft before entering review and renders its result', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('form').trigger('submit');
    await flushPromises();
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('project.create.actions.continue'))
      ?.trigger('click');
    await flushPromises();
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('project.create.actions.review'))
      ?.trigger('click');
    await flushPromises();

    expect(mocks.postProjectCreateValidate).toHaveBeenCalledTimes(1);
    expect(wrapper.text()).toContain('/srv/graft/demo-project');
    expect(wrapper.text()).toContain('compose file uses a deprecated attribute');
  });
});
