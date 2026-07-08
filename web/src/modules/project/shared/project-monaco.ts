import 'monaco-editor/esm/vs/basic-languages/dockerfile/dockerfile.contribution';
import 'monaco-editor/esm/vs/basic-languages/hcl/hcl.contribution';
import 'monaco-editor/esm/vs/basic-languages/ini/ini.contribution';
import 'monaco-editor/esm/vs/basic-languages/markdown/markdown.contribution';
import 'monaco-editor/esm/vs/basic-languages/powershell/powershell.contribution';
import 'monaco-editor/esm/vs/basic-languages/shell/shell.contribution';
import 'monaco-editor/esm/vs/basic-languages/sql/sql.contribution';
import 'monaco-editor/esm/vs/basic-languages/xml/xml.contribution';
import 'monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution';
import 'monaco-editor/esm/vs/language/json/monaco.contribution';
import 'monaco-editor/min/vs/editor/editor.main.css';

import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';
import { configureMonacoYaml } from 'monaco-yaml';
import { nextTick, onBeforeUnmount, onMounted } from 'vue';

import { createLogger } from '@/utils/logger';

import { toMonacoColor } from './project-monaco-color';
import { createProjectMonacoDebugLogger } from './project-monaco-debug';
import { buildProjectMonacoWorker } from './project-monaco-worker';

export type MonacoEditorModule = typeof monaco;
type MonacoCreateWebWorker = typeof monaco.editor.createWebWorker;

type LegacyMonacoWebWorkerOptions = {
  createData?: unknown;
  createWorker?: () => Worker;
  host?: Record<string, (...args: unknown[]) => unknown>;
  keepIdleModels?: boolean;
  label?: string;
  moduleId: string;
};

declare global {
  interface Window {
    MonacoEnvironment?: {
      getWorker?: (_moduleId: string, label: string) => Worker;
    };
  }
}

let monacoConfigured = false;
let monacoYamlConfigured = false;
let monacoYamlConfigurationFailed = false;
let monacoLegacyWorkerCompatInstalled = false;
let modelUriSuffixSeed = 0;
const logger = createLogger('project.monaco');
const logProjectMonacoDebug = createProjectMonacoDebugLogger('project.monaco');

const PROJECT_MONACO_THEME_LIGHT = 'graft-project-workspace-light';
const PROJECT_MONACO_THEME_DARK = 'graft-project-workspace-dark';

export type ProjectMonacoTheme = typeof PROJECT_MONACO_THEME_LIGHT | typeof PROJECT_MONACO_THEME_DARK;

export function ensureProjectMonacoConfigured() {
  if (monacoConfigured) {
    logProjectMonacoDebug('reuse-existing-config', {});
    return monaco;
  }

  globalThis.MonacoEnvironment = {
    getWorker(moduleId: string, label: string) {
      logProjectMonacoDebug('get-worker', {
        moduleId,
        label,
      });

      const worker = buildProjectMonacoWorker(moduleId, label);

      logProjectMonacoDebug('get-worker-resolved', {
        constructorName: worker?.constructor?.name ?? 'unknown',
        moduleId,
        label,
      });

      return worker;
    },
  };

  ensureProjectMonacoLegacyWorkerCompat(monaco);

  if (!monacoYamlConfigured && !monacoYamlConfigurationFailed) {
    try {
      logProjectMonacoDebug('configure-yaml-start', {});
      configureMonacoYaml(monaco, {
        completion: true,
        enableSchemaRequest: false,
        hover: true,
        schemas: [],
        validate: true,
      });
      monacoYamlConfigured = true;
      logProjectMonacoDebug('configure-yaml-success', {});
    } catch (error) {
      monacoYamlConfigurationFailed = true;
      logProjectMonacoDebug('configure-yaml-failed', {
        error,
      });
      logger.error('Failed to configure project Monaco YAML worker integration.', {
        error,
      });
    }
  }

  monacoConfigured = true;
  logProjectMonacoDebug('config-complete', {});
  return monaco;
}

function ensureProjectMonacoLegacyWorkerCompat(monacoInstance: MonacoEditorModule) {
  if (monacoLegacyWorkerCompatInstalled) {
    return;
  }

  const editorApi = monacoInstance.editor as typeof monaco.editor & {
    createWebWorker: MonacoCreateWebWorker;
  };
  const originalCreateWebWorker = editorApi.createWebWorker.bind(editorApi);

  editorApi.createWebWorker = ((options: Parameters<MonacoCreateWebWorker>[0]) => {
    const legacyOptions = asLegacyMonacoWebWorkerOptions(options);

    if (!legacyOptions) {
      return originalCreateWebWorker(options);
    }

    const label = legacyOptions.label ?? 'monaco-editor-worker';
    const worker = Promise.resolve(resolveProjectMonacoWorkerFromLegacyOptions(label, legacyOptions.createWorker)).then(
      (resolvedWorker) => {
        resolvedWorker.postMessage('ignore');
        resolvedWorker.postMessage(legacyOptions.createData);
        return resolvedWorker;
      },
    );

    logProjectMonacoDebug('create-web-worker-compat', {
      label,
      moduleId: legacyOptions.moduleId,
    });

    return originalCreateWebWorker({
      host: legacyOptions.host,
      keepIdleModels: legacyOptions.keepIdleModels,
      worker,
    });
  }) as MonacoCreateWebWorker;

  monacoLegacyWorkerCompatInstalled = true;
}

function asLegacyMonacoWebWorkerOptions(
  options: Parameters<MonacoCreateWebWorker>[0],
): LegacyMonacoWebWorkerOptions | null {
  if (!options || typeof options !== 'object' || 'worker' in options || !('moduleId' in options)) {
    return null;
  }

  return options as unknown as LegacyMonacoWebWorkerOptions;
}

function resolveProjectMonacoWorkerFromLegacyOptions(label: string, createWorker?: () => Worker) {
  if (typeof createWorker === 'function') {
    return createWorker();
  }

  return buildProjectMonacoWorker('workerMain.js', label);
}

function resolveProjectMonacoTheme(): ProjectMonacoTheme {
  return document.documentElement.getAttribute('theme-mode') === 'dark'
    ? PROJECT_MONACO_THEME_DARK
    : PROJECT_MONACO_THEME_LIGHT;
}

function applyProjectMonacoTheme(monacoInstance: MonacoEditorModule, hostElement?: HTMLElement | null) {
  const theme = resolveProjectMonacoTheme();
  defineProjectMonacoTheme(monacoInstance, theme, hostElement);
  monacoInstance.editor.setTheme(theme);
}

export function scheduleProjectMonacoLayout(layout: () => void) {
  return nextTick().then(() => {
    requestAnimationFrame(layout);
  });
}

export function observeProjectMonacoResize(
  container: HTMLElement | null,
  observer: ResizeObserver | null,
  layout: () => void,
) {
  if (typeof ResizeObserver === 'undefined' || !container) {
    return observer;
  }

  observer?.disconnect();
  const nextObserver = new ResizeObserver(() => {
    void scheduleProjectMonacoLayout(layout);
  });
  nextObserver.observe(container);
  return nextObserver;
}

export function useProjectMonacoLifecycle(options: {
  createEditor: () => void | Promise<void>;
  disposeEditor: () => void;
  getMonaco: () => MonacoEditorModule | null;
  getThemeHost?: () => HTMLElement | null;
}) {
  let themeModeObserver: MutationObserver | null = null;

  const applyTheme = () => {
    const monacoInstance = options.getMonaco();
    if (!monacoInstance) {
      return;
    }
    applyProjectMonacoTheme(monacoInstance, options.getThemeHost?.() ?? null);
  };

  const observeThemeMode = () => {
    if (typeof MutationObserver === 'undefined') {
      return;
    }
    themeModeObserver?.disconnect();
    themeModeObserver = new MutationObserver(() => {
      applyTheme();
    });
    themeModeObserver.observe(document.documentElement, {
      attributeFilter: ['theme-mode'],
      attributes: true,
    });
  };

  onMounted(() => {
    applyTheme();
    observeThemeMode();
    void options.createEditor();
  });

  onBeforeUnmount(() => {
    themeModeObserver?.disconnect();
    themeModeObserver = null;
    options.disposeEditor();
  });

  return {
    applyTheme,
  };
}

function defineProjectMonacoTheme(
  monacoInstance: MonacoEditorModule,
  theme: ProjectMonacoTheme,
  hostElement?: HTMLElement | null,
) {
  const host = hostElement ?? document.documentElement;
  const base = theme === PROJECT_MONACO_THEME_DARK ? 'vs-dark' : 'vs';
  const background = resolveThemeColor(host, '--graft-workspace-editor-surface', '#1b2230');
  const raisedSurface = resolveThemeColor(host, '--graft-workspace-editor-surface-raised', '#212b3d');
  const mutedSurface = resolveThemeColor(host, '--graft-workspace-editor-surface-muted', '#162031');
  const border = resolveThemeColor(host, '--graft-workspace-editor-border', '#344056');
  const foreground = resolveThemeColor(host, '--graft-workspace-editor-foreground', '#e7edf7');
  const secondaryForeground = resolveThemeColor(host, '--graft-workspace-editor-foreground-muted', '#8d9bb2');
  const tertiaryForeground = resolveThemeColor(host, '--graft-workspace-editor-foreground-subtle', '#66758f');
  const accent = resolveThemeColor(host, '--graft-workspace-editor-accent', '#5f9bff');
  const lineHighlight = resolveThemeColor(host, '--graft-workspace-editor-line-highlight', 'rgb(95 155 255 / 0.12)');
  const selection = resolveThemeColor(host, '--graft-workspace-editor-selection', 'rgb(95 155 255 / 0.28)');
  const inactiveSelection = resolveThemeColor(
    host,
    '--graft-workspace-editor-selection-inactive',
    'rgb(95 155 255 / 0.18)',
  );
  const indentGuide = resolveThemeColor(host, '--graft-workspace-editor-indent-guide', 'rgb(116 133 160 / 0.28)');
  const activeIndentGuide = resolveThemeColor(
    host,
    '--graft-workspace-editor-indent-guide-active',
    'rgb(95 155 255 / 0.36)',
  );
  const findMatch = resolveThemeColor(host, '--graft-workspace-editor-find-match', 'rgb(95 155 255 / 0.24)');
  const findMatchBorder = resolveThemeColor(
    host,
    '--graft-workspace-editor-find-match-border',
    'rgb(95 155 255 / 0.52)',
  );
  const diffAdded = resolveThemeColor(host, '--graft-workspace-editor-diff-added', 'rgb(46 194 126 / 0.18)');
  const diffRemoved = resolveThemeColor(host, '--graft-workspace-editor-diff-removed', 'rgb(232 93 106 / 0.18)');

  monacoInstance.editor.defineTheme(theme, {
    base,
    inherit: true,
    colors: {
      'diffEditor.insertedLineBackground': diffAdded,
      'diffEditor.insertedTextBackground': diffAdded,
      'diffEditor.removedLineBackground': diffRemoved,
      'diffEditor.removedTextBackground': diffRemoved,
      'editor.background': background,
      'editor.foreground': foreground,
      'editor.lineHighlightBackground': lineHighlight,
      'editor.lineHighlightBorder': '#00000000',
      'editor.selectionBackground': selection,
      'editor.selectionHighlightBackground': inactiveSelection,
      'editor.inactiveSelectionBackground': inactiveSelection,
      'editor.wordHighlightBackground': inactiveSelection,
      'editor.wordHighlightStrongBackground': selection,
      'editorCursor.foreground': accent,
      'editorError.foreground': resolveThemeColor(host, '--td-error-color-6', '#e85d6a'),
      'editor.findMatchBackground': findMatch,
      'editor.findMatchBorder': findMatchBorder,
      'editor.findMatchHighlightBackground': inactiveSelection,
      'editor.findMatchHighlightBorder': '#00000000',
      'editor.findRangeHighlightBackground': inactiveSelection,
      'editorGutter.background': background,
      'editorGutter.modifiedBackground': resolveThemeColor(host, '--td-brand-color-6', '#5f9bff'),
      'editorHoverWidget.background': raisedSurface,
      'editorHoverWidget.border': border,
      'editorIndentGuide.activeBackground1': activeIndentGuide,
      'editorIndentGuide.background1': indentGuide,
      'editorLineNumber.activeForeground': secondaryForeground,
      'editorLineNumber.foreground': tertiaryForeground,
      'editorOverviewRuler.border': '#00000000',
      'editorSuggestWidget.background': raisedSurface,
      'editorSuggestWidget.border': border,
      'editorSuggestWidget.foreground': foreground,
      'editorSuggestWidget.highlightForeground': accent,
      'editorSuggestWidget.selectedBackground': mutedSurface,
      'editorWhitespace.foreground': indentGuide,
      'editorWidget.background': raisedSurface,
      'editorWidget.border': border,
      'input.background': mutedSurface,
      'input.border': border,
      'input.foreground': foreground,
      'scrollbar.shadow': '#00000000',
      'scrollbarSlider.activeBackground': selection,
      'scrollbarSlider.background': inactiveSelection,
      'scrollbarSlider.hoverBackground': findMatch,
    },
    rules: [],
  });
}

function resolveThemeColor(host: HTMLElement, tokenName: string, fallback: string) {
  const styles = getComputedStyle(host);
  const rawValue = styles.getPropertyValue(tokenName).trim();
  return resolveCssColor(host, rawValue || fallback, fallback);
}

function resolveCssColor(host: HTMLElement, value: string, fallback: string) {
  const probe = document.createElement('span');
  probe.style.color = fallback;
  probe.style.color = value;
  probe.style.position = 'absolute';
  probe.style.pointerEvents = 'none';
  probe.style.opacity = '0';
  host.appendChild(probe);
  const resolved = getComputedStyle(probe).color;
  probe.remove();
  return toMonacoColor(resolved, fallback);
}

export function createProjectMonacoModelUriSuffix() {
  modelUriSuffixSeed += 1;
  return `model-${modelUriSuffixSeed}`;
}

export function buildProjectMonacoModelUri(key: string, language: string, suffix?: string) {
  const normalizedKey = key.replace(/[^a-z0-9/_.-]/giu, '-');
  const extension = resolveLanguageExtension(language);
  const normalizedSuffix = suffix ? `-${suffix.replace(/[^a-z0-9/_.-]/giu, '-')}` : '';
  return monaco.Uri.parse(
    `inmemory://project-configuration-workspace/${normalizedKey}${normalizedSuffix}.${extension}`,
  );
}

function resolveLanguageExtension(language: string) {
  if (language === 'yaml') {
    return 'yaml';
  }
  if (language === 'json') {
    return 'json';
  }
  if (language === 'typescript') {
    return 'ts';
  }
  if (language === 'javascript') {
    return 'js';
  }
  if (language === 'shell') {
    return 'sh';
  }
  if (language === 'dockerfile') {
    return 'Dockerfile';
  }
  if (language === 'ini') {
    return 'ini';
  }
  return 'txt';
}
