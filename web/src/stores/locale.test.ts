import { beforeEach, describe, expect, it, vi } from 'vitest';

import { DEFAULT_LOCALE } from '@/app/i18n/messages';
import { createTestingPinia } from '@/test/helpers';

import { useLocaleStore } from './locale';

describe('locale store', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
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

  it('falls back to the default locale when localStorage read throws', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('storage blocked');
    });

    const store = useLocaleStore();

    expect(store.locale).toBe(DEFAULT_LOCALE);
  });

  it('keeps in-memory locale updates when localStorage write throws', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('storage blocked');
    });

    const store = useLocaleStore();

    expect(() => store.setLocale('zh-CN')).not.toThrow();
    expect(store.locale).toBe(DEFAULT_LOCALE);
  });
});
