import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { computed, defineComponent, h, reactive } from 'vue';

import ProjectConfigurationWorkspaceIndex from './index.vue';

const mocks = vi.hoisted(() => ({
  error: vi.fn(),
  getProject: vi.fn(),
  getProjectConfiguration: vi.fn(),
  getProjectConfigurationPreview: vi.fn(),
  getProjectFileContent: vi.fn(),
  getProjectFiles: vi.fn(),
  info: vi.fn(),
  postProjectConfigurationDiff: vi.fn(),
  postProjectConfigurationValidate: vi.fn(),
  postProjectDeploy: vi.fn(),
  putProjectFileAnnotation: vi.fn(),
  putProjectFileContent: vi.fn(),
  success: vi.fn(),
  t: vi.fn((key: string) => key),
  warning: vi.fn(),
}));

const routeState = reactive({
  params: { id: '1' },
  query: { name: 'sub2api' },
});

const pageContextState = reactive({
  locale: 'en-US',
});

const workspaceCopyMessages = {
  'en-US': {
    'project.configurationWorkspace.copy.annotationAction': 'Edit Annotation',
    'project.configurationWorkspace.copy.annotationUnavailableMessage':
      'Annotation editing will be enabled after the workspace note API is wired.',
    'project.configurationWorkspace.copy.deployAction': 'Deploy Project',
    'project.configurationWorkspace.copy.saveAction': 'Save',
    'project.configurationWorkspace.copy.saveThenContinueAction': 'Save',
  },
  'zh-CN': {
    'project.configurationWorkspace.copy.annotationAction': '编辑注释',
    'project.configurationWorkspace.copy.annotationUnavailableMessage': '工作台注释编辑会在后端注释接口接入后启用。',
    'project.configurationWorkspace.copy.deployAction': '部署项目',
    'project.configurationWorkspace.copy.deployDirtyBody': '检测到未保存的修改，是否先保存？',
    'project.configurationWorkspace.copy.saveAction': '保存',
    'project.configurationWorkspace.copy.saveThenContinueAction': '保存',
  },
} as const;

vi.mock('../../api/project', () => ({
  getProject: mocks.getProject,
  getProjectConfiguration: mocks.getProjectConfiguration,
  getProjectConfigurationPreview: mocks.getProjectConfigurationPreview,
  getProjectFileContent: mocks.getProjectFileContent,
  getProjectFiles: mocks.getProjectFiles,
  postProjectConfigurationDiff: mocks.postProjectConfigurationDiff,
  postProjectConfigurationValidate: mocks.postProjectConfigurationValidate,
  postProjectDeploy: mocks.postProjectDeploy,
  putProjectFileAnnotation: mocks.putProjectFileAnnotation,
  putProjectFileContent: mocks.putProjectFileContent,
}));

vi.mock('../../shared/page-context', () => ({
  useProjectPageContext: () => ({
    locale: computed(() => pageContextState.locale),
    t: (key: string) =>
      workspaceCopyMessages[pageContextState.locale as keyof typeof workspaceCopyMessages]?.[
        key as keyof (typeof workspaceCopyMessages)[keyof typeof workspaceCopyMessages]
      ] ?? mocks.t(key),
  }),
}));

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
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
    info: (...args: unknown[]) => mocks.info(...args),
    success: (...args: unknown[]) => mocks.success(...args),
    warning: (...args: unknown[]) => mocks.warning(...args),
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
      loading: { type: Boolean, default: false },
      message: { type: String, default: '' },
      title: { type: String, default: '' },
      visible: { type: Boolean, default: false },
    },
    setup(props, { slots }) {
      return () =>
        h(
          'div',
          {
            'data-stub': name,
            'data-message': props.message,
            'data-title': props.title || props.header,
            'data-visible': props.visible,
          },
          [h('div', slots.header?.()), h('div', slots.default?.()), h('div', slots.footer?.())],
        );
    },
  });
}

const TButtonStub = defineComponent({
  name: 'TButtonStub',
  props: {
    disabled: { type: Boolean, default: false },
  },
  emits: ['click'],
  setup(props, { attrs, emit, slots }) {
    return () =>
      h(
        'button',
        {
          ...attrs,
          disabled: props.disabled,
          onClick: (event: MouseEvent) => !props.disabled && emit('click', event),
        },
        [slots.icon?.(), slots.default?.()],
      );
  },
});

const TTooltipStub = defineComponent({
  name: 'TTooltipStub',
  props: {
    content: { type: String, default: '' },
  },
  setup(props, { slots }) {
    return () => h('span', { 'data-tooltip-content': props.content }, slots.default?.());
  },
});

const TTabsStub = defineComponent({
  name: 'TTabsStub',
  props: {
    value: { type: String, default: '' },
  },
  setup(_props, { slots }) {
    return () => h('div', { class: 't-tabs-stub' }, slots.default?.());
  },
});

const TTabPanelStub = defineComponent({
  name: 'TTabPanelStub',
  props: {
    value: { type: String, default: '' },
  },
  emits: ['remove'],
  setup(_props, { emit, slots }) {
    return () =>
      h('section', { class: 't-tab-panel-stub' }, [
        h('div', { class: 't-tab-panel-label' }, slots.label?.()),
        h(
          'button',
          {
            class: 't-tab-panel-remove',
            onClick: () => emit('remove'),
          },
          'x',
        ),
        slots.default?.(),
      ]);
  },
});

describe('ProjectConfigurationWorkspaceIndex', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    pageContextState.locale = 'en-US';
    mocks.getProject.mockResolvedValue({
      canonical_project_name: 'sub2api',
      display_name: 'sub2api',
      id: 1,
      ownership_mode: 'managed-root-dedicated',
      runtime_status: 'running',
      working_directory: '/srv/sub2api',
    });
    mocks.getProjectConfiguration.mockResolvedValue({
      drift_status: 'clean',
      last_refresh_status: 'success',
      ownership_mode: 'managed-root-dedicated',
      project_id: 1,
    });
    mocks.getProjectConfigurationPreview.mockResolvedValue({
      canonical_project_name: 'sub2api',
      config_hash: '40ddc4d9bc754dc141bd5f7d57842f693b4c19fb6182c',
      normalized_compose_yaml: 'services:\n  api:\n    image: app\n',
      project_id: 1,
      refreshed_at: '2026-07-03T13:12:38Z',
    });
    mocks.getProjectFiles.mockImplementation(async (_id: number, query?: { path?: string; show_hidden?: boolean }) => {
      if (query?.path === 'config') {
        return {
          current_path: 'config',
          items: [
            {
              editable: true,
              file_kind: 'env',
              has_children: false,
              language_hint: 'dotenv',
              name: '.env',
              node_type: 'file',
              relative_path: 'config/.env',
              size_bytes: 0,
            },
          ],
          root_path: '/srv/sub2api',
        };
      }
      return {
        current_path: '',
        items: [
          {
            editable: true,
            file_kind: 'compose',
            has_children: false,
            language_hint: 'yaml',
            name: 'docker-compose.yml',
            node_type: 'file',
            relative_path: 'docker-compose.yml',
            size_bytes: 32,
          },
          {
            editable: false,
            file_kind: 'directory',
            has_children: true,
            name: 'config',
            node_type: 'directory',
            relative_path: 'config',
          },
        ],
        root_path: '/srv/sub2api',
      };
    });
    mocks.getProjectFileContent.mockImplementation(async (_id: number, query: { path: string }) => ({
      content: query.path === 'docker-compose.yml' ? 'services:\n  api:\n    image: app\n' : '',
      editable: query.path !== 'notes.txt',
      encoding: 'utf-8',
      file_kind: query.path === 'docker-compose.yml' ? 'compose' : 'env',
      language_hint: query.path === 'docker-compose.yml' ? 'yaml' : 'dotenv',
      relative_path: query.path,
      size_bytes: query.path === 'docker-compose.yml' ? 32 : 0,
    }));
    mocks.putProjectFileContent.mockResolvedValue({
      content_hash: 'saved-hash',
      relative_path: 'docker-compose.yml',
      saved_at: '2026-07-06T10:00:00Z',
      size_bytes: 40,
    });
    mocks.putProjectFileAnnotation.mockResolvedValue({
      editable: true,
      file_kind: 'compose',
      has_children: false,
      hidden_by_default: false,
      language_hint: 'yaml',
      name: 'docker-compose.yml',
      node_type: 'file',
      project_note: 'Existing note',
      relative_path: 'docker-compose.yml',
      size_bytes: 32,
      tooltip: 'Existing note',
      tooltip_source: 'project-note',
    });
    mocks.postProjectConfigurationDiff.mockResolvedValue({
      canonical_project_name: 'sub2api',
      current_config_hash: '40ddc4d9bc754dc141bd5f7d57842f693b4c19fb6182c',
      files: [
        {
          changed: true,
          current_content: 'services:\n  api:\n    image: old\n',
          current_hash: 'c90a77d4f1e9515ab3e7a02017df9f5c725ab11e90ef',
          kind: 'compose',
          path: 'docker-compose.yml',
          proposed_content: 'services:\n  api:\n    image: app\n',
          proposed_hash: '0dd31a7ef1658f86dcad96522b52d891d6f34f27ca10',
        },
      ],
      has_changes: true,
      ownership_mode: 'managed-root-dedicated',
      project_id: 1,
      proposed_config_hash: '0dd31a7ef1658f86dcad96522b52d891d6f34f27ca10',
      warnings: [],
    });
    mocks.postProjectConfigurationValidate.mockResolvedValue({
      canonical_project_name: 'sub2api',
      declared_service_names: ['api'],
      normalized_compose_yaml: 'services:\n  api:\n    image: app\n',
      ownership_mode: 'managed-root-dedicated',
      project_id: 1,
      proposed_config_hash: '0dd31a7ef1658f86dcad96522b52d891d6f34f27ca10',
      warnings: [],
    });
    mocks.postProjectDeploy.mockResolvedValue({
      message: 'deployed',
    });
  });

  it('loads the root file list and opens the first file buffer', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    expect(mocks.getProjectFiles).toHaveBeenCalledWith(1, { show_hidden: false });
    expect(mocks.getProjectFileContent).toHaveBeenCalledWith(1, { path: 'docker-compose.yml' });
    expect((wrapper.get('[data-testid="workspace-monaco-editor"]').element as HTMLTextAreaElement).value).toBe(
      'services:\n  api:\n    image: app\n',
    );
  });

  it('navigates into nested directories and opens empty file content without a blank editor failure', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    expect(mocks.getProjectFiles).toHaveBeenCalledWith(1, { path: 'config', show_hidden: false });

    await wrapper.get('[data-testid="workspace-entry-config-env"]').trigger('click');
    await flushPromises();

    expect(mocks.getProjectFileContent).toHaveBeenCalledWith(1, { path: 'config/.env' });
    expect(wrapper.find('[data-testid="workspace-monaco-editor"]').exists()).toBe(true);
    expect((wrapper.get('[data-testid="workspace-monaco-editor"]').element as HTMLTextAreaElement).value).toBe('');
  });

  it('keeps .env visible during directory browsing without enabling show_hidden', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="workspace-entry-config-env"]').exists()).toBe(true);
    expect(mocks.getProjectFiles).toHaveBeenNthCalledWith(2, 1, { path: 'config', show_hidden: false });
  });

  it('does not refetch a loaded empty file when reopening the tab with unsaved edits', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-entry-config-env"]').trigger('click');
    await flushPromises();

    const initialCalls = mocks.getProjectFileContent.mock.calls.filter(
      ([, query]) => query.path === 'config/.env',
    ).length;
    expect(initialCalls).toBe(1);

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('EDITOR_ONLY=true\n');
    await wrapper.get('[data-testid="workspace-entry-config-env"]').trigger('click');
    await flushPromises();

    const nextCalls = mocks.getProjectFileContent.mock.calls.filter(([, query]) => query.path === 'config/.env').length;
    expect(nextCalls).toBe(1);
    expect((wrapper.get('[data-testid="workspace-monaco-editor"]').element as HTMLTextAreaElement).value).toBe(
      'EDITOR_ONLY=true\n',
    );
  });

  it('renders compact tree row actions and removes the duplicate editor title header', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    expect(wrapper.find('[data-testid="workspace-entry-docker-compose-yml-annotation"]').exists()).toBe(true);
    expect(wrapper.find('.project-configuration-workspace__editor-head').exists()).toBe(false);
    expect(wrapper.find('.project-configuration-workspace__browser-toolbar').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('current directory still contains default-hidden directories');
  });

  it('shows compact tree toolbar actions and opens the annotation editor', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    const hiddenToggle = wrapper.get('[data-testid="workspace-show-hidden-toggle"]');
    expect(hiddenToggle.element.parentElement?.getAttribute('data-tooltip-content')).toBe(
      'project.configurationWorkspace.copy.showHiddenAction',
    );
    expect(hiddenToggle.text()).toBe('');

    const annotationButton = wrapper.get('[data-testid="workspace-entry-docker-compose-yml-annotation"]');
    expect(annotationButton.element.parentElement?.getAttribute('data-tooltip-content')).toBe('Edit Annotation');

    await annotationButton.trigger('click');
    expect(wrapper.findAll('[data-stub="TDialog"]').at(-1)?.attributes('data-visible')).toBe('true');
  });

  it('truncates configuration hashes while keeping the full value in tooltips', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'project.configurationWorkspace.copy.diffAction')
      ?.trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('40ddc4...b6182c');
    expect(wrapper.text()).toContain('0dd31a...27ca10');
    expect(wrapper.html()).toContain('data-tooltip-content="40ddc4d9bc754dc141bd5f7d57842f693b4c19fb6182c"');
    expect(wrapper.html()).toContain('data-tooltip-content="0dd31a7ef1658f86dcad96522b52d891d6f34f27ca10"');
  });

  it('saves the active file buffer without deploying the project', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: newer\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save')
      ?.trigger('click');
    await flushPromises();

    expect(mocks.putProjectFileContent).toHaveBeenCalledWith(
      1,
      { path: 'docker-compose.yml' },
      { content: 'services:\n  api:\n    image: newer\n' },
    );
    expect(mocks.postProjectDeploy).not.toHaveBeenCalled();
  });

  it('shows the deploy dirty prompt and saves before deploying when requested', async () => {
    pageContextState.locale = 'zh-CN';
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: newer\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === '部署项目')
      ?.trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('检测到未保存的修改，是否先保存？');

    await wrapper
      .findAll('button')
      .filter((button) => button.text().trim() === '保存')
      .at(-1)
      ?.trigger('click');
    await flushPromises();

    expect(mocks.putProjectFileContent).toHaveBeenCalledWith(
      1,
      { path: 'docker-compose.yml' },
      { content: 'services:\n  api:\n    image: newer\n' },
    );
    expect(mocks.postProjectDeploy).toHaveBeenCalledWith(1);
  });
});

function mountWorkspace() {
  return mount(ProjectConfigurationWorkspaceIndex, {
    global: {
      stubs: {
        TAlert: createTStub('TAlert'),
        TButton: TButtonStub,
        TCard: createTStub('TCard'),
        TDescriptions: createTStub('TDescriptions'),
        TDescriptionsItem: createTStub('TDescriptionsItem'),
        TDialog: createTStub('TDialog'),
        TDrawer: createTStub('TDrawer'),
        TEmpty: createTStub('TEmpty'),
        TLoading: createTStub('TLoading'),
        TSpace: createTStub('TSpace'),
        TTabPanel: TTabPanelStub,
        TTabs: TTabsStub,
        TTag: createTStub('TTag'),
        TTextarea: defineComponent({
          name: 'TTextareaStub',
          props: {
            modelValue: { type: String, default: '' },
          },
          emits: ['update:modelValue'],
          setup(props, { emit }) {
            return () =>
              h('textarea', {
                value: props.modelValue,
                onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLTextAreaElement).value),
              });
          },
        }),
        TTooltip: TTooltipStub,
      },
    },
  });
}
