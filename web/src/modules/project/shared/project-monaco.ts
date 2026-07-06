import 'monaco-editor/esm/vs/basic-languages/dockerfile/dockerfile.contribution';
import 'monaco-editor/esm/vs/basic-languages/ini/ini.contribution';
import 'monaco-editor/esm/vs/basic-languages/shell/shell.contribution';
import 'monaco-editor/min/vs/editor/editor.main.css';

import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import { configureMonacoYaml } from 'monaco-yaml';

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

export function buildProjectMonacoModelUri(key: string, language: string) {
  const normalizedKey = key.replace(/[^a-z0-9/_.-]/giu, '-');
  const extension = resolveLanguageExtension(language);
  return monaco.Uri.parse(`inmemory://project-configuration-workspace/${normalizedKey}.${extension}`);
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
