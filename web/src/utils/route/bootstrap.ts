import type { RouteRecordRaw } from 'vue-router';

import type { LocalizedTitle } from '@/contracts/i18n/locales';
import { getBootstrapRouteRegistration } from '@/modules';
import type { BootstrapMenu } from '@/modules/auth/contract/types';
import type { GlobalRouteRegistration } from '@/modules/types';
import { LAYOUT } from '@/utils/route/constant';
import type { AppRouteMeta, MenuRoute, NavigationAncestor } from '@/utils/types';

import { localizeRouteTitle } from './title';

type BootstrapMenuNode = { menu: BootstrapMenu; children: BootstrapMenuNode[] };

// The explicit bootstrap graph owns visible navigation. Groups never become router records.
export function buildBootstrapNavigationTree(menus: BootstrapMenu[]): MenuRoute[] {
  const ancestorsByEntryPath = buildNavigationAncestorsByEntryPath(menus);
  return buildBootstrapMenuTree(menus)
    .map((node) => buildNavigationNode(node, ancestorsByEntryPath))
    .filter((node): node is MenuRoute => Boolean(node));
}

export function transformBootstrapMenusToRoutes(menus: BootstrapMenu[]): RouteRecordRaw[] {
  const ancestorsByEntryPath = buildNavigationAncestorsByEntryPath(menus);
  return menus.flatMap((menu) => {
    if (menu.kind !== 'entry' || !menu.path) return [];
    const registration = getBootstrapRouteRegistration(menu.path);
    if (!registration) return [];
    const routeMeta = buildRouteMeta(menu, registration.meta, ancestorsByEntryPath.get(menu.path));
    return [
      toRouteRecordRaw({
        path: menu.path,
        name: registration.routeName,
        component: LAYOUT,
        meta: routeMeta,
        children: [
          toRouteRecordRaw({
            path: '',
            name: `${registration.routeName}Index`,
            component: registration.loadPage,
            meta: { ...routeMeta, hiddenBreadcrumb: true },
          }),
        ],
      }),
    ];
  });
}

export function transformGlobalRegistrationsToRoutes(
  registrations: GlobalRouteRegistration[],
  menus: BootstrapMenu[] = [],
): RouteRecordRaw[] {
  const ancestorsByEntryPath = buildNavigationAncestorsByEntryPath(menus);
  return registrations.map((registration) =>
    toRouteRecordRaw({
      path: registration.path,
      name: registration.routeName,
      component: LAYOUT,
      meta: {
        ...withNavigationContext(registration.meta, ancestorsByEntryPath.get(registration.navigationParentPath ?? '')),
        hiddenMenu: true,
        single: true,
      },
      children: [
        toRouteRecordRaw({
          path: '',
          name: `${registration.routeName}Index`,
          component: registration.loadPage,
          meta: {
            ...withNavigationContext(
              registration.meta,
              ancestorsByEntryPath.get(registration.navigationParentPath ?? ''),
            ),
            hiddenMenu: true,
            hiddenBreadcrumb:
              !registration.meta?.domainTitle && !ancestorsByEntryPath.has(registration.navigationParentPath ?? ''),
          },
        }),
      ],
    }),
  );
}

function buildBootstrapMenuTree(menus: BootstrapMenu[]): BootstrapMenuNode[] {
  const nodeMap = new Map<string, BootstrapMenuNode>();
  const roots: BootstrapMenuNode[] = [];
  for (const menu of menus) nodeMap.set(menu.code, { menu, children: [] });
  for (const menu of menus) {
    const node = nodeMap.get(menu.code)!;
    const parent = menu.parent_code ? nodeMap.get(menu.parent_code) : undefined;
    if (parent) parent.children.push(node);
    else roots.push(node);
  }
  const sort = (nodes: BootstrapMenuNode[]) => {
    nodes.sort((a, b) => (a.menu.order ?? 0) - (b.menu.order ?? 0));
    nodes.forEach((node) => sort(node.children));
  };
  sort(roots);
  return roots;
}

function buildNavigationNode(
  node: BootstrapMenuNode,
  ancestorsByEntryPath: Map<string, NavigationAncestor[]>,
): MenuRoute | null {
  const children = node.children
    .map((child) => buildNavigationNode(child, ancestorsByEntryPath))
    .filter((child): child is MenuRoute => Boolean(child));
  const registration =
    node.menu.kind === 'entry' && node.menu.path ? getBootstrapRouteRegistration(node.menu.path) : undefined;
  if (node.menu.kind === 'entry' && !registration) return null;
  if (node.menu.kind === 'group' && children.length === 0) return null;
  const targetPath = node.menu.kind === 'entry' ? node.menu.path : children[0]?.meta?.navigationTargetPath;
  if (!targetPath) return null;
  return {
    path: node.menu.code,
    name: registration?.routeName ?? node.menu.code,
    title: localizeRouteTitle(node.menu.title, node.menu.title_key),
    meta: {
      ...buildRouteMeta(
        node.menu,
        registration?.meta,
        node.menu.path ? ancestorsByEntryPath.get(node.menu.path) : undefined,
      ),
      navigationCode: node.menu.code,
      navigationKind: node.menu.kind,
      navigationTargetPath: targetPath,
    },
    children,
  } as MenuRoute;
}

function buildRouteMeta(
  menu: BootstrapMenu,
  metaPatch?: Partial<AppRouteMeta>,
  navigationAncestors?: NavigationAncestor[],
): AppRouteMeta {
  return {
    title: localizeRouteTitle(menu.title, menu.title_key),
    titleKey: menu.title_key,
    icon: menu.icon,
    orderNo: menu.order ?? 0,
    ...withNavigationContext(metaPatch, navigationAncestors, localizeRouteTitle(menu.title, menu.title_key)),
  };
}

function withNavigationContext(
  metaPatch: Partial<AppRouteMeta> | undefined,
  navigationAncestors: NavigationAncestor[] | undefined,
  currentTitle?: LocalizedTitle,
): Partial<AppRouteMeta> {
  const context = navigationAncestors?.length
    ? {
        navigationAncestors,
        navigationTitle: joinLocalizedTitles([
          ...navigationAncestors.map((ancestor) => ancestor.title),
          metaPatch?.tabTitle ?? metaPatch?.semanticTitle ?? metaPatch?.title ?? currentTitle,
        ]),
      }
    : {};
  return { ...metaPatch, ...context };
}

function buildNavigationAncestorsByEntryPath(menus: BootstrapMenu[]) {
  const nodeMap = new Map(menus.map((menu) => [menu.code, menu]));
  const ancestorsByEntryPath = new Map<string, NavigationAncestor[]>();
  for (const menu of menus) {
    if (menu.kind !== 'entry' || !menu.path) continue;
    const ancestors: NavigationAncestor[] = [];
    let parentCode = menu.parent_code;
    while (parentCode) {
      const parent = nodeMap.get(parentCode);
      if (!parent) break;
      ancestors.unshift({
        code: parent.code,
        path: parent.path ?? menu.path,
        title: localizeRouteTitle(parent.title, parent.title_key),
      });
      parentCode = parent.parent_code;
    }
    ancestorsByEntryPath.set(menu.path, ancestors);
  }
  return ancestorsByEntryPath;
}

function joinLocalizedTitles(titles: Array<LocalizedTitle | undefined>): LocalizedTitle | undefined {
  const resolved = titles.filter((title): title is LocalizedTitle => Boolean(title));
  if (resolved.length === 0) return undefined;
  return Object.fromEntries(
    Object.keys(resolved[0]).map((locale) => [
      locale,
      resolved
        .map((title) => title[locale as keyof LocalizedTitle])
        .filter(Boolean)
        .join(' / '),
    ]),
  ) as LocalizedTitle;
}

function toRouteRecordRaw(route: object): RouteRecordRaw {
  return route as RouteRecordRaw;
}
