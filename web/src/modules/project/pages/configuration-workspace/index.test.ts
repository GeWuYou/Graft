import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { computed, defineComponent, h, nextTick, reactive, watch } from 'vue';

import type { ProjectWorkspaceFileContentResponse } from '../../types/project';
import ProjectConfigurationWorkspaceIndex from './index.vue';

const mocks = vi.hoisted(() => ({
  error: vi.fn(),
  getProject: vi.fn(),
  getProjectConfiguration: vi.fn(),
  getProjectFileContent: vi.fn(),
  getProjectFiles: vi.fn(),
  info: vi.fn(),
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
    'project.configurationWorkspace.copy.annotationSaveFailed': 'Failed to save the annotation.',
    'project.configurationWorkspace.copy.deployAction': 'Deploy Project',
    'project.configurationWorkspace.copy.confirmSaveAction': 'Confirm Save',
    'project.configurationWorkspace.copy.diffConfirmBody':
      'Review the changed files below. Changes will be written to the working directory only after you confirm.',
    'project.configurationWorkspace.copy.diffEmptyDirectSaveHint':
      'No file diff was detected. Draft files will be saved directly.',
    'project.configurationWorkspace.copy.diffTreeTitle': 'Changed Files',
    'project.configurationWorkspace.copy.fileValidationFailed': 'Syntax errors were found in the active file.',
    'project.configurationWorkspace.copy.fileValidationPassed': 'No syntax errors were found in the active file.',
    'project.configurationWorkspace.copy.fileValidationUnavailable':
      'Explicit syntax validation is not available for this file type.',
    'project.configurationWorkspace.copy.saveAction': 'Save',
    'project.configurationWorkspace.copy.saveThenContinueAction': 'Save',
    'project.configurationWorkspace.copy.validateAction': 'Validate',
    'project.configurationWorkspace.copy.validateNoFile': 'Open a supported file before running validation.',
  },
  'zh-CN': {
    'project.configurationWorkspace.copy.annotationAction': '编辑注释',
    'project.configurationWorkspace.copy.annotationSaveFailed': '注释保存失败。',
    'project.configurationWorkspace.copy.confirmSaveAction': '确认保存',
    'project.configurationWorkspace.copy.deployAction': '部署项目',
    'project.configurationWorkspace.copy.diffConfirmBody':
      '请先确认下面的变更文件。只有在你确认后，草稿才会写入工作目录。',
    'project.configurationWorkspace.copy.diffEmptyDirectSaveHint': '未检测到文件差异，将直接保存草稿文件。',
    'project.configurationWorkspace.copy.diffTreeTitle': '变更文件',
    'project.configurationWorkspace.copy.fileValidationFailed': '当前文件存在语法错误。',
    'project.configurationWorkspace.copy.fileValidationPassed': '当前文件未检测到语法错误。',
    'project.configurationWorkspace.copy.fileValidationUnavailable': '当前文件类型暂不支持显式语法校验。',
    'project.configurationWorkspace.copy.saveAction': '保存',
    'project.configurationWorkspace.copy.saveThenContinueAction': '保存',
    'project.configurationWorkspace.copy.validateAction': '校验',
    'project.configurationWorkspace.copy.validateNoFile': '请先打开一个可校验的文件。',
  },
} as const;

vi.mock('../../api/project', () => ({
  getProject: mocks.getProject,
  getProjectConfiguration: mocks.getProjectConfiguration,
  getProjectFileContent: mocks.getProjectFileContent,
  getProjectFiles: mocks.getProjectFiles,
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
    props: {
      defaultHeight: { type: Number, default: 0 },
      fillHeight: { type: Boolean, default: false },
      minHeight: { type: Number, default: undefined },
      mobileMinHeight: { type: Number, default: undefined },
    },
    setup(props, { slots }) {
      return () =>
        h(
          'section',
          {
            class: 'content-viewer-frame-stub',
            'data-default-height': String(props.defaultHeight),
            'data-fill-height': String(props.fillHeight),
            'data-min-height': props.minHeight === undefined ? undefined : String(props.minHeight),
            'data-mobile-min-height': props.mobileMinHeight === undefined ? undefined : String(props.mobileMinHeight),
          },
          [
            h('div', { class: 'content-viewer-header' }, slots.header?.()),
            h('div', { class: 'content-viewer-header-actions' }, slots['header-actions']?.()),
            h('div', { class: 'content-viewer-body' }, slots.default?.()),
          ],
        );
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
    setup(props, { emit, expose }) {
      const getMarkers = () =>
        String(props.modelValue).includes('[broken')
          ? [
              {
                endColumn: 15,
                endLineNumber: 3,
                message: 'invalid yaml',
                severity: 8,
                startColumn: 12,
                startLineNumber: 3,
              },
            ]
          : [];

      expose({
        getMarkers,
        relayout: () => Promise.resolve(),
        revealMarker: vi.fn(() => true),
        waitForDiagnostics: () => Promise.resolve(getMarkers()),
      });

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
    setup(props, { expose }) {
      expose({
        getLineChanges: () =>
          props.originalValue === props.modifiedValue
            ? []
            : [
                {
                  modifiedEndLineNumber: 3,
                  modifiedStartLineNumber: 3,
                  originalEndLineNumber: 3,
                  originalStartLineNumber: 3,
                },
              ],
        navigateDiff: vi.fn(() => true),
        relayout: () => Promise.resolve(),
        revealFirstDiff: vi.fn(() => true),
        revealLineChange: vi.fn(() => true),
      });

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

async function flushDiffViewerFrames() {
  await flushPromises();
  await new Promise<void>((resolve) => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        resolve();
      });
    });
  });
  await flushPromises();
}

async function waitForDiffViewer(
  wrapper: ReturnType<typeof mount>,
  selector = '[data-testid="configuration-diff-viewer"]',
  attempts = 6,
) {
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    await flushDiffViewerFrames();
    if (wrapper.find(selector).exists()) {
      return;
    }
  }
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

const TDropdownStub = defineComponent({
  name: 'TDropdownStub',
  props: {
    popupProps: {
      type: Object,
      default: () => ({}),
    },
  },
  setup(props, { attrs, slots }) {
    return () =>
      h(
        'div',
        {
          ...attrs,
          'data-popup-visible':
            props.popupProps && typeof props.popupProps === 'object' && 'visible' in props.popupProps
              ? String((props.popupProps as { visible?: boolean }).visible)
              : 'false',
        },
        [h('div', { class: 't-dropdown-trigger-stub' }, slots.default?.()), h('div', slots.dropdown?.())],
      );
  },
});

const TDropdownItemStub = defineComponent({
  name: 'TDropdownItemStub',
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
          type: 'button',
          onClick: () => !props.disabled && emit('click'),
        },
        slots.default?.(),
      );
  },
});

const TDialogStub = defineComponent({
  name: 'TDialogStub',
  props: {
    cancelBtn: { type: [Boolean, Object], default: false },
    confirmBtn: { type: [Boolean, Object], default: false },
    dialogClassName: { type: String, default: '' },
    dialogStyle: { type: Object, default: undefined },
    header: { type: String, default: '' },
    mode: { type: String, default: 'modal' },
    onOpened: { type: Function, default: undefined },
    top: { type: [Number, String], default: undefined },
    visible: { type: Boolean, default: false },
    width: { type: [Number, String], default: undefined },
  },
  emits: ['close', 'confirm', 'update:visible'],
  setup(props, { emit, slots }) {
    watch(
      () => props.visible,
      (visible) => {
        if (!visible || typeof props.onOpened !== 'function') {
          return;
        }
        void nextTick(() => {
          (props.onOpened as () => void)();
        });
      },
      { immediate: true },
    );

    return () =>
      h(
        'div',
        {
          'data-class-name': props.dialogClassName,
          'data-dialog-style': props.dialogStyle ? JSON.stringify(props.dialogStyle) : undefined,
          'data-mode': props.mode,
          'data-stub': 'TDialog',
          'data-title': props.header,
          'data-top': props.top === undefined ? undefined : String(props.top),
          'data-visible': String(props.visible),
          'data-width': props.width === undefined ? undefined : String(props.width),
        },
        [
          props.visible ? h('div', slots.default?.()) : null,
          props.visible ? h('div', slots.footer?.()) : null,
          props.visible && props.confirmBtn
            ? h(
                'button',
                {
                  'data-testid': 't-dialog-confirm',
                  onClick: () => emit('confirm'),
                },
                typeof props.confirmBtn === 'object' && props.confirmBtn && 'content' in props.confirmBtn
                  ? String(props.confirmBtn.content)
                  : 'Confirm',
              )
            : null,
          props.visible && props.cancelBtn
            ? h(
                'button',
                {
                  'data-testid': 't-dialog-cancel',
                  onClick: () => emit('close'),
                },
                typeof props.cancelBtn === 'object' && props.cancelBtn && 'content' in props.cancelBtn
                  ? String(props.cancelBtn.content)
                  : 'Cancel',
              )
            : null,
        ],
      );
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
      ownership_mode: 'managed-root-dedicated',
      project_id: 1,
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
              readable: true,
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
            readable: true,
            relative_path: 'docker-compose.yml',
            size_bytes: 32,
          },
          {
            editable: false,
            file_kind: 'directory',
            has_children: true,
            name: 'config',
            node_type: 'directory',
            readable: true,
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
      readable: true,
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
      readable: true,
      relative_path: 'docker-compose.yml',
      size_bytes: 32,
      tooltip: 'Existing note',
      tooltip_source: 'project-note',
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

  it('keeps the previous editor content visible while the next file is still loading', async () => {
    let resolveEnvContent!: (value: ProjectWorkspaceFileContentResponse) => void;

    mocks.getProjectFileContent.mockImplementation((_id: number, query: { path: string }) => {
      if (query.path === 'config/.env') {
        return new Promise((resolve) => {
          resolveEnvContent = resolve as (value: ProjectWorkspaceFileContentResponse) => void;
        });
      }

      return Promise.resolve({
        content: 'services:\n  api:\n    image: app\n',
        editable: true,
        encoding: 'utf-8',
        file_kind: 'compose',
        language_hint: 'yaml',
        readable: true,
        relative_path: query.path,
        size_bytes: 32,
      });
    });

    const wrapper = mountWorkspace();
    await flushPromises();

    expect((wrapper.get('[data-testid="workspace-monaco-editor"]').element as HTMLTextAreaElement).value).toBe(
      'services:\n  api:\n    image: app\n',
    );

    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-entry-config-env"]').trigger('click');
    await wrapper.vm.$nextTick();

    expect((wrapper.get('[data-testid="workspace-monaco-editor"]').element as HTMLTextAreaElement).value).toBe(
      'services:\n  api:\n    image: app\n',
    );

    resolveEnvContent({
      content: '',
      editable: true,
      encoding: 'utf-8',
      file_kind: 'env',
      language_hint: 'dotenv',
      readable: true,
      relative_path: 'config/.env',
      size_bytes: 0,
    });
    await flushPromises();

    expect((wrapper.get('[data-testid="workspace-monaco-editor"]').element as HTMLTextAreaElement).value).toBe('');
  });

  it('renders the file tab context menu actions for each open editor tab', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-entry-config-env"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="workspace-file-tab-menu-workspace-entry-docker-compose-yml"]').exists()).toBe(
      true,
    );
    expect(wrapper.find('[data-testid="workspace-file-tab-menu-workspace-entry-config-env"]').exists()).toBe(true);
    expect(wrapper.text()).toContain('layout.tagTabs.refresh');
    expect(wrapper.text()).toContain('layout.tagTabs.closeLeft');
    expect(wrapper.text()).toContain('layout.tagTabs.closeRight');
    expect(wrapper.text()).toContain('layout.tagTabs.closeOther');
    expect(wrapper.text()).toContain('layout.tagTabs.closeAll');
  });

  it('closes file tabs to the right from the context menu actions', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-entry-config-env"]').trigger('click');
    await flushPromises();
    expect(wrapper.findAll('.t-tab-panel-stub')).toHaveLength(2);

    await wrapper
      .get('[data-testid="workspace-file-tab-menu-workspace-entry-docker-compose-yml-close-right"]')
      .trigger('click');
    await flushPromises();

    expect(wrapper.findAll('.t-tab-panel-stub')).toHaveLength(1);
    expect(wrapper.find('[data-testid="workspace-file-tab-menu-workspace-entry-config-env"]').exists()).toBe(false);
  });

  it('refreshes a non-active file tab from the context menu action without changing the active editor', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-entry-config-env"]').trigger('click');
    await flushPromises();

    expect((wrapper.get('[data-testid="workspace-monaco-editor"]').element as HTMLTextAreaElement).value).toBe('');

    const composeCallsBefore = mocks.getProjectFileContent.mock.calls.filter(
      ([, query]) => query.path === 'docker-compose.yml',
    ).length;

    await wrapper
      .get('[data-testid="workspace-file-tab-menu-workspace-entry-docker-compose-yml-refresh"]')
      .trigger('click');
    await flushPromises();

    const composeCallsAfter = mocks.getProjectFileContent.mock.calls.filter(
      ([, query]) => query.path === 'docker-compose.yml',
    ).length;
    expect(composeCallsAfter).toBe(composeCallsBefore + 1);
    expect((wrapper.get('[data-testid="workspace-monaco-editor"]').element as HTMLTextAreaElement).value).toBe('');
  });

  it('lets the workspace page exit fullscreen on escape', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    expect(wrapper.get('.content-viewer-frame-stub').attributes('data-fill-height')).toBe('false');

    await wrapper.get('.content-viewer-header-actions button:nth-of-type(2)').trigger('click');
    await flushPromises();

    expect(wrapper.get('.project-configuration-workspace').classes()).toContain(
      'project-configuration-workspace--fullscreen',
    );
    expect(wrapper.get('.content-viewer-frame-stub').attributes('data-fill-height')).toBe('true');
    expect(document.body.style.overflow).toBe('hidden');

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    await wrapper.vm.$nextTick();

    expect(wrapper.get('.project-configuration-workspace').classes()).not.toContain(
      'project-configuration-workspace--fullscreen',
    );
    expect(wrapper.get('.content-viewer-frame-stub').attributes('data-fill-height')).toBe('false');
    expect(document.body.style.overflow).toBe('');
  });

  it('passes only the editor default height to the shared viewer frame', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    const frame = wrapper.get('.content-viewer-frame-stub');
    expect(Number(frame.attributes('data-default-height'))).toBeGreaterThanOrEqual(560);
    expect(frame.attributes('data-min-height')).toBeUndefined();
    expect(frame.attributes('data-mobile-min-height')).toBeUndefined();
  });

  it('renders compact tree row actions and removes the duplicate editor title header', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    expect(wrapper.find('[data-testid="workspace-entry-docker-compose-yml-annotation"]').exists()).toBe(true);
    expect(wrapper.find('.project-configuration-workspace__editor-head').exists()).toBe(false);
    expect(wrapper.find('.project-configuration-workspace__browser-toolbar').exists()).toBe(false);
    expect(wrapper.find('.project-configuration-workspace__feedback').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('current directory still contains default-hidden directories');
  });

  it('shows compact tree toolbar actions and opens the annotation editor', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    expect(wrapper.find('.project-configuration-workspace__tree.graft-scrollbar').exists()).toBe(true);
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

  it('saves workspace annotations through the dialog flow', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-docker-compose-yml-annotation"]').trigger('click');
    await wrapper.findAll('textarea').at(-1)!.setValue('Updated note');
    await wrapper.get('[data-testid="t-dialog-confirm"]').trigger('click');
    await flushPromises();

    expect(mocks.putProjectFileAnnotation).toHaveBeenCalledWith(
      1,
      { path: 'docker-compose.yml' },
      { annotation: 'Updated note' },
    );
    expect(wrapper.findAll('[data-stub="TDialog"]').at(-1)?.attributes('data-visible')).toBe('false');
    expect(mocks.error).not.toHaveBeenCalled();
  });

  it('shows a localized annotation save failure instead of the retired unavailable fallback copy', async () => {
    const wrapper = mountWorkspace();
    mocks.putProjectFileAnnotation.mockRejectedValueOnce(new Error('save failed'));
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-docker-compose-yml-annotation"]').trigger('click');
    await wrapper.findAll('textarea').at(-1)!.setValue('Updated note');
    await wrapper.get('[data-testid="t-dialog-confirm"]').trigger('click');
    await flushPromises();

    expect(mocks.error).toHaveBeenCalledWith('Failed to save the annotation.');
  });

  it('shows the zh-CN annotation save failure copy when the request fails', async () => {
    pageContextState.locale = 'zh-CN';
    const wrapper = mountWorkspace();
    mocks.putProjectFileAnnotation.mockRejectedValueOnce(new Error('save failed'));
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-docker-compose-yml-annotation"]').trigger('click');
    await wrapper.findAll('textarea').at(-1)!.setValue('更新备注');
    await wrapper.get('[data-testid="t-dialog-confirm"]').trigger('click');
    await flushPromises();

    expect(mocks.error).toHaveBeenCalledWith('注释保存失败。');
  });

  it('hides a directory error after the directory is collapsed', async () => {
    mocks.getProjectFiles.mockImplementationOnce(async () => ({
      current_path: '',
      items: [
        {
          editable: false,
          file_kind: 'directory',
          has_children: true,
          name: 'config',
          node_type: 'directory',
          readable: true,
          relative_path: 'config',
        },
      ],
      root_path: '/srv/sub2api',
    }));
    mocks.getProjectFiles.mockRejectedValueOnce(new Error('Directory load failed'));
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    expect(wrapper.text()).toContain('project.list.retry');

    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    expect(wrapper.text()).not.toContain('project.list.retry');
  });

  it('truncates configuration hashes while keeping the full value in tooltips', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: newer\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save')
      ?.trigger('click');
    await waitForDiffViewer(wrapper);

    const currentHash = wrapper.get('[data-testid="configuration-diff-current-hash"]').text();
    const proposedHash = wrapper.get('[data-testid="configuration-diff-proposed-hash"]').text();
    expect(currentHash).toMatch(/^ws-[0-9a-f]+(?:\.\.\.[0-9a-f]+)?$/);
    expect(proposedHash).toMatch(/^ws-[0-9a-f]+(?:\.\.\.[0-9a-f]+)?$/);
    expect(wrapper.html()).toContain(`data-tooltip-content="${currentHash}"`);
    expect(wrapper.html()).toContain(`data-tooltip-content="${proposedHash}"`);
    expect(currentHash).not.toBe(proposedHash);
  });

  it('opens the diff result in a modal dialog after save', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    expect(
      wrapper
        .findAll('button')
        .some((button) => button.text().trim() === 'project.configurationWorkspace.copy.diffAction'),
    ).toBe(false);

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: newer\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save')
      ?.trigger('click');
    await waitForDiffViewer(wrapper);

    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="configuration-diff-viewer"]').exists()).toBe(true);
    const diffViewer = wrapper.get('[data-testid="configuration-diff-viewer"]');
    expect(diffViewer.find('pre.original').text()).toBe('services:\n  api:\n    image: app');
    expect(diffViewer.find('pre.modified').text()).toBe('services:\n  api:\n    image: newer');
    expect(wrapper.find('[data-testid="configuration-diff-file-workspace-entry-docker-compose-yml"]').exists()).toBe(
      true,
    );
    expect(wrapper.find('[data-testid="configuration-diff-confirm-save"]').exists()).toBe(true);
    const resultDialog = wrapper.find('[data-class-name*="project-configuration-workspace__result-dialog-shell"]');
    expect(resultDialog?.attributes('data-mode')).toBe('modal');
    expect(resultDialog?.attributes('data-top')).toBe('24');
    expect(resultDialog?.attributes('data-width')).toBe('min(80vw, 1600px)');
    expect(resultDialog?.attributes('data-dialog-style')).toContain('"height":"80vh"');
    expect(resultDialog?.attributes('data-dialog-style')).toContain('"maxHeight":"calc(100vh - 48px)"');
    expect(resultDialog?.attributes('data-dialog-style')).not.toContain('"width":"auto"');
  });

  it('expands the diff result dialog to edge-to-edge fullscreen sizing', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: newer\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save')
      ?.trigger('click');
    await waitForDiffViewer(wrapper);

    await wrapper.get('[data-testid="configuration-result-fullscreen-toggle"]').trigger('click');
    await waitForDiffViewer(wrapper);

    const resultDialog = wrapper.find('[data-class-name*="project-configuration-workspace__result-dialog-shell"]');
    expect(resultDialog?.attributes('data-mode')).toBe('full-screen');
    expect(resultDialog?.attributes('data-top')).toBe('0');
    expect(resultDialog?.attributes('data-width')).toBeUndefined();
    expect(resultDialog?.attributes('data-class-name')).toContain(
      'project-configuration-workspace__result-dialog-shell--fullscreen',
    );
    expect(resultDialog?.attributes('data-dialog-style')).toContain('"height":"100vh"');
    expect(resultDialog?.attributes('data-dialog-style')).toContain('"maxHeight":"100vh"');
    expect(resultDialog?.attributes('data-dialog-style')).toContain('"width":"100vw"');
    expect(wrapper.find('[data-testid="configuration-diff-viewer"]').exists()).toBe(true);

    await wrapper.get('[data-testid="configuration-result-fullscreen-toggle"]').trigger('click');
    await flushDiffViewerFrames();

    expect(wrapper.find('[data-testid="configuration-diff-viewer"]').exists()).toBe(true);
    const restoredDialog = wrapper.find('[data-class-name*="project-configuration-workspace__result-dialog-shell"]');
    expect(restoredDialog?.attributes('data-mode')).toBe('modal');
  });

  it('validates the active file syntax without saving dirty drafts first', async () => {
    pageContextState.locale = 'zh-CN';
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: [broken\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === '校验')
      ?.trigger('click');
    await flushPromises();

    expect(mocks.putProjectFileContent).not.toHaveBeenCalled();
    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="syntax-monaco-viewer"]').exists()).toBe(true);
    expect(mocks.error).toHaveBeenCalledWith('当前文件存在语法错误。');
  });

  it('does not open a result dialog when the active file has no syntax errors', async () => {
    pageContextState.locale = 'en-US';
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Validate')
      ?.trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="syntax-monaco-viewer"]').exists()).toBe(false);
    expect(mocks.success).toHaveBeenCalledWith('No syntax errors were found in the active file.');
  });

  it('keeps the diff file list in workspace tree form instead of compare cards', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: newer\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save')
      ?.trigger('click');
    await flushDiffViewerFrames();

    expect(wrapper.find('[data-testid="configuration-diff-viewer"]').exists()).toBe(true);
    expect(wrapper.text()).toContain('docker-compose.yml');
    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(true);
    expect(wrapper.find('.project-configuration-workspace__diff-sidebar').exists()).toBe(true);
    expect(wrapper.find('.project-configuration-workspace__diff-file').exists()).toBe(false);
  });

  it('previews the active dirty buffers before saving them to disk', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: newer\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save')
      ?.trigger('click');
    await flushPromises();

    expect(mocks.putProjectFileContent).not.toHaveBeenCalled();
    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(true);
    expect(mocks.postProjectDeploy).not.toHaveBeenCalled();
  });

  it('deploys only after preview confirm saves the dirty draft', async () => {
    pageContextState.locale = 'zh-CN';
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: newer\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === '部署项目')
      ?.trigger('click');
    await flushPromises();

    expect(mocks.putProjectFileContent).not.toHaveBeenCalled();
    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(true);

    await wrapper.get('[data-testid="configuration-diff-confirm-save"]').trigger('click');
    await flushPromises();

    expect(mocks.putProjectFileContent).toHaveBeenCalledWith(
      1,
      { path: 'docker-compose.yml' },
      { content: 'services:\n  api:\n    image: newer\n' },
    );
    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(false);
    expect(mocks.postProjectDeploy).toHaveBeenCalledWith(1);
  });

  it('cancels the preview without saving dirty files', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: newer\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save')
      ?.trigger('click');
    await flushPromises();

    await wrapper.get('[data-testid="configuration-diff-cancel"]').trigger('click');
    await flushPromises();

    expect(mocks.putProjectFileContent).not.toHaveBeenCalled();
    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(false);
    expect((wrapper.get('[data-testid="workspace-monaco-editor"]').element as HTMLTextAreaElement).value).toBe(
      'services:\n  api:\n    image: newer\n',
    );
  });

  it('saves directly without opening preview when normalization removes the dirty diff', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper
      .get('[data-testid="workspace-monaco-editor"]')
      .setValue('services:\r\n  api:  \r\n    image: app\r\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save')
      ?.trigger('click');
    await flushPromises();

    expect(mocks.putProjectFileContent).toHaveBeenCalledWith(
      1,
      { path: 'docker-compose.yml' },
      { content: 'services:\n  api:\n    image: app\n' },
    );
    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(false);
    expect(mocks.info).toHaveBeenCalledWith('No file diff was detected. Draft files will be saved directly.');
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
        TDialog: TDialogStub,
        TDropdown: TDropdownStub,
        TDropdownItem: TDropdownItemStub,
        TDropdownMenu: createTStub('TDropdownMenu'),
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
