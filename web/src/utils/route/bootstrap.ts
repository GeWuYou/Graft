import type { RouteRecordRaw } from 'vue-router';

import { getBootstrapRouteRegistration } from '@/modules';
import type { BootstrapMenu } from '@/modules/auth/contract/types';
import type { GlobalRouteRegistration } from '@/modules/types';
import { LAYOUT } from '@/utils/route/constant';
import type { AppRouteMeta, MenuRoute } from '@/utils/types';

import { localizeRouteTitle } from './title';

type BootstrapMenuNode = { menu: BootstrapMenu; children: BootstrapMenuNode[] };

// The explicit bootstrap graph owns visible navigation. Groups never become router records.
export function buildBootstrapNavigationTree(menus: BootstrapMenu[]): MenuRoute[] {
  return buildBootstrapMenuTree(menus)
    .map(buildNavigationNode)
    .filter((node): node is MenuRoute => Boolean(node));
}

export function transformBootstrapMenusToRoutes(menus: BootstrapMenu[]): RouteRecordRaw[] {
  return menus.flatMap((menu) => {
    if (menu.kind !== 'entry' || !menu.path) return [];
    const registration = getBootstrapRouteRegistration(menu.path);
    if (!registration) return [];
    const routeMeta = buildRouteMeta(menu, registration.meta);
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

export function transformGlobalRegistrationsToRoutes(registrations: GlobalRouteRegistration[]): RouteRecordRaw[] {
  return registrations.map((registration) =>
    toRouteRecordRaw({
      path: registration.path,
      name: registration.routeName,
      component: LAYOUT,
      meta: { ...registration.meta, hiddenMenu: true, single: true },
      children: [
        toRouteRecordRaw({
          path: '',
          name: `${registration.routeName}Index`,
          component: registration.loadPage,
          meta: {
            ...registration.meta,
            hiddenMenu: true,
            hiddenBreadcrumb: !registration.meta?.domainTitle,
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

function buildNavigationNode(node: BootstrapMenuNode): MenuRoute | null {
  const children = node.children.map(buildNavigationNode).filter((child): child is MenuRoute => Boolean(child));
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
      ...buildRouteMeta(node.menu, registration?.meta),
      navigationCode: node.menu.code,
      navigationKind: node.menu.kind,
      navigationTargetPath: targetPath,
    },
    children,
  } as MenuRoute;
}

function buildRouteMeta(menu: BootstrapMenu, metaPatch?: Partial<AppRouteMeta>): AppRouteMeta {
  return {
    title: localizeRouteTitle(menu.title, menu.title_key),
    titleKey: menu.title_key,
    icon: menu.icon,
    orderNo: menu.order ?? 0,
    ...metaPatch,
  };
}

function toRouteRecordRaw(route: object): RouteRecordRaw {
  return route as RouteRecordRaw;
}
