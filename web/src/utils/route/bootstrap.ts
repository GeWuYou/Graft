import type { RouteRecordRaw } from 'vue-router';

import type { BootstrapMenu } from '@/api/model/authModel';
import { getBootstrapRouteRegistration } from '@/modules';
import { BLANK_LAYOUT, LAYOUT } from '@/utils/route/constant';
import type { AppRouteMeta } from '@/utils/types';

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

  const builtChildren = node.children
    .map((childNode) => buildNestedRoute(childNode, node.menu.path))
    .filter((child): child is BootstrapLeaf => Boolean(child));

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

function buildNestedRoute(node: BootstrapMenuNode, parentFullPath: string): BootstrapLeaf | null {
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
        meta: buildRouteMeta(node.menu, false),
      }),
    };
  }

  const builtChildren = node.children
    .map((childNode) => buildNestedRoute(childNode, node.menu.path))
    .filter((child): child is BootstrapLeaf => Boolean(child));

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

function buildTopLevelLeafRoute(menu: BootstrapMenu, routeName: string, loadPage: RouteRecordRaw['component']) {
  return toRouteRecordRaw({
    path: menu.path,
    component: LAYOUT,
    redirect: `${menu.path}/index`,
    name: routeName,
    meta: buildRouteMeta(menu, true),
    children: [
      toRouteRecordRaw({
        path: 'index',
        name: `${routeName}Index`,
        component: loadPage,
        meta: {
          hidden: true,
          hiddenBreadcrumb: true,
          title: localizeRouteTitle(menu.title, menu.title_key),
          titleKey: menu.title_key,
        },
      }),
    ],
  });
}

function buildRouteMeta(menu: BootstrapMenu, single: boolean): AppRouteMeta {
  return {
    title: localizeRouteTitle(menu.title, menu.title_key),
    titleKey: menu.title_key,
    icon: menu.icon,
    single,
  };
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
