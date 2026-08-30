import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { nextTick } from 'vue';

vi.mock('@/utils/color', () => ({
  composeThemeTokenMap: (tokens: Record<string, string>) => tokens,
  generateBrandColorMap: (brandTheme: string) => ({
    '--td-brand-color': brandTheme,
  }),
  insertThemeStylesheet: vi.fn(),
  syncFaviconColor: vi.fn(),
}));

import { THEME_PRESET_DEFINITIONS, THEME_TOKEN_DEFINITIONS } from '@/config/theme-workbench';
import { insertThemeStylesheet, syncFaviconColor } from '@/utils/color';

import { useSettingStore } from './setting';
import {
  createPersistedThemeAuthoritySnapshot,
  STYLE_CONFIG_KEYS,
  WORKBENCH_STYLE_CONFIG_KEYS,
} from './setting-theme-authority';

const insertThemeStylesheetMock = insertThemeStylesheet as unknown as ReturnType<typeof vi.fn>;
const syncFaviconColorMock = syncFaviconColor as unknown as ReturnType<typeof vi.fn>;

type StubMatchMediaOptions = {
  reducedMotion?: boolean;
};

const stubMatchMedia = (matches: boolean, options: StubMatchMediaOptions = {}) => {
  const matchMedia = vi.fn(() => ({ matches }));
  const classList = {
    add: vi.fn(),
    remove: vi.fn(),
  };
  const documentElement = {
    animate: vi.fn(),
    classList,
    setAttribute: vi.fn(),
    toggleAttribute: vi.fn(),
  };

  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: {
      innerHeight: 600,
      innerWidth: 800,
      matchMedia: vi.fn((query: string) => ({
        matches: query === '(prefers-reduced-motion: reduce)' ? Boolean(options.reducedMotion) : matches,
      })),
      setTimeout: (callback: () => void) => {
        callback();
        return 0;
      },
    },
  });
  Object.defineProperty(globalThis, 'document', {
    configurable: true,
    value: { documentElement },
  });
  Object.defineProperty(globalThis, 'matchMedia', {
    configurable: true,
    value: matchMedia,
  });
};

describe('setting store theme authority', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    stubMatchMedia(false);
    syncFaviconColorMock.mockReset();
  });

  it('uses the standard font size preset by default', () => {
    const store = useSettingStore();

    expect(store.fontSizePreset).toBe('standard');
    expect(store.menuAlwaysExpanded).toBe(false);
    expect(store.isAcrylicEnabled).toBe(true);
    expect(store.tabIndicatorPosition).toBe('none');
    expect(store.selectedThemePresetId).toBe('one-dark-pro');
    expect(store.createThemeAuthoritySnapshot().fontSizePreset).toBe('standard');
  });

  it('uses standard shadow intensity by default without changing the current floating shadow tokens', () => {
    const store = useSettingStore();

    store.initializeThemeWorkbenchRuntime();

    expect(store.shadowIntensity).toBe('standard');
    expect(store.createThemeAuthoritySnapshot().shadowIntensity).toBe('standard');
    expect(store.themeResolvedTokens.light).toMatchObject({
      '--td-shadow-1': '0 6px 16px rgba(15, 23, 42, 0.12)',
      '--td-shadow-2': '0 16px 36px rgba(15, 23, 42, 0.18)',
      '--td-shadow-3': '0 24px 56px rgba(15, 23, 42, 0.24)',
    });
  });

  it('restores standard shadow intensity when reading a legacy persisted snapshot', () => {
    const store = useSettingStore();
    const legacySnapshot = { ...store.createThemeAuthoritySnapshot() } as Record<string, unknown>;
    delete legacySnapshot.shadowIntensity;

    const restored = createPersistedThemeAuthoritySnapshot(legacySnapshot as never);

    expect(restored.shadowIntensity).toBe('standard');
  });

  it('normalizes persisted continuous overrides before the workbench consumes them', () => {
    const store = useSettingStore();
    const persistedSnapshot = {
      ...store.createThemeAuthoritySnapshot(),
      densityOverride: 0.5,
      radiusOverride: -1,
    };

    const restored = createPersistedThemeAuthoritySnapshot(persistedSnapshot);

    expect(restored.radiusPreset).toBe('square');
    expect(restored.radiusOverride).toBeNull();
    expect(restored.densityPreset).toBe('compact');
    expect(restored.densityOverride).toBeNull();
  });

  it('scales hard-offset shadows and the neo hard surface together', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('style');
    store.updateThemeDraftAppearance({ shadowPreset: 'hard-offset', shadowIntensity: 'subtle' });
    expect(store.themeResolvedTokens.light).toMatchObject({
      '--td-shadow-1': '1px 1px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
      '--td-shadow-2': '2px 2px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
      '--td-shadow-3': '3px 3px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
      '--graft-neo-shadow': '2px 2px 0 var(--graft-neo-ink)',
    });

    store.updateThemeDraftAppearance({ shadowIntensity: 'strong' });
    expect(store.themeResolvedTokens.light).toMatchObject({
      '--td-shadow-1': '3px 3px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
      '--td-shadow-2': '6px 6px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
      '--td-shadow-3': '9px 9px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
      '--graft-neo-shadow': '6px 6px 0 var(--graft-neo-ink)',
    });
  });

  it('changes soft shadow recipes by intensity while flat remains shadowless', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('style');
    store.updateThemeDraftAppearance({ shadowPreset: 'standard', shadowIntensity: 'standard' });
    expect(store.themeResolvedTokens.light['--td-shadow-2']).toBe('0 10px 24px rgba(15, 23, 42, 0.12)');

    store.updateThemeDraftAppearance({ shadowPreset: 'standard', shadowIntensity: 'subtle' });
    expect(store.themeResolvedTokens.light['--td-shadow-2']).toBe('0 6px 16px rgba(15, 23, 42, 0.08)');

    store.updateThemeDraftAppearance({ shadowIntensity: 'strong' });
    expect(store.themeResolvedTokens.light['--td-shadow-2']).toBe('0 14px 32px rgba(15, 23, 42, 0.18)');

    store.updateThemeDraftAppearance({ shadowPreset: 'floating', shadowIntensity: 'subtle' });
    expect(store.themeResolvedTokens.light['--td-shadow-2']).toBe('0 10px 24px rgba(15, 23, 42, 0.12)');

    store.updateThemeDraftAppearance({ shadowIntensity: 'strong' });
    expect(store.themeResolvedTokens.light['--td-shadow-2']).toBe('0 22px 48px rgba(15, 23, 42, 0.24)');

    store.updateThemeDraftAppearance({ shadowPreset: 'flat' });
    expect(store.themeResolvedTokens.light).toMatchObject({
      '--td-shadow-1': 'none',
      '--td-shadow-2': 'none',
      '--td-shadow-3': 'none',
    });
  });

  it('interpolates soft shadow tokens for a continuous intensity override', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('style');
    store.updateThemeDraftAppearance({ shadowPreset: 'standard', shadowIntensityOverride: 1.25 });

    expect(store.themeResolvedTokens.light['--td-shadow-2']).toBe('0 12px 28px rgba(15, 23, 42, 0.15)');
  });

  it('tracks, applies, and persists shadow intensity as theme authority', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('style');
    store.updateThemeDraftAppearance({ shadowIntensity: 'strong' });

    expect(store.themeAuthorityDiff).toEqual(
      expect.arrayContaining([expect.objectContaining({ key: 'shadowIntensity', toValue: 'strong' })]),
    );
    expect(store.hasThemeDraftPendingChanges).toBe(true);

    store.applyThemeDraft();

    expect(store.shadowIntensity).toBe('strong');
    expect(store.createThemeAuthoritySnapshot().shadowIntensity).toBe('strong');
  });

  it('interpolates continuous radius and density values while clearing overrides at preset anchors', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('style');
    store.updateThemeDraftAppearance({ radiusOverride: 6, densityOverride: 0.94 });

    expect(store.radiusOverride).toBe(6);
    expect(store.themeResolvedTokens.light).toMatchObject({
      '--td-radius-default': '6px',
      '--td-radius-medium': '8px',
      '--td-comp-size-m': '30.08px',
    });
    expect(store.densityOverride).toBe(0.94);

    store.updateThemeDraftAppearance({ radiusOverride: 12, densityOverride: 1 });

    expect(store.radiusPreset).toBe('rounded');
    expect(store.radiusOverride).toBeNull();
    expect(store.densityPreset).toBe('standard');
    expect(store.densityOverride).toBeNull();
  });

  it('keeps a flat shadow override for later visual styles and scales hard-offset surfaces continuously', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('style');
    store.updateThemeDraftAppearance({ shadowPreset: 'flat', shadowIntensityOverride: 1.25 });

    expect(store.shadowIntensityOverride).toBe(1.25);
    expect(store.themeResolvedTokens.light['--td-shadow-2']).toBe('none');

    store.updateThemeDraftAppearance({ shadowPreset: 'hard-offset' });

    expect(store.themeResolvedTokens.light).toMatchObject({
      '--td-shadow-2': '5px 5px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
      '--graft-neo-shadow': '5px 5px 0 var(--graft-neo-ink)',
    });

    store.updateThemeDraftAppearance({ shadowIntensityOverride: 1 });

    expect(store.shadowIntensity).toBe('standard');
    expect(store.shadowIntensityOverride).toBeNull();
    expect(store.themeResolvedTokens.light['--graft-neo-shadow']).toBe('4px 4px 0 var(--graft-neo-ink)');
  });

  it('preserves continuous overrides for palette presets and clears them for complete presets', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('presets');
    store.updateThemeDraftAppearance({
      radiusOverride: 6,
      shadowIntensityOverride: 0.75,
      densityOverride: 0.94,
    });
    store.selectThemePreset('tokyo-night');

    expect(store.radiusOverride).toBe(6);
    expect(store.shadowIntensityOverride).toBe(0.75);
    expect(store.densityOverride).toBe(0.94);

    store.selectThemePreset('atom-one-dark', 'complete');

    expect(store.radiusOverride).toBeNull();
    expect(store.shadowIntensityOverride).toBeNull();
    expect(store.densityOverride).toBeNull();
  });

  it('removes hard surfaces when Atom One Dark is applied as a complete preset', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('presets');
    store.selectThemePreset('industrial-yellow', 'complete');
    expect(document.documentElement.toggleAttribute).toHaveBeenLastCalledWith('data-graft-hard-surface', true);

    store.selectThemePreset('atom-one-dark', 'complete');

    expect(store.radiusPreset).toBe('standard');
    expect(store.radiusOverride).toBeNull();
    expect(store.shadowPreset).toBe('floating');
    expect(store.shadowIntensityOverride).toBeNull();
    expect(document.documentElement.toggleAttribute).toHaveBeenLastCalledWith('data-graft-hard-surface', false);
  });

  it('keeps acrylic as a persisted workbench style preference instead of theme authority', () => {
    const store = useSettingStore();

    expect(STYLE_CONFIG_KEYS).toContain('isAcrylicEnabled');
    expect(store.themeAuthorityDiff).toHaveLength(0);

    store.openThemeWorkbench('appearance');
    store.updateConfig({ isAcrylicEnabled: false });

    expect(store.isAcrylicEnabled).toBe(false);
    expect(store.hasThemeWorkbenchPendingChanges).toBe(true);
    expect(store.themeAuthorityDiff).toHaveLength(0);

    store.cancelThemeDraft();

    expect(store.isAcrylicEnabled).toBe(true);
  });

  it('keeps the tab indicator position in the persisted shell configuration', () => {
    const store = useSettingStore();

    expect(STYLE_CONFIG_KEYS).toContain('tabIndicatorPosition');

    store.updateConfig({ tabIndicatorPosition: 'top' });

    expect(store.tabIndicatorPosition).toBe('top');
  });

  it('keeps the preset application scope as a persisted preference outside the visual draft', () => {
    const store = useSettingStore();

    expect(STYLE_CONFIG_KEYS).toContain('themePresetApplicationScope');
    expect(WORKBENCH_STYLE_CONFIG_KEYS).not.toContain('themePresetApplicationScope');

    store.openThemeWorkbench('presets');
    store.updateConfig({ themePresetApplicationScope: 'complete' });
    store.cancelThemeDraft();

    expect(store.themePresetApplicationScope).toBe('complete');
  });

  it('applies and resets the acrylic workbench preference with the draft lifecycle', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('appearance');
    store.updateConfig({ isAcrylicEnabled: true });
    store.applyThemeDraft();

    expect(store.isAcrylicEnabled).toBe(true);
    expect(store.showThemeWorkbench).toBe(false);

    store.openThemeWorkbench('appearance');
    store.resetThemeDraftToDefault();

    expect(store.isAcrylicEnabled).toBe(true);
    expect(store.hasThemeWorkbenchPendingChanges).toBe(false);

    store.applyThemeDraft();

    expect(store.isAcrylicEnabled).toBe(true);
  });

  it('keeps always-expanded and auto-collapse preferences mutually exclusive', () => {
    const store = useSettingStore();

    store.updateConfig({ menuAutoCollapsed: true });
    store.updateConfig({ menuAlwaysExpanded: true });

    expect(store.menuAlwaysExpanded).toBe(true);
    expect(store.menuAutoCollapsed).toBe(false);

    store.updateConfig({ menuAlwaysExpanded: false });

    expect(store.menuAlwaysExpanded).toBe(false);
    expect(store.menuAutoCollapsed).toBe(false);

    store.updateConfig({ menuAutoCollapsed: true });

    expect(store.menuAlwaysExpanded).toBe(false);
    expect(store.menuAutoCollapsed).toBe(true);

    store.updateConfig({ menuAlwaysExpanded: true });
    store.updateConfig({ layout: 'top' });

    expect(store.menuAlwaysExpanded).toBe(false);
  });

  it('clears persisted always-expanded navigation when the restored layout is top', () => {
    const store = useSettingStore();
    store.layout = 'top';
    store.menuAlwaysExpanded = true;

    store.initializeThemeWorkbenchRuntime();

    expect(store.menuAlwaysExpanded).toBe(false);
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
    expect(store.themeResolvedTokens.light['--graft-density-gap-16']).toBe('14.08px');
    expect(store.themeResolvedTokens.light['--graft-density-card-padding']).toBe('14.08px');
    expect(store.themeResolvedTokens.dark['--td-comp-paddingLR-l']).toBe('14.08px');
    expect(store.themeResolvedTokens.dark['--graft-density-section-padding']).toBe('21.12px');
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

  it('tracks pending draft changes against the saved theme baseline', () => {
    const store = useSettingStore();

    store.beginThemeDraft();

    expect(store.hasThemeDraftPendingChanges).toBe(false);
    expect(store.hasThemeWorkbenchPendingChanges).toBe(false);

    store.updateThemeDraftAppearance({ fontSizePreset: 'extra-large' });

    expect(store.hasThemeDraftPendingChanges).toBe(true);
    expect(store.hasThemeWorkbenchPendingChanges).toBe(true);
  });

  it('tracks layout config preview changes against the workbench open baseline', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('layout');

    expect(store.layout).toBe('mix');
    expect(store.hasThemeWorkbenchPendingChanges).toBe(false);

    store.updateConfig({ layout: 'side' });

    expect(store.layout).toBe('side');
    expect(store.hasThemeDraftPendingChanges).toBe(false);
    expect(store.hasThemeWorkbenchPendingChanges).toBe(true);

    store.updateConfig({ layout: 'mix' });

    expect(store.hasThemeWorkbenchPendingChanges).toBe(false);
  });

  it('rolls back previewed layout config when the workbench is canceled', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('layout');
    store.updateConfig({ layout: 'mix', splitMenu: true, isSidebarFixed: false });

    expect(store.layout).toBe('mix');
    expect(store.splitMenu).toBe(true);
    expect(store.isSidebarFixed).toBe(false);
    expect(store.hasThemeWorkbenchPendingChanges).toBe(true);

    store.cancelThemeDraft();

    expect(store.layout).toBe('mix');
    expect(store.splitMenu).toBe(true);
    expect(store.isSidebarFixed).toBe(true);
    expect(store.showThemeWorkbench).toBe(false);
    expect(store.hasThemeWorkbenchPendingChanges).toBe(false);
  });

  it('applies and cancels the floating personalization entry through the layout draft', () => {
    const store = useSettingStore();

    expect(store.showThemeWorkbenchDock).toBe(false);

    store.openThemeWorkbench('layout');
    store.updateConfig({ showThemeWorkbenchDock: true });

    expect(store.showThemeWorkbenchDock).toBe(true);
    expect(store.hasThemeWorkbenchPendingChanges).toBe(true);

    store.cancelThemeDraft();
    expect(store.showThemeWorkbenchDock).toBe(false);

    store.openThemeWorkbench('layout');
    store.updateConfig({ showThemeWorkbenchDock: true });
    store.applyThemeDraft();
    expect(store.showThemeWorkbenchDock).toBe(true);
  });

  it('keeps a personalization entry available when configuration hides both Header and Dock', () => {
    const store = useSettingStore();

    store.updateConfig({ showHeader: false, showThemeWorkbenchDock: false });

    expect(store.showHeader).toBe(false);
    expect(store.showThemeWorkbenchDock).toBe(true);
  });

  it('keeps the Dock enabled after applying a Header-hidden workbench draft', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('layout');

    store.updateConfig({ showHeader: false });
    store.applyThemeDraft();

    expect(store.showHeader).toBe(false);
    expect(store.showThemeWorkbenchDock).toBe(true);
  });

  it('repairs a persisted unreachable personalization configuration during runtime initialization', () => {
    const store = useSettingStore();
    store.showHeader = false;
    store.showThemeWorkbenchDock = false;

    store.initializeThemeWorkbenchRuntime();

    expect(store.showHeader).toBe(false);
    expect(store.showThemeWorkbenchDock).toBe(true);
  });

  it('keeps previewed layout config after applying the workbench changes', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('layout');
    store.updateConfig({ layout: 'side', splitMenu: false });
    const modifiedBeforeApply = store.themeAuthorityLastModifiedAt;

    store.applyThemeDraft();

    expect(store.layout).toBe('side');
    expect(store.splitMenu).toBe(false);
    expect(store.showThemeWorkbench).toBe(false);
    expect(store.themeDraft).toBeNull();
    expect(store.hasThemeWorkbenchPendingChanges).toBe(false);
    expect(store.themeAuthorityLastModifiedAt).not.toBe(modifiedBeforeApply);
  });

  it('applies or cancels combined theme and layout workbench changes together', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('layout');
    store.updateConfig({ layout: 'mix' });
    store.updateThemeDraftAppearance({ fontSizePreset: 'extra-large' });

    expect(store.layout).toBe('mix');
    expect(store.fontSizePreset).toBe('extra-large');
    expect(store.hasThemeWorkbenchPendingChanges).toBe(true);

    store.cancelThemeDraft();

    expect(store.layout).toBe('mix');
    expect(store.fontSizePreset).toBe('standard');
    expect(store.hasThemeWorkbenchPendingChanges).toBe(false);
  });

  it('rolls back previewed always-expanded navigation changes when the workbench is canceled', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('layout');
    store.updateConfig({ menuAutoCollapsed: true });
    store.updateConfig({ menuAlwaysExpanded: true });

    expect(store.menuAlwaysExpanded).toBe(true);
    expect(store.menuAutoCollapsed).toBe(false);

    store.cancelThemeDraft();

    expect(store.menuAlwaysExpanded).toBe(false);
    expect(store.menuAutoCollapsed).toBe(false);
  });

  it('includes advanced token overrides in draft diff tracking', () => {
    const store = useSettingStore();

    store.beginThemeDraft();
    store.updateThemeToken('light', '--td-brand-color', '#0062ff');

    expect(store.themeAuthorityDiff).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          key: 'themeTokenOverrides',
          fromValue: expect.any(String),
          toValue: expect.any(String),
        }),
      ]),
    );
    expect(store.themeIdentitySummary.modifiedCount).toBeGreaterThan(0);
  });

  it('resolves display tokens using the actual display mode when mode is auto', () => {
    const store = useSettingStore();

    stubMatchMedia(true);
    store.themeResolvedTokens = {
      light: { '--td-brand-color': '#ffffff' },
      dark: { '--td-brand-color': '#000000' },
    };
    store.mode = 'auto';

    expect(store.resolvedThemeTokensForDisplayMode['--td-brand-color']).toBe('#000000');
  });

  it('refreshes chart colors when the brand theme changes directly', () => {
    const store = useSettingStore();

    store.themeTokenOverrides = {
      light: {
        '--graft-chart-text-color': '#123456',
      },
      dark: {},
    };
    store.chartColors = {
      textColor: '#stale',
      placeholderColor: '#stale',
      borderColor: '#stale',
      containerColor: '#stale',
    };
    store.mode = 'light';

    store.changeBrandTheme('#2BA471');

    expect(store.brandTheme).toBe('#2BA471');
    expect(store.chartColors.textColor).toBe('#123456');
    expect(store.chartColors.placeholderColor).toBe('#8a94a6');
    expect(insertThemeStylesheet).toHaveBeenCalledWith(
      '#2BA471',
      expect.objectContaining({
        '--graft-chart-text-color': '#123456',
      }),
      'light',
    );
  });

  it('syncs the favicon through theme preview and restores it when the draft is canceled', () => {
    const store = useSettingStore();
    const initialBrandTheme = store.brandTheme;

    store.initializeThemeWorkbenchRuntime();
    expect(syncFaviconColorMock).toHaveBeenLastCalledWith(initialBrandTheme);

    store.openThemeWorkbench('appearance');
    store.setCustomBrandTheme('#2BA471');
    expect(syncFaviconColorMock).toHaveBeenLastCalledWith('#2BA471');

    store.cancelThemeDraft();
    expect(syncFaviconColorMock).toHaveBeenLastCalledWith(initialBrandTheme);
  });

  it('refreshes theme runtime only once when applying draft preview and final draft', () => {
    const store = useSettingStore();
    insertThemeStylesheetMock.mockClear();

    store.beginThemeDraft();
    store.updateThemeDraftAppearance({ radiusPreset: 'rounded' });
    expect(insertThemeStylesheet).toHaveBeenCalledTimes(1);

    insertThemeStylesheetMock.mockClear();
    store.applyThemeDraft();

    expect(insertThemeStylesheet).toHaveBeenCalledTimes(1);
  });

  it('resets font size preset to the default theme authority', () => {
    const store = useSettingStore();
    store.updateThemeDraftAppearance({ fontSizePreset: 'small' });
    store.resetThemeDraftToDefault();

    expect(store.fontSizePreset).toBe('standard');
    expect(store.themeResolvedTokens.light['--graft-theme-font-scale']).toBe('100%');
    expect(store.themeResolvedTokens.light['--td-font-size-body-medium']).toBe('14px');
  });

  it('keeps reset-to-default applicable when the saved theme differs from the default authority', () => {
    const store = useSettingStore();
    store.assignThemeAuthorityState({
      ...store.createThemeAuthoritySnapshot(),
      fontSizePreset: 'large',
      themeSource: 'customized',
    });
    store.beginThemeDraft();

    store.resetThemeDraftToDefault();

    expect(store.themeAuthorityDiff).toHaveLength(0);
    expect(store.hasThemeDraftPendingChanges).toBe(true);
  });

  it('does not mark reset-to-default as pending when the saved theme is already default', () => {
    const store = useSettingStore();

    store.beginThemeDraft();
    store.resetThemeDraftToDefault();

    expect(store.themeAuthorityDiff).toHaveLength(0);
    expect(store.hasThemeDraftPendingChanges).toBe(false);
  });

  it('clears stale reset feedback state when reset-to-default is called directly', () => {
    const store = useSettingStore();

    store.themeResetting = true;
    store.beginThemeDraft();

    store.resetThemeDraftToDefault();

    expect(store.themeResetting).toBe(false);
    expect(store.fontSizePreset).toBe('standard');
  });

  it('tracks reset-to-default feedback while keeping the draft semantics', async () => {
    const store = useSettingStore();
    let finishResetFeedback: (() => void) | undefined;

    Object.defineProperty(window, 'setTimeout', {
      configurable: true,
      value: vi.fn((callback: () => void) => {
        finishResetFeedback = callback;
        return 0;
      }),
    });

    store.assignThemeAuthorityState({
      ...store.createThemeAuthoritySnapshot(),
      fontSizePreset: 'large',
      themeSource: 'customized',
    });
    store.beginThemeDraft();

    const resetPromise = store.resetDefaultThemeWithFeedback();

    expect(store.themeResetting).toBe(true);
    expect(store.themeResetFeedbackKey).toBe(1);
    expect(store.fontSizePreset).toBe('standard');
    expect(store.hasThemeDraftPendingChanges).toBe(true);

    await nextTick();
    expect(store.themeResetting).toBe(true);

    finishResetFeedback?.();
    await resetPromise;

    expect(store.themeResetting).toBe(false);
    expect(store.themeResetFeedbackKey).toBe(1);
  });

  it('does not use full-page theme transitions for reset-to-default feedback', async () => {
    const store = useSettingStore();
    const startViewTransition = vi.fn((callback: () => void) => {
      callback();
      return { finished: Promise.resolve(), ready: Promise.resolve() };
    });

    Object.defineProperty(document, 'startViewTransition', {
      configurable: true,
      value: startViewTransition,
    });

    await store.resetDefaultThemeWithFeedback();

    expect(startViewTransition).not.toHaveBeenCalled();
    expect(document.documentElement.animate).not.toHaveBeenCalled();
    expect(document.documentElement.classList.add).not.toHaveBeenCalledWith('graft-theme-view-transition');
    expect(document.documentElement.classList.add).not.toHaveBeenCalledWith('graft-theme-css-transition');
  });

  it('persists reset-to-default drafts and closes the workbench after apply', () => {
    const store = useSettingStore();

    store.assignThemeAuthorityState({
      ...store.createThemeAuthoritySnapshot(),
      mode: 'dark',
      selectedThemePresetId: 'midnight-blue',
      brandTheme: '#3B82F6',
      fontSizePreset: 'large',
      themeSource: 'customized',
    });
    store.openThemeWorkbench('overview');
    store.resetThemeDraftToDefault();
    const modifiedBeforeApply = store.themeAuthorityLastModifiedAt;

    store.applyThemeDraft();

    expect(store.mode).toBe('dark');
    expect(store.brandTheme).toBe('#61AFEF');
    expect(store.selectedThemePresetId).toBe('one-dark-pro');
    expect(store.fontSizePreset).toBe('standard');
    expect(store.themeSource).toBe('preset');
    expect(store.showThemeWorkbench).toBe(false);
    expect(store.themeDraft).toBeNull();
    expect(store.themeAuthorityLastModifiedAt).not.toBe(modifiedBeforeApply);
  });

  it('uses the selected preset mode for preview, cancellation, and apply', () => {
    const store = useSettingStore();

    store.assignThemeAuthorityState({
      ...store.createThemeAuthoritySnapshot(),
      mode: 'dark',
      selectedThemePresetId: 'midnight-blue',
      brandTheme: '#3B82F6',
    });
    store.openThemeWorkbench('presets');
    store.selectThemePreset('tdesign-default');

    expect(store.mode).toBe('light');
    expect(store.selectedThemePresetId).toBe('tdesign-default');

    store.cancelThemeDraft();

    expect(store.mode).toBe('dark');
    expect(store.selectedThemePresetId).toBe('midnight-blue');

    store.openThemeWorkbench('presets');
    store.selectThemePreset('tencent-cloud');
    store.applyThemeDraft();

    expect(store.mode).toBe('light');
    expect(store.selectedThemePresetId).toBe('tencent-cloud');
  });

  it('applies overview quick adjustments through the shared draft state', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('overview');
    store.applyWorkbenchQuickAppearance({ densityPreset: 'compact', mode: 'dark' });
    store.applyWorkbenchQuickLayout({ layout: 'mix' });

    expect(store.densityPreset).toBe('compact');
    expect(store.mode).toBe('dark');
    expect(store.layout).toBe('mix');
    expect(store.hasThemeWorkbenchPendingChanges).toBe(true);

    store.cancelThemeDraft();

    expect(store.densityPreset).toBe('comfortable');
    expect(store.mode).toBe('dark');
    expect(store.layout).toBe('mix');
  });

  it('applies scenario presets to both theme authority and shell layout draft state', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('overview');
    store.applyThemeWorkbenchScenarioPreset('high-density');

    expect(store.layout).toBe('side');
    expect(store.showFooter).toBe(false);
    expect(store.fontSizePreset).toBe('small');
    expect(store.densityPreset).toBe('compact');
    expect(store.hasThemeWorkbenchPendingChanges).toBe(true);

    store.applyThemeDraft();

    expect(store.showThemeWorkbench).toBe(false);
    expect(store.showFooter).toBe(false);
    expect(store.fontSizePreset).toBe('small');
    expect(store.densityPreset).toBe('compact');
  });

  it('applies official theme presets with their bundled appearance and layout defaults', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('overview');
    store.updateConfig({ menuAlwaysExpanded: true });

    store.selectThemePreset('graphite-slate', 'complete');
    expect(store.selectedThemePresetId).toBe('graphite-slate');
    expect(store.brandTheme).toBe('#4F6B8A');
    expect(store.mode).toBe('dark');
    expect(store.fontFamilyPreset).toBe('inter');
    expect(store.fontSizePreset).toBe('small');
    expect(store.radiusPreset).toBe('business');
    expect(store.shadowPreset).toBe('flat');
    expect(store.densityPreset).toBe('compact');
    expect(store.layout).toBe('side');
    expect(store.isUseTabsRouter).toBe(true);
    expect(store.menuAutoCollapsed).toBe(true);
    expect(store.menuAlwaysExpanded).toBe(false);
    expect(store.splitMenu).toBe(false);

    store.selectThemePreset('sunset-amber', 'complete');
    expect(store.selectedThemePresetId).toBe('sunset-amber');
    expect(store.brandTheme).toBe('#D97706');
    expect(store.mode).toBe('light');
    expect(store.fontFamilyPreset).toBe('source-han-sans');
    expect(store.fontSizePreset).toBe('standard');
    expect(store.radiusPreset).toBe('rounded');
    expect(store.shadowPreset).toBe('standard');
    expect(store.densityPreset).toBe('comfortable');
    expect(store.layout).toBe('side');
    expect(store.isUseTabsRouter).toBe(false);
    expect(store.menuAutoCollapsed).toBe(false);
    expect(store.splitMenu).toBe(false);

    store.selectThemePreset('ocean-teal', 'complete');
    expect(store.selectedThemePresetId).toBe('ocean-teal');
    expect(store.brandTheme).toBe('#0F8A83');
    expect(store.mode).toBe('light');
    expect(store.fontFamilyPreset).toBe('harmonyos');
    expect(store.fontSizePreset).toBe('standard');
    expect(store.radiusPreset).toBe('standard');
    expect(store.shadowPreset).toBe('floating');
    expect(store.densityPreset).toBe('standard');
    expect(store.layout).toBe('mix');
    expect(store.isUseTabsRouter).toBe(true);
    expect(store.menuAutoCollapsed).toBe(false);
    expect(store.splitMenu).toBe(true);

    store.selectThemePreset('frost-silver', 'complete');
    expect(store.selectedThemePresetId).toBe('frost-silver');
    expect(store.brandTheme).toBe('#7A8CA5');
    expect(store.mode).toBe('light');
    expect(store.fontFamilyPreset).toBe('system');
    expect(store.fontSizePreset).toBe('large');
    expect(store.radiusPreset).toBe('capsule');
    expect(store.shadowPreset).toBe('flat');
    expect(store.densityPreset).toBe('comfortable');
    expect(store.layout).toBe('side');
    expect(store.isUseTabsRouter).toBe(false);
    expect(store.menuAutoCollapsed).toBe(false);
    expect(store.splitMenu).toBe(false);

    store.selectThemePreset('signal-rose', 'complete');
    expect(store.selectedThemePresetId).toBe('signal-rose');
    expect(store.brandTheme).toBe('#D65B8C');
    expect(store.mode).toBe('dark');
    expect(store.fontFamilyPreset).toBe('inter');
    expect(store.fontSizePreset).toBe('small');
    expect(store.radiusPreset).toBe('business');
    expect(store.densityPreset).toBe('compact');
    expect(store.layout).toBe('mix');
    expect(store.splitMenu).toBe(true);

    store.selectThemePreset('forest-night', 'complete');
    expect(store.selectedThemePresetId).toBe('forest-night');
    expect(store.brandTheme).toBe('#3FA879');
    expect(store.mode).toBe('dark');
    expect(store.fontFamilyPreset).toBe('harmonyos');
    expect(store.radiusPreset).toBe('rounded');
    expect(store.shadowPreset).toBe('flat');
    expect(store.densityPreset).toBe('comfortable');
    expect(store.layout).toBe('side');
    expect(store.splitMenu).toBe(false);
  });

  it('applies Industrial Yellow as a complete personalized configuration baseline', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');
    store.selectThemePreset('industrial-yellow', 'complete');

    expect(store.selectedThemePresetId).toBe('industrial-yellow');
    expect(store.mode).toBe('light');
    expect(store.sideMode).toBe('light');
    expect(store.radiusPreset).toBe('square');
    expect(store.shadowPreset).toBe('hard-offset');
    expect(store.shadowIntensity).toBe('standard');
    expect(store.themeResolvedTokens.light).toMatchObject({
      '--graft-neo-accent': '#FFE45C',
      '--graft-neo-shadow': '4px 4px 0 var(--graft-neo-ink)',
    });
    expect(store.isAcrylicEnabled).toBe(false);
    expect(store.showHeader).toBe(false);
    expect(store.showThemeWorkbenchDock).toBe(true);
  });

  it('applies the Industrial Yellow personalized baseline with its palette', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');
    store.selectThemePreset('industrial-yellow', 'complete');

    expect(store.radiusPreset).toBe('square');
    expect(store.shadowPreset).toBe('hard-offset');
    expect(store.shadowIntensity).toBe('standard');
    expect(store.selectedThemePresetId).toBe('industrial-yellow');
  });

  it('exposes every Industrial Yellow-only surface token for manual personalization', () => {
    const editableTokenKeys = new Set(THEME_TOKEN_DEFINITIONS.map((definition) => definition.key));
    const industrialPreset = THEME_PRESET_DEFINITIONS.find((preset) => preset.id === 'industrial-yellow');

    expect(industrialPreset).toBeDefined();
    if (!industrialPreset) {
      throw new Error('Industrial Yellow preset is missing');
    }

    const industrialLightTokens = industrialPreset.tokenOverrides?.light;
    if (!industrialLightTokens) {
      throw new Error('Industrial Yellow light token baseline is missing');
    }

    Object.keys(industrialLightTokens).forEach((tokenKey) => {
      expect(editableTokenKeys).toContain(tokenKey);
    });
  });

  it('declares a fixed light or dark mode for every built-in theme preset', () => {
    expect(THEME_PRESET_DEFINITIONS).toHaveLength(21);
    expect(THEME_PRESET_DEFINITIONS.every((preset) => preset.mode === 'light' || preset.mode === 'dark')).toBe(true);
    expect(
      THEME_PRESET_DEFINITIONS.every((preset) => (preset.authorityPatch?.shadowIntensity ?? 'standard') === 'standard'),
    ).toBe(true);
  });

  it.each(THEME_PRESET_DEFINITIONS)('aligns the $id complete preset sidebar with its fixed theme mode', (preset) => {
    if (preset.mode !== 'light' && preset.mode !== 'dark') {
      throw new Error(`Built-in preset ${preset.id} must declare a fixed theme mode`);
    }

    const store = useSettingStore();
    const oppositeMode = preset.mode === 'light' ? 'dark' : 'light';
    store.updateConfig({ sideMode: oppositeMode });
    store.openThemeWorkbench('presets');

    store.selectThemePreset(preset.id, 'complete');

    expect(store.mode).toBe(preset.mode);
    expect(store.sideMode).toBe(preset.mode);
    expect(document.documentElement.setAttribute).toHaveBeenLastCalledWith(
      'side-mode',
      preset.mode === 'dark' ? 'dark' : '',
    );
  });

  it('resolves scrollbar tracks and hover states from the selected theme preset', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('presets');
    store.selectThemePreset('tencent-cloud', 'complete');

    expect(store.themeResolvedTokens.light['--graft-scrollbar-track-color']).toBe('#E0F1FB');
    expect(store.themeResolvedTokens.light['--graft-scrollbar-thumb-hover-color']).toBe('#7EC1E8');

    store.selectThemePreset('forest-night', 'complete');

    expect(store.themeResolvedTokens.dark['--graft-scrollbar-track-color']).toBe('#111A15');
    expect(store.themeResolvedTokens.dark['--graft-scrollbar-thumb-hover-color']).toBe('#5D8D73');
  });

  it('does not use selected preset metadata when resolving equivalent editable theme state', () => {
    const store = useSettingStore();

    store.initializeThemeWorkbenchRuntime();
    const resolvedBeforeMetadataChange = {
      light: { ...store.themeResolvedTokens.light },
      dark: { ...store.themeResolvedTokens.dark },
    };

    store.selectedThemePresetId = 'tokyo-night';
    store.refreshThemeWorkbenchRuntime();

    expect(store.themeResolvedTokens).toEqual(resolvedBeforeMetadataChange);
  });

  it('resolves separate light and dark acrylic glass tokens', () => {
    const store = useSettingStore();

    store.initializeThemeWorkbenchRuntime();

    expect(store.themeResolvedTokens.light).toMatchObject({
      '--graft-glass-ambient-color': 'color-mix(in srgb, var(--td-brand-color) 7%, transparent)',
      '--graft-glass-bg': 'rgba(255, 255, 255, 0.54)',
      '--graft-glass-blur': '28px',
      '--graft-glass-content-bg': 'rgba(255, 255, 255, 0.72)',
      '--graft-glass-content-blur': '22px',
    });
    expect(store.themeResolvedTokens.dark).toMatchObject({
      '--graft-glass-ambient-color': 'color-mix(in srgb, var(--td-brand-color) 10%, transparent)',
      '--graft-glass-bg': 'rgba(33, 37, 43, 0.72)',
      '--graft-glass-blur': '30px',
      '--graft-glass-content-bg': 'rgba(40, 44, 52, 0.78)',
      '--graft-glass-content-blur': '24px',
    });
    expect(store.themeResolvedTokens.dark['--graft-glass-content-bg']).not.toBe(
      store.themeResolvedTokens.dark['--graft-glass-bg'],
    );
  });

  it('persists and resets the theme workbench dock position', () => {
    const store = useSettingStore();

    expect(store.themeWorkbenchDockPosition).toBeNull();

    store.setThemeWorkbenchDockPosition({ xRatio: 1.2, yRatio: -0.2 });

    expect(store.themeWorkbenchDockPosition).toEqual({ xRatio: 1, yRatio: 0 });

    store.resetThemeWorkbenchDockPosition();

    expect(store.themeWorkbenchDockPosition).toBeNull();
  });

  it('animates theme mode changes from the click position when View Transitions are available', async () => {
    const store = useSettingStore();
    const finished = Promise.resolve();
    const ready = Promise.resolve();
    const startViewTransition = vi.fn((callback: () => void) => {
      callback();
      return { finished, ready };
    });

    Object.defineProperty(document, 'startViewTransition', {
      configurable: true,
      value: startViewTransition,
    });

    await store.updateThemeDraftModeWithTransition('dark', { clientX: 120, clientY: 160 } as MouseEvent);

    expect(store.mode).toBe('dark');
    expect(startViewTransition).toHaveBeenCalledTimes(1);
    const [keyframes, options] = (document.documentElement.animate as ReturnType<typeof vi.fn>).mock.calls[0];

    expect(keyframes.clipPath[0]).toBe('circle(0px at 120px 160px)');
    expect(keyframes.clipPath[1]).toMatch(/^circle\([\d.]+px at 120px 160px\)$/);
    expect(Number(keyframes.clipPath[1].match(/^circle\(([\d.]+)px/)?.[1])).toBeCloseTo(809.9382692526635);
    expect(options).toEqual({
      duration: 420,
      easing: 'cubic-bezier(0.4, 0, 0.2, 1)',
      pseudoElement: '::view-transition-new(root)',
    });
    expect(document.documentElement.classList.add).toHaveBeenCalledWith('graft-theme-view-transition');
    expect(document.documentElement.classList.remove).toHaveBeenCalledWith('graft-theme-view-transition');
  });

  it('falls back to CSS theme transitions when View Transitions are unavailable', async () => {
    const store = useSettingStore();

    Object.defineProperty(document, 'startViewTransition', {
      configurable: true,
      value: undefined,
    });

    await store.updateThemeDraftModeWithTransition('dark');

    expect(store.mode).toBe('dark');
    expect(document.documentElement.classList.add).toHaveBeenCalledWith('graft-theme-css-transition');
    expect(document.documentElement.classList.remove).toHaveBeenCalledWith('graft-theme-css-transition');
    expect(document.documentElement.animate).not.toHaveBeenCalled();
  });

  it('skips theme transition animation when reduced motion is preferred', async () => {
    stubMatchMedia(false, { reducedMotion: true });
    const store = useSettingStore();
    const startViewTransition = vi.fn();

    Object.defineProperty(document, 'startViewTransition', {
      configurable: true,
      value: startViewTransition,
    });

    await store.updateThemeDraftModeWithTransition('dark');

    expect(store.mode).toBe('dark');
    expect(startViewTransition).not.toHaveBeenCalled();
    expect(document.documentElement.animate).not.toHaveBeenCalled();
    expect(document.documentElement.classList.add).not.toHaveBeenCalledWith('graft-theme-css-transition');
    expect(document.documentElement.classList.add).not.toHaveBeenCalledWith('graft-theme-view-transition');
  });

  it('uses One Dark Pro as the complete first-run default', () => {
    const store = useSettingStore();

    store.initializeThemeWorkbenchRuntime();

    expect(store.selectedThemePresetId).toBe('one-dark-pro');
    expect(store.mode).toBe('dark');
    expect(store.brandTheme).toBe('#61AFEF');
    expect(store.isAcrylicEnabled).toBe(true);
    expect(store.layout).toBe('mix');
    expect(store.themeResolvedTokens.dark['--graft-glass-blur']).toBe('30px');
  });

  it('preserves personalization while applying a preset palette by default', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');
    store.updateConfig({ isAcrylicEnabled: false, layout: 'side', sideMode: 'light' });
    store.updateThemeDraftAppearance({ fontSizePreset: 'large', densityPreset: 'compact', shadowIntensity: 'strong' });
    store.updateThemeToken('dark', '--graft-chart-text-color', '#123456');

    store.selectThemePreset('tokyo-night');

    expect(store.mode).toBe('dark');
    expect(store.brandTheme).toBe('#7AA2F7');
    expect(store.fontSizePreset).toBe('large');
    expect(store.densityPreset).toBe('compact');
    expect(store.shadowIntensity).toBe('strong');
    expect(store.layout).toBe('side');
    expect(store.sideMode).toBe('light');
    expect(store.isAcrylicEnabled).toBe(false);
    expect(store.themeTokenOverrides.dark['--graft-chart-text-color']).toBe('#123456');
    expect(store.themeTokenOverrides.dark['--graft-glass-blur']).toBeUndefined();
  });

  it('applies the complete preset package into editable authority and token state', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');
    store.updateConfig({ isAcrylicEnabled: false, layout: 'side', sideMode: 'light' });
    store.updateThemeDraftAppearance({ fontSizePreset: 'large' });
    store.updateThemeToken('dark', '--graft-chart-text-color', '#123456');

    store.selectThemePreset('one-dark-pro', 'complete');

    expect(store.fontSizePreset).toBe('standard');
    expect(store.densityPreset).toBe('comfortable');
    expect(store.shadowIntensity).toBe('standard');
    expect(store.layout).toBe('mix');
    expect(store.sideMode).toBe('dark');
    expect(store.isAcrylicEnabled).toBe(true);
    expect(store.themeTokenOverrides.dark['--graft-glass-blur']).toBe('30px');
    expect(store.themeResolvedTokens.dark['--graft-glass-blur']).toBe('30px');
  });

  it('clears previous material tokens when applying only a new palette', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');

    store.selectThemePreset('one-dark-pro', 'complete');
    expect(store.themeTokenOverrides.dark['--graft-glass-blur']).toBe('30px');

    store.selectThemePreset('industrial-yellow', 'palette');

    expect(store.themeTokenOverrides.dark['--graft-glass-blur']).toBeUndefined();
  });

  it('preserves custom material tokens when applying only a new palette', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');

    store.selectThemePreset('one-dark-pro', 'complete');
    store.updateThemeToken('dark', '--graft-glass-blur', '42px');
    store.selectThemePreset('industrial-yellow', 'palette');

    expect(store.themeTokenOverrides.dark['--graft-glass-blur']).toBe('42px');
  });

  it('preserves custom material tokens whose value matches another material token', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');

    store.selectThemePreset('one-dark-pro', 'complete');
    store.updateThemeToken('dark', '--graft-glass-bg', '30px');
    store.selectThemePreset('industrial-yellow', 'palette');

    expect(store.themeTokenOverrides.dark['--graft-glass-bg']).toBe('30px');
  });

  it('restores the target preset style values when leaving Industrial Yellow', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');

    store.selectThemePreset('industrial-yellow', 'complete');
    expect(store.radiusPreset).toBe('square');
    expect(store.shadowPreset).toBe('hard-offset');
    expect(store.shadowIntensity).toBe('standard');

    store.selectThemePreset('one-dark-pro', 'complete');
    expect(store.radiusPreset).toBe('standard');
    expect(store.shadowPreset).toBe('floating');
  });

  it('keeps manually selected style values while changing color presets', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');
    store.updateThemeDraftAppearance({
      radiusPreset: 'square',
      shadowPreset: 'hard-offset',
      shadowIntensity: 'strong',
    });

    store.selectThemePreset('tokyo-night');

    expect(store.radiusPreset).toBe('square');
    expect(store.shadowPreset).toBe('hard-offset');
    expect(store.shadowIntensity).toBe('strong');
  });

  it('clears the active preset when an authority value is customized', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');
    store.selectThemePreset('one-dark-pro', 'complete');

    store.updateThemeDraftAppearance({ fontSizePreset: 'large' });

    expect(store.selectedThemePresetId).toBeNull();
    expect(store.effectiveSelectedThemePreset).toBeNull();
  });

  it('reselects the matching preset when a customized value is restored', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');
    store.selectThemePreset('one-dark-pro', 'complete');

    store.updateThemeDraftAppearance({ fontSizePreset: 'large' });
    store.updateThemeDraftAppearance({ fontSizePreset: 'standard' });

    expect(store.selectedThemePresetId).toBe('one-dark-pro');
    expect(store.effectiveSelectedThemePreset?.id).toBe('one-dark-pro');
  });

  it('does not fall back to the default preset after a token customization', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');
    store.selectThemePreset('one-dark-pro', 'complete');

    store.updateThemeToken('dark', '--graft-chart-text-color', '#123456');

    expect(store.selectedThemePresetId).toBeNull();
    expect(store.effectiveSelectedThemePreset).toBeNull();
  });

  it('applies Atom One Dark with its green accent and distinct graphite surfaces', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');
    store.selectThemePreset('atom-one-dark', 'complete');

    expect(store.mode).toBe('dark');
    expect(store.brandTheme).toBe('#98C379');
    expect(store.themeResolvedTokens.dark['--graft-shell-sidebar-bg']).toBe('#1B1F24');
    expect(store.themeResolvedTokens.dark['--td-component-stroke']).toBe('#3A414B');
    expect(store.themeResolvedTokens.dark['--graft-glass-bg']).toBe('rgba(27, 31, 36, 0.74)');
  });

  it('resets to the complete default configuration', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');
    store.updateConfig({ isAcrylicEnabled: false, layout: 'side', sideMode: 'light' });
    store.updateThemeDraftAppearance({ fontSizePreset: 'large' });
    store.updateThemeToken('dark', '--graft-chart-text-color', '#123456');

    store.resetThemeDraftToDefault();

    expect(store.selectedThemePresetId).toBe('one-dark-pro');
    expect(store.mode).toBe('dark');
    expect(store.fontSizePreset).toBe('standard');
    expect(store.layout).toBe('mix');
    expect(store.sideMode).toBe('dark');
    expect(store.isAcrylicEnabled).toBe(true);
    expect(store.themeTokenOverrides.dark['--graft-chart-text-color']).toBe('#E6EAF0');
  });

  it('prefers the default preset authority mode when resetting a personalized draft', () => {
    const store = useSettingStore();
    const defaultPreset = THEME_PRESET_DEFINITIONS.find((preset) => preset.id === 'one-dark-pro');

    expect(defaultPreset).toBeDefined();
    if (!defaultPreset) {
      return;
    }

    const authorityPatch = defaultPreset.authorityPatch;
    defaultPreset.authorityPatch = { ...authorityPatch, mode: 'light' };
    try {
      store.openThemeWorkbench('presets');
      store.updateThemeDraftAppearance({ fontSizePreset: 'large' });

      store.resetThemeDraftToDefault();

      expect(store.mode).toBe('light');
    } finally {
      defaultPreset.authorityPatch = authorityPatch;
    }
  });
});
