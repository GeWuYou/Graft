import type { MenuRoute } from '@/utils/types';

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

export function resolveMenuNavigationPath(menu: MenuRoute, parentPath = ''): string {
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
    const visibleChildren = (menu.children ?? []).filter((child) => child.meta?.hidden !== true);
    const childMatch =
      visibleChildren.length > 0
        ? findExpandedMenuMatch(visibleChildren, activePath, fullPath)
        : { matched: false, expandedPaths: [] };

    if (childMatch.matched) {
      return {
        matched: true,
        expandedPaths: menu.meta?.single ? childMatch.expandedPaths : [fullPath, ...childMatch.expandedPaths],
      };
    }

    const navigationPath = resolveMenuNavigationPath(menu, parentPath);
    const isExpandableMenu = visibleChildren.length > 0 && menu.meta?.single !== true;
    const matchesCurrentMenu =
      activePath === fullPath ||
      activePath === navigationPath ||
      activePath.startsWith(`${fullPath}/`) ||
      activePath.startsWith(`${navigationPath}/`);

    if (matchesCurrentMenu) {
      return {
        matched: true,
        expandedPaths: isExpandableMenu ? [fullPath] : [],
      };
    }
  }

  return { matched: false, expandedPaths: [] };
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
