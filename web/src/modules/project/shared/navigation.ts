import type { LocationQueryRaw, Router } from 'vue-router';

import type { LocalizedTitle } from '@/contracts/i18n/locales';
import { useTabsRouterStore } from '@/store/modules/tabs-router';
import { buildDetailTitleWithFallback, localizeRouteTitleKey } from '@/utils/route/title';
import type { AppRouteMeta, TRouterInfo } from '@/utils/types';

import { PROJECT_BOOTSTRAP_ROUTE } from '../contract/bootstrap';

type ApplicationCreateRouter = Pick<Router, 'push' | 'replace'>;
type ApplicationRouteRouter = Pick<Router, 'push' | 'resolve'>;

/**
 * 导航回项目创建方式选择页，并保留当前创建流程的查询上下文。
 */
export function navigateToApplicationCreateSource(router: ApplicationCreateRouter, query: LocationQueryRaw) {
  void router.push({
    name: PROJECT_BOOTSTRAP_ROUTE.CREATE_SOURCE.pageRouteName,
    query,
  });
}

/**
 * 返回应用管理列表，并在路由过渡前同步列表标签，避免动态 bootstrap 列表页在标签状态切换间隙失去渲染宿主。
 */
export function navigateToApplicationList(
  router: ApplicationRouteRouter,
  tabs: { appendTabRouterList: (route: TRouterInfo) => void; setActiveTabKey: (key: string) => void },
) {
  const target = { name: PROJECT_BOOTSTRAP_ROUTE.LIST.pageRouteName };
  const resolved = router.resolve(target);
  appendResolvedTab(tabs, resolved, localizeRouteTitleKey('project.route.list.title'));
  void router.push(target);
}

/**
 * 追加项目创建流程标签并导航至目标页面。
 */
function navigateApplicationCreateRoute(
  router: ApplicationRouteRouter,
  tabs: { appendTabRouterList: (route: TRouterInfo) => void; setActiveTabKey: (key: string) => void },
  target: Parameters<Router['push']>[0],
  titleKey: string,
) {
  const resolved = router.resolve(target);
  appendResolvedTab(tabs, resolved, localizeRouteTitleKey(titleKey));
  void router.push(target);
}

/**
 * 创建用于项目创建流程的路由导航函数。
 *
 * @returns 接收目标路由和标题键的导航函数；导航前会添加并激活对应标签页。
 */
export function useApplicationCreateRouteNavigation(router: ApplicationRouteRouter) {
  const tabs = useTabsRouterStore();

  return (target: Parameters<Router['push']>[0], titleKey: string) => {
    navigateApplicationCreateRoute(router, tabs, target, titleKey);
  };
}

/**
 * 使用指定的路由名称和查询参数替换当前项目创建页面。
 *
 * @param router - 用于执行路由替换的路由器
 * @param pageRouteName - 目标页面的路由名称
 * @param query - 目标页面的查询参数
 */
export function refreshApplicationCreatePage(
  router: ApplicationCreateRouter,
  pageRouteName: string,
  query: LocationQueryRaw,
) {
  void router.replace({ name: pageRouteName, query });
}

/**
 * 生成带回退规则的详情页标题。
 *
 * @param routeTitleKey - 路由标题的本地化键
 * @param name - 用于拼接到基础标题后的名称
 * @returns 基础标题；当 `name` 为空、或与中文/英文基础标题一致时直接返回基础标题，否则返回追加了 `name` 的中英文标题
 */
export { buildDetailTitleWithFallback };

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
