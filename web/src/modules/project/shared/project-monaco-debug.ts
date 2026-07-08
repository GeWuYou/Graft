import type * as Monaco from 'monaco-editor';

import { emitDebugLog, formatDebugLine, isDebugFlagEnabled } from '@/shared/debug/runtime';
import { createLogger } from '@/utils/logger';

export function isProjectMonacoDebugEnabled() {
  return isDebugFlagEnabled('project.monaco');
}

function isProjectMonacoCancellationError(error: unknown) {
  return error instanceof Error && error.name === 'Canceled' && error.message === 'Canceled';
}

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

export function formatProjectMonacoDebugMessage(event: string, detail: Record<string, unknown>) {
  return formatDebugLine('project.monaco', event, detail);
}

export function describeProjectMonacoElement(element: HTMLElement | null) {
  if (!element) {
    return 'null';
  }

  const classSuffix = Array.from(element.classList).join('.');
  const idSuffix = element.id ? `#${element.id}` : '';
  return `${element.tagName.toLowerCase()}${idSuffix}${classSuffix ? `.${classSuffix}` : ''}`;
}

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
