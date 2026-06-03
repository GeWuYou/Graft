import type { SupportedLocale } from '@/contracts/i18n/locales';
import { createLogger } from '@/utils/logger';

export type TranslationLengthBudgetScope = 'button' | 'navigation' | 'tab';

const TRANSLATION_LENGTH_BUDGETS: Record<TranslationLengthBudgetScope, Record<SupportedLocale, number>> = {
  navigation: {
    'zh-CN': 4,
    'en-US': 14,
  },
  tab: {
    'zh-CN': 6,
    'en-US': 18,
  },
  button: {
    'zh-CN': 6,
    'en-US': 20,
  },
};

const logger = createLogger('locales.lengthBudget');

export function warnTranslationLengthBudget(
  scope: TranslationLengthBudgetScope,
  locale: SupportedLocale,
  key: string,
  value: string,
) {
  if (!import.meta.env.DEV) {
    return;
  }

  const budget = TRANSLATION_LENGTH_BUDGETS[scope][locale];
  const length = Array.from(value.trim()).length;
  if (length <= budget) {
    return;
  }

  logger.warn('[UI Warning] Translation length exceeds budget', {
    budget,
    key,
    length,
    locale,
    scope,
    value,
  });
}
