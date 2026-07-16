import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { ApplicationLifecycleConfigurationDraft } from '../../types/project';
import ApplicationCreateIndex from './index.vue';

const routeQuery = vi.hoisted(() => ({
  runtime_target_id: '7' as string | string[],
  application_name: undefined as string | undefined,
}));
const mocks = vi.hoisted(() => ({
  getApplicationTemplates: vi.fn(),
  postApplicationApplicationNameAvailability: vi.fn(),
  postApplicationCreate: vi.fn(),
  push: vi.fn(),
}));
const lifecycleStepDraft = vi.hoisted(() => ({ value: null as ApplicationLifecycleConfigurationDraft | null }));

vi.mock('../../api/project', () => ({
  getApplicationTemplates: mocks.getApplicationTemplates,
  postApplicationApplicationNameAvailability: mocks.postApplicationApplicationNameAvailability,
  postApplicationCreate: mocks.postApplicationCreate,
}));

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: routeQuery }),
}));

vi.mock('../../shared/page-context', () => ({
  useApplicationPageContext: () => ({
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
    name: 'ApplicationCreateWorkspaceEditorStub',
    setup() {
      return () => h('div');
    },
  }),
}));

vi.mock('../../components/ProjectLifecycleConfigurationReview.vue', () => ({
  default: defineComponent({
    name: 'ApplicationLifecycleConfigurationReviewStub',
    setup() {
      return () => h('div');
    },
  }),
}));

vi.mock('../../components/ProjectLifecycleConfigurationStep.vue', () => ({
  default: defineComponent({
    name: 'ApplicationLifecycleConfigurationStepStub',
    props: {
      continueLabel: { type: String, default: '' },
      draft: { type: Object, required: true },
    },
    emits: ['back', 'continue'],
    setup(props, { emit }) {
      lifecycleStepDraft.value = props.draft as ApplicationLifecycleConfigurationDraft;
      return () =>
        h('div', [
          h('button', { onClick: () => emit('back') }, 'project.create.actions.back'),
          h('button', { onClick: () => emit('continue') }, props.continueLabel),
        ]);
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
  return mount(ApplicationCreateIndex, {
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

describe('ApplicationCreateIndex', () => {
  beforeEach(() => {
    mocks.postApplicationApplicationNameAvailability.mockClear();
    mocks.postApplicationCreate.mockClear();
    mocks.getApplicationTemplates.mockResolvedValue({ items: [] });
    routeQuery.runtime_target_id = '7';
    delete routeQuery.application_name;
    lifecycleStepDraft.value = null;
    mocks.postApplicationApplicationNameAvailability.mockResolvedValue({
      status: 'available',
      workspace_path: '/var/lib/graft/applications/demo-project',
      workspace_non_empty: false,
    });
    mocks.postApplicationCreate.mockResolvedValue({
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
      .find((button) => button.text().includes('project.create.actions.next'))
      ?.trigger('click');
    await flushPromises();
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('project.create.actions.next'))
      ?.trigger('click');
    await flushPromises();
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('project.create.actions.next'))
      ?.trigger('click');
    await flushPromises();
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('project.create.actions.create'))
      ?.trigger('click');
    await flushPromises();

    expect(mocks.postApplicationCreate).toHaveBeenCalledTimes(1);
    const request = mocks.postApplicationCreate.mock.calls[0]?.[0];
    expect(request).toEqual(expect.objectContaining({ runtime_target_id: 7 }));
    expect(request.lifecycle_configuration).toEqual(
      expect.objectContaining({
        strategy_kind: 'standard',
        down_before_redeploy: true,
        remove_orphans: true,
        additional_args: [],
      }),
    );
    expect(request.workspace_entries).toEqual(
      expect.arrayContaining([
        { path: '.env', node_type: 'file', content: '' },
        { path: 'compose.yaml', node_type: 'file', content: JSON.stringify({ services: {} }) },
      ]),
    );
  });

  it('does not create when the runtime target id is not a safe positive integer', async () => {
    routeQuery.runtime_target_id = '1.5';
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('form').trigger('submit');
    await flushPromises();

    expect(mocks.postApplicationCreate).not.toHaveBeenCalled();
  });

  it('keeps lifecycle edits when returning from the lifecycle step', async () => {
    routeQuery.application_name = 'demo-project';
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('form').trigger('submit');
    await flushPromises();
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('project.create.actions.next'))
      ?.trigger('click');
    await flushPromises();
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('project.create.actions.next'))
      ?.trigger('click');
    await flushPromises();

    const draft = lifecycleStepDraft.value;
    expect(draft).not.toBeNull();
    if (!draft) return;
    draft.profiles = ['production'];
    draft.additional_args = "--label 'release channel'";
    draft.wait_after_up = true;

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('project.create.actions.back'))
      ?.trigger('click');
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('project.create.actions.next'))
      ?.trigger('click');
    await flushPromises();

    expect(lifecycleStepDraft.value).toMatchObject({
      compose_files: ['compose.yaml'],
      compose_project_name: 'demo-project',
      profiles: ['production'],
      additional_args: "--label 'release channel'",
      wait_after_up: true,
    });
  });

  it('returns to the source page with the current query', async () => {
    mocks.push.mockClear();
    const wrapper = mountPage();

    await wrapper.get('[data-testid="project-create-back-source"]').trigger('click');

    expect(mocks.push).toHaveBeenCalledWith({
      name: 'ApplicationCreateSourceIndex',
      query: { runtime_target_id: '7' },
    });
  });
});
