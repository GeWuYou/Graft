import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { computed, defineComponent, h, reactive } from 'vue';

import ProjectConfigurationWorkspaceIndex from './index.vue';

const mocks = vi.hoisted(() => ({
  confirm: vi.fn(),
  copyText: vi.fn(),
  error: vi.fn(),
  getProject: vi.fn(),
  getProjectConfiguration: vi.fn(),
  getProjectConfigurationFile: vi.fn(),
  getProjectConfigurationPreview: vi.fn(),
  postProjectConfigurationDiff: vi.fn(),
  postProjectConfigurationValidate: vi.fn(),
  postProjectDeploy: vi.fn(),
  success: vi.fn(),
  t: vi.fn((key: string) => key),
}));

const routeState = reactive({
  params: { id: '1' },
  query: { name: 'sub2api' },
});

vi.mock('../../api/project', () => ({
  getProject: mocks.getProject,
  getProjectConfiguration: mocks.getProjectConfiguration,
  getProjectConfigurationFile: mocks.getProjectConfigurationFile,
  getProjectConfigurationPreview: mocks.getProjectConfigurationPreview,
  postProjectConfigurationDiff: mocks.postProjectConfigurationDiff,
  postProjectConfigurationValidate: mocks.postProjectConfigurationValidate,
  postProjectDeploy: mocks.postProjectDeploy,
}));

vi.mock('../../shared/page-context', () => ({
  useProjectPageContext: () => ({
    locale: computed(() => 'en-US'),
    t: mocks.t,
  }),
}));

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

vi.mock('@/shared/observability/copy', () => ({
  copyText: mocks.copyText,
}));

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router');
  return {
    ...actual,
    useRoute: () => routeState,
  };
});

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: {
    error: (...args: unknown[]) => mocks.error(...args),
    success: (...args: unknown[]) => mocks.success(...args),
    warning: vi.fn(),
  },
}));

vi.mock('tdesign-vue-next/es/dialog', () => ({
  DialogPlugin: {
    confirm: (...args: unknown[]) => mocks.confirm(...args),
  },
}));

vi.mock('@/shared/components/management', () => ({
  formatCompactDateTime: (value?: string | null) => value || '-',
  ManagementPageContent: defineComponent({
    name: 'ManagementPageContentStub',
    setup(_props, { slots }) {
      return () => h('section', { class: 'management-page-content-stub' }, slots.default?.());
    },
  }),
  ManagementPageHeader: defineComponent({
    name: 'ManagementPageHeaderStub',
    props: {
      title: { type: String, default: '' },
      description: { type: String, default: '' },
    },
    setup(props, { slots }) {
      return () =>
        h('header', { class: 'management-page-header-stub' }, [
          h('h1', props.title),
          h('p', props.description),
          h('div', slots.meta?.()),
          h('div', slots.actions?.()),
        ]);
    },
  }),
}));

vi.mock('@/shared/components/viewer/ContentViewerFrame.vue', () => ({
  default: defineComponent({
    name: 'ContentViewerFrameStub',
    setup(_props, { slots }) {
      return () =>
        h('section', { class: 'content-viewer-frame-stub' }, [
          h('div', { class: 'content-viewer-header' }, slots.header?.()),
          h('div', { class: 'content-viewer-header-actions' }, slots['header-actions']?.()),
          h('div', { class: 'content-viewer-toolbar' }, slots.toolbar?.()),
          h('div', { class: 'content-viewer-body' }, slots.default?.()),
        ]);
    },
  }),
}));

vi.mock('../../components/ProjectMonacoSurface.vue', () => ({
  default: defineComponent({
    name: 'ProjectMonacoSurfaceStub',
    props: {
      modelValue: { type: String, default: '' },
      testId: { type: String, default: 'monaco-stub' },
      readOnly: { type: Boolean, default: false },
    },
    emits: ['update:modelValue'],
    setup(props, { emit }) {
      return () =>
        h('textarea', {
          'data-testid': props.testId,
          disabled: props.readOnly,
          value: props.modelValue,
          onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLTextAreaElement).value),
        });
    },
  }),
}));

vi.mock('../../components/ProjectMonacoDiffSurface.vue', () => ({
  default: defineComponent({
    name: 'ProjectMonacoDiffSurfaceStub',
    props: {
      modifiedValue: { type: String, default: '' },
      originalValue: { type: String, default: '' },
      testId: { type: String, default: 'diff-stub' },
    },
    setup(props) {
      return () =>
        h('div', { 'data-testid': props.testId }, [
          h('pre', { class: 'original' }, props.originalValue),
          h('pre', { class: 'modified' }, props.modifiedValue),
        ]);
    },
  }),
}));

function createTStub(name: string) {
  return defineComponent({
    name,
    props: {
      description: { type: String, default: '' },
      header: { type: String, default: '' },
      label: { type: String, default: '' },
      loading: { type: Boolean, default: false },
      message: { type: String, default: '' },
      theme: { type: String, default: '' },
      title: { type: String, default: '' },
      value: { type: [String, Number, Boolean], default: '' },
      visible: { type: Boolean, default: false },
    },
    emits: ['click', 'change', 'update:value', 'update:visible'],
    setup(props, { emit, slots }) {
      return () =>
        h(
          'div',
          {
            'data-stub': name,
            'data-header': props.header,
            'data-message': props.message,
            'data-visible': props.visible,
            onClick: () => emit('click'),
          },
          [
            h('div', slots.header?.()),
            h('div', slots.meta?.()),
            h('div', slots.actions?.()),
            h('div', slots.default?.()),
          ],
        );
    },
  });
}

const TButtonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_props, { emit, slots }) {
    return () =>
      h(
        'button',
        {
          onClick: () => emit('click'),
        },
        slots.default?.(),
      );
  },
});

const TTabsStub = defineComponent({
  name: 'TTabsStub',
  props: {
    value: { type: String, default: '' },
  },
  emits: ['change', 'update:value'],
  setup(_props, { slots }) {
    return () => h('div', { class: 't-tabs-stub' }, slots.default?.());
  },
});

const TTabPanelStub = defineComponent({
  name: 'TTabPanelStub',
  props: {
    label: { type: String, default: '' },
    value: { type: String, default: '' },
  },
  setup(props, { slots }) {
    return () => h('section', { 'data-tab-panel': props.value }, [h('h3', props.label), slots.default?.()]);
  },
});

describe('ProjectConfigurationWorkspaceIndex', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getProject.mockResolvedValue({
      canonical_project_name: 'sub2api',
      display_name: 'sub2api',
      id: 1,
      ownership_mode: 'managed-root-dedicated',
      runtime_status: 'running',
    });
    mocks.getProjectConfiguration.mockResolvedValue({
      compose_files: [
        {
          absolute_path: '/srv/sub2api/docker-compose.yml',
          display_path: '/srv/sub2api/docker-compose.yml',
          exists_on_last_refresh: true,
          id: 11,
          kind: 'compose',
          last_observed_hash: 'compose-hash',
          order_index: 0,
          role: 'primary',
        },
      ],
      diagnostics_summary: [],
      drift_status: 'clean',
      env_files: [
        {
          absolute_path: '/srv/sub2api/.env',
          display_path: '/srv/sub2api/.env',
          exists_on_last_refresh: true,
          id: 12,
          kind: 'env',
          last_observed_hash: 'env-hash',
          order_index: 0,
          role: 'env',
        },
      ],
      last_refresh_at: '2026-07-03T13:12:38Z',
      last_refresh_status: 'success',
      ownership_mode: 'managed-root-dedicated',
      project_id: 1,
    });
    mocks.getProjectConfigurationFile.mockImplementation(async (_id: number, fileId: number) => ({
      content: fileId === 11 ? 'services:\n  api:\n    image: app\n' : 'APP_ENV=prod\n',
      download_name: fileId === 11 ? 'docker-compose.yml' : '.env',
      encoding: 'utf-8',
      file_id: fileId,
      kind: fileId === 11 ? 'compose' : 'env',
      path: fileId === 11 ? '/srv/sub2api/docker-compose.yml' : '/srv/sub2api/.env',
      read_only: true,
    }));
    mocks.getProjectConfigurationPreview.mockResolvedValue({
      canonical_project_name: 'sub2api',
      config_hash: 'preview-hash',
      normalized_compose_yaml: 'services:\n  api:\n    image: app\n',
      project_id: 1,
      refreshed_at: '2026-07-03T13:12:38Z',
    });
    mocks.postProjectConfigurationDiff.mockResolvedValue({
      canonical_project_name: 'sub2api',
      current_config_hash: 'current-hash',
      files: [
        {
          changed: true,
          current_content: 'services:\n  api:\n    image: old\n',
          current_hash: 'old-hash',
          kind: 'compose',
          path: '/srv/sub2api/docker-compose.yml',
          proposed_content: 'services:\n  api:\n    image: app\n',
          proposed_hash: 'new-hash',
        },
      ],
      has_changes: true,
      ownership_mode: 'managed-root-dedicated',
      project_id: 1,
      proposed_config_hash: 'proposed-hash',
      warnings: [],
    });
    mocks.postProjectConfigurationValidate.mockResolvedValue({
      canonical_project_name: 'sub2api',
      declared_service_names: ['api'],
      normalized_compose_yaml: 'services:\n  api:\n    image: app\n',
      ownership_mode: 'managed-root-dedicated',
      project_id: 1,
      proposed_config_hash: 'validated-hash',
      warnings: [],
    });
    mocks.copyText.mockResolvedValue(true);
    mocks.confirm.mockImplementation(({ onConfirm }: { onConfirm: () => Promise<void> }) => {
      void onConfirm();
      return { destroy: vi.fn() };
    });
  });

  it('loads current files into the draft editors', async () => {
    const wrapper = mountWorkspace();

    await flushPromises();

    expect(mocks.getProject).toHaveBeenCalledWith(1);
    expect(mocks.getProjectConfiguration).toHaveBeenCalledWith(1);
    expect(mocks.getProjectConfigurationFile).toHaveBeenCalledWith(1, 11);
    expect(mocks.getProjectConfigurationFile).toHaveBeenCalledWith(1, 12);
    expect((wrapper.get('[data-testid="compose-monaco-editor"]').element as HTMLTextAreaElement).value).toBe(
      'services:\n  api:\n    image: app\n',
    );
  });

  it('runs diff with normalized draft content', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="compose-monaco-editor"]').setValue('services:\n  api:\n    image: app   \n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('project.detail.configuration.runDiff'))
      ?.trigger('click');
    await flushPromises();

    expect(mocks.postProjectConfigurationDiff).toHaveBeenCalledWith(1, {
      compose_file_content: 'services:\n  api:\n    image: app\n',
      env_file_content: 'APP_ENV=prod\n',
    });
    expect(wrapper.get('[data-testid="configuration-diff-viewer"]').text()).toContain('image: old');
    expect(wrapper.get('[data-testid="configuration-diff-viewer"]').text()).toContain('image: app');
  });

  it('loads snapshot lazily when the snapshot action is used', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    const snapshotButton = wrapper.findAll('button').find((button) => button.text().includes('View Current Snapshot'));
    expect(snapshotButton).toBeTruthy();
    await snapshotButton!.trigger('click');
    await flushPromises();

    expect(mocks.getProjectConfigurationPreview).toHaveBeenCalledWith(1);
    expect((wrapper.get('[data-testid="snapshot-monaco-viewer"]').element as HTMLTextAreaElement).value).toBe(
      'services:\n  api:\n    image: app\n',
    );
  });
});

function mountWorkspace() {
  return mount(ProjectConfigurationWorkspaceIndex, {
    global: {
      stubs: {
        TAlert: createTStub('TAlert'),
        TCard: createTStub('TCard'),
        TDescriptions: createTStub('TDescriptions'),
        TDescriptionsItem: createTStub('TDescriptionsItem'),
        TDrawer: createTStub('TDrawer'),
        TEmpty: createTStub('TEmpty'),
        TLoading: createTStub('TLoading'),
        TSpace: createTStub('TSpace'),
        TTag: createTStub('TTag'),
        TTabs: TTabsStub,
        TTabPanel: TTabPanelStub,
        TButton: TButtonStub,
      },
    },
  });
}
