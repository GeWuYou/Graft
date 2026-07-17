import { emitDebugLog } from '@/shared/debug/runtime';

const APPLICATION_TEMPLATE_DEBUG_FLAG = 'project.templates';

/**
 * 在启用模板调试开关时发出创建、路由与详情加载的诊断事件。
 *
 * 详情只包含模板标识、路由和结果状态，避免把名称、定义内容或接口响应正文写入控制台。
 */
export function emitApplicationTemplateDebug(event: string, detail: Record<string, unknown> = {}) {
  emitDebugLog(APPLICATION_TEMPLATE_DEBUG_FLAG, event, detail);
}
