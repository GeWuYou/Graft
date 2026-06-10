// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { useRouter } from 'vue-router';

import { ROOT_ENTRY_PATH } from '@/contracts/app/routes';
import { useTabsRouterStore } from '@/store/modules/tabs-router';

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
