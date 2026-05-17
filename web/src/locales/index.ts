import type { DropdownOption } from 'tdesign-vue-next';
import { computed } from 'vue';
import type { I18nOptions } from 'vue-i18n';
import { createI18n } from 'vue-i18n';

import {
  getDefaultLocale,
  normalizeLocale,
  type SupportedLocale,
  supportedLocales,
  toTDesignLocale,
} from '@/contracts/i18n/locales';
import { STORAGE_KEY } from '@/contracts/storage/keys';

export const localeConfigKey = STORAGE_KEY.LOCALE;
export type { LocalizedTitle, SupportedLocale } from '@/contracts/i18n/locales';
export { supportedLocales } from '@/contracts/i18n/locales';

const langModules = import.meta.glob<{ default: Record<string, unknown> }>('./lang/*.json', { eager: true });

const langCode: SupportedLocale[] = [];
const messages: I18nOptions['messages'] = {};
const langList: DropdownOption[] = [];

Object.entries(langModules).forEach(([path, module]) => {
  const code = path.match(/\.\/lang\/([^.]+)\.json$/)?.[1] as SupportedLocale | undefined;
  if (!code || !supportedLocales.includes(code)) return;
  langCode.push(code);
  messages[code] = { ...module.default, componentsLocale: toTDesignLocale(code) };
  langList.push({ content: module.default.lang as string, value: code });
});

export { langCode };

// 获取初始语言：优先本地存储，其次浏览器偏好，最后默认中文
const getInitialLocale = (): SupportedLocale => {
  try {
    const stored = normalizeLocale(localStorage.getItem(localeConfigKey));
    if (stored) {
      localStorage.setItem(localeConfigKey, stored);
      return stored;
    }
  } catch {
    // 某些受限环境会禁用本地存储，此时回退到浏览器语言。
  }

  const preferred = normalizeLocale(navigator.languages?.[0] ?? navigator.language);
  if (preferred) {
    return preferred;
  }

  return getDefaultLocale();
};

const initialLocale = getInitialLocale();

export const i18n = createI18n({
  legacy: false,
  locale: initialLocale,
  fallbackLocale: getDefaultLocale(),
  messages,
  globalInjection: true,
});

export const languageList = computed(() => langList);
export const { t } = i18n.global;
export default i18n;
