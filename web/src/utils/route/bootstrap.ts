import type { RouteRecordRaw } from 'vue-router';

import type { LocalizedTitle } from '@/contracts/i18n/locales';
import { getBootstrapRouteRegistration } from '@/modules';
import type { BootstrapMenu } from '@/modules/auth/contract/types';
import type { GlobalRouteRegistration } from '@/modules/types';
import { LAYOUT } from '@/utils/route/constant';
import type { AppRouteMeta, MenuRoute, NavigationAncestor } from '@/utils/types';

import { localizeRouteTitle } from './title';

type BootstrapMenuNode = { menu: BootstrapMenu; children: BootstrapMenuNode[] };

/**
 * 将 Bootstrap 菜单转换为可见的导航树。
 *
 * @param menus - Bootstrap 菜单项列表
 * @returns 可用于构建导航的菜单路由列表
 */
export function buildBootstrapNavigationTree(menus: BootstrapMenu[]): MenuRoute[] {
  const ancestorsByEntryPath = buildNavigationAncestorsByEntryPath(menus);
  return buildBootstrapMenuTree(menus)
    .map((node) => buildNavigationNode(node, ancestorsByEntryPath))
    .filter((node): node is MenuRoute => Boolean(node));
}

/**
 * 将 Bootstrap 菜单条目转换为 Vue Router 路由记录。
 *
 * @param menus - 待转换的 Bootstrap 菜单列表
 * @returns 已注册菜单条目对应的路由记录列表
 */
export function transformBootstrapMenusToRoutes(menus: BootstrapMenu[]): RouteRecordRaw[] {
  const ancestorsByEntryPath = buildNavigationAncestorsByEntryPath(menus);
  return menus.flatMap((menu) => {
    if (menu.kind !== 'entry' || !menu.path) return [];
    const registration = getBootstrapRouteRegistration(menu.path);
    if (!registration) return [];
    const routeMeta = buildRouteMeta(menu, registration.meta, ancestorsByEntryPath.get(menu.path));
    return [
      toRouteRecordRaw({
        path: menu.path,
        name: registration.routeName,
        component: LAYOUT,
        meta: routeMeta,
        children: [
          toRouteRecordRaw({
            path: '',
            name: `${registration.routeName}Index`,
            component: registration.loadPage,
            meta: { ...routeMeta, hiddenBreadcrumb: true },
          }),
        ],
      }),
    ];
  });
}

/**
 * 将全局路由注册项转换为 Vue Router 路由记录。
 *
 * @param registrations - 全局路由注册项列表
 * @param menus - 用于补充导航祖先上下文的菜单列表
 * @returns 转换后的路由记录列表
 */
export function transformGlobalRegistrationsToRoutes(
  registrations: GlobalRouteRegistration[],
  menus: BootstrapMenu[] = [],
): RouteRecordRaw[] {
  const ancestorsByEntryPath = buildNavigationAncestorsByEntryPath(menus);
  return registrations.map((registration) =>
    toRouteRecordRaw({
      path: registration.path,
      name: registration.routeName,
      component: LAYOUT,
      meta: {
        ...withNavigationContext(registration.meta, ancestorsByEntryPath.get(registration.navigationParentPath ?? '')),
        navigationTargetPath: registration.navigationParentPath,
        hiddenMenu: true,
        single: true,
      },
      children: [
        toRouteRecordRaw({
          path: '',
          name: `${registration.routeName}Index`,
          component: registration.loadPage,
          meta: {
            ...withNavigationContext(
              registration.meta,
              ancestorsByEntryPath.get(registration.navigationParentPath ?? ''),
            ),
            navigationTargetPath: registration.navigationParentPath,
            hiddenMenu: true,
            hiddenBreadcrumb:
              !registration.meta?.domainTitle && !ancestorsByEntryPath.has(registration.navigationParentPath ?? ''),
          },
        }),
      ],
    }),
  );
}

/**
 * 将 Bootstrap 菜单构建为按父子关系组织并按顺序排列的树。
 *
 * @param menus - 待构建的菜单项
 * @returns 排序后的顶层菜单节点列表
 */
function buildBootstrapMenuTree(menus: BootstrapMenu[]): BootstrapMenuNode[] {
  const nodeMap = new Map<string, BootstrapMenuNode>();
  const roots: BootstrapMenuNode[] = [];
  for (const menu of menus) nodeMap.set(menu.code, { menu, children: [] });
  for (const menu of menus) {
    const node = nodeMap.get(menu.code)!;
    const parent = menu.parent_code ? nodeMap.get(menu.parent_code) : undefined;
    if (parent) parent.children.push(node);
    else roots.push(node);
  }
  const sort = (nodes: BootstrapMenuNode[]) => {
    nodes.sort((a, b) => (a.menu.order ?? 0) - (b.menu.order ?? 0));
    nodes.forEach((node) => sort(node.children));
  };
  sort(roots);
  return roots;
}

/**
 * 将 Bootstrap 菜单节点转换为导航路由节点。
 *
 * @param node - 待转换的菜单节点
 * @param ancestorsByEntryPath - 按条目路径索引的导航祖先信息
 * @returns 转换后的导航路由节点；节点无有效目标或无法关联路由时返回 `null`
 */
function buildNavigationNode(
  node: BootstrapMenuNode,
  ancestorsByEntryPath: Map<string, NavigationAncestor[]>,
): MenuRoute | null {
  const children = node.children
    .map((child) => buildNavigationNode(child, ancestorsByEntryPath))
    .filter((child): child is MenuRoute => Boolean(child));
  const registration =
    node.menu.kind === 'entry' && node.menu.path ? getBootstrapRouteRegistration(node.menu.path) : undefined;
  if (node.menu.kind === 'entry' && !registration) return null;
  if (node.menu.kind === 'group' && children.length === 0) return null;
  const targetPath = node.menu.kind === 'entry' ? node.menu.path : children[0]?.meta?.navigationTargetPath;
  if (!targetPath) return null;
  return {
    path: node.menu.code,
    name: registration?.routeName ?? node.menu.code,
    title: localizeRouteTitle(node.menu.title, node.menu.title_key),
    meta: {
      ...buildRouteMeta(
        node.menu,
        registration?.meta,
        node.menu.path ? ancestorsByEntryPath.get(node.menu.path) : undefined,
      ),
      navigationCode: node.menu.code,
      navigationKind: node.menu.kind,
      navigationTargetPath: targetPath,
    },
    children,
  } as MenuRoute;
}

/**
 * 根据菜单信息构建路由元数据。
 *
 * @param menu - 用于生成标题、图标和排序信息的菜单
 * @param metaPatch - 覆盖或补充默认路由元数据的内容
 * @param navigationAncestors - 菜单对应的导航祖先信息
 * @returns 包含菜单基础信息和导航上下文的路由元数据
 */
function buildRouteMeta(
  menu: BootstrapMenu,
  metaPatch?: Partial<AppRouteMeta>,
  navigationAncestors?: NavigationAncestor[],
): AppRouteMeta {
  return {
    ...withNavigationContext(metaPatch, navigationAncestors, localizeRouteTitle(menu.title, menu.title_key)),
    title: localizeRouteTitle(menu.title, menu.title_key),
    titleKey: menu.title_key,
    icon: menu.icon,
    orderNo: menu.order ?? 0,
  };
}

/**
 * 将导航祖先信息补充到路由元数据中。
 *
 * @param metaPatch - 要合并的路由元数据
 * @param navigationAncestors - 当前路由的导航祖先列表
 * @param currentTitle - 当前路由的本地化标题
 * @returns 合并后的路由元数据；存在导航祖先时包含导航标题和祖先信息
 */
function withNavigationContext(
  metaPatch: Partial<AppRouteMeta> | undefined,
  navigationAncestors: NavigationAncestor[] | undefined,
  currentTitle?: LocalizedTitle,
): Partial<AppRouteMeta> {
  const context = navigationAncestors?.length
    ? {
        navigationAncestors,
        navigationTitle: joinLocalizedTitles([
          ...navigationAncestors.map((ancestor) => ancestor.title),
          metaPatch?.tabTitle ?? metaPatch?.semanticTitle ?? metaPatch?.title ?? currentTitle,
        ]),
      }
    : {};
  return { ...metaPatch, ...context };
}

/**
 * 构建条目路径到其导航祖先列表的映射。
 *
 * @param menus - 用于解析条目父级关系的菜单列表
 * @returns 以条目路径为键、按层级从上到下排列的导航祖先列表映射
 */
function buildNavigationAncestorsByEntryPath(menus: BootstrapMenu[]) {
  const nodeMap = new Map(menus.map((menu) => [menu.code, menu]));
  const ancestorsByEntryPath = new Map<string, NavigationAncestor[]>();
  for (const menu of menus) {
    if (menu.kind !== 'entry' || !menu.path) continue;
    const ancestors: NavigationAncestor[] = [];
    let parentCode = menu.parent_code;
    while (parentCode) {
      const parent = nodeMap.get(parentCode);
      if (!parent) break;
      ancestors.unshift({
        code: parent.code,
        path: parent.path ?? menu.path,
        title: localizeRouteTitle(parent.title, parent.title_key),
      });
      parentCode = parent.parent_code;
    }
    ancestorsByEntryPath.set(menu.path, ancestors);
  }
  return ancestorsByEntryPath;
}

/**
 * 将多个本地化标题按语言合并为以“ / ”分隔的标题。
 *
 * @param titles - 待合并的本地化标题，空值将被忽略
 * @returns 合并后的本地化标题；没有有效标题时返回 `undefined`
 */
function joinLocalizedTitles(titles: Array<LocalizedTitle | undefined>): LocalizedTitle | undefined {
  const resolved = titles.filter((title): title is LocalizedTitle => Boolean(title));
  if (resolved.length === 0) return undefined;
  return Object.fromEntries(
    Object.keys(resolved[0]).map((locale) => [
      locale,
      resolved
        .map((title) => title[locale as keyof LocalizedTitle])
        .filter(Boolean)
        .join(' / '),
    ]),
  ) as LocalizedTitle;
}

/**
 * 将路由对象转换为 Vue Router 路由记录。
 *
 * @param route - 要转换的路由对象
 * @returns 作为 `RouteRecordRaw` 使用的路由对象
 */
function toRouteRecordRaw(route: object): RouteRecordRaw {
  return route as RouteRecordRaw;
}
