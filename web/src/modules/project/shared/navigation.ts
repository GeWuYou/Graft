import type { LocationQueryRaw, Router } from 'vue-router';

import { LOCALE, type LocalizedTitle } from '@/contracts/i18n/locales';
import { localizeRouteTitleKey } from '@/utils/route/title';
import type { AppRouteMeta, TRouterInfo } from '@/utils/types';

type ProjectCreateRouter = Pick<Router, 'push' | 'replace'>;

/**
 * 导航至项目创建列表页。
 *
 * @param listRouteName - 目标列表页的路由名称
 */
export function navigateProjectCreateList(router: ProjectCreateRouter, listRouteName: string) {
  void router.push({ name: listRouteName });
}

/**
 * 使用指定的路由名称和查询参数替换当前项目创建页面。
 *
 * @param router - 用于执行路由替换的路由器
 * @param pageRouteName - 目标页面的路由名称
 * @param query - 目标页面的查询参数
 */
export function refreshProjectCreatePage(router: ProjectCreateRouter, pageRouteName: string, query: LocationQueryRaw) {
  void router.replace({ name: pageRouteName, query });
}

/**
 * 生成带回退规则的详情页标题。
 *
 * @param routeTitleKey - 路由标题的本地化键
 * @param name - 用于拼接到基础标题后的名称
 * @returns 基础标题；当 `name` 为空、或与中文/英文基础标题一致时直接返回基础标题，否则返回追加了 `name` 的中英文标题
 */
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

/**
 * 将解析后的路由信息追加到标签页列表并激活对应标签。
 *
 * @param tabs - 标签页管理器
 * @param resolved - 解析后的路由信息
 * @param title - 要写入标签页的标题
 */
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
