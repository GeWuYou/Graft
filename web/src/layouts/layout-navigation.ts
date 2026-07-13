import { ACCESS_LOG_ROUTE_PATH } from '@/modules/access-log/contract/paths';
import { APP_LOG_ROUTE_PATH } from '@/modules/app-log/contract/paths';
import { AUDIT_ROUTE_PATH } from '@/modules/audit/contract/paths';
import { CONTAINER_ROUTE_PATH } from '@/modules/container/contract/paths';
import type { MenuRoute } from '@/utils/types';

export type SidebarMotionMode = 'default' | 'wide-table';

export type SidebarMotionPhase =
  | 'expanded'
  | 'collapsing-width'
  | 'collapsing-submenu'
  | 'collapsing-topmenu'
  | 'compact'
  | 'expanding-width'
  | 'expanding-topmenu'
  | 'expanding-submenu';

const WIDE_TABLE_SIDEBAR_ROUTE_PATHS = new Set<string>([
  CONTAINER_ROUTE_PATH.LIST,
  APP_LOG_ROUTE_PATH.LIST,
  ACCESS_LOG_ROUTE_PATH.LIST,
  AUDIT_ROUTE_PATH.LOGS,
]);

export function resolveSidebarMotionMode(routePath: string): SidebarMotionMode {
  return WIDE_TABLE_SIDEBAR_ROUTE_PATHS.has(routePath) ? 'wide-table' : 'default';
}

/**
 * 将混合布局的顶级菜单转换为扁平化导航菜单。
 *
 * @param menus - 待转换的菜单列表
 * @returns 扁平化后的菜单列表
 */
export function flattenMixHeaderMenus(menus: MenuRoute[]): MenuRoute[] {
  return menus.map((menu) => ({
    ...menu,
    path: resolveMenuNavigationPath(menu),
    redirect: undefined,
    children: [],
    meta: {
      ...menu.meta,
      single: true,
    },
  }));
}

/**
 * Selects the top-level menu branch that owns the current route for the mixed-layout sidebar.
 *
 * A group navigation target points to its first visible leaf, so ownership must be resolved from
 * the complete menu tree rather than from that single target path.
 */
export function selectMixSidebarMenu(menus: MenuRoute[], activePath: string): MenuRoute[] {
  const activeMenu = menus.find((menu) => menu.children?.length && menuContainsActivePath(menu, activePath));

  if (!activeMenu) {
    return menus;
  }

  return [
    {
      ...activeMenu,
      meta: {
        ...activeMenu.meta,
        expanded: true,
      },
    },
  ];
}

/**
 * 确定菜单项用于导航的路径。
 *
 * @param menu - 要解析导航路径的菜单项
 * @param parentPath - 菜单项的父级路径
 * @returns 菜单项的显式目标路径、重定向路径、首个可见子菜单路径或自身完整路径
 */
export function resolveMenuNavigationPath(menu: MenuRoute, parentPath = ''): string {
  const explicitTarget = menu.meta?.navigationTargetPath;
  if (typeof explicitTarget === 'string' && explicitTarget) {
    return explicitTarget;
  }
  const fullPath = normalizeMenuPath(parentPath, menu.path);

  if (typeof menu.redirect === 'string' && menu.redirect.trim()) {
    const redirectedPath = normalizeMenuPath(fullPath, menu.redirect);
    const redirectedChild = findRedirectedChild(menu.children ?? [], fullPath, redirectedPath);

    if (redirectedChild) {
      return resolveMenuNavigationPath(redirectedChild, fullPath);
    }

    return redirectedPath;
  }

  const firstVisibleChild = menu.children?.find((child) => child.meta?.hidden !== true);
  if (firstVisibleChild) {
    return resolveMenuNavigationPath(firstVisibleChild, fullPath);
  }

  return fullPath;
}

export function findExpandedMenuPaths(menus: MenuRoute[], activePath: string, parentPath = ''): string[] {
  return findExpandedMenuMatch(menus, activePath, parentPath).expandedPaths;
}

/**
 * 查找与当前路径匹配的菜单，并确定需要展开的菜单路径。
 *
 * @param menus - 要搜索的菜单列表
 * @param activePath - 当前激活的路由路径
 * @param parentPath - 菜单父级路径
 * @returns 包含匹配状态和待展开菜单路径的结果
 */
function findExpandedMenuMatch(
  menus: MenuRoute[],
  activePath: string,
  parentPath: string,
): { matched: boolean; expandedPaths: string[] } {
  if (!activePath) {
    return { matched: false, expandedPaths: [] };
  }

  for (const menu of menus) {
    if (menu.meta?.hidden === true) {
      continue;
    }

    const fullPath = normalizeMenuPath(parentPath, menu.path);
    const targetPath = resolveMenuNavigationPath(menu, parentPath);
    const visibleChildren = (menu.children ?? []).filter((child) => child.meta?.hidden !== true);
    const childMatch =
      visibleChildren.length > 0
        ? findExpandedMenuMatch(visibleChildren, activePath, fullPath)
        : { matched: false, expandedPaths: [] };

    if (childMatch.matched) {
      return {
        matched: true,
        expandedPaths: menu.meta?.single ? childMatch.expandedPaths : [menu.path, ...childMatch.expandedPaths],
      };
    }

    const isExpandableMenu = visibleChildren.length > 0 && menu.meta?.single !== true;
    const matchesCurrentMenu = activePath === targetPath || activePath.startsWith(`${targetPath}/`);

    if (matchesCurrentMenu) {
      return {
        matched: true,
        expandedPaths: isExpandableMenu ? [menu.path] : [],
      };
    }
  }

  return { matched: false, expandedPaths: [] };
}

function menuContainsActivePath(menu: MenuRoute, activePath: string, parentPath = ''): boolean {
  if (!activePath || menu.meta?.hidden === true) {
    return false;
  }

  const fullPath = normalizeMenuPath(parentPath, menu.path);
  const visibleChildren = (menu.children ?? []).filter((child) => child.meta?.hidden !== true);

  if (visibleChildren.some((child) => menuContainsActivePath(child, activePath, fullPath))) {
    return true;
  }

  const targetPath = resolveMenuNavigationPath(menu, parentPath);
  return activePath === targetPath || activePath.startsWith(`${targetPath}/`);
}

function normalizeMenuPath(parentPath: string, routePath: string) {
  if (!routePath) {
    return parentPath || '/';
  }

  if (routePath.startsWith('/')) {
    return routePath === '/' ? routePath : routePath.replace(/\/+$/, '');
  }

  if (!parentPath || parentPath === '/') {
    return `/${routePath}`.replace(/\/+$/, '');
  }

  return `${parentPath.replace(/\/$/, '')}/${routePath}`.replace(/\/+$/, '');
}

function findRedirectedChild(children: MenuRoute[], parentPath: string, redirectedPath: string) {
  return children.find((child) => {
    if (child.meta?.hidden === true) {
      return false;
    }

    const childFullPath = normalizeMenuPath(parentPath, child.path);
    return redirectedPath === childFullPath;
  });
}
