import type { Pinia } from 'pinia';
import type { App as VueApp, ComputedRef, InjectionKey } from 'vue';
import { computed, inject } from 'vue';

import { useLocaleStore } from '@/stores/locale';

import { DEFAULT_FALLBACK_LOCALE, resolveMessageTemplate } from './messages';
import { createLocaleRequestHeaders } from './request';

type TranslateValues = Record<
  string,
  string | number | boolean | null | undefined
>;

interface TranslateOptions {
  fallback?: string;
  values?: TranslateValues;
}

export interface I18nContext {
  locale: ComputedRef<string>;
  fallbackLocale: ComputedRef<string>;
  setLocale: (locale: string) => void;
  t: (key: string, options?: TranslateOptions) => string;
  resolveRequestHeaders: (
    headers?: Record<string, string>,
  ) => Record<string, string>;
}

const I18N_CONTEXT_KEY: InjectionKey<I18nContext> = Symbol('graft-i18n');

function interpolateMessage(
  template: string,
  values?: TranslateValues,
): string {
  if (!values) {
    return template;
  }

  return template.replace(/\{(\w+)\}/g, (_, token: string) => {
    const value = values[token];

    return value === null || value === undefined ? '' : String(value);
  });
}

/**
 * 解析消息时先查当前 locale，再查 fallback locale，最后回退到显式 fallback 或 key 本身。
 * 这样在后续逐步补齐多语言词条时，壳层仍能保持可预测的显示行为。
 */
export function translateMessage(
  locale: string,
  fallbackLocale: string,
  key: string,
  options: TranslateOptions = {},
): string {
  const template =
    resolveMessageTemplate(locale, key) ??
    resolveMessageTemplate(fallbackLocale, key) ??
    options.fallback ??
    key;

  return interpolateMessage(template, options.values);
}

/**
 * 在应用启动阶段安装最小 i18n 上下文：
 * locale 状态继续显式落在 Pinia，上层组件只通过注入上下文获取翻译和请求头能力。
 */
export function setupI18n(app: VueApp<Element>, pinia: Pinia) {
  const localeStore = useLocaleStore(pinia);

  const context: I18nContext = {
    locale: computed(() => localeStore.locale),
    fallbackLocale: computed(
      () => localeStore.fallbackLocale || DEFAULT_FALLBACK_LOCALE,
    ),
    setLocale: (locale: string) => localeStore.setLocale(locale),
    t: (key: string, options?: TranslateOptions) =>
      translateMessage(
        localeStore.locale,
        localeStore.fallbackLocale || DEFAULT_FALLBACK_LOCALE,
        key,
        options,
      ),
    resolveRequestHeaders: (headers?: Record<string, string>) =>
      createLocaleRequestHeaders(localeStore.locale, headers),
  };

  app.provide(I18N_CONTEXT_KEY, context);
}

export function useI18n(): I18nContext {
  const context = inject(I18N_CONTEXT_KEY);

  if (!context) {
    throw new Error('graft i18n has not been installed');
  }

  return context;
}
