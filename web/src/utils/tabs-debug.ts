import { LOCALE } from '@/contracts/i18n/locales';
import { emitDebugLog } from '@/shared/debug/runtime';
import type { TRouterInfo } from '@/utils/types';

type TabsDebugRoute = Pick<TRouterInfo, 'fullPath' | 'name' | 'path' | 'tabKey' | 'title'>;

export function formatTabDebugTitle(title?: TRouterInfo['title']) {
  return title?.[LOCALE.ZH_CN] || title?.[LOCALE.EN_US] || '';
}

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

export function logTabsDebug(flagId: 'tabs.layout' | 'tabs.store', message: string | (() => string)) {
  emitDebugLog(flagId, 'trace', {
    message: typeof message === 'function' ? message() : message,
  });
}

function normalizeTabKey(value?: string) {
  return typeof value === 'string' ? value.trim() : '';
}
