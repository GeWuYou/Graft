import { useLocalStorage } from '@vueuse/core';
import type { GlobalConfigProvider } from 'tdesign-vue-next';
import { computed } from 'vue';

import { getDefaultLocale, normalizeLocale, type SupportedLocale } from '@/contracts/i18n/locales';
import { STORAGE_KEY } from '@/contracts/storage/keys';
import { i18n } from '@/locales/index';
import { useNotificationStore } from '@/store/modules/notification';

export function useLocale() {
  const locale = computed({
    get: () => normalizeLocale(i18n.global.locale.value) ?? getDefaultLocale(),
    set: (val: string) => {
      i18n.global.locale.value = normalizeLocale(val) ?? getDefaultLocale();
    },
  });
  const storedLocale = useLocalStorage<SupportedLocale>(STORAGE_KEY.LOCALE, getDefaultLocale());

  const changeLocale = (lang: string) => {
    const validLang = normalizeLocale(lang) ?? getDefaultLocale();
    locale.value = validLang;
    storedLocale.value = validLang;
    // 刷新持久化的翻译数据
    useNotificationStore().refreshMsgData();
  };

  const getComponentsLocale = computed(() => {
    return i18n.global.getLocaleMessage(locale.value).componentsLocale as GlobalConfigProvider;
  });

  return {
    changeLocale,
    getComponentsLocale,
    locale,
  };
}
