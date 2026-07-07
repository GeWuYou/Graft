import { createLogger } from '@/utils/logger';

const PROJECT_MONACO_DEBUG_KEY = '__GRAFT_MONACO_DEBUG__';

export function isProjectMonacoDebugEnabled() {
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

export function createProjectMonacoDebugLogger(name: string) {
  const logger = createLogger(name);

  return (event: string, detail: Record<string, unknown>) => {
    if (!isProjectMonacoDebugEnabled()) {
      return;
    }

    logger.debug(`[ProjectMonaco] ${event}`, detail);
  };
}
