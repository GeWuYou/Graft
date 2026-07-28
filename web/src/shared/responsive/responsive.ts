import { RESPONSIVE_CONTAINER_THRESHOLDS } from './breakpoints';

export type ResponsiveDensity = 'compact' | 'comfortable' | 'spacious';
export type ResponsiveLayout = 'stack' | 'flow' | 'split' | 'wide-split' | 'grid';
export type ResponsiveSurface = 'page' | 'dialog' | 'drawer' | 'sheet';
export type ResponsivePresentation = 'data' | 'entity';
export type ResponsiveInteraction = 'readonly' | 'interactive' | 'workspace';

export interface ResponsiveVariant {
  density: ResponsiveDensity;
  interaction: ResponsiveInteraction;
  layout: ResponsiveLayout;
  presentation: ResponsivePresentation;
  surface: ResponsiveSurface;
}

export interface ResponsiveVariantOptions {
  interaction?: ResponsiveInteraction;
  layout?: ResponsiveLayout;
  presentation?: ResponsivePresentation;
  surface?: ResponsiveSurface;
}

const DEFAULT_VARIANT_OPTIONS: Required<ResponsiveVariantOptions> = {
  interaction: 'interactive',
  layout: 'flow',
  presentation: 'data',
  surface: 'page',
};

export function resolveResponsiveDensity(width: number): ResponsiveDensity {
  if (width < RESPONSIVE_CONTAINER_THRESHOLDS.comfortable) {
    return 'compact';
  }

  if (width < RESPONSIVE_CONTAINER_THRESHOLDS.spacious) {
    return 'comfortable';
  }

  return 'spacious';
}

export function resolveResponsiveVariant(width: number, options: ResponsiveVariantOptions = {}): ResponsiveVariant {
  return {
    ...DEFAULT_VARIANT_OPTIONS,
    ...options,
    density: resolveResponsiveDensity(width),
  };
}
