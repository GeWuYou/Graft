import { LOCALE } from '@/contracts/i18n/locales';
import { createLogger } from '@/utils/logger';
import type { TRouterInfo } from '@/utils/types';

type TabsDebugRoute = Pick<TRouterInfo, 'fullPath' | 'name' | 'path' | 'tabKey' | 'title'>;
type TabsDebugLogger = Pick<ReturnType<typeof createLogger>, 'debug'>;

const tabsDebugEnabled = import.meta.env.VITE_TABS_DEBUG === 'true';

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

export function logTabsDebug(logger: TabsDebugLogger, message: string | (() => string)) {
  if (!tabsDebugEnabled) {
    return;
  }

  logger.debug(typeof message === 'function' ? message() : message);
}

function normalizeTabKey(value?: string) {
  return typeof value === 'string' ? value.trim() : '';
}
