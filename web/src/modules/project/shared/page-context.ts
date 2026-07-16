import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { useTabsRouterStore } from '@/store/modules/tabs-router';

export function useApplicationPageContext() {
  return {
    router: useRouter(),
    tabsRouterStore: useTabsRouterStore(),
    ...useI18n(),
  };
}
