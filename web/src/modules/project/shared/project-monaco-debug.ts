import type * as Monaco from 'monaco-editor';

import { createLogger } from '@/utils/logger';

const PROJECT_MONACO_DEBUG_KEY = '__GRAFT_MONACO_DEBUG__';
const PROJECT_MONACO_DEBUG_ENV_KEY = 'VITE_PROJECT_MONACO_DEBUG';

function isEnabledValue(value: unknown) {
  return value === true || value === 'true' || value === '1' || value === 1;
}

export function isProjectMonacoDebugEnabled() {
  if (isEnabledValue(import.meta.env[PROJECT_MONACO_DEBUG_ENV_KEY])) {
    return true;
  }

  if (import.meta.env.DEV) {
    return true;
  }

  const debugFlag = (globalThis as typeof globalThis & Record<string, unknown>)[PROJECT_MONACO_DEBUG_KEY];

  if (isEnabledValue(debugFlag)) {
    return true;
  }

  if (typeof localStorage === 'undefined') {
    return false;
  }

  try {
    return isEnabledValue(localStorage.getItem(PROJECT_MONACO_DEBUG_KEY));
  } catch {
    return false;
  }
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
  const summary = Object.entries(detail)
    .map(([key, value]) => `${key}=${String(value)}`)
    .join(' ');
  return summary ? `${event} ${summary}` : event;
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

    logger.warn(`[ProjectMonaco] ${event}`, detail);
  };
}
