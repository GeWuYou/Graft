import { emitDebugLog } from '@/shared/debug/runtime';

const PROJECT_WORKSPACE_DEBUG_FLAG = 'project.workspace';

/**
 * 在启用项目工作台调试开关时发出工作台状态诊断事件。
 *
 * 调用方只应传入路径、数量和布尔状态，不得传入工作区文件内容。
 */
export function emitProjectWorkspaceDebug(event: string, detail: Record<string, unknown> = {}) {
  emitDebugLog(PROJECT_WORKSPACE_DEBUG_FLAG, event, detail);
}
