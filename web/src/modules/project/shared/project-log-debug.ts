import { emitDebugLog } from '@/shared/debug/runtime';

const APPLICATION_LOG_DEBUG_FLAG = 'project.logs';

export function emitApplicationLogDebug(event: string, detail: Record<string, unknown> = {}) {
  emitDebugLog(APPLICATION_LOG_DEBUG_FLAG, event, detail);
}
