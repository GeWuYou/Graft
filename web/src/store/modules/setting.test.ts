import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { ModeType } from '@/utils/types';

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

  it('resolves density preset into TDesign spacing and size tokens', () => {
    const store = useSettingStore();

    store.updateThemeDraftAppearance({ densityPreset: 'compact' });

    expect(store.densityPreset).toBe('compact');
    expect(store.themeResolvedTokens.light['--graft-theme-density-scale']).toBe('0.88');
    expect(store.themeResolvedTokens.light['--td-comp-size-m']).toBe('28.16px');
    expect(store.themeResolvedTokens.dark['--td-comp-paddingLR-l']).toBe('14.08px');
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

  it('includes advanced token overrides in draft diff tracking', () => {
    const store = useSettingStore();

    store.beginThemeDraft();
    store.updateThemeToken('light', '--td-brand-color', '#0062ff');

    expect(store.themeAuthorityDiff).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          key: 'themeTokenOverrides',
          fromValue: '0',
          toValue: '1',
        }),
      ]),
    );
    expect(store.themeIdentitySummary.modifiedCount).toBeGreaterThan(0);
  });

  it('resolves display tokens using the actual display mode when mode is auto', () => {
    const store = useSettingStore();

    vi.stubGlobal(
      'matchMedia',
      vi.fn(() => ({ matches: true })),
    );
    store.themeResolvedTokens = {
      light: { '--td-brand-color': '#ffffff' },
      dark: { '--td-brand-color': '#000000' },
    };
    store.mode = 'auto';

    expect(store.resolvedThemeTokensForDisplayMode['--td-brand-color']).toBe('#000000');
  });

  it('refreshes theme runtime only once when applying draft preview and final draft', () => {
    const store = useSettingStore();
    const refreshSpy = vi.spyOn(store, 'refreshThemeWorkbenchRuntime');
    const changeMode = store.changeMode.bind(store);
    vi.spyOn(store, 'changeMode').mockImplementation(async (mode: ModeType | 'auto') => {
      await changeMode(mode);
    });

    store.beginThemeDraft();
    store.updateThemeDraftAppearance({ radiusPreset: 'rounded' });
    expect(refreshSpy).toHaveBeenCalledTimes(1);

    refreshSpy.mockClear();
    store.applyThemeDraft();

    expect(refreshSpy).toHaveBeenCalledTimes(1);
  });

  it('resets font size preset to the default theme authority', () => {
    const store = useSettingStore();

    store.updateThemeDraftAppearance({ fontSizePreset: 'small' });
    store.resetThemeDraftToDefault();

    expect(store.fontSizePreset).toBe('standard');
    expect(store.themeResolvedTokens.light['--graft-theme-font-scale']).toBe('100%');
    expect(store.themeResolvedTokens.light['--td-font-size-body-medium']).toBe('14px');
  });

  it('persists and resets the theme workbench dock position', () => {
    const store = useSettingStore();

    expect(store.themeWorkbenchDockPosition).toBeNull();

    store.setThemeWorkbenchDockPosition({ xRatio: 1.2, yRatio: -0.2 });

    expect(store.themeWorkbenchDockPosition).toEqual({ xRatio: 1, yRatio: 0 });

    store.resetThemeWorkbenchDockPosition();

    expect(store.themeWorkbenchDockPosition).toBeNull();
  });
});
