import { LOCALE, type LocalizedTitle } from '@/contracts/i18n/locales';
import { localizeRouteTitleKey } from '@/utils/route/title';
import type { AppRouteMeta, TRouterInfo } from '@/utils/types';

export function buildDetailTitleWithFallback(routeTitleKey: string, name: string): LocalizedTitle {
  const normalizedName = name.trim();
  const baseTitle = localizeRouteTitleKey(routeTitleKey);
  if (!normalizedName || normalizedName === baseTitle[LOCALE.ZH_CN] || normalizedName === baseTitle[LOCALE.EN_US]) {
    return baseTitle;
  }

  return {
    [LOCALE.ZH_CN]: `${baseTitle[LOCALE.ZH_CN]} - ${normalizedName}`,
    [LOCALE.EN_US]: `${baseTitle[LOCALE.EN_US]} - ${normalizedName}`,
  };
}

export function appendResolvedTab(
  tabs: { appendTabRouterList: (route: TRouterInfo) => void; setActiveTabKey: (key: string) => void },
  resolved: {
    path: string;
    fullPath: string;
    query: TRouterInfo['query'];
    params: TRouterInfo['params'];
    name: TRouterInfo['name'];
    meta: unknown;
  },
  title: LocalizedTitle,
) {
  tabs.appendTabRouterList({
    tabKey: resolved.path,
    path: resolved.path,
    fullPath: resolved.fullPath,
    query: resolved.query,
    params: resolved.params,
    title,
    name: resolved.name,
    isAlive: true,
    meta: resolved.meta as AppRouteMeta,
  });
  tabs.setActiveTabKey(resolved.path);
}
