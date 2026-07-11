import { emitDebugLog } from '@/shared/debug/runtime';

const PROJECT_LOG_DEBUG_FLAG = 'project.logs';

/** Emits project-log diagnostics only when the project log debug flag is enabled. */
export function emitProjectLogDebug(event: string, detail: Record<string, unknown> = {}) {
  emitDebugLog(PROJECT_LOG_DEBUG_FLAG, event, detail);
}
