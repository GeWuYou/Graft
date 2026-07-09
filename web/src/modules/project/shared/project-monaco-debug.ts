import type * as Monaco from 'monaco-editor';

import { emitDebugLog, formatDebugLine, isDebugFlagEnabled } from '@/shared/debug/runtime';
import { createLogger } from '@/utils/logger';

/**
 * 判断是否启用 Project Monaco 调试。
 *
 * @returns `true` 如果已启用 `project.monaco` 调试标志，否则为 `false`。
 */
export function isProjectMonacoDebugEnabled() {
  return isDebugFlagEnabled('project.monaco');
}

/**
 * 判断错误是否为 Monaco 取消错误。
 *
 * @param error - 待检查的错误对象
 * @returns `true` if the error is a `Canceled` 错误且消息为 `Canceled`, `false` otherwise.
 */
function isProjectMonacoCancellationError(error: unknown) {
  return error instanceof Error && error.name === 'Canceled' && error.message === 'Canceled';
}

/**
 * 判断是否为可忽略的 Monaco 取消错误。
 *
 * @param error - 待检查的错误
 * @returns `true` 如果错误符合取消错误且堆栈命中已知的 Monaco 相关来源，`false` 否则
 */
export function isProjectMonacoBenignCancellationError(error: unknown) {
  if (!isProjectMonacoCancellationError(error)) {
    return false;
  }

  const stack = error instanceof Error ? (error.stack ?? '') : '';
  return (
    stack.includes('ProjectMonacoSurface.vue') ||
    stack.includes('ProjectMonacoDiffSurface.vue') ||
    stack.includes('monaco-editor') ||
    stack.includes('monaco-yaml') ||
    stack.includes('chunk-N7RXFHJR') ||
    stack.includes('chunk-64EI5KNP') ||
    stack.includes('chunk-MBPJUX45')
  );
}

/**
 * 格式化 Project Monaco 的调试消息。
 *
 * @param event - 事件名称
 * @param detail - 事件的详细信息
 * @returns 使用 `project.monaco` 分类生成的调试行文本
 */
export function formatProjectMonacoDebugMessage(event: string, detail: Record<string, unknown>) {
  return formatDebugLine('project.monaco', event, detail);
}

/**
 * 描述一个 DOM 元素的标识特征。
 *
 * @param element - 要描述的元素。
 * @returns 元素描述字符串；当 `element` 为空时返回 `null`。
 */
export function describeProjectMonacoElement(element: HTMLElement | null) {
  if (!element) {
    return 'null';
  }

  const classSuffix = Array.from(element.classList).join('.');
  const idSuffix = element.id ? `#${element.id}` : '';
  return `${element.tagName.toLowerCase()}${idSuffix}${classSuffix ? `.${classSuffix}` : ''}`;
}

/**
 * 延迟处置 Monaco 文本模型并根据结果触发对应回调。
 *
 * @param targetModel - 要处置的文本模型；为 `null` 时直接返回。
 * @param reason - 处置原因，会包含在回调详情中。
 * @param handlers - 处置完成、取消或出错时要调用的回调集合。
 */
export function disposeProjectMonacoModelDeferred(
  targetModel: Monaco.editor.ITextModel | null,
  reason: string,
  handlers: {
    onDispose(detail: Record<string, unknown>): void;
    onCancellation(detail: Record<string, unknown>): void;
    onError(error: Error, detail: Record<string, unknown>): void;
  },
) {
  if (!targetModel) {
    return;
  }

  const detail = {
    language: targetModel.getLanguageId(),
    reason,
    textLength: targetModel.getValue().length,
    uri: String(targetModel.uri),
  };

  queueMicrotask(() => {
    try {
      targetModel.dispose();
      handlers.onDispose(detail);
    } catch (error) {
      if (isProjectMonacoCancellationError(error)) {
        handlers.onCancellation(detail);
        return;
      }

      handlers.onError(error instanceof Error ? error : new Error(String(error)), detail);
    }
  });
}

/**
 * 创建用于输出 Project Monaco 调试信息的记录器。
 *
 * @param name - 记录器名称
 * @returns 接收事件名和详情并输出调试日志的函数
 */
export function createProjectMonacoDebugLogger(name: string) {
  const logger = createLogger(name);

  return (event: string, detail: Record<string, unknown>) => {
    if (!isProjectMonacoDebugEnabled()) {
      return;
    }

    emitDebugLog('project.monaco', event, {
      logger: name,
      ...detail,
    });
    logger.debug(`[ProjectMonaco] ${event}`, detail);
  };
}
