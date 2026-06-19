// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import {
  buildThemeTokenColorValue,
  formatThemeTokenSummaryValue,
  isThemeTokenColorKey,
  parseThemeTokenColor,
} from './theme-token-color';

describe('theme-token-color', () => {
  it('parses hex and rgba values into a normalized preview payload', () => {
    expect(parseThemeTokenColor('#e37327')).toMatchObject({
      alpha: 1,
      hex: '#e37327',
      red: 227,
      green: 115,
      blue: 39,
    });

    expect(parseThemeTokenColor('rgba(227, 115, 39, 0.42)')).toMatchObject({
      alpha: 0.42,
      hex: '#e37327',
    });
  });

  it('builds canonical output values from hex plus opacity', () => {
    expect(buildThemeTokenColorValue('#e37327', 100)).toBe('#e37327');
    expect(buildThemeTokenColorValue('#e37327', 42)).toBe('rgba(227, 115, 39, 0.42)');
    expect(buildThemeTokenColorValue('invalid', 42)).toBeNull();
  });

  it('formats compact summary values without mutating non-color tokens', () => {
    expect(formatThemeTokenSummaryValue('--td-brand-color', 'rgba(227, 115, 39, 0.42)')).toBe('#e37327 / 42%');
    expect(formatThemeTokenSummaryValue('--td-font-family', 'Inter, sans-serif')).toBe('Inter, sans-serif');
  });

  it('detects editable color tokens without treating shadow tokens as colors', () => {
    expect(isThemeTokenColorKey('--td-brand-color')).toBe(true);
    expect(isThemeTokenColorKey('--td-bg-color-page')).toBe(true);
    expect(isThemeTokenColorKey('--td-shadow-1')).toBe(false);
  });
});
