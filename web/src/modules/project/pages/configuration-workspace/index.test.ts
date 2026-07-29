import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { computed, defineComponent, h, nextTick, reactive, watch } from 'vue';

import type { ApplicationWorkspaceFileContentResponse } from '../../types/project';
import ApplicationConfigurationWorkspaceIndex from './index.vue';

const mocks = vi.hoisted(() => ({
  deleteApplicationWorkspaceEntry: vi.fn(),
  error: vi.fn(),
  getApplication: vi.fn(),
  getApplicationConfiguration: vi.fn(),
  getApplicationFileContent: vi.fn(),
  getApplicationFiles: vi.fn(),
  info: vi.fn(),
  postApplicationRedeploy: vi.fn(),
  postApplicationWorkspaceEntry: vi.fn(),
  postApplicationWorkspaceRename: vi.fn(),
  putApplicationFileAnnotation: vi.fn(),
  putApplicationFileContent: vi.fn(),
  success: vi.fn(),
  t: vi.fn((key: string) => key),
  warning: vi.fn(),
}));

const monacoSurfaceState = vi.hoisted(() => ({
  diagnosticsResolver: null as
    | null
    | ((context: { modelKey: string; modelValue: string }) => Array<{
        endColumn: number;
        endLineNumber: number;
        message: string;
        severity: number;
        startColumn: number;
        startLineNumber: number;
      }>),
  modelKeyLagByPath: {} as Record<string, number>,
}));

const routeState = reactive({
  fullPath: '/applications/app_1/configuration?name=sub2api',
  name: 'ApplicationConfigurationWorkspaceIndex',
  params: { applicationId: 'app_1' },
  path: '/applications/app_1/configuration',
  query: { name: 'sub2api' },
});

const tabsRouterStoreMock = vi.hoisted(() => ({
  updateActiveTabTitle: vi.fn(),
  tabRouterList: [
    {
      fullPath: '/applications/app_1/configuration?name=sub2api',
      path: '/applications/app_1/configuration',
      tabKey: '/applications/app_1/configuration',
      title: { 'en-US': 'Application Detail - sub2api', 'zh-CN': '项目详情 - sub2api' },
    },
  ],
}));

const pageContextState = reactive({
  locale: 'en-US',
});

const workspaceCopyMessages = {
  'en-US': {
    'project.route.configurationWorkspace.title': 'Configuration Workspace',
    'project.configurationWorkspace.copy.annotationAction': 'Edit Annotation',
    'project.configurationWorkspace.copy.annotationSaveFailed': 'Failed to save the annotation.',
    'project.configurationWorkspace.copy.batchFileValidationRiskBody':
      'These files still have syntax errors. Saving them may leave the project in a state that later validation or deploy cannot parse correctly.',
    'project.configurationWorkspace.copy.batchFileValidationRiskTitle': 'Save Files with Syntax Errors',
    'project.configurationWorkspace.copy.batchFileValidationTitle': 'File Validation Errors',
    'project.configurationWorkspace.copy.redeployAction': 'Redeploy Application',
    'project.configurationWorkspace.copy.confirmSaveAllAction': 'Confirm Save All',
    'project.configurationWorkspace.copy.confirmSaveAllWithErrorsAction': 'Save All Anyway',
    'project.configurationWorkspace.copy.confirmSaveCurrentAction': 'Confirm Save',
    'project.configurationWorkspace.copy.confirmSaveRedeployWithErrorsAction': 'Save and Redeploy Anyway',
    'project.configurationWorkspace.copy.confirmSaveAction': 'Confirm Save',
    'project.configurationWorkspace.copy.confirmSaveWithErrorsAction': 'Save Anyway',
    'project.configurationWorkspace.copy.diffCurrentFileConfirmBody':
      'Review the current file diff below. Changes will be written only after you confirm.',
    'project.configurationWorkspace.copy.diffCurrentFileTitle': 'Current File Diff',
    'project.configurationWorkspace.copy.diffConfirmBody':
      'Review the changed files below. Changes will be written to the working directory only after you confirm.',
    'project.configurationWorkspace.copy.diffEmptyDirectSaveHint':
      'No file diff was detected. Draft files will be saved directly.',
    'project.configurationWorkspace.copy.diffTreeTitle': 'Changed Files',
    'project.configurationWorkspace.copy.fileValidationEmbeddedHint':
      'The current file has syntax errors. Review the highlighted locations before saving or deploying.',
    'project.configurationWorkspace.copy.fileValidationFailed': 'Syntax errors were found in the active file.',
    'project.configurationWorkspace.copy.fileValidationPassed': 'No syntax errors were found in the active file.',
    'project.configurationWorkspace.copy.fileValidationTitle': 'File Validation',
    'project.configurationWorkspace.copy.fileValidationUnavailable':
      'Explicit syntax validation is not available for this file type.',
    'project.configurationWorkspace.copy.resultSummaryChangedFilesLabel': 'Changed Files',
    'project.configurationWorkspace.copy.resultSummaryCurrentErrorsLabel': 'Current Errors',
    'project.configurationWorkspace.copy.resultSummaryCurrentFileLabel': 'Current File',
    'project.configurationWorkspace.copy.resultSummaryErrorFilesLabel': 'Error Files',
    'project.configurationWorkspace.copy.saveAction': 'Save',
    'project.configurationWorkspace.copy.saveAllAction': 'Save All',
    'project.configurationWorkspace.copy.savePartialHint': 'Only the current file will be saved.',
    'project.configurationWorkspace.copy.selectSyntaxFile': 'Choose a file to inspect its syntax errors',
    'project.configurationWorkspace.copy.syntaxErrorCountLabel': '{count} Error(s)',
    'project.configurationWorkspace.copy.syntaxFileTreeTitle': 'Files with Syntax Errors',
    'project.configurationWorkspace.copy.saveThenContinueAction': 'Save',
    'project.configurationWorkspace.copy.validateAction': 'Validate',
    'project.configurationWorkspace.copy.validateSkipUnsupportedHint':
      'Files without supported syntax diagnostics are skipped silently during save.',
    'project.configurationWorkspace.copy.validateNoFile': 'Open a supported file before running validation.',
  },
  'zh-CN': {
    'project.route.configurationWorkspace.title': '配置工作台',
    'project.configurationWorkspace.copy.annotationAction': '编辑注释',
    'project.configurationWorkspace.copy.annotationSaveFailed': '注释保存失败。',
    'project.configurationWorkspace.copy.batchFileValidationRiskBody':
      '这些文件仍存在语法错误。继续保存后，后续校验或部署可能无法正确解析项目配置。',
    'project.configurationWorkspace.copy.batchFileValidationRiskTitle': '带语法错误保存文件',
    'project.configurationWorkspace.copy.batchFileValidationTitle': '文件语法错误',
    'project.configurationWorkspace.copy.confirmSaveAllAction': '确认保存全部',
    'project.configurationWorkspace.copy.confirmSaveAllWithErrorsAction': '仍然保存全部',
    'project.configurationWorkspace.copy.confirmSaveCurrentAction': '确认保存',
    'project.configurationWorkspace.copy.confirmSaveRedeployWithErrorsAction': '仍然保存并重新部署',
    'project.configurationWorkspace.copy.confirmSaveAction': '确认保存',
    'project.configurationWorkspace.copy.confirmSaveWithErrorsAction': '仍然保存',
    'project.configurationWorkspace.copy.redeployAction': '重新部署项目',
    'project.configurationWorkspace.copy.diffCurrentFileConfirmBody':
      '请先确认当前文件差异。只有在你确认后，修改才会写入工作目录。',
    'project.configurationWorkspace.copy.diffCurrentFileTitle': '当前文件差异',
    'project.configurationWorkspace.copy.diffConfirmBody':
      '请先确认下面的变更文件。只有在你确认后，草稿才会写入工作目录。',
    'project.configurationWorkspace.copy.diffEmptyDirectSaveHint': '未检测到文件差异，将直接保存草稿文件。',
    'project.configurationWorkspace.copy.diffTreeTitle': '变更文件',
    'project.configurationWorkspace.copy.fileValidationEmbeddedHint':
      '当前文件存在语法错误，请先检查高亮位置，再决定是否保存或部署。',
    'project.configurationWorkspace.copy.fileValidationFailed': '当前文件存在语法错误。',
    'project.configurationWorkspace.copy.fileValidationPassed': '当前文件未检测到语法错误。',
    'project.configurationWorkspace.copy.fileValidationTitle': '文件校验',
    'project.configurationWorkspace.copy.fileValidationUnavailable': '当前文件类型暂不支持显式语法校验。',
    'project.configurationWorkspace.copy.resultSummaryChangedFilesLabel': '变更文件',
    'project.configurationWorkspace.copy.resultSummaryCurrentErrorsLabel': '当前错误',
    'project.configurationWorkspace.copy.resultSummaryCurrentFileLabel': '当前文件',
    'project.configurationWorkspace.copy.resultSummaryErrorFilesLabel': '错误文件',
    'project.configurationWorkspace.copy.saveAction': '保存',
    'project.configurationWorkspace.copy.saveAllAction': '保存全部',
    'project.configurationWorkspace.copy.savePartialHint': '这次只会保存当前文件。',
    'project.configurationWorkspace.copy.selectSyntaxFile': '选择一个文件查看语法错误',
    'project.configurationWorkspace.copy.syntaxErrorCountLabel': '{count} 处错误',
    'project.configurationWorkspace.copy.syntaxFileTreeTitle': '存在语法错误的文件',
    'project.configurationWorkspace.copy.saveThenContinueAction': '保存',
    'project.configurationWorkspace.copy.validateAction': '校验',
    'project.configurationWorkspace.copy.validateSkipUnsupportedHint': '保存时会静默跳过不支持语法诊断的文件。',
    'project.configurationWorkspace.copy.validateNoFile': '请先打开一个可校验的文件。',
  },
} as const;

type PendingApplicationDetail = {
  compose_project_name: string;
  display_name: string;
  application_id: string;
  ownership_mode: string;
  runtime_status: string;
  workspace_path: string;
};

vi.mock('../../api/project', () => ({
  deleteApplicationWorkspaceEntry: mocks.deleteApplicationWorkspaceEntry,
  getApplication: mocks.getApplication,
  getApplicationConfiguration: mocks.getApplicationConfiguration,
  getApplicationFileContent: mocks.getApplicationFileContent,
  getApplicationFiles: mocks.getApplicationFiles,
  postApplicationRedeploy: mocks.postApplicationRedeploy,
  postApplicationWorkspaceEntry: mocks.postApplicationWorkspaceEntry,
  postApplicationWorkspaceRename: mocks.postApplicationWorkspaceRename,
  putApplicationFileAnnotation: mocks.putApplicationFileAnnotation,
  putApplicationFileContent: mocks.putApplicationFileContent,
}));

vi.mock('../../shared/page-context', () => ({
  useApplicationPageContext: () => ({
    locale: computed(() => pageContextState.locale),
    tabsRouterStore: tabsRouterStoreMock,
    t: (key: string) =>
      workspaceCopyMessages[pageContextState.locale as keyof typeof workspaceCopyMessages]?.[
        key as keyof (typeof workspaceCopyMessages)[keyof typeof workspaceCopyMessages]
      ] ?? mocks.t(key),
  }),
}));

const dialogMocks = vi.hoisted(() => ({
  confirm: vi.fn(),
}));

vi.mock('tdesign-vue-next/es/dialog', () => ({
  DialogPlugin: {
    confirm: (options: unknown) => {
      dialogMocks.confirm(options);
      return { destroy: vi.fn() };
    },
  },
}));

vi.mock('@/modules/task/contract/task-ui', () => ({
  TaskDetailDrawer: defineComponent({
    name: 'TaskDetailDrawerStub',
    props: {
      taskId: { type: Number, default: null },
      visible: { type: Boolean, default: false },
    },
    setup(props) {
      return () => h('div', { 'data-task-id': String(props.taskId ?? ''), 'data-visible': String(props.visible) });
    },
  }),
}));

vi.mock('../../shared/navigation', () => ({
  buildDetailTitleWithFallback: (titleKey: string, name: string) => {
    const baseZh =
      workspaceCopyMessages['zh-CN'][titleKey as keyof (typeof workspaceCopyMessages)['zh-CN']] ?? titleKey;
    const baseEn =
      workspaceCopyMessages['en-US'][titleKey as keyof (typeof workspaceCopyMessages)['en-US']] ?? titleKey;
    const normalizedName = name.trim();
    if (!normalizedName || normalizedName === baseZh || normalizedName === baseEn) {
      return {
        'en-US': String(baseEn),
        'zh-CN': String(baseZh),
      };
    }
    return {
      'en-US': `${String(baseEn)} - ${normalizedName}`,
      'zh-CN': `${String(baseZh)} - ${normalizedName}`,
    };
  },
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
      modelKey: { type: String, default: '' },
      modelValue: { type: String, default: '' },
      testId: { type: String, default: 'monaco-stub' },
      readOnly: { type: Boolean, default: false },
    },
    emits: ['update:modelValue'],
    setup(props, { emit, expose }) {
      let renderedModelKey = String(props.modelKey);
      let renderedModelValue = String(props.modelValue);
      let pendingModelKey: string | null = null;
      let pendingModelValue = '';
      let pendingLagSteps = 0;

      const resolveRenderedModel = () => {
        if (props.readOnly) {
          renderedModelKey = String(props.modelKey);
          renderedModelValue = String(props.modelValue);
          return { modelKey: renderedModelKey, modelValue: renderedModelValue };
        }

        if (pendingModelKey && pendingLagSteps > 0) {
          pendingLagSteps -= 1;
          return { modelKey: renderedModelKey, modelValue: renderedModelValue };
        }

        if (pendingModelKey) {
          renderedModelKey = pendingModelKey;
          renderedModelValue = pendingModelValue;
          pendingModelKey = null;
        }

        return { modelKey: renderedModelKey, modelValue: renderedModelValue };
      };

      watch(
        () => [String(props.modelKey), String(props.modelValue)] as const,
        ([nextModelKey, nextModelValue]) => {
          if (props.readOnly) {
            renderedModelKey = nextModelKey;
            renderedModelValue = nextModelValue;
            pendingModelKey = null;
            pendingLagSteps = 0;
            return;
          }

          const lagSteps = monacoSurfaceState.modelKeyLagByPath[nextModelKey] ?? 0;
          if (lagSteps > 0) {
            pendingModelKey = nextModelKey;
            pendingModelValue = nextModelValue;
            pendingLagSteps = lagSteps;
            return;
          }

          renderedModelKey = nextModelKey;
          renderedModelValue = nextModelValue;
          pendingModelKey = null;
          pendingLagSteps = 0;
        },
        { immediate: true },
      );

      const getMarkers = () => {
        const { modelKey, modelValue } = resolveRenderedModel();
        return monacoSurfaceState.diagnosticsResolver
          ? monacoSurfaceState.diagnosticsResolver({
              modelKey,
              modelValue,
            })
          : modelValue.includes('[broken')
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
      };

      expose({
        getModelKey: () => resolveRenderedModel().modelKey,
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

function mockTwoFileSyntaxFixture() {
  mocks.getApplicationFiles.mockImplementation(
    async (_id: string, query?: { path?: string; show_hidden?: boolean }) => {
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
            {
              editable: true,
              file_kind: 'config',
              has_children: false,
              language_hint: 'yaml',
              name: 'app.yaml',
              node_type: 'file',
              readable: true,
              relative_path: 'config/app.yaml',
              size_bytes: 24,
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
    },
  );
  mocks.getApplicationFileContent.mockImplementation(async (_id: string, query: { path: string }) => ({
    content:
      query.path === 'docker-compose.yml'
        ? 'services:\n  api:\n    image: app\n'
        : query.path === 'config/app.yaml'
          ? 'name: demo\nenabled: true\n'
          : query.path === 'config/.env'
            ? 'APP_ENV=development\n'
            : '',
    editable: true,
    encoding: 'utf-8',
    file_kind: query.path === 'docker-compose.yml' ? 'compose' : query.path === 'config/.env' ? 'env' : 'config',
    language_hint: query.path === 'config/.env' ? 'dotenv' : 'yaml',
    readable: true,
    relative_path: query.path,
    size_bytes: 32,
  }));
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
    return () => h('div', { class: 't-tabs-stub' }, [slots.default?.(), slots.action?.()]);
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

describe('ApplicationConfigurationWorkspaceIndex', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    monacoSurfaceState.diagnosticsResolver = null;
    monacoSurfaceState.modelKeyLagByPath = {};
    pageContextState.locale = 'en-US';
    routeState.query = { name: 'sub2api' };
    tabsRouterStoreMock.tabRouterList = [
      {
        fullPath: '/applications/app_1/configuration?name=sub2api',
        path: '/applications/app_1/configuration',
        tabKey: '/applications/app_1/configuration',
        title: { 'en-US': 'Application Detail - sub2api', 'zh-CN': '项目详情 - sub2api' },
      },
    ];
    mocks.getApplication.mockResolvedValue({
      compose_project_name: 'sub2api',
      display_name: 'sub2api',
      application_id: 'app_1',
      ownership_mode: 'managed-root-dedicated',
      runtime_status: 'running',
      workspace_path: '/srv/sub2api',
    });
    mocks.getApplicationConfiguration.mockResolvedValue({
      drift_status: 'clean',
      ownership_mode: 'managed-root-dedicated',
      application_id: 'app_1',
    });
    mocks.getApplicationFiles.mockImplementation(
      async (_id: string, query?: { path?: string; show_hidden?: boolean }) => {
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
      },
    );
    mocks.getApplicationFileContent.mockImplementation(async (_id: string, query: { path: string }) => ({
      content: query.path === 'docker-compose.yml' ? 'services:\n  api:\n    image: app\n' : '',
      editable: query.path !== 'notes.txt',
      encoding: 'utf-8',
      file_kind: query.path === 'docker-compose.yml' ? 'compose' : 'env',
      language_hint: query.path === 'docker-compose.yml' ? 'yaml' : 'dotenv',
      readable: true,
      relative_path: query.path,
      size_bytes: query.path === 'docker-compose.yml' ? 32 : 0,
    }));
    mocks.putApplicationFileContent.mockResolvedValue({
      content_hash: 'saved-hash',
      relative_path: 'docker-compose.yml',
      saved_at: '2026-07-06T10:00:00Z',
      size_bytes: 40,
    });
    mocks.putApplicationFileAnnotation.mockResolvedValue({
      editable: true,
      file_kind: 'compose',
      has_children: false,
      hidden_by_default: false,
      language_hint: 'yaml',
      name: 'docker-compose.yml',
      node_type: 'file',
      application_note: 'Existing note',
      readable: true,
      relative_path: 'docker-compose.yml',
      size_bytes: 32,
      tooltip: 'Existing note',
      tooltip_source: 'application-note',
    });
    mocks.postApplicationRedeploy.mockResolvedValue({
      task_id: 42,
    });
    mocks.postApplicationWorkspaceEntry.mockResolvedValue({ path: 'notes.txt' });
    mocks.postApplicationWorkspaceRename.mockResolvedValue({ path: 'renamed.txt' });
    mocks.deleteApplicationWorkspaceEntry.mockResolvedValue({ path: 'docker-compose.yml' });
  });

  it('loads the root file list and opens the first file buffer', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    expect(mocks.getApplicationFiles).toHaveBeenCalledWith('app_1', { path: undefined, show_hidden: false });
    expect(mocks.getApplicationFileContent).toHaveBeenCalledWith('app_1', { path: 'docker-compose.yml' });
    expect((wrapper.get('[data-testid="workspace-monaco-editor"]').element as HTMLTextAreaElement).value).toBe(
      'services:\n  api:\n    image: app\n',
    );
  });

  it('saves the active dirty file only when Ctrl+S originates in the workspace root', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: newer\n');

    const outsideShortcut = new KeyboardEvent('keydown', {
      bubbles: true,
      cancelable: true,
      code: 'KeyS',
      ctrlKey: true,
      key: 's',
    });
    document.body.dispatchEvent(outsideShortcut);
    await flushPromises();

    expect(outsideShortcut.defaultPrevented).toBe(false);
    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(false);

    const workspaceShortcut = new KeyboardEvent('keydown', {
      bubbles: true,
      cancelable: true,
      code: 'KeyS',
      ctrlKey: true,
      key: 's',
    });
    wrapper.get('.project-configuration-workspace').element.dispatchEvent(workspaceShortcut);
    await flushPromises();

    expect(workspaceShortcut.defaultPrevented).toBe(true);
    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(true);
  });

  it('renders the workspace header with the standalone route title', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    expect(wrapper.find('h1').text()).toBe('sub2api · Configuration Workspace');
  });

  it('middle-truncates long workspace paths while keeping full values in tooltips', async () => {
    const fullWorkspacePath = '/srv/graft/releases/2026/06/14/shared-postgres/configuration';
    mocks.getApplication.mockResolvedValueOnce({
      compose_project_name: 'sub2api',
      display_name: 'sub2api',
      application_id: 'app_1',
      ownership_mode: 'managed-root-dedicated',
      runtime_status: 'running',
      workspace_path: fullWorkspacePath,
    });

    const wrapper = mountWorkspace();
    await flushPromises();

    expect(wrapper.get('[data-testid="workspace-working-directory"]').text()).toBe('/srv/g...ration');
    expect(wrapper.get('[data-testid="workspace-working-directory"]').attributes('aria-label')).toBe(fullWorkspacePath);
    expect(wrapper.text()).not.toContain(fullWorkspacePath);
    expect(wrapper.find(`[data-tooltip-content="${fullWorkspacePath}"]`).exists()).toBe(true);
  });

  it('delegates the current tab title update to the active workspace tab', async () => {
    mountWorkspace();
    await flushPromises();

    expect(tabsRouterStoreMock.updateActiveTabTitle).toHaveBeenCalledWith(
      'ApplicationConfigurationWorkspaceIndex',
      routeState,
      {
        'en-US': 'Configuration Workspace - sub2api',
        'zh-CN': '配置工作台 - sub2api',
      },
    );
  });

  it('keeps the standalone workspace header title before the detail request resolves', async () => {
    let resolveApplication: (value: PendingApplicationDetail) => void = () => {
      throw new Error('Expected pending project resolver to be assigned');
    };
    mocks.getApplication.mockReturnValueOnce(
      new Promise<PendingApplicationDetail>((resolve) => {
        resolveApplication = resolve;
      }),
    );

    const wrapper = mountWorkspace();
    await flushPromises();

    expect(wrapper.find('h1').text()).toBe('sub2api · Configuration Workspace');

    resolveApplication({
      compose_project_name: 'sub2api',
      display_name: 'sub2api',
      application_id: 'app_1',
      ownership_mode: 'managed-root-dedicated',
      runtime_status: 'running',
      workspace_path: '/srv/sub2api',
    });
    await flushPromises();

    expect(wrapper.find('h1').text()).toBe('sub2api · Configuration Workspace');
  });

  it('navigates into nested directories and opens empty file content without a blank editor failure', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    expect(mocks.getApplicationFiles).toHaveBeenCalledWith('app_1', { path: 'config', show_hidden: false });

    await wrapper.get('[data-testid="workspace-entry-config-env"]').trigger('click');
    await flushPromises();

    expect(mocks.getApplicationFileContent).toHaveBeenCalledWith('app_1', { path: 'config/.env' });
    expect(wrapper.find('[data-testid="workspace-monaco-editor"]').exists()).toBe(true);
    expect((wrapper.get('[data-testid="workspace-monaco-editor"]').element as HTMLTextAreaElement).value).toBe('');
  });

  it('keeps .env visible during directory browsing without enabling show_hidden', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="workspace-entry-config-env"]').exists()).toBe(true);
    expect(mocks.getApplicationFiles).toHaveBeenNthCalledWith(2, 'app_1', { path: 'config', show_hidden: false });
  });

  it('does not refetch a loaded empty file when reopening the tab with unsaved edits', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-entry-config-env"]').trigger('click');
    await flushPromises();

    const initialCalls = mocks.getApplicationFileContent.mock.calls.filter(
      ([, query]) => query.path === 'config/.env',
    ).length;
    expect(initialCalls).toBe(1);

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('EDITOR_ONLY=true\n');
    await wrapper.get('[data-testid="workspace-entry-config-env"]').trigger('click');
    await flushPromises();

    const nextCalls = mocks.getApplicationFileContent.mock.calls.filter(
      ([, query]) => query.path === 'config/.env',
    ).length;
    expect(nextCalls).toBe(1);
    expect((wrapper.get('[data-testid="workspace-monaco-editor"]').element as HTMLTextAreaElement).value).toBe(
      'EDITOR_ONLY=true\n',
    );
  });

  it('keeps the previous editor content visible while the next file is still loading', async () => {
    let resolveEnvContent!: (value: ApplicationWorkspaceFileContentResponse) => void;

    mocks.getApplicationFileContent.mockImplementation((_id: string, query: { path: string }) => {
      if (query.path === 'config/.env') {
        return new Promise((resolve) => {
          resolveEnvContent = resolve as (value: ApplicationWorkspaceFileContentResponse) => void;
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

    const composeCallsBefore = mocks.getApplicationFileContent.mock.calls.filter(
      ([, query]) => query.path === 'docker-compose.yml',
    ).length;

    await wrapper
      .get('[data-testid="workspace-file-tab-menu-workspace-entry-docker-compose-yml-refresh"]')
      .trigger('click');
    await flushPromises();

    const composeCallsAfter = mocks.getApplicationFileContent.mock.calls.filter(
      ([, query]) => query.path === 'docker-compose.yml',
    ).length;
    expect(composeCallsAfter).toBe(composeCallsBefore + 1);
    expect((wrapper.get('[data-testid="workspace-monaco-editor"]').element as HTMLTextAreaElement).value).toBe('');
  });

  it('lets the workspace page exit fullscreen on escape', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    expect(wrapper.get('.content-viewer-frame-stub').attributes('data-fill-height')).toBe('false');

    await wrapper.get('[data-testid="workspace-fullscreen-toggle"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('.project-configuration-workspace').classes()).toContain(
      'project-configuration-workspace--fullscreen',
    );
    expect(wrapper.get('.content-viewer-frame-stub').attributes('data-fill-height')).toBe('true');
    expect(document.body.style.overflow).toBe('hidden');

    window.dispatchEvent(new KeyboardEvent('keydown', { code: 'Escape', key: 'Escape' }));
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

    expect(wrapper.find('[data-testid="workspace-entry-docker-compose-yml-annotation"]').exists()).toBe(false);
    await wrapper.get('[data-testid="workspace-entry-docker-compose-yml"]').trigger('contextmenu', {
      clientX: 24,
      clientY: 24,
    });
    expect(wrapper.find('[data-testid="workspace-entry-docker-compose-yml-annotation"]').exists()).toBe(true);
    expect(wrapper.find('.project-configuration-workspace__editor-head').exists()).toBe(false);
    expect(wrapper.find('.project-configuration-workspace__browser-toolbar').exists()).toBe(false);
    expect(wrapper.find('.project-configuration-workspace__feedback').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('current directory still contains default-hidden directories');
  });

  it('shows compact tree toolbar actions and opens the annotation editor', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    expect(wrapper.find('.project-workspace-editor__tree.graft-scrollbar').exists()).toBe(true);
    const hiddenToggle = wrapper.get('[data-testid="workspace-show-hidden-toggle"]');
    expect(hiddenToggle.element.parentElement?.getAttribute('data-tooltip-content')).toBe(
      'project.configurationWorkspace.copy.showHiddenAction',
    );
    expect(hiddenToggle.text()).toBe('');

    await wrapper.get('[data-testid="workspace-entry-docker-compose-yml"]').trigger('contextmenu', {
      clientX: 24,
      clientY: 24,
    });
    const annotationButton = wrapper.get('[data-testid="workspace-entry-docker-compose-yml-annotation"]');
    expect(annotationButton.text()).toBe('Edit Annotation');

    await annotationButton.trigger('click');
    expect(
      wrapper
        .findAll('[data-stub="TDialog"]')
        .find((dialog) => dialog.attributes('data-title') === 'Edit Annotation')
        ?.attributes('data-visible'),
    ).toBe('true');
  });

  it('creates a workspace file through the shared tree context menu and refreshes the browser', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.find('.project-workspace-editor__tree').trigger('contextmenu', { clientX: 24, clientY: 24 });
    await nextTick();
    await wrapper.get('[role="menuitem"]').trigger('click');
    await wrapper.find('input').setValue('notes.txt');
    wrapper.findComponent({ name: 'ApplicationWorkspaceEditor' }).vm.$emit('inline-edit-submit');
    await flushPromises();

    expect(mocks.postApplicationWorkspaceEntry).toHaveBeenCalledWith('app_1', {
      content: '',
      node_type: 'file',
      path: 'notes.txt',
    });
    expect(mocks.getApplicationFiles).toHaveBeenCalledWith('app_1', { path: undefined, show_hidden: false });
  });

  it('renames a workspace entry through the shared inline editor', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-docker-compose-yml"]').trigger('contextmenu', {
      clientX: 24,
      clientY: 24,
    });
    await wrapper
      .findAll('[role="menuitem"]')
      .find((menuItem) => menuItem.text() === 'project.create.workspace.rename')
      ?.trigger('click');
    await wrapper.find('input').setValue('runtime.yml');
    wrapper.findComponent({ name: 'ApplicationWorkspaceEditor' }).vm.$emit('inline-edit-submit');
    await flushPromises();

    expect(mocks.postApplicationWorkspaceRename).toHaveBeenCalledWith('app_1', {
      path: 'docker-compose.yml',
      new_path: 'runtime.yml',
    });
  });

  it('saves workspace annotations through the dialog flow', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-docker-compose-yml"]').trigger('contextmenu', {
      clientX: 24,
      clientY: 24,
    });
    await wrapper.get('[data-testid="workspace-entry-docker-compose-yml-annotation"]').trigger('click');
    await wrapper.findAll('textarea').at(-1)!.setValue('Updated note');
    await wrapper.get('[data-testid="t-dialog-confirm"]').trigger('click');
    await flushPromises();

    expect(mocks.putApplicationFileAnnotation).toHaveBeenCalledWith(
      'app_1',
      { path: 'docker-compose.yml' },
      { annotation: 'Updated note' },
    );
    expect(
      wrapper
        .findAll('[data-stub="TDialog"]')
        .find((dialog) => dialog.attributes('data-title') === 'Edit Annotation')
        ?.attributes('data-visible'),
    ).toBe('false');
    expect(mocks.error).not.toHaveBeenCalled();
  });

  it('shows a localized annotation save failure instead of the retired unavailable fallback copy', async () => {
    const wrapper = mountWorkspace();
    mocks.putApplicationFileAnnotation.mockRejectedValueOnce(new Error('save failed'));
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-docker-compose-yml"]').trigger('contextmenu', {
      clientX: 24,
      clientY: 24,
    });
    await wrapper.get('[data-testid="workspace-entry-docker-compose-yml-annotation"]').trigger('click');
    await wrapper.findAll('textarea').at(-1)!.setValue('Updated note');
    await wrapper.get('[data-testid="t-dialog-confirm"]').trigger('click');
    await flushPromises();

    expect(mocks.error).toHaveBeenCalledWith('Failed to save the annotation.');
  });

  it('shows the zh-CN annotation save failure copy when the request fails', async () => {
    pageContextState.locale = 'zh-CN';
    const wrapper = mountWorkspace();
    mocks.putApplicationFileAnnotation.mockRejectedValueOnce(new Error('save failed'));
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-docker-compose-yml"]').trigger('contextmenu', {
      clientX: 24,
      clientY: 24,
    });
    await wrapper.get('[data-testid="workspace-entry-docker-compose-yml-annotation"]').trigger('click');
    await wrapper.findAll('textarea').at(-1)!.setValue('更新备注');
    await wrapper.get('[data-testid="t-dialog-confirm"]').trigger('click');
    await flushPromises();

    expect(mocks.error).toHaveBeenCalledWith('注释保存失败。');
  });

  it('hides a directory error after the directory is collapsed', async () => {
    mocks.getApplicationFiles.mockImplementationOnce(async () => ({
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
    mocks.getApplicationFiles.mockRejectedValueOnce(new Error('Directory load failed'));
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

    expect(mocks.putApplicationFileContent).not.toHaveBeenCalled();
    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="syntax-monaco-viewer"]').exists()).toBe(true);
    expect(wrapper.text()).toContain('文件校验');
    expect(wrapper.html()).toContain('data-message="当前文件存在语法错误，请先检查高亮位置，再决定是否保存或部署。"');
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

  it('intercepts save when the current file has syntax errors and only saves after explicit confirmation', async () => {
    pageContextState.locale = 'zh-CN';
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: [broken\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === '保存')
      ?.trigger('click');
    await flushPromises();

    expect(mocks.putApplicationFileContent).not.toHaveBeenCalled();
    await wrapper.get('[data-testid="configuration-diff-confirm-save"]').trigger('click');
    await flushPromises();
    expect(wrapper.find('[data-testid="syntax-monaco-viewer"]').exists()).toBe(true);

    await wrapper.get('[data-testid="configuration-syntax-confirm-save"]').trigger('click');
    await flushPromises();
    await wrapper
      .find('[data-title="带语法错误保存文件"]')
      .findAll('button')
      .find((button) => button.text().trim() === '仍然保存')
      ?.trigger('click');
    await flushPromises();

    expect(mocks.putApplicationFileContent).toHaveBeenCalledWith(
      'app_1',
      { path: 'docker-compose.yml' },
      { content: 'services:\n  api:\n    image: [broken\n' },
    );
  });

  it('uses a single-pane diff for saving only the current file', async () => {
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
    expect(wrapper.find('.project-configuration-workspace__diff-sidebar').exists()).toBe(false);
    expect(wrapper.find('[data-class-name*="project-configuration-workspace__result-dialog-shell"]').html()).toContain(
      'Current File Diff',
    );
  });

  it('keeps the diff file list in workspace tree form for save all', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: newer\n');
    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushDiffViewerFrames();
    await wrapper.get('[data-testid="workspace-entry-config-env"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('APP_ENV=prod\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save All')
      ?.trigger('click');
    await flushDiffViewerFrames();

    expect(wrapper.find('[data-testid="configuration-diff-viewer"]').exists()).toBe(true);
    expect(wrapper.find('.project-configuration-workspace__diff-sidebar').exists()).toBe(true);
    expect(wrapper.text()).toContain('docker-compose.yml');
    expect(wrapper.text()).toContain('.env');
    expect(wrapper.find('.project-configuration-workspace__diff-sidebar .t-icon-folder').exists()).toBe(true);
    expect(wrapper.find('.project-configuration-workspace__diff-sidebar .t-icon-command').exists()).toBe(true);
  });

  it('shows the syntax file list for save all when multiple files have syntax errors', async () => {
    mockTwoFileSyntaxFixture();

    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: [broken\n');
    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-entry-config-app-yaml"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('name: [broken\nenabled: true\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save All')
      ?.trigger('click');
    await waitForDiffViewer(wrapper, '[data-testid="configuration-diff-confirm-save"]');
    await wrapper.get('[data-testid="configuration-diff-confirm-save"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="syntax-monaco-viewer"]').exists()).toBe(true);
    expect(wrapper.find('.project-configuration-workspace__diff-sidebar').exists()).toBe(true);
    expect(wrapper.text()).toContain('File Validation Errors');
    expect(wrapper.text()).toContain(
      'These files still have syntax errors. Saving them may leave the project in a state that later validation or deploy cannot parse correctly.',
    );
    expect(wrapper.text()).toContain('docker-compose.yml');
    expect(wrapper.text()).toContain('app.yaml');
    expect(wrapper.find('.project-configuration-workspace__diff-sidebar .t-icon-folder').exists()).toBe(true);
    expect(wrapper.find('.project-configuration-workspace__diff-sidebar .t-icon-file-code').exists()).toBe(true);
  });

  it('rechecks unresolved files so the first batch validation still includes every error file', async () => {
    const diagnosticsCallCount = new Map<string, number>();
    monacoSurfaceState.diagnosticsResolver = ({ modelKey, modelValue }) => {
      const nextCount = (diagnosticsCallCount.get(modelKey) ?? 0) + 1;
      diagnosticsCallCount.set(modelKey, nextCount);
      if (modelKey === 'config/app.yaml' && nextCount === 1) {
        return [];
      }

      return String(modelValue).includes('[broken')
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
    };

    mockTwoFileSyntaxFixture();

    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: [broken\n');
    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-entry-config-app-yaml"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('name: [broken\nenabled: true\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save All')
      ?.trigger('click');
    await waitForDiffViewer(wrapper, '[data-testid="configuration-diff-confirm-save"]');
    await wrapper.get('[data-testid="configuration-diff-confirm-save"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('Error Files');
    expect(wrapper.text()).toContain('2');
    expect(wrapper.text()).toContain('docker-compose.yml');
    expect(wrapper.text()).toContain('app.yaml');
  });

  it('warns when save all includes dirty files without supported syntax diagnostics', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: newer\n');
    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-entry-config-env"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('APP_ENV=prod\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save All')
      ?.trigger('click');
    await flushDiffViewerFrames();
    await wrapper.get('[data-testid="configuration-diff-confirm-save"]').trigger('click');
    await flushPromises();

    expect(mocks.warning).toHaveBeenCalledWith(
      'Files without supported syntax diagnostics are skipped silently during save. config/.env',
    );
    expect(mocks.putApplicationFileContent).not.toHaveBeenCalled();
    expect(wrapper.text()).toContain('config/.env');
    expect(wrapper.text()).toContain('Confirm Save All');

    await wrapper.get('[data-testid="workspace-dialog-save"]').trigger('click');
    await flushPromises();

    expect(mocks.putApplicationFileContent).toHaveBeenCalledWith(
      'app_1',
      { path: 'docker-compose.yml' },
      { content: 'services:\n  api:\n    image: newer\n' },
    );
    expect(mocks.putApplicationFileContent).toHaveBeenCalledWith(
      'app_1',
      { path: 'config/.env' },
      { content: 'APP_ENV=prod\n' },
    );
  });

  it('requires explicit confirmation before saving the current unsupported file type', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-entry-config-env"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('APP_ENV=prod\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save')
      ?.trigger('click');
    await flushDiffViewerFrames();
    await wrapper.get('[data-testid="configuration-diff-confirm-save"]').trigger('click');
    await flushPromises();

    expect(mocks.warning).toHaveBeenCalledWith(
      'Files without supported syntax diagnostics are skipped silently during save. config/.env',
    );
    expect(mocks.putApplicationFileContent).not.toHaveBeenCalled();
    expect(wrapper.text()).toContain('config/.env');
    expect(wrapper.text()).toContain('Confirm Save');

    await wrapper.get('[data-testid="workspace-dialog-save"]').trigger('click');
    await flushPromises();

    expect(mocks.putApplicationFileContent).toHaveBeenCalledWith(
      'app_1',
      { path: 'config/.env' },
      { content: 'APP_ENV=prod\n' },
    );
  });

  it('requires skipped-file confirmation after syntax confirmation for mixed save-all batches', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: [broken\n');
    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushDiffViewerFrames();
    await wrapper.get('[data-testid="workspace-entry-config-env"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('APP_ENV=prod\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save All')
      ?.trigger('click');
    await flushDiffViewerFrames();
    await wrapper.get('[data-testid="configuration-diff-confirm-save"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="syntax-monaco-viewer"]').exists()).toBe(true);

    await wrapper.get('[data-testid="configuration-syntax-confirm-save"]').trigger('click');
    await flushPromises();
    await wrapper
      .find('[data-title="Save Files with Syntax Errors"]')
      .findAll('button')
      .find((button) => button.text().trim() === 'Save All Anyway')
      ?.trigger('click');
    await flushPromises();

    expect(mocks.putApplicationFileContent).not.toHaveBeenCalled();
    expect(wrapper.text()).toContain('config/.env');
    expect(wrapper.text()).toContain('Confirm Save All');

    await wrapper.get('[data-testid="workspace-dialog-save"]').trigger('click');
    await flushPromises();

    expect(mocks.putApplicationFileContent).toHaveBeenCalledWith(
      'app_1',
      { path: 'docker-compose.yml' },
      { content: 'services:\n  api:\n    image: [broken\n' },
    );
    expect(mocks.putApplicationFileContent).toHaveBeenCalledWith(
      'app_1',
      { path: 'config/.env' },
      { content: 'APP_ENV=prod\n' },
    );
  });

  it('waits for delayed editor rebinding before collecting batch diagnostics and restoring the active tab', async () => {
    mocks.getApplicationFiles.mockImplementation(
      async (_id: string, query?: { path?: string; show_hidden?: boolean }) => {
        if (query?.path === 'config') {
          return {
            current_path: 'config',
            items: [
              {
                editable: true,
                file_kind: 'config',
                has_children: false,
                language_hint: 'yaml',
                name: 'app.yaml',
                node_type: 'file',
                readable: true,
                relative_path: 'config/app.yaml',
                size_bytes: 24,
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
      },
    );
    mocks.getApplicationFileContent.mockImplementation(async (_id: string, query: { path: string }) => ({
      content:
        query.path === 'docker-compose.yml'
          ? 'services:\n  api:\n    image: app\n'
          : query.path === 'config/app.yaml'
            ? 'name: demo\nenabled: true\n'
            : '',
      editable: true,
      encoding: 'utf-8',
      file_kind: query.path === 'docker-compose.yml' ? 'compose' : 'config',
      language_hint: 'yaml',
      readable: true,
      relative_path: query.path,
      size_bytes: 32,
    }));

    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: [broken\n');
    await wrapper.get('[data-testid="workspace-entry-config"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-entry-config-app-yaml"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('name: [broken\nenabled: true\n');

    monacoSurfaceState.modelKeyLagByPath = {
      'config/app.yaml': 8,
      'docker-compose.yml': 8,
    };

    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Save All')
      ?.trigger('click');
    await flushDiffViewerFrames();
    await wrapper.get('[data-testid="configuration-diff-confirm-save"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('Error Files');
    expect(wrapper.text()).toContain('2');
    expect(wrapper.text()).toContain('docker-compose.yml');
    expect(wrapper.text()).toContain('app.yaml');
    expect((wrapper.get('[data-testid="workspace-monaco-editor"]').element as HTMLTextAreaElement).value).toBe(
      'name: [broken\nenabled: true\n',
    );
  });

  it('previews the active dirty buffers before saving them to disk', async () => {
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: newer\n');
    wrapper.get('.project-configuration-workspace').element.dispatchEvent(
      new KeyboardEvent('keydown', {
        bubbles: true,
        cancelable: true,
        code: 'KeyS',
        ctrlKey: true,
        key: 's',
      }),
    );
    await flushPromises();

    expect(mocks.putApplicationFileContent).not.toHaveBeenCalled();
    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(true);
    expect(mocks.postApplicationRedeploy).not.toHaveBeenCalled();
  });

  it('confirms and redeploys only after preview confirm saves the dirty draft', async () => {
    pageContextState.locale = 'zh-CN';
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper.get('[data-testid="workspace-monaco-editor"]').setValue('services:\n  api:\n    image: newer\n');
    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === '重新部署项目')
      ?.trigger('click');
    await flushPromises();

    expect(dialogMocks.confirm).toHaveBeenCalledTimes(1);
    expect(mocks.putApplicationFileContent).not.toHaveBeenCalled();
    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(false);

    const dialogOptions = dialogMocks.confirm.mock.calls[0]?.[0] as { onConfirm?: () => void };
    dialogOptions.onConfirm?.();
    await flushPromises();

    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(true);

    await wrapper.get('[data-testid="configuration-diff-confirm-save"]').trigger('click');
    await flushPromises();

    expect(mocks.putApplicationFileContent).toHaveBeenCalledWith(
      'app_1',
      { path: 'docker-compose.yml' },
      { content: 'services:\n  api:\n    image: newer\n' },
    );
    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(false);
    expect(mocks.postApplicationRedeploy).toHaveBeenCalledWith('app_1');
    expect(wrapper.find('[data-task-id="42"]').attributes('data-visible')).toBe('true');
  });

  it('does not save or submit a redeploy when the confirmation is cancelled', async () => {
    pageContextState.locale = 'zh-CN';
    const wrapper = mountWorkspace();
    await flushPromises();

    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === '重新部署项目')
      ?.trigger('click');
    await vi.waitFor(() => expect(dialogMocks.confirm).toHaveBeenCalled());

    const dialogOptions = dialogMocks.confirm.mock.calls[0]?.[0] as { onCancel?: () => void };
    dialogOptions.onCancel?.();
    await flushPromises();

    expect(mocks.putApplicationFileContent).not.toHaveBeenCalled();
    expect(mocks.postApplicationRedeploy).not.toHaveBeenCalled();
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

    expect(mocks.putApplicationFileContent).not.toHaveBeenCalled();
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

    expect(mocks.putApplicationFileContent).toHaveBeenCalledWith(
      'app_1',
      { path: 'docker-compose.yml' },
      { content: 'services:\n  api:\n    image: app\n' },
    );
    expect(wrapper.find('[data-testid="configuration-diff-modal"]').exists()).toBe(false);
    expect(mocks.info).toHaveBeenCalledWith('No file diff was detected. Draft files will be saved directly.');
  });
});

function mountWorkspace() {
  return mount(ApplicationConfigurationWorkspaceIndex, {
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
        TInput: defineComponent({
          name: 'TInputStub',
          props: { modelValue: { type: String, default: '' } },
          emits: ['update:modelValue'],
          setup(props, { emit }) {
            return () =>
              h('input', {
                value: props.modelValue,
                onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
              });
          },
        }),
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
