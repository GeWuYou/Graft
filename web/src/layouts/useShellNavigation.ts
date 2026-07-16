import { useRouter } from 'vue-router';

import { ROOT_ENTRY_PATH } from '@/contracts/app/routes';
import { useTabsRouterStore } from '@/store/modules/tabs-router';

/**
 * 提供后台壳层的导航动作。
 *
 * 返回首页前先激活固定首页 Tab，确保 tabs 路由状态与实际导航目标保持一致。
 */
export function useShellNavigation() {
  const router = useRouter();
  const tabsRouterStore = useTabsRouterStore();

  const goHome = async () => {
    tabsRouterStore.activateHomeTab();
    await router.push(ROOT_ENTRY_PATH);
  };

  return {
    goHome,
  };
}
