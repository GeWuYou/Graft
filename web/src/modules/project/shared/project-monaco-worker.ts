import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import JsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';

import { createLogger } from '@/utils/logger';

import { createProjectMonacoDebugLogger } from './project-monaco-debug';
// YAML worker 必须保留为独立的 Vite `?worker` 入口，不能改成普通模块导入，否则浏览器构建时无法获得 Worker 构造器。
import YamlWorker from './project-yaml.worker.js?worker';

type MonacoWorkerFactory = () => Worker;

type ProjectMonacoWorkerFactories = {
  createEditorWorker: MonacoWorkerFactory;
  createJsonWorker: MonacoWorkerFactory;
  createYamlWorker: MonacoWorkerFactory;
};

const logProjectMonacoWorkerDebug = createProjectMonacoDebugLogger('project.monaco.worker');
const logger = createLogger('project.monaco.worker');

/**
 * 创建编辑器 Monaco worker。
 *
 * @returns 编辑器 worker 实例。
 */
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

/**
 * 为 Monaco worker 附加调试日志事件处理。
 *
 * @param worker - 要附加监听器的 worker
 * @param label - worker 的标识
 * @param kind - worker 类型
 * @returns 已附加调试监听器的 worker
 */
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

/**
 * 根据模块标识和标签确定 Monaco worker 类型。
 *
 * @param moduleId - worker 模块标识
 * @param label - worker 标签
 * @returns `yaml`、`json` 或 `editor`
 */
function resolveWorkerKind(moduleId: string, label: string) {
  if (label === 'yaml' || moduleId.includes('yaml.worker')) {
    return 'yaml';
  }

  if (label === 'json' || moduleId.includes('json.worker')) {
    return 'json';
  }

  return 'editor';
}

/**
 * 根据模块标识和标签构建并返回对应的 Monaco Web Worker。
 *
 * @param moduleId - 用于解析 worker 类型的模块标识。
 * @param label - worker 的标签，用于辅助路由到 JSON、YAML 或编辑器 worker。
 * @param factories - 用于创建各类 worker 的工厂集合。
 * @returns 构建得到的 worker 实例。
 */
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
