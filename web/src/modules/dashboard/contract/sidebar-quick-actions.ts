// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';

import { getDefaultLocale, type SupportedLocale } from '@/contracts/i18n/locales';
import { renderLocalizedTitle, resolveRouteLocalizedTitle } from '@/utils/route/meta';
import type { AppRouteMeta } from '@/utils/types';

import type { DashboardQuickActionLink } from './quick-action-links';
import { normalizeDashboardRoutePath, normalizeJoinedDashboardRoutePath } from './route-paths';

type QuickActionSource = Pick<RouteRecordRaw, 'path' | 'children' | 'name' | 'meta'>;
interface QuickActionParent {
  groupKey?: string;
  groupLabel?: string;
}
type QuickActionRouteMeta = AppRouteMeta & {
  permission?: string;
  required_permissions?: string[];
  requiredPermissions?: string[];
};

/**
 * Builds a sorted list of dashboard quick action links from Vue Router routes.
 *
 * @param routes - The array of Vue Router route records to extract quick action links from.
 * @returns An array of DashboardQuickActionLink objects sorted by order (ascending), then by id (lexicographically).
 */
export function buildDashboardQuickActionLinks(routes: RouteRecordRaw[], locale: SupportedLocale = getDefaultLocale()) {
  return collectLeafLinks(routes, locale).sort(compareQuickActions);
}

/**
 * Recursively collects visible leaf routes and transforms them into dashboard quick action links.
 *
 * @returns An array of dashboard quick action links for visible leaf routes
 */
function collectLeafLinks(
  routes: QuickActionSource[],
  locale: SupportedLocale,
  parentPath = '',
  parent?: QuickActionParent,
): DashboardQuickActionLink[] {
  return routes.flatMap((route) => {
    const routeMeta = toRouteMeta(route.meta);
    const fullPath = normalizeJoinedDashboardRoutePath(parentPath, String(route.path ?? ''));
    if (!fullPath || routeMeta?.hidden || routeMeta?.hiddenMenu) {
      return [];
    }

    const nextParent = resolveParent(routeMeta, locale, parent);
    if (routeMeta?.single && isQuickActionLeaf(route, fullPath)) {
      return [buildQuickActionLink(route, routeMeta, fullPath, locale, parent)];
    }

    const visibleChildren = collectLeafLinks(route.children ?? [], locale, fullPath, nextParent);
    if (visibleChildren.length > 0) {
      return visibleChildren;
    }

    if (!isQuickActionLeaf(route, fullPath)) {
      return [];
    }

    return [buildQuickActionLink(route, routeMeta, fullPath, locale, parent)];
  });
}

/**
 * Determines if a route is eligible as a quick-action leaf node.
 *
 * @returns `true` if the route has a name and valid path, is not hidden, and either is marked as single or has no children, `false` otherwise.
 */
function isQuickActionLeaf(route: QuickActionSource, fullPath: string) {
  const routeMeta = toRouteMeta(route.meta);
  if (!route.name || !fullPath) {
    return false;
  }

  if (routeMeta?.single) {
    return !routeMeta.hidden && !routeMeta.hiddenMenu;
  }

  if ((route.children?.length ?? 0) > 0) {
    return false;
  }

  return !routeMeta?.hidden && !routeMeta?.hiddenMenu;
}

/**
 * Types route metadata as AppRouteMeta.
 *
 * @param meta - The raw metadata value
 * @returns The metadata as `AppRouteMeta`, or `undefined` if the input was nullish
 */
function toRouteMeta(meta: unknown) {
  return (meta ?? undefined) as QuickActionRouteMeta | undefined;
}

function buildQuickActionLink(
  route: QuickActionSource,
  routeMeta: QuickActionRouteMeta | undefined,
  fullPath: string,
  locale: SupportedLocale,
  parent?: QuickActionParent,
): DashboardQuickActionLink {
  const title =
    renderLocalizedTitle(resolveRouteLocalizedTitle(routeMeta, 'breadcrumb'), locale) ||
    renderLocalizedTitle(resolveRouteLocalizedTitle(routeMeta, 'page'), locale) ||
    fullPath;
  const fullLabel = renderLocalizedTitle(resolveRouteLocalizedTitle(routeMeta, 'tab'), locale) || title;
  const routeGroup = renderLocalizedTitle(resolveRouteLocalizedTitle(routeMeta, 'page'), locale) || parent?.groupLabel;
  const isSingleLeaf = Boolean(routeMeta?.single);

  return {
    id: String(route.name ?? fullPath),
    full_label: fullLabel,
    group: isSingleLeaf ? routeGroup : parent?.groupLabel,
    group_key: isSingleLeaf ? routeMeta?.titleKey : parent?.groupKey,
    icon: typeof routeMeta?.icon === 'string' ? routeMeta.icon : undefined,
    module_key: inferModuleKey(route, fullPath, routeMeta),
    order: routeMeta?.orderNo ?? 0,
    required_permissions: resolveRequiredPermissions(routeMeta),
    route_location: fullPath,
    title,
    title_key: routeMeta?.titleKey,
  };
}

/**
 * Resolves grouping context for a route based on its metadata and locale.
 *
 * @returns A grouping context with `groupLabel` and `groupKey`, or the parent context if no new values are available
 */
function resolveParent(
  routeMeta: QuickActionRouteMeta | undefined,
  locale: SupportedLocale,
  parent?: QuickActionParent,
) {
  const groupLabel =
    renderLocalizedTitle(resolveRouteLocalizedTitle(routeMeta, 'page'), locale) ||
    renderLocalizedTitle(routeMeta?.title, locale) ||
    parent?.groupLabel;
  const groupKey = routeMeta?.titleKey || parent?.groupKey;

  if (!groupLabel && !groupKey) {
    return parent;
  }

  return {
    groupKey,
    groupLabel,
  };
}

function resolveRequiredPermissions(routeMeta: QuickActionRouteMeta | undefined) {
  const requiredPermissions = routeMeta?.required_permissions ?? routeMeta?.requiredPermissions;
  if (Array.isArray(requiredPermissions) && requiredPermissions.length > 0) {
    return [...requiredPermissions];
  }

  if (typeof routeMeta?.permission === 'string' && routeMeta.permission.trim()) {
    return [routeMeta.permission];
  }

  return undefined;
}

/**
 * Derives a module key from dashboard route authority.
 *
 * Prefers route authority that is already stable in the frontend contract:
 * `titleKey`, then route name, then the normalized path as the last fallback.
 *
 * @param route - The quick-action source route
 * @param path - The normalized dashboard route path
 * @param routeMeta - The typed route metadata
 * @returns The module key derived from the best available authority source
 */
function inferModuleKey(route: QuickActionSource, path: string, routeMeta: QuickActionRouteMeta | undefined) {
  const byTitleKey = inferModuleKeyFromTitleKey(routeMeta?.titleKey);
  if (byTitleKey) {
    return byTitleKey;
  }

  const byRouteName = inferModuleKeyFromRouteName(route.name);
  if (byRouteName) {
    return byRouteName;
  }
  const segments = normalizeDashboardRoutePath(path).split('/').filter(Boolean);
  if (segments.length === 0) {
    return 'dashboard';
  }

  if (segments[0] === 'logs' && segments[1]) {
    return `${segments[1]}-log`;
  }

  return segments[0];
}

function inferModuleKeyFromTitleKey(titleKey?: string) {
  if (!titleKey) {
    return '';
  }

  const [prefix] = titleKey.split('.');
  if (!prefix || prefix === 'menu') {
    return '';
  }

  return normalizeModuleKey(prefix);
}

function inferModuleKeyFromRouteName(name: QuickActionSource['name']) {
  if (typeof name !== 'string' || !name.trim()) {
    return '';
  }

  const tokens = name.match(/[A-Z][a-z0-9]*/g) ?? [];
  const meaningfulTokens = tokens.filter((token) => !ROUTE_NAME_NOISE_TOKENS.has(token));
  if (meaningfulTokens.length === 0) {
    return '';
  }

  return normalizeModuleKey(meaningfulTokens.join('-'));
}

function normalizeModuleKey(value: string) {
  return value
    .replace(/([a-z0-9])([A-Z])/g, '$1-$2')
    .replace(/[_\s]+/g, '-')
    .toLowerCase();
}

/**
 * Orders two dashboard quick action links by their order field, then by their ID.
 *
 * @returns A negative number if `left` should come before `right`, zero if equal, positive otherwise.
 */
function compareQuickActions(left: DashboardQuickActionLink, right: DashboardQuickActionLink) {
  if (left.order !== right.order) {
    return left.order - right.order;
  }

  return left.id.localeCompare(right.id);
}

const ROUTE_NAME_NOISE_TOKENS = new Set([
  'Bootstrap',
  'Group',
  'Index',
  'List',
  'Overview',
  'Detail',
  'Runtime',
  'Dependencies',
  'Management',
  'Page',
]);
