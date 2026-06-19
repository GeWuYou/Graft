// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export interface ParsedThemeTokenColor {
  alpha: number;
  blue: number;
  css: string;
  green: number;
  hex: string;
  red: number;
}

const FALLBACK_HEX = '#0052d9';

let sharedCanvasContext: CanvasRenderingContext2D | null | undefined;

/**
 * Retrieves or creates a cached 2D canvas rendering context.
 *
 * Uses a module-level cache to avoid creating multiple canvas elements.
 * In non-browser environments where document is unavailable, returns null.
 *
 * @returns The cached 2D canvas context, or null in non-browser environments
 */
function getCanvasContext() {
  if (sharedCanvasContext !== undefined) {
    return sharedCanvasContext;
  }

  if (typeof document === 'undefined') {
    sharedCanvasContext = null;
    return sharedCanvasContext;
  }

  const canvas = document.createElement('canvas');
  sharedCanvasContext = canvas.getContext('2d');
  return sharedCanvasContext;
}

/**
 * Converts a numeric channel value to a 2-digit lowercase hexadecimal string.
 *
 * @returns A 2-digit lowercase hexadecimal string, with the value clamped to [0, 255] and rounded.
 */
function toHexSegment(value: number) {
  return Math.max(0, Math.min(255, Math.round(value)))
    .toString(16)
    .padStart(2, '0');
}

/**
 * Normalizes a hex color string to canonical `#RRGGBB` format.
 *
 * Accepts hex colors with 3, 4, 6, or 8 digits, with or without a leading `#`.
 *
 * @returns A lowercase `#RRGGBB` hex string, or `null` if the input is invalid.
 */
function normalizeHexInput(value: string) {
  const normalized = value.trim().replace(/^#?/, '#');
  const compact = normalized.slice(1);

  if (/^[0-9a-fA-F]{3}$/.test(compact)) {
    return `#${compact
      .split('')
      .map((item) => `${item}${item}`)
      .join('')
      .toLowerCase()}`;
  }

  if (/^[0-9a-fA-F]{4}$/.test(compact)) {
    const [red, green, blue] = compact
      .slice(0, 3)
      .split('')
      .map((item) => `${item}${item}`);
    return `#${red}${green}${blue}`.toLowerCase();
  }

  if (/^[0-9a-fA-F]{6}$/.test(compact)) {
    return normalized.toLowerCase();
  }

  if (/^[0-9a-fA-F]{8}$/.test(compact)) {
    return `#${compact.slice(0, 6)}`.toLowerCase();
  }

  return null;
}

/**
 * Parses a hex color string into structured color data.
 *
 * Accepts hex colors in 3, 4, 6, or 8 digit formats with an optional `#` prefix.
 * The alpha channel is extracted from the last two digits in 8-digit format; otherwise alpha is 1.
 *
 * @returns The parsed color containing red, green, blue, and alpha values, plus `hex` (#RRGGBB) and `css` (hex if opaque, rgba otherwise), or `null` if the input is invalid.
 */
function parseHexColor(value: string): ParsedThemeTokenColor | null {
  const normalized = value.trim().replace(/^#?/, '#');
  const compact = normalized.slice(1);

  if (!/^[0-9a-fA-F]{3,8}$/.test(compact) || ![3, 4, 6, 8].includes(compact.length)) {
    return null;
  }

  const expanded =
    compact.length === 3 || compact.length === 4
      ? compact
          .split('')
          .map((item) => `${item}${item}`)
          .join('')
      : compact;
  const red = Number.parseInt(expanded.slice(0, 2), 16);
  const green = Number.parseInt(expanded.slice(2, 4), 16);
  const blue = Number.parseInt(expanded.slice(4, 6), 16);
  const alpha = expanded.length === 8 ? Number.parseInt(expanded.slice(6, 8), 16) / 255 : 1;

  return {
    alpha,
    blue,
    css: alpha >= 0.999 ? `#${expanded.slice(0, 6).toLowerCase()}` : `rgba(${red}, ${green}, ${blue}, ${alpha})`,
    green,
    hex: `#${expanded.slice(0, 6).toLowerCase()}`,
    red,
  };
}

/**
 * Parses an RGB or RGBA color string into structured color data.
 *
 * @param value - An RGB or RGBA color string (e.g., "rgb(255, 0, 0)" or "rgba(255, 0, 0, 0.5)")
 * @returns The parsed color data with hex, CSS, and channel values, or `null` if the input is not a valid RGB(A) color or any channel is out of range.
 */
function parseRgbColor(value: string): ParsedThemeTokenColor | null {
  const matched = value.match(
    /^rgba?\(\s*(\d+(?:\.\d+)?)\s*,\s*(\d+(?:\.\d+)?)\s*,\s*(\d+(?:\.\d+)?)\s*(?:,\s*(\d*(?:\.\d+)?)\s*)?\)$/i,
  );

  if (!matched) {
    return null;
  }

  const red = Number(matched[1]);
  const green = Number(matched[2]);
  const blue = Number(matched[3]);
  const alpha = matched[4] === undefined ? 1 : Number(matched[4]);

  if ([red, green, blue].some((item) => Number.isNaN(item) || item < 0 || item > 255)) {
    return null;
  }

  if (Number.isNaN(alpha) || alpha < 0 || alpha > 1) {
    return null;
  }

  return {
    alpha,
    blue,
    css: alpha >= 0.999 ? `#${toHexSegment(red)}${toHexSegment(green)}${toHexSegment(blue)}` : value,
    green,
    hex: `#${toHexSegment(red)}${toHexSegment(green)}${toHexSegment(blue)}`,
    red,
  };
}

/**
 * Normalizes a CSS color string to its browser-resolved form.
 *
 * @param value - The CSS color string to normalize
 * @returns The resolved color string if valid, `null` if the color is unrecognized or if a browser environment is unavailable
 */
function normalizeCssColor(value: string) {
  if (typeof document === 'undefined') {
    return null;
  }

  const style = document.createElement('span').style;
  style.color = '';
  style.color = value.trim();

  if (!style.color) {
    return null;
  }

  const context = getCanvasContext();

  if (!context) {
    return null;
  }

  context.fillStyle = '#010203';

  try {
    context.fillStyle = style.color;
  } catch {
    return null;
  }

  return context.fillStyle;
}

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

  const hexColor = parseHexColor(trimmed);

  if (hexColor) {
    return hexColor;
  }

  const rgbColor = parseRgbColor(trimmed);

  if (rgbColor) {
    return rgbColor;
  }

  const normalized = normalizeCssColor(trimmed);

  if (!normalized) {
    return null;
  }

  return parseHexColor(normalized) ?? parseRgbColor(normalized);
}

/**
 * Builds a CSS color value from a hex color and opacity percentage.
 *
 * @param opacityValue - The opacity as a percentage value; will be clamped to [0, 100] and rounded to the nearest integer.
 * @returns A hex color string if opacity is 100%, or an rgba color string otherwise, or null if the hex input is invalid.
 */
export function buildThemeTokenColorValue(hexValue: string, opacityValue: number) {
  const normalizedHex = normalizeHexInput(hexValue);

  if (!normalizedHex) {
    return null;
  }

  const parsed = parseHexColor(normalizedHex);

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
