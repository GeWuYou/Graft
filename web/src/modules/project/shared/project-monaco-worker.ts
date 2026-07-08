import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import JsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';

import { createLogger } from '@/utils/logger';

import { createProjectMonacoDebugLogger } from './project-monaco-debug';
import YamlWorker from './project-yaml.worker.js?worker';

type MonacoWorkerFactory = () => Worker;

type ProjectMonacoWorkerFactories = {
  createEditorWorker: MonacoWorkerFactory;
  createJsonWorker: MonacoWorkerFactory;
  createYamlWorker: MonacoWorkerFactory;
};

const logProjectMonacoWorkerDebug = createProjectMonacoDebugLogger('project.monaco.worker');
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

function createJsonWorker() {
  return new JsonWorker({
    name: 'json',
  });
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

function resolveWorkerKind(moduleId: string, label: string) {
  if (label === 'yaml' || moduleId.includes('yaml.worker')) {
    return 'yaml';
  }

  if (label === 'json' || moduleId.includes('json.worker')) {
    return 'json';
  }

  return 'editor';
}

export function buildProjectMonacoWorker(
  moduleId: string,
  label: string,
  factories: ProjectMonacoWorkerFactories = {
    createEditorWorker,
    createJsonWorker,
    createYamlWorker,
  },
) {
  const workerKind = resolveWorkerKind(moduleId, label);

  logProjectMonacoWorkerDebug('route-worker', {
    moduleId,
    label,
    workerKind,
  });

  try {
    switch (workerKind) {
      case 'json':
        return attachProjectMonacoWorkerDebug(factories.createJsonWorker(), label, 'json');
      case 'yaml':
        return attachProjectMonacoWorkerDebug(factories.createYamlWorker(), label, 'yaml');
      default:
        return attachProjectMonacoWorkerDebug(factories.createEditorWorker(), label, 'editor');
    }
  } catch (error) {
    logger.error(error instanceof Error ? error : new Error(String(error)), {
      label,
      moduleId,
      workerKind,
    });
    throw error;
  }
}
