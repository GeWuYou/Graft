import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { nextTick } from 'vue';

vi.mock('@/utils/color', () => ({
  composeThemeTokenMap: (tokens: Record<string, string>) => tokens,
  generateBrandColorMap: (brandTheme: string) => ({
    '--td-brand-color': brandTheme,
  }),
  insertThemeStylesheet: vi.fn(),
}));

import { insertThemeStylesheet } from '@/utils/color';

import { useSettingStore } from './setting';

const insertThemeStylesheetMock = insertThemeStylesheet as unknown as ReturnType<typeof vi.fn>;

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

    expect(store.layout).toBe('side');
    expect(store.hasThemeWorkbenchPendingChanges).toBe(false);

    store.updateConfig({ layout: 'mix' });

    expect(store.layout).toBe('mix');
    expect(store.hasThemeDraftPendingChanges).toBe(false);
    expect(store.hasThemeWorkbenchPendingChanges).toBe(true);

    store.updateConfig({ layout: 'side' });

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

    expect(store.layout).toBe('side');
    expect(store.splitMenu).toBe(false);
    expect(store.isSidebarFixed).toBe(true);
    expect(store.showThemeWorkbench).toBe(false);
    expect(store.hasThemeWorkbenchPendingChanges).toBe(false);
  });

  it('keeps previewed layout config after applying the workbench changes', () => {
    const store = useSettingStore();

    store.openThemeWorkbench('layout');
    store.updateConfig({ layout: 'mix', splitMenu: true });
    const modifiedBeforeApply = store.themeAuthorityLastModifiedAt;

    store.applyThemeDraft();

    expect(store.layout).toBe('mix');
    expect(store.splitMenu).toBe(true);
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

    expect(store.layout).toBe('side');
    expect(store.fontSizePreset).toBe('standard');
    expect(store.hasThemeWorkbenchPendingChanges).toBe(false);
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

    expect(store.mode).toBe('light');
    expect(store.brandTheme).toBe('#0052D9');
    expect(store.selectedThemePresetId).toBe('tdesign-default');
    expect(store.fontSizePreset).toBe('standard');
    expect(store.themeSource).toBe('preset');
    expect(store.showThemeWorkbench).toBe(false);
    expect(store.themeDraft).toBeNull();
    expect(store.themeAuthorityLastModifiedAt).not.toBe(modifiedBeforeApply);
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

    expect(store.densityPreset).toBe('standard');
    expect(store.mode).toBe('light');
    expect(store.layout).toBe('side');
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

    store.selectThemePreset('graphite-slate');
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
    expect(store.splitMenu).toBe(false);

    store.selectThemePreset('sunset-amber');
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

    store.selectThemePreset('ocean-teal');
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

    store.selectThemePreset('frost-silver');
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
});
