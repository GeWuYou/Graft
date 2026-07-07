import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';

import { createProjectMonacoDebugLogger } from './project-monaco-debug';
import YamlWorker from './project-yaml.worker?worker';

type MonacoWorkerFactory = () => Worker;

type ProjectMonacoWorkerFactories = {
  createEditorWorker: MonacoWorkerFactory;
  createYamlWorker: MonacoWorkerFactory;
};

const logProjectMonacoWorkerDebug = createProjectMonacoDebugLogger('project.monaco.worker');

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
