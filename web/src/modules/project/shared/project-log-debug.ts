import { emitDebugLog } from '@/shared/debug/runtime';

const PROJECT_LOG_DEBUG_FLAG = 'project.logs';

export function emitProjectLogDebug(event: string, detail: Record<string, unknown> = {}) {
  emitDebugLog(PROJECT_LOG_DEBUG_FLAG, event, detail);
}
