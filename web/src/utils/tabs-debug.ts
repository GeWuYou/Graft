import { LOCALE } from '@/contracts/i18n/locales';
import { emitDebugLog } from '@/shared/debug/runtime';
import type { TRouterInfo } from '@/utils/types';

type TabsDebugRoute = Pick<TRouterInfo, 'fullPath' | 'name' | 'path' | 'tabKey' | 'title'>;

/**
 * 生成用于调试的标签标题。
 *
 * @param title - 路由标题的多语言映射
 * @returns 中文标题优先，其次英文标题；都不存在时返回空字符串
 */
export function formatTabDebugTitle(title?: TRouterInfo['title']) {
  return title?.[LOCALE.ZH_CN] || title?.[LOCALE.EN_US] || '';
}

/**
 * 生成标签路由的调试摘要字符串。
 *
 * @param routes - 标签路由列表
 * @returns 路由数量及每个路由关键信息组成的单行摘要；当列表为空时返回 `count=0`
 */
export function formatTabsDebugSummary(routes: TabsDebugRoute[]) {
  if (!routes.length) {
    return 'count=0';
  }

  return `count=${routes.length} ${routes
    .map(
      (route, index) =>
        `[#${index} key=${normalizeTabKey(route.tabKey) || normalizeTabKey(route.path) || '/'} path=${route.path} fullPath=${
          route.fullPath || ''
        } name=${String(route.name || '')} title=${formatTabDebugTitle(route.title)}]`,
    )
    .join(' ')}`;
}

/**
 * 记录指定调试标志下的追踪日志。
 *
 * @param flagId - 调试标志 ID
 * @param message - 日志内容，或用于延迟生成日志内容的回调
 */
export function logTabsDebug(flagId: 'tabs.layout' | 'tabs.store', message: string | (() => string)) {
  emitDebugLog(flagId, 'trace', {
    message: typeof message === 'function' ? message() : message,
  });
}

/**
 * 规范化标签键或路径为去除首尾空白的字符串。
 *
 * @param value - 待规范化的字符串值
 * @returns 去除首尾空白后的字符串；当未提供值时返回空字符串
 */
function normalizeTabKey(value?: string) {
  return typeof value === 'string' ? value.trim() : '';
}
