import cloneDeep from 'lodash/cloneDeep';
import type { RouteRecordRaw } from 'vue-router';

import { createLogger } from '@/utils/logger';
import {
  BLANK_LAYOUT,
  EXCEPTION_COMPONENT,
  IFRAME,
  LAYOUT,
  PAGE_NOT_FOUND_ROUTE,
  PARENT_LAYOUT,
} from '@/utils/route/constant';
import type { MenuRoute } from '@/utils/types';

type DynamicRouteItem = Omit<MenuRoute, 'children' | 'component'> & {
  component?: string | (() => Promise<unknown>);
  children?: DynamicRouteItem[];
};

const routeLogger = createLogger('route');

// 动态从包内引入单个Icon,如果没有网络环境可以使用这种方式 但是会导致产物存在多个chunk
// const iconsPath = import.meta.glob('../../../node_modules/tdesign-icons-vue-next/esm/components/*.js');

// async function getMenuIcon(iconName: string): Promise<string> {
//   const RenderIcon = iconsPath[`../../../node_modules/tdesign-icons-vue-next/esm/components/${iconName}.js`];

//   const Icon = await RenderIcon();
//   return shallowRef(Icon.default);
// }

const LayoutMap = new Map<string, () => Promise<typeof import('*.vue')>>();

LayoutMap.set('LAYOUT', LAYOUT);
LayoutMap.set('BLANK', BLANK_LAYOUT);
LayoutMap.set('IFRAME', IFRAME);

let dynamicViewsModules: Record<string, () => Promise<Record<string, unknown>>>;

// 动态引入路由组件
function asyncImportRoute(routes: DynamicRouteItem[] | undefined) {
  dynamicViewsModules = dynamicViewsModules || import.meta.glob('../../pages/**/*.vue');
  if (!routes) return;

  routes.forEach((item) => {
    const { component, name } = item;
    const { children } = item;

    if (typeof component === 'string') {
      const layoutFound = LayoutMap.get(component.toUpperCase());
      if (layoutFound) {
        item.component = layoutFound;
      } else {
        item.component = dynamicImport(dynamicViewsModules, component);
      }
    } else if (name) {
      item.component = PARENT_LAYOUT;
    }

    // 动态从包内引入单个Icon,如果没有网络环境可以使用这种方式 但是会导致产物存在多个chunk
    // if (item.meta.icon) item.meta.icon = await getMenuIcon(item.meta.icon);

    if (children) {
      asyncImportRoute(children);
    }
  });
}

function dynamicImport(dynamicViewsModules: Record<string, () => Promise<Record<string, unknown>>>, component: string) {
  const keys = Object.keys(dynamicViewsModules);
  const matchKeys = keys.filter((key) => {
    const k = key.replace('../../pages', '');
    const startFlag = component.startsWith('/');
    const endFlag = component.endsWith('.vue') || component.endsWith('.tsx');
    const startIndex = startFlag ? 0 : 1;
    const lastIndex = endFlag ? k.length : k.lastIndexOf('.');
    return k.substring(startIndex, lastIndex) === component;
  });
  if (matchKeys?.length === 1) {
    const matchKey = matchKeys[0];
    return dynamicViewsModules[matchKey];
  }
  if (matchKeys?.length > 1) {
    throw new Error(
      'Please do not create `.vue` and `.TSX` files with the same file name in the same hierarchical directory under the views folder. This will cause dynamic introduction failure',
    );
  } else {
    routeLogger.warn(`Can't find ${component} in pages folder`, {
      component,
    });
  }
  return EXCEPTION_COMPONENT;
}

// 将背景对象变成路由对象
export function transformObjectToRoute<T = DynamicRouteItem>(routeList: DynamicRouteItem[]): T[] {
  routeList.forEach((route) => {
    const component = route.component as string;

    if (component) {
      if (component.toUpperCase() === 'LAYOUT') {
        route.component = LayoutMap.get(component.toUpperCase());
      } else {
        route.children = [cloneDeep(route)];
        route.component = LAYOUT;
        route.name = `${String(route.name)}Parent`;
        route.path = '';
        route.meta = route.meta || {};
      }
    } else {
      throw new Error('component is undefined');
    }

    if (route.children) {
      asyncImportRoute(route.children);
    }

    // 动态从包内引入单个Icon,如果没有网络环境可以使用这种方式 但是会导致产物存在多个chunk
    // if (route.meta.icon)
    // route.meta.icon = await getMenuIcon(route.meta.icon);
  });

  return [PAGE_NOT_FOUND_ROUTE, ...routeList] as unknown as T[];
}

export const RUNTIME_ENTRY_FALLBACK_PATH = '/result/404';

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
