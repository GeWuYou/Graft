// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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
