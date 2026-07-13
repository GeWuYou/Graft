import type { AppRouteMeta, MenuRoute } from '@/utils/types';

export type SidebarMotionMode = NonNullable<AppRouteMeta['sidebarMotion']>;

export type SidebarMotionPhase =
  | 'expanded'
  | 'collapsing-width'
  | 'collapsing-submenu'
  | 'collapsing-topmenu'
  | 'compact'
  | 'expanding-width'
  | 'expanding-topmenu'
  | 'expanding-submenu';

/**
 * 根据页面元数据解析侧边栏动效模式。
 *
 * @param meta - 页面路由元数据
 * @returns 配置的动效模式；列表页且页面表面类型适用时返回 `wide-table`，否则返回 `default`
 */
export function resolveSidebarMotionMode(meta?: AppRouteMeta): SidebarMotionMode {
  if (meta?.sidebarMotion) {
    return meta.sidebarMotion;
  }

  return meta?.pageKind === 'list' && (meta.pageSurface === undefined || meta.pageSurface === 'paged-table')
    ? 'wide-table'
    : 'default';
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
 * 选择混合布局侧边栏中包含当前路由的顶级菜单分支。
 *
 * @param menus - 顶级菜单列表
 * @param activePath - 当前激活路由路径
 * @returns 仅包含匹配分支且将其标记为展开的菜单列表；未找到匹配分支时返回原菜单列表
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
 * 返回菜单树中所有实际渲染为子菜单的可见节点值。
 *
 * @param menus - 待遍历的菜单列表
 * @returns 可用于受控展开的子菜单值
 */
export function findAllExpandedMenuPaths(menus: MenuRoute[]): string[] {
  return menus.flatMap((menu) => {
    const visibleChildren = getExpandableMenuChildren(menu);
    if (!visibleChildren) {
      return [];
    }

    return [menu.path, ...findAllExpandedMenuPaths(visibleChildren)];
  });
}

/**
 * 基于当前已展开的顶层值，补全对应分支中全部可展开的后代。
 *
 * @param menus - 待遍历的菜单列表
 * @param expandedPaths - 菜单组件当前报告的展开值
 * @returns 当前分支及其可见后代的完整展开值
 */
export function findExpandedMenuBranchPaths(
  menus: MenuRoute[],
  expandedPaths: ReadonlyArray<string | number>,
): string[] {
  const expandedPathSet = new Set(expandedPaths);

  return menus.flatMap((menu) => {
    const visibleChildren = getExpandableMenuChildren(menu);
    if (!visibleChildren) {
      return [];
    }

    if (expandedPathSet.has(menu.path)) {
      return [menu.path, ...findAllExpandedMenuPaths(visibleChildren)];
    }

    return findExpandedMenuBranchPaths(visibleChildren, expandedPaths);
  });
}

function getExpandableMenuChildren(menu: MenuRoute): MenuRoute[] | null {
  if (menu.meta?.hidden === true || menu.meta?.single === true) {
    return null;
  }

  const visibleChildren = (menu.children ?? []).filter((child) => child.meta?.hidden !== true);
  return visibleChildren.length > 0 ? visibleChildren : null;
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

/**
 * 判断菜单项或其可见子菜单是否包含当前激活路径。
 *
 * @param menu - 要检查的菜单项
 * @param activePath - 当前激活的路由路径
 * @param parentPath - 父级菜单的路径
 * @returns 如果激活路径对应菜单项或其子路径则为 `true`，否则为 `false`
 */
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

/**
 * 根据父级路径和路由路径生成规范化的菜单路径。
 *
 * @param parentPath - 父级菜单路径
 * @param routePath - 当前菜单的路由路径
 * @returns 规范化后的绝对菜单路径
 */
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
