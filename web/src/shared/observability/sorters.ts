// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export type SortDirection = 'asc' | 'desc';

export type QuerySorter<Field extends string = string> = {
  field: Field;
  direction?: SortDirection;
};

export type SorterState<Field extends string, TState extends { sorters: QuerySorter<Field>[] }> = TState;

export type SortOption<Field extends string> = {
  label: string;
  value: Field;
};

export type SorterUiState<Field extends string> = {
  sortAddDisabled: boolean;
  sortFieldOptionsByIndex: Array<SortOption<Field>[]>;
  sortMoveDownDisabled: boolean[];
  sortMoveUpDisabled: boolean[];
  sorters: QuerySorter<Field>[];
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

export function normalizeSorters<Field extends string>(
  sorters: QuerySorter<Field>[],
  sortOptions: Array<SortOption<Field>>,
): QuerySorter<Field>[] {
  const allowedFields = new Set(sortOptions.map((option) => option.value));
  const usedFields = new Set<Field>();
  const normalized: QuerySorter<Field>[] = [];

  sorters.forEach((sorter) => {
    const field = String(sorter?.field || '').trim() as Field | '';
    if (!field || !allowedFields.has(field) || usedFields.has(field)) {
      return;
    }

    usedFields.add(field);
    normalized.push({
      field,
      direction: sorter.direction === 'asc' ? 'asc' : 'desc',
    });
  });

  return normalized;
}

function hasAvailableSortField<Field extends string>(
  sorters: QuerySorter<Field>[],
  sortOptions: Array<SortOption<Field>>,
) {
  return getNextAvailableSortField(sorters, sortOptions) !== null;
}

function getNextAvailableSortField<Field extends string>(
  sorters: QuerySorter<Field>[],
  sortOptions: Array<SortOption<Field>>,
) {
  const normalized = normalizeSorters(sorters, sortOptions);
  const usedFields = new Set(normalized.map((sorter) => sorter.field));

  return sortOptions.find((option) => !usedFields.has(option.value)) ?? null;
}

function buildSortFieldOptionsByIndex<Field extends string>(
  sorters: QuerySorter<Field>[],
  sortOptions: Array<SortOption<Field>>,
) {
  const normalized = normalizeSorters(sorters, sortOptions);

  return normalized.map((sorter, index) => {
    const usedByOthers = new Set(
      normalized.filter((_, sorterIndex) => sorterIndex !== index).map((item) => item.field),
    );

    return sortOptions.filter((option) => option.value === sorter.field || !usedByOthers.has(option.value));
  });
}

function isMoveUpDisabled<Field extends string>(sorters: QuerySorter<Field>[], index: number) {
  return sorters.length <= 1 || index <= 0 || index >= sorters.length;
}

function isMoveDownDisabled<Field extends string>(sorters: QuerySorter<Field>[], index: number) {
  return sorters.length <= 1 || index < 0 || index >= sorters.length - 1;
}

export function buildSorterUiState<Field extends string>(
  sorters: QuerySorter<Field>[],
  sortOptions: Array<SortOption<Field>>,
): SorterUiState<Field> {
  const normalized = normalizeSorters(sorters, sortOptions);

  return {
    sorters: normalized,
    sortFieldOptionsByIndex: buildSortFieldOptionsByIndex(normalized, sortOptions),
    sortAddDisabled: !hasAvailableSortField(normalized, sortOptions),
    sortMoveUpDisabled: normalized.map((_, index) => isMoveUpDisabled(normalized, index)),
    sortMoveDownDisabled: normalized.map((_, index) => isMoveDownDisabled(normalized, index)),
  };
}

function withUpdatedSorters<Field extends string, TState extends { sorters: QuerySorter<Field>[] }>(
  state: TState,
  sorters: QuerySorter<Field>[],
): TState {
  return {
    ...state,
    sorters,
  };
}

export function appendSorterToState<Field extends string, TState extends { sorters: QuerySorter<Field>[] }>(
  state: TState,
  sortOptions: Array<SortOption<Field>>,
): TState {
  const normalized = normalizeSorters(state.sorters, sortOptions);
  const nextOption = getNextAvailableSortField(normalized, sortOptions);

  if (!nextOption) {
    return withUpdatedSorters(state, normalized);
  }

  return withUpdatedSorters(state, [
    ...normalized,
    {
      field: nextOption.value,
      direction: 'desc',
    },
  ]);
}

export function removeSorterFromState<Field extends string, TState extends { sorters: QuerySorter<Field>[] }>(
  state: TState,
  index: number,
  sortOptions: Array<SortOption<Field>>,
): TState {
  const normalized = normalizeSorters(state.sorters, sortOptions);
  return withUpdatedSorters(
    state,
    normalizeSorters(
      normalized.filter((_, sorterIndex) => sorterIndex !== index),
      sortOptions,
    ),
  );
}

export function moveSorterInState<Field extends string, TState extends { sorters: QuerySorter<Field>[] }>(
  state: TState,
  index: number,
  direction: -1 | 1,
  sortOptions: Array<SortOption<Field>>,
): TState {
  const normalized = normalizeSorters(state.sorters, sortOptions);
  const disabled = direction === -1 ? isMoveUpDisabled(normalized, index) : isMoveDownDisabled(normalized, index);

  if (disabled) {
    return withUpdatedSorters(state, normalized);
  }

  const targetIndex = index + direction;
  const nextSorters = [...normalized];
  const [item] = nextSorters.splice(index, 1);
  nextSorters.splice(targetIndex, 0, item);
  return withUpdatedSorters(state, nextSorters);
}

export function withSorterFieldFromInput<Field extends string, TState extends { sorters: QuerySorter<Field>[] }>(
  state: TState,
  index: number,
  value: string | number | Array<string | number> | undefined,
  normalizeField: (value: string) => Field | '',
  sortOptions: Array<SortOption<Field>>,
  fallbackDirection: SortDirection = 'desc',
): TState {
  const normalized = normalizeSorters(state.sorters, sortOptions);
  if (index < 0 || index >= normalized.length) {
    return withUpdatedSorters(state, normalized);
  }
  const field = typeof value === 'string' ? normalizeField(value) : '';

  if (!field) {
    return removeSorterFromState(withUpdatedSorters(state, normalized), index, sortOptions);
  }

  const duplicated = normalized.some((sorter, sorterIndex) => sorterIndex !== index && sorter.field === field);
  if (duplicated) {
    return withUpdatedSorters(state, normalized);
  }

  const nextSorters = [...normalized];
  nextSorters[index] = {
    field,
    direction: nextSorters[index]?.direction ?? fallbackDirection,
  };

  return withUpdatedSorters(state, normalizeSorters(nextSorters, sortOptions));
}

export function withSorterDirectionFromInput<Field extends string, TState extends { sorters: QuerySorter<Field>[] }>(
  state: TState,
  index: number,
  value: string | number | Array<string | number> | undefined,
  normalizeDirection: (value: string) => SortDirection,
  sortOptions: Array<SortOption<Field>>,
): TState {
  const normalized = normalizeSorters(state.sorters, sortOptions);
  const sorter = normalized[index];

  if (!sorter) {
    return withUpdatedSorters(state, normalized);
  }

  const nextSorters = [...normalized];
  nextSorters[index] = {
    field: sorter.field,
    direction: normalizeDirection(typeof value === 'string' ? value : ''),
  };

  return withUpdatedSorters(state, nextSorters);
}

function buildSorterTagLabel<Field extends string>(
  sorter: QuerySorter<Field>,
  options: Array<SortOption<Field>>,
  prefix: string,
  index: number,
) {
  const fieldLabel = options.find((option) => option.value === sorter.field)?.label ?? sorter.field;
  const arrow = sorter.direction === 'asc' ? '↑' : '↓';
  return `${prefix} ${index + 1}: ${[fieldLabel, arrow].filter(Boolean).join(' ')}`;
}

export function prependSorterTags<Key extends string, Field extends string>(
  tags: Array<{ key: Key; label: string; closable?: boolean }>,
  sorters: QuerySorter<Field>[],
  options: Array<SortOption<Field>>,
  prefix: string,
): Array<{ key: Key | `sorter:${number}`; label: string; closable?: boolean }> {
  const normalized = normalizeSorters(sorters, options);
  if (!normalized.length) {
    return tags;
  }

  return [
    ...normalized.map((sorter, index) => ({
      key: `sorter:${index}` as const,
      label: buildSorterTagLabel(sorter, options, prefix, index),
      closable: true,
    })),
    ...tags,
  ];
}

export function assignEncodedSorters<Field extends string>(
  query: { sort?: string[] },
  sorters: QuerySorter<Field>[],
  options: Array<SortOption<Field>>,
) {
  const encodedSorters = encodeSorters(normalizeSorters(sorters, options), options);
  if (encodedSorters.length) {
    query.sort = encodedSorters;
    return;
  }
  delete query.sort;
}

export function encodeSorters<Field extends string>(sorters: QuerySorter<Field>[], options?: Array<SortOption<Field>>) {
  const normalized = options ? normalizeSorters(sorters, options) : sorters;

  return normalized
    .map((sorter) => {
      const field = String(sorter.field || '').trim();
      if (!field) {
        return '';
      }

      const direction = sorter.direction === 'asc' ? 'asc' : 'desc';
      return `${field}:${direction}`;
    })
    .filter(Boolean);
}

export function decodeSorters<Field extends string>(
  rawValue: string | string[] | undefined,
  normalizeField: (value: string) => Field | '',
  normalizeDirection: (value: string) => SortDirection,
) {
  const values = Array.isArray(rawValue) ? rawValue : rawValue ? [rawValue] : [];

  return values.reduce<QuerySorter<Field>[]>((acc, value) => {
    const candidate = value.trim();
    if (!candidate) {
      return acc;
    }

    const [rawField = '', rawDirection = 'desc'] = candidate.split(':');
    const field = normalizeField(rawField.trim());
    if (!field) {
      return acc;
    }

    acc.push({
      field,
      direction: normalizeDirection(rawDirection.trim() || 'desc'),
    });
    return acc;
  }, []);
}
