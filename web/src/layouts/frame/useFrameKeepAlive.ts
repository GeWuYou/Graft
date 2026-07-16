import uniqBy from 'lodash/uniqBy';
import { computed, toRaw, unref } from 'vue';
import { useRouter } from 'vue-router';

import { useSettingStore, useTabsRouterStore } from '@/store';
import type { MenuRoute } from '@/utils/types';

/**
 * 管理壳层 iframe 页面与 tabs 路由之间的保活关系。
 *
 * 非 tabs 模式只渲染当前 iframe；启用 tabs 后按已打开的外链页保留实例，避免切换页面时丢失外部系统状态。
 */
export function useFrameKeepAlive() {
  const router = useRouter();
  const { currentRoute } = router;
  const { isUseTabsRouter } = useSettingStore();
  const tabStore = useTabsRouterStore();
  const getFramePages = computed(() => {
    const ret = getAllFramePages(toRaw(router.getRoutes()) as unknown as MenuRoute[]) || [];
    return ret;
  });

  const getOpenTabList = computed((): string[] => {
    return tabStore.tabRouters.reduce((prev: string[], next) => {
      if (next.meta && Reflect.has(next.meta, 'frameSrc')) {
        prev.push(next.name as string);
      }
      return prev;
    }, []);
  });

  function getAllFramePages(routes: MenuRoute[]): MenuRoute[] {
    let res: MenuRoute[] = [];
    for (const route of routes) {
      const { meta: { frameSrc, frameBlank } = {}, children } = route;
      if (frameSrc && !frameBlank) {
        res.push(route);
      }
      if (children && children.length) {
        res.push(...getAllFramePages(children));
      }
    }
    res = uniqBy(res, 'name');
    return res;
  }

  function showIframe(item: MenuRoute) {
    return item.name === unref(currentRoute).name;
  }

  function hasRenderFrame(name: string) {
    if (!unref(isUseTabsRouter)) {
      return router.currentRoute.value.name === name;
    }
    return unref(getOpenTabList).includes(name);
  }

  return { hasRenderFrame, getFramePages, showIframe, getAllFramePages };
}
