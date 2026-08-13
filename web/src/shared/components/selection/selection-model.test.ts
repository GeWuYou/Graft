import { describe, expect, it } from 'vitest';

import { createExplicitSelection, replaceExplicitPageSelection } from './selection-model';

describe('selection model', () => {
  it('creates an explicit selection with de-duplicated IDs', () => {
    const selection = createExplicitSelection(['a', 'a', 'b']);

    expect(selection.mode).toBe('explicit');
    expect(Array.from(selection.selectedIds)).toEqual(['a', 'b']);
  });

  it('replaces only the current page while retaining selections from other pages', () => {
    const selection = createExplicitSelection([1, 2, 7]);

    const next = replaceExplicitPageSelection(selection, [1, 2, 3], [3]);

    expect(Array.from(next.selectedIds)).toEqual([7, 3]);
    expect(selection.selectedIds).toEqual(new Set([1, 2, 7]));
  });
});
