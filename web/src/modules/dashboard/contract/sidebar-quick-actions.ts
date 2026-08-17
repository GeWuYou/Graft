import { getDefaultLocale, type SupportedLocale } from '@/contracts/i18n/locales';
import { renderLocalizedTitle, resolveRouteLocalizedTitle } from '@/utils/route/meta';
import type { AppRouteMeta, MenuRoute } from '@/utils/types';

import type { DashboardQuickActionLink } from './quick-action-links';
import { normalizeDashboardRoutePath, normalizeJoinedDashboardRoutePath } from './route-paths';

type QuickActionSource = Pick<MenuRoute, 'path' | 'children' | 'name' | 'meta'>;
interface QuickActionParent {
  groupKey?: string;
  groupLabel?: string;
  order: number;
}
type QuickActionRouteMeta = AppRouteMeta & {
  permission?: string;
  required_permissions?: string[];
  requiredPermissions?: string[];
};

/** 从可见叶子路由生成快速入口，并保持配置顺序与标识符排序的稳定性。 */
export function buildDashboardQuickActionLinks(
  routes: Array<QuickActionSource | RouteRecordRaw>,
  locale: SupportedLocale = getDefaultLocale(),
) {
  return collectLeafLinks(routes as QuickActionSource[], locale).sort(compareQuickActions);
}

function collectLeafLinks(
  routes: QuickActionSource[],
  locale: SupportedLocale,
  parentPath = '',
  parent?: QuickActionParent,
  section?: QuickActionParent,
): DashboardQuickActionLink[] {
  return routes.flatMap((route) => {
    const routeMeta = toRouteMeta(route.meta);
    const fullPath =
      routeMeta?.navigationTargetPath ?? normalizeJoinedDashboardRoutePath(parentPath, String(route.path ?? ''));
    if (!fullPath || routeMeta?.hidden || routeMeta?.hiddenMenu) {
      return [];
    }

    const nextParent = resolveParent(routeMeta, locale, parent);
    const nextSection = section ?? nextParent;
    if (routeMeta?.single && isQuickActionLeaf(route, fullPath)) {
      return [buildQuickActionLink(route, routeMeta, fullPath, locale, parent, nextSection)];
    }

    const visibleChildren = collectLeafLinks(route.children ?? [], locale, fullPath, nextParent, nextSection);
    if (visibleChildren.length > 0) {
      return visibleChildren;
    }

    if (!isQuickActionLeaf(route, fullPath)) {
      return [];
    }

    return [buildQuickActionLink(route, routeMeta, fullPath, locale, parent, nextSection)];
  });
}

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

function toRouteMeta(meta: unknown) {
  return (meta ?? undefined) as QuickActionRouteMeta | undefined;
}

function buildQuickActionLink(
  route: QuickActionSource,
  routeMeta: QuickActionRouteMeta | undefined,
  fullPath: string,
  locale: SupportedLocale,
  parent?: QuickActionParent,
  section?: QuickActionParent,
): DashboardQuickActionLink {
  const title =
    renderLocalizedTitle(resolveRouteLocalizedTitle(routeMeta, 'breadcrumb'), locale) ||
    renderLocalizedTitle(resolveRouteLocalizedTitle(routeMeta, 'page'), locale) ||
    fullPath;
  const fullLabel = renderLocalizedTitle(resolveRouteLocalizedTitle(routeMeta, 'tab'), locale) || title;
  const routeGroup = renderLocalizedTitle(resolveRouteLocalizedTitle(routeMeta, 'page'), locale) || parent?.groupLabel;
  const isSingleLeaf = Boolean(routeMeta?.single);
  const requiredPermissions = resolveRequiredPermissions(routeMeta);

  return {
    id: String(route.name ?? fullPath),
    full_label: fullLabel,
    group: isSingleLeaf ? routeGroup : parent?.groupLabel,
    group_key: isSingleLeaf ? routeMeta?.titleKey : parent?.groupKey,
    icon: typeof routeMeta?.icon === 'string' ? routeMeta.icon : undefined,
    module_key: inferModuleKey(route, fullPath, routeMeta),
    order: routeMeta?.orderNo ?? 0,
    ...(requiredPermissions ? { required_permissions: requiredPermissions } : {}),
    route_location: fullPath,
    ...(section?.groupLabel
      ? {
          section: section.groupLabel,
          ...(section.groupKey ? { section_key: section.groupKey } : {}),
          section_order: section.order,
        }
      : {}),
    title,
    title_key: routeMeta?.titleKey,
  };
}

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
    order: routeMeta?.orderNo ?? parent?.order ?? 0,
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

/** 按稳定的路由标题契约、路由名、规范化路径依次推导模块标识，避免依赖展示文案。 */
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
import type { RouteRecordRaw } from 'vue-router';
