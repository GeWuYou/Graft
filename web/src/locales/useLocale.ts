import { useLocalStorage } from '@vueuse/core';
import type { GlobalConfigProvider } from 'tdesign-vue-next';
import { computed } from 'vue';

import { getDefaultLocale, normalizeLocale, type SupportedLocale, toTDesignLocale } from '@/contracts/i18n/locales';
import { STORAGE_KEY } from '@/contracts/storage/keys';
import { i18n } from '@/locales/index';

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
  };

  const getComponentsLocale = computed<GlobalConfigProvider>(() => {
    const componentsLocale = toTDesignLocale(locale.value);
    // 语言包导出只读元组，ConfigProvider 接受可写列表，因此在 provider 边界复制这些数组。
    return {
      ...componentsLocale,
      datePicker: {
        ...componentsLocale.datePicker,
        weekdays: [...componentsLocale.datePicker.weekdays],
        months: [...componentsLocale.datePicker.months],
        quarters: [...componentsLocale.datePicker.quarters],
      },
      rate: { ...componentsLocale.rate, rateText: [...componentsLocale.rate.rateText] },
    };
  });

  return {
    changeLocale,
    getComponentsLocale,
    locale,
  };
}
