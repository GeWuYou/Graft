// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { type SupportedLocale } from '@/contracts/i18n/locales';
import { renderLocalizedTitle, resolveRouteLocalizedTitle } from '@/utils/route/meta';
import type { AppRouteMeta, MenuRoute } from '@/utils/types';

export type GlobalMenuSearchItem = {
  key: string;
  title: string;
  titleKey?: string;
  path: string;
  navigationPath: string;
  routeName?: string;
  icon?: string;
  parentTitles: string[];
  parentTitleKeys?: string[];
  module?: string;
  keywords: string[];
  hidden?: boolean;
};

export type BuildGlobalMenuSearchIndexOptions = {
  locale: SupportedLocale;
};

type SearchableRouteMeta = AppRouteMeta & {
  keywords?: string[];
};

type GlobalMenuSearchInternalItem = GlobalMenuSearchItem & {
  order: number;
};

type GlobalMenuSearchMatchedItem = {
  item: GlobalMenuSearchItem;
  score: number;
  titleLength: number;
};

export function buildGlobalMenuSearchIndex(routes: MenuRoute[], options: BuildGlobalMenuSearchIndexOptions) {
  const items: GlobalMenuSearchInternalItem[] = [];
  const seenPaths = new Set<string>();
  const seenRouteNames = new Set<string>();

  collectGlobalMenuSearchItems(routes, options.locale).forEach((item) => {
    if (seenPaths.has(item.path)) {
      return;
    }

    if (item.routeName && seenRouteNames.has(item.routeName)) {
      return;
    }

    seenPaths.add(item.path);
    if (item.routeName) {
      seenRouteNames.add(item.routeName);
    }
    items.push(item);
  });

  return items.map(({ order: _order, ...item }) => item);
}

export function searchGlobalMenuItems(items: GlobalMenuSearchItem[], keyword: string) {
  const normalizedKeyword = normalizeGlobalMenuSearchKeyword(keyword);
  if (!normalizedKeyword) {
    return [];
  }

  const matchedItems = items
    .map((item, index) => matchGlobalMenuSearchItem(item, normalizedKeyword, index))
    .filter((matched): matched is GlobalMenuSearchMatchedItem & { order: number } => Boolean(matched));

  matchedItems.sort((left, right) => {
    if (left.score !== right.score) {
      return right.score - left.score;
    }

    if (left.titleLength !== right.titleLength) {
      return left.titleLength - right.titleLength;
    }

    return left.order - right.order;
  });

  return matchedItems.map(({ item }) => item);
}

export function normalizeGlobalMenuSearchKeyword(keyword: string) {
  return keyword.trim().toLowerCase();
}

function collectGlobalMenuSearchItems(
  routes: MenuRoute[],
  locale: SupportedLocale,
  parentPath = '',
  parentTitles: string[] = [],
  parentTitleKeys: string[] = [],
  orderRef = { value: 0 },
): GlobalMenuSearchInternalItem[] {
  return [...routes]
    .sort((left, right) => (left.meta?.orderNo ?? 0) - (right.meta?.orderNo ?? 0))
    .flatMap((route) => {
      const meta = toSearchableRouteMeta(route.meta);
      if (meta?.hidden || meta?.hiddenMenu) {
        return [];
      }

      const fullPath = normalizeJoinedMenuPath(parentPath, route.path);
      if (!fullPath) {
        return [];
      }

      const visibleChildren = (route.children ?? []).filter((child) => {
        const childMeta = toSearchableRouteMeta(child.meta);
        return childMeta?.hidden !== true && childMeta?.hiddenMenu !== true;
      });
      const routeTitle = resolveSearchRouteTitle(route, meta, locale);
      const routeTitleKey =
        typeof meta?.titleKey === 'string' && meta.titleKey.trim() ? meta.titleKey.trim() : undefined;
      const nextParentTitles = routeTitle ? [...parentTitles, routeTitle] : [...parentTitles];
      const nextParentTitleKeys = routeTitleKey ? [...parentTitleKeys, routeTitleKey] : [...parentTitleKeys];
      const currentItem: GlobalMenuSearchInternalItem[] = isSearchableMenuLeaf(route, fullPath, visibleChildren)
        ? [
            {
              hidden: meta?.hidden,
              icon: typeof meta?.icon === 'string' ? meta.icon : undefined,
              key: routeTitleKey || String(route.name ?? fullPath),
              keywords: extractSearchKeywords(route, meta),
              module: inferSearchModuleKey(route, meta, fullPath),
              navigationPath: resolveSearchNavigationPath(route, fullPath),
              order: orderRef.value++,
              parentTitleKeys,
              parentTitles,
              path: fullPath,
              routeName: typeof route.name === 'string' ? route.name : undefined,
              title: routeTitle,
              titleKey: routeTitleKey,
            },
          ]
        : [];

      if (visibleChildren.length === 0 || meta?.single) {
        return currentItem;
      }

      return currentItem.concat(
        collectGlobalMenuSearchItems(
          visibleChildren,
          locale,
          fullPath,
          nextParentTitles,
          nextParentTitleKeys,
          orderRef,
        ),
      );
    });
}

function matchGlobalMenuSearchItem(item: GlobalMenuSearchItem, normalizedKeyword: string, order: number) {
  const title = normalizeGlobalMenuSearchKeyword(item.title);
  const parents = normalizeGlobalMenuSearchKeyword(item.parentTitles.join(' / '));
  const path = normalizeGlobalMenuSearchKeyword(item.path);
  const routeName = normalizeGlobalMenuSearchKeyword(item.routeName ?? '');
  const titleKey = normalizeGlobalMenuSearchKeyword(item.titleKey ?? '');
  const moduleKey = normalizeGlobalMenuSearchKeyword(item.module ?? '');
  const keywordPool = item.keywords.map(normalizeGlobalMenuSearchKeyword);

  let score = 0;

  if (title.startsWith(normalizedKeyword)) {
    score = Math.max(score, 1000);
  } else if (title.includes(normalizedKeyword)) {
    score = Math.max(score, 800);
  }

  if (parents.startsWith(normalizedKeyword)) {
    score = Math.max(score, 700);
  } else if (parents.includes(normalizedKeyword)) {
    score = Math.max(score, 600);
  }

  if (path.startsWith(normalizedKeyword)) {
    score = Math.max(score, 500);
  } else if (path.includes(normalizedKeyword)) {
    score = Math.max(score, 420);
  }

  if (routeName.startsWith(normalizedKeyword)) {
    score = Math.max(score, 410);
  } else if (routeName.includes(normalizedKeyword)) {
    score = Math.max(score, 360);
  }

  if (titleKey.includes(normalizedKeyword)) {
    score = Math.max(score, 340);
  }

  if (moduleKey.includes(normalizedKeyword)) {
    score = Math.max(score, 320);
  }

  if (keywordPool.some((keyword) => keyword.includes(normalizedKeyword))) {
    score = Math.max(score, 300);
  }

  if (score <= 0) {
    return null;
  }

  return {
    item,
    order,
    score,
    titleLength: item.title.length,
  };
}

function isSearchableMenuLeaf(route: MenuRoute, fullPath: string, visibleChildren: MenuRoute[]) {
  if (!fullPath) {
    return false;
  }

  const meta = toSearchableRouteMeta(route.meta);
  if (meta?.single) {
    return true;
  }

  if (visibleChildren.length > 0) {
    return false;
  }

  return !route.redirect;
}

function resolveSearchRouteTitle(route: MenuRoute, meta: SearchableRouteMeta | undefined, locale: SupportedLocale) {
  return (
    renderLocalizedTitle(resolveRouteLocalizedTitle(meta, 'breadcrumb'), locale, '') ||
    renderLocalizedTitle(resolveRouteLocalizedTitle(meta, 'page'), locale, '') ||
    renderLocalizedTitle(route.title, locale, '') ||
    renderLocalizedTitle(meta?.title, locale, '')
  );
}

function extractSearchKeywords(route: MenuRoute, meta: SearchableRouteMeta | undefined) {
  const keywords = new Set<string>();

  const routeName = typeof route.name === 'string' ? route.name.trim() : '';
  if (routeName) {
    keywords.add(routeName);
  }

  const titleKey = typeof meta?.titleKey === 'string' ? meta.titleKey.trim() : '';
  if (titleKey) {
    keywords.add(titleKey);
  }

  const metaKeywords = Array.isArray(meta?.keywords) ? meta.keywords : [];
  metaKeywords
    .filter((keyword): keyword is string => typeof keyword === 'string' && Boolean(keyword.trim()))
    .forEach((keyword) => keywords.add(keyword.trim()));

  return [...keywords];
}

function inferSearchModuleKey(route: MenuRoute, meta: SearchableRouteMeta | undefined, fullPath: string) {
  const titleKey = meta?.titleKey?.trim();
  if (titleKey) {
    const [prefix] = titleKey.split('.');
    if (prefix && prefix !== 'menu') {
      return normalizeSearchModuleKey(prefix);
    }
  }

  if (typeof route.name === 'string' && route.name.trim()) {
    const tokens = route.name.match(/[A-Z][a-z0-9]*/g) ?? [];
    const normalizedTokens = tokens.filter((token) => !SEARCH_ROUTE_NAME_NOISE_TOKENS.has(token));
    if (normalizedTokens.length > 0) {
      return normalizeSearchModuleKey(normalizedTokens.join('-'));
    }
  }

  const [firstSegment, secondSegment] = fullPath.split('/').filter(Boolean);
  if (!firstSegment) {
    return '';
  }

  if (firstSegment === 'logs' && secondSegment) {
    return `${secondSegment}-log`;
  }

  return firstSegment;
}

function normalizeSearchModuleKey(value: string) {
  return value
    .replace(/([a-z0-9])([A-Z])/g, '$1-$2')
    .replace(/[_\s]+/g, '-')
    .toLowerCase();
}

function resolveSearchNavigationPath(route: MenuRoute, fullPath: string): string {
  if (typeof route.redirect === 'string' && route.redirect.trim()) {
    const redirectedPath = normalizeJoinedMenuPath(fullPath, route.redirect);
    const redirectedChild = (route.children ?? []).find((child) => {
      const childMeta = toSearchableRouteMeta(child.meta);
      if (childMeta?.hidden === true || childMeta?.hiddenMenu === true) {
        return false;
      }

      return normalizeJoinedMenuPath(fullPath, child.path) === redirectedPath;
    });

    if (redirectedChild) {
      return resolveSearchNavigationPath(redirectedChild, normalizeJoinedMenuPath(fullPath, redirectedChild.path));
    }

    return redirectedPath || fullPath;
  }

  const firstVisibleChild = (route.children ?? []).find((child) => {
    const childMeta = toSearchableRouteMeta(child.meta);
    return childMeta?.hidden !== true && childMeta?.hiddenMenu !== true;
  });
  if (firstVisibleChild) {
    return resolveSearchNavigationPath(firstVisibleChild, normalizeJoinedMenuPath(fullPath, firstVisibleChild.path));
  }

  return fullPath;
}

function normalizeJoinedMenuPath(parentPath: string, routePath: string) {
  const trimmedRoutePath = routePath.trim();
  if (!trimmedRoutePath) {
    return parentPath;
  }

  if (trimmedRoutePath.startsWith('/')) {
    return trimmedRoutePath === '/' ? trimmedRoutePath : trimmedRoutePath.replace(/\/+$/, '');
  }

  if (!parentPath || parentPath === '/') {
    return `/${trimmedRoutePath}`.replace(/\/+$/, '');
  }

  return `${parentPath.replace(/\/$/, '')}/${trimmedRoutePath}`.replace(/\/+$/, '');
}

function toSearchableRouteMeta(meta: MenuRoute['meta']) {
  return (meta ?? undefined) as SearchableRouteMeta | undefined;
}

const SEARCH_ROUTE_NAME_NOISE_TOKENS = new Set([
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
