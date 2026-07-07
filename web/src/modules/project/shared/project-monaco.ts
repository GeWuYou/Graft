import 'monaco-editor/esm/vs/basic-languages/dockerfile/dockerfile.contribution';
import 'monaco-editor/esm/vs/basic-languages/ini/ini.contribution';
import 'monaco-editor/esm/vs/basic-languages/shell/shell.contribution';
import 'monaco-editor/min/vs/editor/editor.main.css';

import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import { configureMonacoYaml } from 'monaco-yaml';
import { nextTick, onBeforeUnmount, onMounted } from 'vue';

import yamlWorker from './project-yaml.worker?worker';

export type MonacoEditorModule = typeof monaco;

declare global {
  interface Window {
    MonacoEnvironment?: {
      getWorker?: (_moduleId: string, label: string) => Worker;
    };
  }
}

let monacoConfigured = false;
let modelUriSuffixSeed = 0;

export type ProjectMonacoTheme = 'vs' | 'vs-dark';

function buildWorker(label: string) {
  switch (label) {
    case 'yaml':
      return new yamlWorker();
    default:
      return new editorWorker();
  }
}

export function ensureProjectMonacoConfigured() {
  if (monacoConfigured) {
    return monaco;
  }

  globalThis.MonacoEnvironment = {
    getWorker(_moduleId: string, label: string) {
      return buildWorker(label);
    },
  };

  configureMonacoYaml(monaco, {
    completion: true,
    enableSchemaRequest: false,
    hover: true,
    schemas: [],
    validate: true,
  });

  monacoConfigured = true;
  return monaco;
}

function resolveProjectMonacoTheme(): ProjectMonacoTheme {
  return document.documentElement.getAttribute('theme-mode') === 'dark' ? 'vs-dark' : 'vs';
}

function applyProjectMonacoTheme(monacoInstance: MonacoEditorModule) {
  monacoInstance.editor.setTheme(resolveProjectMonacoTheme());
}

export function scheduleProjectMonacoLayout(layout: () => void) {
  return nextTick().then(() => {
    requestAnimationFrame(layout);
  });
}

export function useProjectMonacoLifecycle(options: {
  createEditor: () => void | Promise<void>;
  disposeEditor: () => void;
  getMonaco: () => MonacoEditorModule | null;
}) {
  let themeModeObserver: MutationObserver | null = null;

  const applyTheme = () => {
    const monacoInstance = options.getMonaco();
    if (!monacoInstance) {
      return;
    }
    applyProjectMonacoTheme(monacoInstance);
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
