import { defineStore } from 'pinia';

import {
  DEFAULT_FALLBACK_LOCALE,
  DEFAULT_LOCALE,
  normalizeLocale,
} from '@/app/i18n/messages';

const LOCALE_STORAGE_KEY = 'graft:locale';

interface LocalePayload {
  locale: string;
}

function isLocalePayload(value: unknown): value is LocalePayload {
  if (!value || typeof value !== 'object') {
    return false;
  }

  return typeof (value as Record<string, unknown>).locale === 'string';
}

function loadLocale(): string {
  if (typeof window === 'undefined') {
    return DEFAULT_LOCALE;
  }

  const raw = window.localStorage.getItem(LOCALE_STORAGE_KEY);

  if (!raw) {
    return DEFAULT_LOCALE;
  }

  try {
    const parsed = JSON.parse(raw) as unknown;

    if (!isLocalePayload(parsed)) {
      window.localStorage.removeItem(LOCALE_STORAGE_KEY);
      return DEFAULT_LOCALE;
    }

    return normalizeLocale(parsed.locale);
  } catch {
    window.localStorage.removeItem(LOCALE_STORAGE_KEY);
    return DEFAULT_LOCALE;
  }
}

function persistLocale(locale: string) {
  if (typeof window === 'undefined') {
    return;
  }

  window.localStorage.setItem(
    LOCALE_STORAGE_KEY,
    JSON.stringify({
      locale,
    }),
  );
}

/**
 * 这里集中保存跨页面共享的 locale 状态。
 * 后续如果语言切换器、服务端用户偏好或租户级默认语言接入，
 * 只需要扩展这个 store，而不必让页面直接读写 localStorage。
 */
export const useLocaleStore = defineStore('locale', {
  state: () => ({
    locale: loadLocale(),
    fallbackLocale: DEFAULT_FALLBACK_LOCALE,
  }),
  actions: {
    setLocale(locale: string) {
      const normalizedLocale = normalizeLocale(locale);

      this.locale = normalizedLocale;
      persistLocale(normalizedLocale);
    },
  },
});
