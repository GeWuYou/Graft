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

/**
 * Builds a searchable index of global menu items from routes, deduplicating by path and route name.
 *
 * @returns Array of global menu search items
 */
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

/**
 * Finds menu items matching a keyword and ranks them by relevance.
 *
 * @param keyword - The search term to match against menu items
 * @returns Matching items, sorted by relevance score (highest first), title length (shortest first), and original item order.
 */
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

/**
 * Normalizes a keyword by removing whitespace and converting to lowercase.
 *
 * @returns The normalized keyword
 */
export function normalizeGlobalMenuSearchKeyword(keyword: string) {
  return keyword.trim().toLowerCase();
}

/**
 * Recursively collects searchable menu items from a route hierarchy.
 *
 * Filters out hidden routes, computes full paths, derives titles and title keys, and builds hierarchical metadata. Returns a flattened array of leaf items with associated keywords, module information, and parent context. Uses the `orderRef` object to maintain consistent global ordering across recursive calls.
 *
 * @param routes - The routes to process
 * @param locale - The locale for resolving localized titles
 * @returns A flattened array of searchable menu items with computed metadata
 */
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

/**
 * Scores a menu item's relevance to a search keyword.
 *
 * @param order - The item's collection order, used for stable sorting when scores are equal
 * @returns An object with the relevance score and sort metadata if the item matches, `null` otherwise
 */
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

/**
 * Determines whether a route should be treated as a menu leaf node for search indexing.
 *
 * @returns `true` if the route is a leaf node for menu indexing, `false` otherwise.
 */
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

/**
 * Resolves the localized title for a route by attempting breadcrumb, page, and fallback sources in priority order.
 *
 * @param route - The route definition
 * @param meta - The route's metadata containing potential title sources
 * @param locale - The target locale for title rendering
 * @returns The localized route title, or an empty string if no title is available from any source
 */
function resolveSearchRouteTitle(route: MenuRoute, meta: SearchableRouteMeta | undefined, locale: SupportedLocale) {
  return (
    renderLocalizedTitle(resolveRouteLocalizedTitle(meta, 'breadcrumb'), locale, '') ||
    renderLocalizedTitle(resolveRouteLocalizedTitle(meta, 'page'), locale, '') ||
    renderLocalizedTitle(route.title, locale, '') ||
    renderLocalizedTitle(meta?.title, locale, '')
  );
}

/**
 * Collects searchable keywords from a route definition.
 *
 * @returns An array of keywords derived from the route name, title key, and any additional keywords in meta.
 */
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

/**
 * Derives a module key for a menu item based on its title key, route name, or path.
 *
 * @returns A normalized module key string, or an empty string if no key could be derived
 */
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

/**
 * Normalizes a module key to kebab-case format.
 *
 * @param value - The module key to normalize
 * @returns The normalized module key in kebab-case
 */
function normalizeSearchModuleKey(value: string) {
  return value
    .replace(/([a-z0-9])([A-Z])/g, '$1-$2')
    .replace(/[_\s]+/g, '-')
    .toLowerCase();
}

/**
 * Determines the navigation destination for a route, following redirects and descending to the first visible child.
 *
 * @param route - The route to resolve
 * @param fullPath - The current full path of the route
 * @returns The navigation path to use
 */
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

/**
 * Joins a parent path with a route path, normalizing the result.
 *
 * @param parentPath - The base path
 * @param routePath - The route path to append
 * @returns The normalized joined path with trailing slashes removed, except the root path remains `/`
 */
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

/**
 * Casts a route meta object to the searchable variant.
 *
 * @returns The meta object cast as `SearchableRouteMeta`, or `undefined` if the input is `null` or `undefined`
 */
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
