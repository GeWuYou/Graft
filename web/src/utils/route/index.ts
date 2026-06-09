// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';

import { APP_RESULT_ROUTE_PATH } from '@/contracts/app/routes';

export const RUNTIME_ENTRY_FALLBACK_PATH = APP_RESULT_ROUTE_PATH.NOT_FOUND;
export const RUNTIME_HOME_PATH = '/';

export function isRootEntryPath(path: string) {
  return path === '/' || path === '';
}

export function resolveRuntimeHomePath(routes: RouteRecordRaw[], fallback = RUNTIME_ENTRY_FALLBACK_PATH) {
  for (const route of routes) {
    const resolvedPath = resolveVisibleRoutePath(route);
    if (resolvedPath) {
      return RUNTIME_HOME_PATH;
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
