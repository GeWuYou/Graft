import type { RouteRecordRaw } from 'vue-router';

import type { BootstrapMenu } from '@/api/model/authModel';
import type { LocalizedTitle } from '@/locales';

const bootstrapRouteComponentMap: Record<string, RouteRecordRaw['component']> = {
  '/users': () => import('@/pages/user/index.vue'),
} as const;

const bootstrapLayout: RouteRecordRaw['component'] = () => import('@/layouts/index.vue');

function localizeTitle(title: string): LocalizedTitle {
  return {
    zh_CN: title,
    en_US: title,
  };
}

// transformBootstrapMenusToRoutes 把后端 bootstrap 菜单快照映射为当前 web 可消费的最小动态路由。
//
// 当前阶段只接入已在 `web` 内存在页面实现的真实菜单项，避免继续沿用 starter demo 菜单树。
export function transformBootstrapMenusToRoutes(menus: BootstrapMenu[]): RouteRecordRaw[] {
  return menus.flatMap((menu) => {
    const pageComponent = bootstrapRouteComponentMap[menu.path];
    if (!pageComponent) {
      return [];
    }

    const routeName = menu.code
      .split('.')
      .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
      .join('');

    return [
      {
        path: menu.path,
        component: bootstrapLayout,
        redirect: `${menu.path}/index`,
        name: routeName,
        meta: {
          title: localizeTitle(menu.title),
          icon: menu.icon,
          single: true,
        },
        children: [
          {
            path: 'index',
            name: `${routeName}Index`,
            component: pageComponent,
            meta: {
              hidden: true,
              title: localizeTitle(menu.title),
            },
          },
        ],
      },
    ] as unknown as RouteRecordRaw[];
  });
}
