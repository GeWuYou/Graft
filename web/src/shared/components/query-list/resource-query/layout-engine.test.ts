import { describe, expect, it } from 'vitest';

import { resolveQueryLayout } from './layout-engine';
import type { QueryFieldSize, ResourceQueryFilterDefinition } from './types';

const fieldSizes: QueryFieldSize[] = ['sm', 'md', 'lg', 'xl', 'full'];
const fields = fieldSizes.map((size): ResourceQueryFilterDefinition => ({
  key: size,
  label: size,
  type: 'input',
  layout: { size },
}));

describe('resolveQueryLayout', () => {
  it.each([
    [767, 'compact', 'compact', 1, [1, 1, 1, 1, 1]],
    [768, 'narrow', 'stacked', 6, [3, 3, 6, 6, 6]],
    [991, 'narrow', 'stacked', 6, [3, 3, 6, 6, 6]],
    [992, 'medium', 'split', 12, [3, 4, 6, 6, 12]],
    [1199, 'medium', 'split', 12, [3, 4, 6, 6, 12]],
    [1200, 'wide', 'inline', 12, [2, 3, 4, 6, 12]],
  ] as const)('resolves the %ipx %s tier boundary and size spans', (width, tier, commandBar, columns, spans) => {
    const result = resolveQueryLayout(fields, width);

    expect(result).toMatchObject({ tier, commandBar, columns });
    expect(result.fields.map((item) => item.span)).toEqual(spans);
  });

  it('caps configured spans to the current tier columns and starts a new row when needed', () => {
    const result = resolveQueryLayout(
      [
        { key: 'first', label: 'First', type: 'input', layout: { order: 2, span: { narrow: 99 } } },
        { key: 'second', label: 'Second', type: 'input', layout: { order: 1, span: { narrow: 4 } } },
        { key: 'third', label: 'Third', type: 'input', layout: { order: 3, span: { narrow: 3 } } },
      ],
      768,
    );

    expect(result.fields.map(({ field, row, span }) => [field.key, row, span])).toEqual([
      ['second', 1, 4],
      ['first', 2, 6],
      ['third', 3, 3],
    ]);
  });
});
