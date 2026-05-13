import { beforeEach, describe, expect, it } from 'vitest';

import { DEFAULT_LOCALE } from '@/app/i18n/messages';
import { createTestingPinia } from '@/test/helpers';

import { useLocaleStore } from './locale';

describe('locale store', () => {
  beforeEach(() => {
    window.localStorage.clear();
    createTestingPinia();
  });

  it('falls back to the default locale for malformed persisted state', () => {
    window.localStorage.setItem('graft:locale', JSON.stringify({ locale: 42 }));

    const store = useLocaleStore();

    expect(store.locale).toBe(DEFAULT_LOCALE);
    expect(window.localStorage.getItem('graft:locale')).toBeNull();
  });

  it('normalizes and persists locale changes', () => {
    const store = useLocaleStore();

    store.setLocale('en-US');

    expect(store.locale).toBe(DEFAULT_LOCALE);
    expect(window.localStorage.getItem('graft:locale')).toBe(
      JSON.stringify({
        locale: DEFAULT_LOCALE,
      }),
    );
  });
});
