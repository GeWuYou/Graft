import { storeToRefs } from 'pinia';
import { computed } from 'vue';

import { type SupportedLocale } from '@/contracts/i18n/locales';
import { useLocale } from '@/locales/useLocale';
import { usePermissionStore } from '@/store';
import type { MenuRoute } from '@/utils/types';

import type { GlobalMenuSearchItem } from './global-menu-search';
import { buildGlobalMenuSearchIndex, searchGlobalMenuItems } from './global-menu-search';

export type { GlobalMenuSearchItem } from './global-menu-search';
export { normalizeGlobalMenuSearchKeyword } from './global-menu-search';

/**
 * Provides a composable for searching and accessing global menu routes.
 *
 * @returns An object containing the routes initialization state, search index, and a method to search routes by keyword.
 */
export function useGlobalMenuSearch() {
  const permissionStore = usePermissionStore();
  const { routers } = storeToRefs(permissionStore);
  const { locale } = useLocale();

  const searchIndex = computed(() => {
    return buildGlobalMenuSearchIndex(routers.value as MenuRoute[], {
      locale: locale.value as SupportedLocale,
    });
  });

  return {
    routesInitialized: computed(() => permissionStore.routesInitialized),
    searchIndex,
    searchItems: (keyword: string): GlobalMenuSearchItem[] => searchGlobalMenuItems(searchIndex.value, keyword),
  };
}
