import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import ProjectCreateIndex from './index.vue';

const routeQuery = vi.hoisted(() => ({ runtime_target_id: '7' as string | string[] }));
const mocks = vi.hoisted(() => ({
  postProjectCreate: vi.fn(),
  push: vi.fn(),
}));

vi.mock('../../api/project', () => ({
  postProjectCreate: mocks.postProjectCreate,
  postProjectDeploy: vi.fn(),
}));

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: routeQuery }),
}));

vi.mock('../../shared/page-context', () => ({
  useProjectPageContext: () => ({
    router: { push: mocks.push, resolve: vi.fn() },
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
    return () => h('div', [slots.default?.(), slots.actions?.()]);
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
    mocks.postProjectCreate.mockClear();
    routeQuery.runtime_target_id = '7';
    mocks.postProjectCreate.mockResolvedValue({
      application_id: 'app_42',
      display_name: 'demo-project',
    });
  });

  it('creates a project from the workspace draft', async () => {
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
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('project.create.actions.create'))
      ?.trigger('click');
    await flushPromises();

    expect(mocks.postProjectCreate).toHaveBeenCalledTimes(1);
    expect(mocks.postProjectCreate).toHaveBeenCalledWith(expect.objectContaining({ runtime_target_id: 7 }));
  });

  it('does not create when the runtime target id is not a safe positive integer', async () => {
    routeQuery.runtime_target_id = '1.5';
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('form').trigger('submit');
    await flushPromises();

    expect(mocks.postProjectCreate).not.toHaveBeenCalled();
  });

  it('returns to the source page with the current query', async () => {
    mocks.push.mockClear();
    const wrapper = mountPage();

    await wrapper.get('[data-testid="project-create-back-source"]').trigger('click');

    expect(mocks.push).toHaveBeenCalledWith({
      name: 'ProjectCreateSourceIndex',
      query: { runtime_target_id: '7' },
    });
  });
});
