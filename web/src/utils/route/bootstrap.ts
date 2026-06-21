import type { RouteRecordRaw } from 'vue-router';

import type { LocalizedTitle } from '@/contracts/i18n/locales';
import { getBootstrapRouteRegistration } from '@/modules';
import type { BootstrapMenu } from '@/modules/auth/contract/types';
import type { GlobalRouteRegistration } from '@/modules/types';
import { BLANK_LAYOUT, LAYOUT } from '@/utils/route/constant';
import type { AppRouteMeta } from '@/utils/types';

import { resolveRouteLocalizedTitle } from './meta';
import { localizeRouteTitle } from './title';

type BootstrapMenuNode = {
  menu: BootstrapMenu;
  children: BootstrapMenuNode[];
};

type BootstrapLeaf = {
  fullPath: string;
  relativePath: string;
  route: RouteRecordRaw;
};

// transformBootstrapMenusToRoutes 把后端 bootstrap 菜单快照映射为当前 web 可消费的最小动态路由。
//
// 当前阶段只接入已在 `web` 内存在页面实现的真实菜单项，避免继续沿用 starter demo 菜单树。
export function transformBootstrapMenusToRoutes(menus: BootstrapMenu[]): RouteRecordRaw[] {
  const rootNodes = buildBootstrapMenuTree(menus);
  const routes = rootNodes
    .map((node) => buildRootRoute(node))
    .filter((route): route is RouteRecordRaw => Boolean(route));

  return routes;
}

/**
 * Converts global route registrations into route record definitions.
 *
 * Each registration is transformed into a top-level route with an index child route.
 * The breadcrumb title is derived from the domain title, breadcrumb title, or title
 * in that priority order.
 *
 * @returns An array of route records corresponding to the input registrations.
 */
export function transformGlobalRegistrationsToRoutes(registrations: GlobalRouteRegistration[]): RouteRecordRaw[] {
  return registrations.map((registration) =>
    toRouteRecordRaw({
      path: registration.path,
      name: registration.routeName,
      component: LAYOUT,
      meta: {
        ...registration.meta,
        breadcrumbTitle:
          registration.meta?.domainTitle ?? registration.meta?.breadcrumbTitle ?? registration.meta?.title,
        hiddenMenu: true,
        single: true,
      },
      children: [
        toRouteRecordRaw({
          path: '',
          name: `${registration.routeName}Index`,
          component: registration.loadPage,
          meta: {
            ...registration.meta,
            hiddenMenu: true,
            hiddenBreadcrumb: !registration.meta?.domainTitle,
          },
        }),
      ],
    }),
  );
}

function toRouteRecordRaw(route: object): RouteRecordRaw {
  return route as unknown as RouteRecordRaw;
}

function buildBootstrapMenuTree(menus: BootstrapMenu[]) {
  const orderedMenus = menus
    .filter((menu) => normalizePath(menu.path))
    .map((menu) => ({
      ...menu,
      path: normalizePath(menu.path),
    }));
  const nodeMap = new Map<string, BootstrapMenuNode>();
  const roots: BootstrapMenuNode[] = [];

  for (const menu of orderedMenus) {
    nodeMap.set(menu.path, {
      menu,
      children: [],
    });
  }

  for (const menu of orderedMenus) {
    const node = nodeMap.get(menu.path);
    if (!node) {
      continue;
    }

    const parentPath = parentMenuPath(menu.path);
    if (!parentPath) {
      roots.push(node);
      continue;
    }

    const parentNode = nodeMap.get(parentPath);
    if (!parentNode) {
      roots.push(node);
      continue;
    }

    parentNode.children.push(node);
  }

  return roots;
}

function buildRootRoute(node: BootstrapMenuNode) {
  const registration = getBootstrapRouteRegistration(node.menu.path);
  if (registration && node.children.length === 0) {
    return buildTopLevelLeafRoute(node.menu, registration.routeName, registration.loadPage);
  }

  const builtChildren = buildNestedChildren(node, node.menu.path);

  if (builtChildren.length === 0) {
    return null;
  }

  return toRouteRecordRaw({
    path: node.menu.path,
    component: LAYOUT,
    redirect: builtChildren[0]?.fullPath,
    name: buildGroupRouteName(node.menu.path),
    meta: buildRouteMeta(node.menu, false),
    children: builtChildren.map((child) => child.route),
  });
}

function buildNestedRoute(
  node: BootstrapMenuNode,
  parentFullPath: string,
  parentMenu: BootstrapMenu,
): BootstrapLeaf | null {
  const registration = getBootstrapRouteRegistration(node.menu.path);
  const relativePath = childSegment(parentFullPath, node.menu.path);
  if (!relativePath) {
    return null;
  }

  if (registration && node.children.length === 0) {
    return {
      fullPath: node.menu.path,
      relativePath,
      route: toRouteRecordRaw({
        path: relativePath,
        name: `${registration.routeName}Index`,
        component: registration.loadPage,
        meta: buildRouteMeta(node.menu, false, registration.meta, parentMenu),
      }),
    };
  }

  const builtChildren = buildNestedChildren(node, node.menu.path);

  if (builtChildren.length === 0) {
    return null;
  }

  return {
    fullPath: node.menu.path,
    relativePath,
    route: toRouteRecordRaw({
      path: relativePath,
      name: buildGroupRouteName(node.menu.path),
      component: BLANK_LAYOUT,
      redirect: builtChildren[0]?.relativePath,
      meta: buildRouteMeta(node.menu, false),
      children: builtChildren.map((child) => child.route),
    }),
  };
}

function buildNestedChildren(node: BootstrapMenuNode, parentFullPath: string): BootstrapLeaf[] {
  return node.children
    .map((childNode) => buildNestedRoute(childNode, parentFullPath, node.menu))
    .filter((child): child is BootstrapLeaf => Boolean(child));
}

function buildTopLevelLeafRoute(menu: BootstrapMenu, routeName: string, loadPage: RouteRecordRaw['component']) {
  const registration = getBootstrapRouteRegistration(menu.path);
  const routeMeta = buildRouteMeta(menu, true, registration?.meta);
  return toRouteRecordRaw({
    path: menu.path,
    component: LAYOUT,
    redirect: `${menu.path}/index`,
    name: routeName,
    meta: routeMeta,
    children: [
      toRouteRecordRaw({
        path: 'index',
        name: `${routeName}Index`,
        component: loadPage,
        meta: {
          ...registration?.meta,
          hidden: true,
          hiddenBreadcrumb: true,
          title: routeMeta.title,
          titleKey: menu.title_key,
          semanticTitle: resolveRouteLocalizedTitle(routeMeta, 'page'),
          breadcrumbTitle: resolveRouteLocalizedTitle(routeMeta, 'breadcrumb'),
          tabTitle: resolveRouteLocalizedTitle(routeMeta, 'tab'),
        },
      }),
    ],
  });
}

function buildRouteMeta(
  menu: BootstrapMenu,
  single: boolean,
  metaPatch?: Partial<AppRouteMeta>,
  parentMenu?: BootstrapMenu,
): AppRouteMeta {
  const title = localizeRouteTitle(menu.title, menu.title_key);
  const parentTitle = parentMenu ? localizeRouteTitle(parentMenu.title, parentMenu.title_key) : undefined;
  const derivedTitlePatch = buildDerivedTitleMeta(title, parentTitle);

  return {
    title,
    titleKey: menu.title_key,
    icon: menu.icon,
    orderNo: menu.order ?? 0,
    single,
    ...derivedTitlePatch,
    ...metaPatch,
  };
}

function buildDerivedTitleMeta(title: LocalizedTitle, parentTitle?: LocalizedTitle): Partial<AppRouteMeta> {
  if (!parentTitle) {
    return {};
  }

  return {
    semanticTitle: combineLocalizedTitles(parentTitle, title),
    breadcrumbTitle: title,
    tabTitle: combineLocalizedTitles(parentTitle, title),
  };
}

function combineLocalizedTitles(parentTitle: LocalizedTitle, title: LocalizedTitle): LocalizedTitle {
  return Object.entries(title).reduce<LocalizedTitle>((combined, [locale, localeTitle]) => {
    const parentLocaleTitle = parentTitle[locale as keyof LocalizedTitle];
    combined[locale as keyof LocalizedTitle] = parentLocaleTitle
      ? `${parentLocaleTitle} - ${localeTitle}`
      : localeTitle;
    return combined;
  }, {} as LocalizedTitle);
}

function normalizePath(path: string) {
  const trimmed = path.trim();
  if (!trimmed) {
    return '';
  }

  const withLeadingSlash = trimmed.startsWith('/') ? trimmed : `/${trimmed}`;
  return withLeadingSlash === '/' ? withLeadingSlash : withLeadingSlash.replace(/\/+$/, '');
}

function parentMenuPath(path: string) {
  const segments = path.split('/').filter(Boolean);
  if (segments.length <= 1) {
    return '';
  }

  return `/${segments.slice(0, -1).join('/')}`;
}

function childSegment(parentFullPath: string, childFullPath: string) {
  if (!childFullPath.startsWith(`${parentFullPath}/`)) {
    return '';
  }

  return childFullPath.slice(parentFullPath.length + 1);
}

function buildGroupRouteName(path: string) {
  const suffix = path
    .split('/')
    .filter(Boolean)
    .map((segment) => segment.replace(/(^\w)|-(\w)/g, (_, start, afterDash) => (start || afterDash).toUpperCase()))
    .join('');

  return `BootstrapGroup${suffix}`;
}
