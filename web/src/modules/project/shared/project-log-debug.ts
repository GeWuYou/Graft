import { emitDebugLog } from '@/shared/debug/runtime';

const PROJECT_LOG_DEBUG_FLAG = 'project.logs';

/**
 * 在启用项目日志调试开关时发出项目日志诊断事件。
 *
 * @param event - 诊断事件名称
 * @param detail - 诊断事件的结构化元数据
 */
export function emitProjectLogDebug(event: string, detail: Record<string, unknown> = {}) {
  emitDebugLog(PROJECT_LOG_DEBUG_FLAG, event, detail);
}
