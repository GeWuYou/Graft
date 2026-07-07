import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';

import { createLogger } from '@/utils/logger';

import YamlWorker from './project-yaml.worker?worker';

type MonacoWorkerFactory = () => Worker;

type ProjectMonacoWorkerFactories = {
  createEditorWorker: MonacoWorkerFactory;
  createYamlWorker: MonacoWorkerFactory;
};

const PROJECT_MONACO_DEBUG_KEY = '__GRAFT_MONACO_DEBUG__';
const logger = createLogger('project.monaco.worker');

function createEditorWorker() {
  return new EditorWorker({
    name: 'editorWorkerService',
  });
}

function createYamlWorker() {
  return new YamlWorker({
    name: 'yaml',
  });
}

function isProjectMonacoDebugEnabled() {
  const debugFlag = (globalThis as typeof globalThis & Record<string, unknown>)[PROJECT_MONACO_DEBUG_KEY];

  if (debugFlag === true) {
    return true;
  }

  if (typeof localStorage === 'undefined') {
    return false;
  }

  try {
    return localStorage.getItem(PROJECT_MONACO_DEBUG_KEY) === 'true';
  } catch {
    return false;
  }
}

function logProjectMonacoWorkerDebug(event: string, detail: Record<string, unknown>) {
  if (!isProjectMonacoDebugEnabled()) {
    return;
  }

  logger.debug(`[ProjectMonaco] ${event}`, detail);
}

function attachProjectMonacoWorkerDebug(worker: Worker, label: string, kind: string) {
  logProjectMonacoWorkerDebug('worker-created', {
    kind,
    label,
  });

  if (typeof worker.addEventListener !== 'function') {
    return worker;
  }

  worker.addEventListener('error', (event) => {
    logProjectMonacoWorkerDebug('worker-error', {
      filename: event.filename,
      kind,
      label,
      lineno: event.lineno,
      message: event.message,
    });
  });

  worker.addEventListener('messageerror', () => {
    logProjectMonacoWorkerDebug('worker-messageerror', {
      kind,
      label,
    });
  });

  return worker;
}

export function buildProjectMonacoWorker(
  label: string,
  factories: ProjectMonacoWorkerFactories = {
    createEditorWorker,
    createYamlWorker,
  },
) {
  logProjectMonacoWorkerDebug('route-worker', {
    label,
  });

  switch (label) {
    case 'yaml':
      return attachProjectMonacoWorkerDebug(factories.createYamlWorker(), label, 'yaml');
    case 'editorWorkerService':
    default:
      return attachProjectMonacoWorkerDebug(factories.createEditorWorker(), label, 'editor');
  }
}
