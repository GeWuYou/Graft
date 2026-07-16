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
 * 从路由构建全局菜单搜索索引，并按路径和路由名称去重。
 *
 * @returns 全局菜单搜索项数组
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
 * 查找匹配关键词的菜单项，并按相关性排序。
 *
 * @param keyword - 用于匹配菜单项的搜索词
 * @returns 匹配项，依次按相关性分数降序、标题长度升序和原始顺序排序
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
 * 去除关键词首尾空白并转换为小写。
 *
 * @returns 规范化后的关键词
 */
export function normalizeGlobalMenuSearchKeyword(keyword: string) {
  return keyword.trim().toLowerCase();
}

/**
 * 从路由层级中递归收集可搜索的菜单项。
 *
 * 过滤隐藏路由，解析导航目标，并为菜单项构建标题、层级上下文、关键词及模块信息。
 *
 * @param routes - 要处理的路由
 * @param locale - 用于解析本地化标题的区域设置
 * @returns 扁平化的可搜索菜单项数组
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
      const navigationPath = resolveSearchNavigationPath(route, fullPath);
      const currentItem: GlobalMenuSearchInternalItem[] = isSearchableMenuLeaf(route, fullPath, visibleChildren)
        ? [
            {
              hidden: meta?.hidden,
              icon: typeof meta?.icon === 'string' ? meta.icon : undefined,
              key: routeTitleKey || String(route.name ?? fullPath),
              keywords: extractSearchKeywords(route, meta),
              module: inferSearchModuleKey(route, meta, fullPath),
              navigationPath,
              order: orderRef.value++,
              parentTitleKeys,
              parentTitles,
              path: navigationPath,
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
 * 计算菜单项与搜索关键词的相关性分数。
 *
 * @param order - 菜单项收集顺序，分数相同时用于保持稳定排序
 * @returns 匹配时返回相关性分数及排序元数据，否则返回 `null`
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
 * 判断路由是否应作为菜单叶节点加入搜索索引。
 *
 * @returns 路由可作为菜单叶节点时返回 `true`，否则返回 `false`
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
 * 按面包屑标题、页面标题和兜底来源的优先级解析路由本地化标题。
 *
 * @param route - 路由定义
 * @param meta - 可能包含标题来源的路由元数据
 * @param locale - 用于渲染标题的区域设置
 * @returns 解析出的本地化路由标题；所有来源均无标题时返回空字符串
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
 * 从路由定义收集可搜索关键词。
 *
 * @returns 从路由名称、标题键和元数据附加关键词中提取的关键词数组
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
 * 根据标题键、路由名称或路径推导菜单项的模块键。
 *
 * @returns 规范化后的模块键；无法推导时返回空字符串
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
 * 将模块键规范化为 kebab-case 格式。
 *
 * @param value - 待规范化的模块键
 * @returns kebab-case 格式的模块键
 */
function normalizeSearchModuleKey(value: string) {
  return value
    .replace(/([a-z0-9])([A-Z])/g, '$1-$2')
    .replace(/[_\s]+/g, '-')
    .toLowerCase();
}

/**
 * 解析路由最终使用的导航路径。
 *
 * @param route - 要解析的路由
 * @param fullPath - 路由当前的完整路径
 * @returns 路由配置的导航目标路径、重定向目标路径、首个可见子路由的导航路径，或当前完整路径
 */
function resolveSearchNavigationPath(route: MenuRoute, fullPath: string): string {
  const navigationTargetPath = route.meta?.navigationTargetPath?.trim();
  if (navigationTargetPath) {
    return navigationTargetPath;
  }

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
 * 拼接父级路径和路由路径，并规范化结果。
 *
 * @param parentPath - 基础路径
 * @param routePath - 要追加的路由路径
 * @returns 去除尾部斜杠后的规范化路径；根路径仍保留为 `/`
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
 * 将路由元数据转换为可搜索的元数据变体。
 *
 * @returns 转换为 `SearchableRouteMeta` 的元数据；输入为 `null` 或 `undefined` 时返回 `undefined`
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
