import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { RESPONSIVE_CONTAINER_THRESHOLDS } from './breakpoints';
import { resolveResponsiveDensity, resolveResponsiveVariant } from './responsive';

describe('responsive variants', () => {
  it('maps container widths to semantic density thresholds', () => {
    expect(resolveResponsiveDensity(0)).toBe('compact');
    expect(resolveResponsiveDensity(767)).toBe('compact');
    expect(resolveResponsiveDensity(768)).toBe('comfortable');
    expect(resolveResponsiveDensity(991)).toBe('comfortable');
    expect(resolveResponsiveDensity(992)).toBe('spacious');
  });

  it('keeps runtime thresholds aligned with the Less source of truth', () => {
    const variables = readFileSync(resolve(process.cwd(), 'src/style/variables.less'), 'utf8');

    expect(variables).toContain('@screen-sm: 768px;');
    expect(variables).toContain('@screen-md: 992px;');
    expect(variables).toContain('@screen-lg: 1200px;');
    expect(RESPONSIVE_CONTAINER_THRESHOLDS).toEqual({
      comfortable: 768,
      spacious: 992,
      wide: 1200,
    });
  });

  it('exposes only semantic variant fields', () => {
    expect(resolveResponsiveVariant(768, { layout: 'grid', presentation: 'entity' })).toEqual({
      density: 'comfortable',
      interaction: 'interactive',
      layout: 'grid',
      presentation: 'entity',
      surface: 'page',
    });
    expect(Object.keys(resolveResponsiveVariant(0))).not.toEqual(
      expect.arrayContaining(['isMobile', 'isTablet', 'isDesktop']),
    );
  });
});
