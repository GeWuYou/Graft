import { normalizeCssColorValue, parseResolvedCssColor } from '@/shared/theme/css-color';

export type ParsedThemeTokenColor = NonNullable<ReturnType<typeof parseResolvedCssColor>>;

const FALLBACK_HEX = '#0052d9';

/**
 * Determines whether a token key represents a theme color property.
 *
 * @returns `true` if the token key indicates a color theme property, `false` otherwise.
 */
export function isThemeTokenColorKey(tokenKey: string) {
  return /color|background|border/i.test(tokenKey) && !/shadow/i.test(tokenKey);
}

/**
 * Parses a color string into structured color data.
 *
 * @returns `ParsedThemeTokenColor` containing the color's RGB channels, opacity, and CSS representation, or `null` if parsing fails.
 */
export function parseThemeTokenColor(value: string) {
  const trimmed = value.trim();

  if (!trimmed) {
    return null;
  }

  return parseResolvedCssColor(trimmed);
}

/**
 * Builds a CSS color value from a hex color and opacity percentage.
 *
 * @param opacityValue - The opacity as a percentage value; will be clamped to [0, 100] and rounded to the nearest integer.
 * @returns A hex color string if opacity is 100%, or an rgba color string otherwise, or null if the hex input is invalid.
 */
export function buildThemeTokenColorValue(hexValue: string, opacityValue: number) {
  const normalizedHex = normalizeCssColorValue(hexValue);

  if (!normalizedHex) {
    return null;
  }

  const parsed = parseResolvedCssColor(normalizedHex);

  if (!parsed) {
    return null;
  }

  const opacity = Math.max(0, Math.min(100, Math.round(opacityValue)));

  if (opacity >= 100) {
    return parsed.hex;
  }

  return `rgba(${parsed.red}, ${parsed.green}, ${parsed.blue}, ${opacity / 100})`;
}

/**
 * Formats a token value for UI display, with special handling for color tokens.
 *
 * For color tokens, displays the hex color with opacity percentage (e.g., `#0052d9 / 50%`), or just the hex if fully opaque. For other tokens, returns the value unchanged. Empty values return '--'.
 *
 * @param tokenKey - The token key used to determine if the token is a color
 * @param value - The token value to format
 * @returns The formatted display string
 */
export function formatThemeTokenSummaryValue(tokenKey: string, value: string) {
  const trimmed = value.trim();

  if (!trimmed) {
    return '--';
  }

  if (!isThemeTokenColorKey(tokenKey)) {
    return trimmed;
  }

  const parsed = parseThemeTokenColor(trimmed);

  if (!parsed) {
    return trimmed;
  }

  const opacity = Math.round(parsed.alpha * 100);
  return opacity >= 100 ? parsed.hex : `${parsed.hex} / ${opacity}%`;
}

/**
 * Produces a hex color for a theme token preview.
 *
 * @returns The hex color string in `#RRGGBB` format, or a fallback color if parsing fails.
 */
export function resolveThemeTokenPreviewHex(value: string) {
  return parseThemeTokenColor(value)?.hex ?? FALLBACK_HEX;
}
