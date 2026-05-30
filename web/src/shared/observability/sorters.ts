export type SortDirection = 'asc' | 'desc';

export type QuerySorter<Field extends string = string> = {
  field: Field;
  direction?: SortDirection;
};

type SortOption<Field extends string> = {
  label: string;
  value: Field;
};

export function createSingleSorter<Field extends string>(
  field?: Field | null,
  direction: SortDirection = 'desc',
): QuerySorter<Field>[] {
  if (!field) {
    return [];
  }

  return [{ field, direction }];
}

export function getSingleSorter<Field extends string>(sorters: QuerySorter<Field>[]) {
  return sorters[0] ?? null;
}

function buildSingleSorterTagLabel<Field extends string>(
  sorter: QuerySorter<Field> | null,
  options: Array<SortOption<Field>>,
  prefix: string,
) {
  if (!sorter) {
    return '';
  }

  const fieldLabel = options.find((option) => option.value === sorter.field)?.label;
  const arrow = sorter.direction === 'asc' ? '↑' : sorter.direction === 'desc' ? '↓' : '';
  return `${prefix}：${[fieldLabel, arrow].filter(Boolean).join(' ')}`;
}

export function prependSingleSorterTag<Key extends string, Field extends string>(
  tags: Array<{ key: Key; label: string }>,
  sorter: QuerySorter<Field> | null,
  options: Array<SortOption<Field>>,
  prefix: string,
): Array<{ key: Key | 'sorter'; label: string }> {
  if (!sorter) {
    return tags;
  }

  return [{ key: 'sorter', label: buildSingleSorterTagLabel(sorter, options, prefix) }, ...tags];
}

export function normalizeSingleSorterField<Field extends string>(
  value: string | number | Array<string | number> | undefined,
  currentDirection: SortDirection | undefined,
  normalizeField: (value: string) => Field,
) {
  const candidate = typeof value === 'string' ? value : '';
  if (!candidate) {
    return createSingleSorter<Field>();
  }

  return createSingleSorter(normalizeField(candidate), currentDirection ?? 'desc');
}

export function normalizeSingleSorterDirection<Field extends string>(
  value: string | number | Array<string | number> | undefined,
  currentField: Field | undefined,
  normalizeDirection: (value: string) => SortDirection,
) {
  if (!currentField) {
    return createSingleSorter<Field>();
  }

  const candidate = typeof value === 'string' ? value : '';
  if (!candidate) {
    return createSingleSorter(currentField);
  }

  return createSingleSorter(currentField, normalizeDirection(candidate));
}
