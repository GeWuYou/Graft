import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/utils/color', () => ({
  composeThemeTokenMap: (tokens: Record<string, string>) => tokens,
  generateBrandColorMap: (brandTheme: string) => ({
    '--td-brand-color': brandTheme,
  }),
  insertThemeStylesheet: vi.fn(),
}));

import { useSettingStore } from './setting';

describe('setting store theme authority', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.stubGlobal(
      'matchMedia',
      vi.fn(() => ({ matches: false })),
    );
  });

  it('uses the standard font size preset by default', () => {
    const store = useSettingStore();

    expect(store.fontSizePreset).toBe('standard');
    expect(store.createThemeAuthoritySnapshot().fontSizePreset).toBe('standard');
  });

  it('resolves font size preset into TDesign font tokens', () => {
    const store = useSettingStore();

    store.updateThemeDraftAppearance({ fontSizePreset: 'large' });

    expect(store.fontSizePreset).toBe('large');
    expect(store.themeResolvedTokens.light['--graft-theme-font-scale']).toBe('106%');
    expect(store.themeResolvedTokens.light['--td-font-size-body-medium']).toBe('14.84px');
    expect(store.themeResolvedTokens.light['--td-font-body-medium']).toBe(
      'var(--td-font-size-body-medium) / var(--td-line-height-body-medium) var(--td-font-family)',
    );
    expect(store.themeResolvedTokens.dark['--graft-theme-font-scale']).toBe('106%');
    expect(store.themeResolvedTokens.dark['--td-font-size-title-large']).toBe('19.08px');
  });

  it('includes font size preset in draft diff tracking', () => {
    const store = useSettingStore();

    store.beginThemeDraft();
    store.updateThemeDraftAppearance({ fontSizePreset: 'extra-large' });

    expect(store.themeAuthorityDiff).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          key: 'fontSizePreset',
          fromValue: 'standard',
          toValue: 'extra-large',
        }),
      ]),
    );
  });

  it('resets font size preset to the default theme authority', () => {
    const store = useSettingStore();

    store.updateThemeDraftAppearance({ fontSizePreset: 'small' });
    store.resetThemeDraftToDefault();

    expect(store.fontSizePreset).toBe('standard');
    expect(store.themeResolvedTokens.light['--graft-theme-font-scale']).toBe('100%');
    expect(store.themeResolvedTokens.light['--td-font-size-body-medium']).toBe('14px');
  });
});
