import { describe, expect, it } from 'vitest';

import { normalizeCssColorValue, parseResolvedCssColor } from './css-color';

describe('css-color', () => {
  it('parses hex and rgba values into a normalized payload', () => {
    expect(parseResolvedCssColor('#e37327')).toMatchObject({
      alpha: 1,
      blue: 39,
      green: 115,
      hex: '#e37327',
      red: 227,
    });

    expect(parseResolvedCssColor('rgba(227, 115, 39, 0.42)')).toMatchObject({
      alpha: 0.42,
      hex: '#e37327',
    });
  });

  it('parses css color 4 rgb syntax with slash alpha', () => {
    expect(parseResolvedCssColor('rgb(95 155 255 / 0.28)')).toMatchObject({
      alpha: 0.28,
      hex: '#5f9bff',
    });
  });

  it('parses srgb color() output into channel-scaled hex', () => {
    expect(parseResolvedCssColor('color(srgb 0.1379 0.197 0.2236 / 1)')).toMatchObject({
      alpha: 1,
      hex: '#233239',
      red: 35,
      green: 50,
      blue: 57,
    });
  });

  it('normalizes browser-supported named colors and rejects invalid values', () => {
    expect(normalizeCssColorValue('rebeccapurple')).toBeTruthy();
    expect(parseResolvedCssColor('not-a-color')).toBeNull();
  });
});
