import type { TChartColor } from '@/config/color';
import { DEFAULT_CHART_COLORS } from '@/config/color';
import type { ThemeTokenMap } from '@/types/theme';

const THEME_TRANSITION_DURATION_MS = 420;
const THEME_TRANSITION_EASING = 'cubic-bezier(0.4, 0, 0.2, 1)';
const THEME_VIEW_TRANSITION_CLASS = 'graft-theme-view-transition';
const THEME_CSS_TRANSITION_CLASS = 'graft-theme-css-transition';

export const THEME_RESET_FEEDBACK_DURATION_MS = 640;

type ThemeViewTransition = {
  ready: Promise<void>;
  finished: Promise<void>;
};

type ThemeViewTransitionDocument = Document & {
  startViewTransition?: (callback: () => void) => ThemeViewTransition;
};

function prefersReducedMotion() {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

function resolveThemeTransitionOrigin(event?: MouseEvent) {
  const x = event?.clientX ?? window.innerWidth;
  const y = event?.clientY ?? 0;

  return { x, y };
}

function safelyApplyThemeChange(applyThemeChange: () => void) {
  applyThemeChange();
}

async function runThemeCssFallbackTransition(applyThemeChange: () => void) {
  const root = document.documentElement;

  root.classList.add(THEME_CSS_TRANSITION_CLASS);
  try {
    safelyApplyThemeChange(applyThemeChange);
    await new Promise((resolve) => {
      window.setTimeout(resolve, THEME_TRANSITION_DURATION_MS);
    });
  } finally {
    root.classList.remove(THEME_CSS_TRANSITION_CLASS);
  }
}

async function runThemeViewTransition(applyThemeChange: () => void, event?: MouseEvent) {
  const transitionDocument = document as ThemeViewTransitionDocument;

  if (!transitionDocument.startViewTransition || prefersReducedMotion()) {
    safelyApplyThemeChange(applyThemeChange);
    return;
  }

  const { x, y } = resolveThemeTransitionOrigin(event);
  const endRadius = Math.hypot(Math.max(x, window.innerWidth - x), Math.max(y, window.innerHeight - y));
  const root = document.documentElement;

  root.classList.add(THEME_VIEW_TRANSITION_CLASS);

  try {
    const transition = transitionDocument.startViewTransition(() => {
      safelyApplyThemeChange(applyThemeChange);
    });
    await transition.ready;
    root
      .animate(
        {
          clipPath: [`circle(0px at ${x}px ${y}px)`, `circle(${endRadius}px at ${x}px ${y}px)`],
        },
        {
          duration: THEME_TRANSITION_DURATION_MS,
          easing: THEME_TRANSITION_EASING,
          pseudoElement: '::view-transition-new(root)',
        },
      )
      .finished.catch(() => undefined);
    await transition.finished;
  } catch {
    safelyApplyThemeChange(applyThemeChange);
  } finally {
    root.classList.remove(THEME_VIEW_TRANSITION_CLASS);
  }
}

export async function runThemeTransition(applyThemeChange: () => void, event?: MouseEvent) {
  const transitionDocument = document as ThemeViewTransitionDocument;

  if (prefersReducedMotion()) {
    safelyApplyThemeChange(applyThemeChange);
    return;
  }

  if (!transitionDocument.startViewTransition) {
    await runThemeCssFallbackTransition(applyThemeChange);
    return;
  }

  await runThemeViewTransition(applyThemeChange, event);
}

export function buildChartColorsFromTokens(tokens: ThemeTokenMap): TChartColor {
  return {
    textColor: tokens['--graft-chart-text-color'] ?? DEFAULT_CHART_COLORS.textColor,
    placeholderColor: tokens['--graft-chart-placeholder-color'] ?? DEFAULT_CHART_COLORS.placeholderColor,
    borderColor: tokens['--graft-chart-border-color'] ?? DEFAULT_CHART_COLORS.borderColor,
    containerColor: tokens['--graft-chart-container-color'] ?? DEFAULT_CHART_COLORS.containerColor,
  };
}
