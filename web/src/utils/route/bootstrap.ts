import type { RouteRecordRaw } from 'vue-router';

import type { BootstrapMenu } from '@/api/model/authModel';
import { getBootstrapRouteRegistration } from '@/modules';
import { ACCESS_CONTROL_ROUTE_PATH } from '@/modules/access-control/contract/bootstrap';
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
  const rootNodes = buildBootstrapMenuTree(normalizeAccessControlMenus(menus));
  const routes = rootNodes
    .map((node) => buildRootRoute(node))
    .filter((route): route is RouteRecordRaw => Boolean(route));

  return routes;
}

function normalizeAccessControlMenus(menus: BootstrapMenu[]): BootstrapMenu[] {
  const managedPaths = new Set<string>([
    ACCESS_CONTROL_ROUTE_PATH.OVERVIEW,
    ACCESS_CONTROL_ROUTE_PATH.USERS,
    ACCESS_CONTROL_ROUTE_PATH.ROLES,
    ACCESS_CONTROL_ROUTE_PATH.PERMISSIONS,
  ]);
  const normalizedMenus = menus.map((menu) => {
    switch (normalizePath(menu.path)) {
      case ACCESS_CONTROL_ROUTE_PATH.LEGACY_USERS:
        return withAccessControlIcon({
          ...menu,
          path: ACCESS_CONTROL_ROUTE_PATH.USERS,
          title_key:
            !menu.title_key || menu.title_key === 'menu.user_list.title'
              ? 'menu.access_control.users.title'
              : menu.title_key,
        });
      case ACCESS_CONTROL_ROUTE_PATH.LEGACY_ROLES:
        return withAccessControlIcon({
          ...menu,
          path: ACCESS_CONTROL_ROUTE_PATH.ROLES,
          title_key:
            !menu.title_key || menu.title_key === 'menu.role_list.title'
              ? 'menu.access_control.roles.title'
              : menu.title_key,
        });
      case ACCESS_CONTROL_ROUTE_PATH.LEGACY_PERMISSIONS:
        return withAccessControlIcon({
          ...menu,
          path: ACCESS_CONTROL_ROUTE_PATH.PERMISSIONS,
          title_key:
            !menu.title_key || menu.title_key === 'menu.permission_list.title'
              ? 'menu.access_control.permissions.title'
              : menu.title_key,
        });
      default:
        return withAccessControlIcon(menu);
    }
  });

  const hasAccessControlRoot = normalizedMenus.some(
    (menu) => normalizePath(menu.path) === ACCESS_CONTROL_ROUTE_PATH.ROOT,
  );
  const hasAccessControlOverview = normalizedMenus.some(
    (menu) => normalizePath(menu.path) === ACCESS_CONTROL_ROUTE_PATH.OVERVIEW,
  );
  const hasManagedChildren = normalizedMenus.some((menu) => managedPaths.has(normalizePath(menu.path)));

  if (!hasManagedChildren) {
    return normalizedMenus;
  }

  const synthesizedMenus: BootstrapMenu[] = [];

  if (!hasAccessControlRoot) {
    synthesizedMenus.push({
      code: 'access-control.group',
      title: '访问控制',
      title_key: 'menu.access_control.title',
      path: ACCESS_CONTROL_ROUTE_PATH.ROOT,
      icon: 'secured',
      permission: '',
    });
  }

  if (!hasAccessControlOverview) {
    synthesizedMenus.push({
      code: 'access-control.overview',
      title: '概览',
      title_key: 'menu.access_control.overview.title',
      path: ACCESS_CONTROL_ROUTE_PATH.OVERVIEW,
      icon: 'dashboard',
      permission: '',
    });
  }

  const dedupedMenus = new Map<string, BootstrapMenu>();
  [...synthesizedMenus, ...normalizedMenus].forEach((menu) => {
    const normalizedPath = normalizePath(menu.path);
    if (normalizedPath && !dedupedMenus.has(normalizedPath)) {
      dedupedMenus.set(normalizedPath, menu);
    }
  });

  return Array.from(dedupedMenus.values()).sort(compareAccessControlMenus);
}

function compareAccessControlMenus(left: BootstrapMenu, right: BootstrapMenu) {
  const leftPath = normalizePath(left.path);
  const rightPath = normalizePath(right.path);
  const accessControlOrder = new Map<string, number>([
    [ACCESS_CONTROL_ROUTE_PATH.ROOT, 0],
    [ACCESS_CONTROL_ROUTE_PATH.OVERVIEW, 1],
    [ACCESS_CONTROL_ROUTE_PATH.USERS, 2],
    [ACCESS_CONTROL_ROUTE_PATH.ROLES, 3],
    [ACCESS_CONTROL_ROUTE_PATH.PERMISSIONS, 4],
  ]);

  const leftOrder = accessControlOrder.get(leftPath);
  const rightOrder = accessControlOrder.get(rightPath);
  if (leftOrder !== undefined && rightOrder !== undefined) {
    return leftOrder - rightOrder;
  }

  if (leftOrder !== undefined) {
    return -1;
  }

  if (rightOrder !== undefined) {
    return 1;
  }

  return 0;
}

function withAccessControlIcon(menu: BootstrapMenu): BootstrapMenu {
  switch (normalizePath(menu.path)) {
    case ACCESS_CONTROL_ROUTE_PATH.ROOT:
      return { ...menu, icon: 'secured' };
    case ACCESS_CONTROL_ROUTE_PATH.OVERVIEW:
      return { ...menu, icon: 'dashboard' };
    case ACCESS_CONTROL_ROUTE_PATH.USERS:
      return { ...menu, icon: 'usergroup' };
    case ACCESS_CONTROL_ROUTE_PATH.ROLES:
      return { ...menu, icon: 'secured' };
    case ACCESS_CONTROL_ROUTE_PATH.PERMISSIONS:
      return { ...menu, icon: 'lock-on' };
    default:
      return menu;
  }
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
    .map((childNode) => buildNestedRoute(childNode, parentFullPath))
    .filter((child): child is BootstrapLeaf => Boolean(child));
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
