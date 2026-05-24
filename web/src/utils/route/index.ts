import type { RouteRecordRaw } from 'vue-router';

import { PAGE_NOT_FOUND_ROUTE } from '@/utils/route/constant';

export const RUNTIME_ENTRY_FALLBACK_PATH = PAGE_NOT_FOUND_ROUTE.redirect;

export function isRootEntryPath(path: string) {
  return path === '/' || path === '';
}

export function resolveRuntimeHomePath(routes: RouteRecordRaw[], fallback = RUNTIME_ENTRY_FALLBACK_PATH) {
  for (const route of routes) {
    const resolvedPath = resolveVisibleRoutePath(route);
    if (resolvedPath) {
      return resolvedPath;
    }
  }

  return fallback;
}

function resolveVisibleRoutePath(route: RouteRecordRaw, parentPath = ''): string | null {
  const currentPath = normalizeRoutePath(parentPath, route.path);

  if (currentPath !== '/' && route.meta?.hidden !== true) {
    return currentPath;
  }

  for (const child of route.children ?? []) {
    const childPath = resolveVisibleRoutePath(child, currentPath);
    if (childPath) {
      return childPath;
    }
  }

  return null;
}

function normalizeRoutePath(parentPath: string, routePath: string) {
  if (!routePath) {
    return parentPath || '/';
  }

  if (routePath.startsWith('/')) {
    return routePath;
  }

  if (!parentPath || parentPath === '/') {
    return `/${routePath}`;
  }

  return `${parentPath.replace(/\/$/, '')}/${routePath}`;
}
