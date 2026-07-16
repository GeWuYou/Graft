import type { DropdownOption } from 'tdesign-vue-next';
import { computed, readonly } from 'vue';
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
const moduleLangModules = import.meta.glob<{ default: Record<string, unknown> }>('../modules/*/locales/**/*.json', {
  eager: true,
});

const langCode: SupportedLocale[] = [];
const messages: I18nOptions['messages'] = {};
const langList: DropdownOption[] = [];

function isPlainMessageRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value));
}

export function mergeLocaleMessages(
  target: Record<string, unknown>,
  source: Record<string, unknown>,
): Record<string, unknown> {
  const merged = { ...target };

  for (const [key, value] of Object.entries(source)) {
    const existing = merged[key];
    if (isPlainMessageRecord(existing) && isPlainMessageRecord(value)) {
      merged[key] = mergeLocaleMessages(existing, value);
      continue;
    }

    merged[key] = value;
  }

  return merged;
}

Object.entries(langModules).forEach(([path, module]) => {
  const code = path.match(/\.\/lang\/([^.]+)\.json$/)?.[1] as SupportedLocale | undefined;
  if (!code || !supportedLocales.includes(code)) return;
  langCode.push(code);
  messages[code] = { ...module.default, componentsLocale: toTDesignLocale(code) };
  langList.push({ content: module.default.lang as string, value: code });
});

Object.entries(moduleLangModules).forEach(([path, module]) => {
  const code = path.match(/\/([^./]+)\.json$/)?.[1] as SupportedLocale | undefined;
  if (!code || !supportedLocales.includes(code)) return;

  messages[code] = {
    ...mergeLocaleMessages((messages[code] ?? {}) as Record<string, unknown>, module.default),
  } as NonNullable<I18nOptions['messages']>[SupportedLocale];
});

export { langCode };

// 启动时只接受受支持的持久化 locale；存储不可用或值失效时回退到平台默认语言。
const getInitialLocale = (): SupportedLocale => {
  try {
    const stored = normalizeLocale(localStorage.getItem(localeConfigKey));
    if (stored) {
      localStorage.setItem(localeConfigKey, stored);
      return stored;
    }
  } catch {
    // 某些受限环境会禁用本地存储，此时继续使用内存态默认语言。
  }

  return getDefaultLocale();
};

const initialLocale = getInitialLocale();

function persistCanonicalLocale(locale: SupportedLocale) {
  try {
    localStorage.setItem(localeConfigKey, locale);
  } catch {
    // 持久化失败不应阻断当前会话的语言切换，内存态仍由 i18n provider 负责生效。
  }
}

export const i18n = createI18n({
  legacy: false,
  locale: initialLocale,
  fallbackLocale: getDefaultLocale(),
  messages,
  globalInjection: true,
});

persistCanonicalLocale(initialLocale);

export const languageList = computed(() => langList);
export const currentLocale = readonly(computed(() => i18n.global.locale.value));
export const { t } = i18n.global;
export default i18n;
