import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { useTabsRouterStore } from '@/store/modules/tabs-router';

/**
 * 获取项目页面的共享上下文。
 *
 * @returns 包含 `router`、`tabsRouterStore` 以及 `useI18n()` 返回值的上下文对象
 */
export function useProjectPageContext() {
  return {
    router: useRouter(),
    tabsRouterStore: useTabsRouterStore(),
    ...useI18n(),
  };
}
