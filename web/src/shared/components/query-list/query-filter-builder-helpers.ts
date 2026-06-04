import { computed, type Ref } from 'vue';

import {
  appendSorterToState,
  buildSorterUiState,
  moveSorterInState,
  prependSorterTags,
  type QuerySorter,
  removeSorterFromState,
  type SortDirection,
  type SortOption,
  withSorterDirectionFromInput,
  withSorterFieldFromInput,
} from '@/shared/observability/sorters';

import type { AdvancedQueryFilterBuilderFrameState } from './AdvancedQueryFilterBuilderFrame.vue';
import type {
  AdvancedQueryFilterFieldDefinition,
  AdvancedQueryFilterPreset,
  AdvancedQueryFilterTag,
  AdvancedQueryTimeRangeField,
} from './query-filter-builder';

type SorterInputValue = string | number | Array<string | number> | undefined;

type BuilderListenerConfig<TPreset extends string, TFieldKey extends string, TTimeValue> = {
  addSorter: () => void;
  clearTag: (key: string) => void;
  emitApplyPreset: (preset: TPreset) => void;
  emitReset: () => void;
  emitSearch: () => void;
  handleFieldUpdate: (payload: { key: string; value: string | string[] }) => void;
  moveSorterDown: (index: number) => void;
  moveSorterUp: (index: number) => void;
  removeSorter: (index: number) => void;
  selectedFieldKey: Ref<TFieldKey>;
  updateKeyword: (value: string) => void;
  updateSortDirection: (index: number, value: string | number | Array<string | number> | undefined) => void;
  updateSortField: (index: number, value: string | number | Array<string | number> | undefined) => void;
  updateTimeField: (payload: TTimeValue) => void;
};

export function useAdvancedQuerySorterUiState<Field extends string>(
  sorters: () => QuerySorter<Field>[],
  sortOptions: () => Array<SortOption<Field>>,
) {
  const sorterUiState = computed(() => buildSorterUiState(sorters(), sortOptions()));

  return {
    normalizedSorters: computed(() => sorterUiState.value.sorters),
    sortAddDisabled: computed(() => sorterUiState.value.sortAddDisabled),
    sortFieldOptionsByIndex: computed(() => sorterUiState.value.sortFieldOptionsByIndex),
    sortMoveDownDisabled: computed(() => sorterUiState.value.sortMoveDownDisabled),
    sortMoveUpDisabled: computed(() => sorterUiState.value.sortMoveUpDisabled),
  };
}

function useAdvancedQuerySorterControls<Field extends string, State extends { sorters: QuerySorter<Field>[] }>(config: {
  emit: (state: State) => void;
  normalizeField: (value: string) => Field | '';
  sortOptions: () => Array<SortOption<Field>>;
  state: () => State;
}) {
  return {
    mutators: createSorterStateMutators(config),
    ui: useAdvancedQuerySorterUiState(
      () => config.state().sorters,
      () => config.sortOptions(),
    ),
  };
}

export function useAdvancedQuerySorterControlsForModel<
  Field extends string,
  State extends { sorters: QuerySorter<Field>[] },
>(
  model: () => State,
  emit: (state: State) => void,
  normalizeField: (value: string) => Field | '',
  sortOptions: () => Array<SortOption<Field>>,
) {
  return useAdvancedQuerySorterControls({
    emit,
    normalizeField,
    sortOptions,
    state: model,
  });
}

export function createAdvancedQueryBuilderListeners<TPreset extends string, TFieldKey extends string, TTimeValue>(
  config: BuilderListenerConfig<TPreset, TFieldKey, TTimeValue>,
) {
  return {
    'add-sorter': config.addSorter,
    'apply-preset': (preset: string) => config.emitApplyPreset(preset as TPreset),
    'close-tag': (key: string) => config.clearTag(key),
    'move-sorter-down': config.moveSorterDown,
    'move-sorter-up': config.moveSorterUp,
    'remove-sorter': config.removeSorter,
    reset: config.emitReset,
    search: config.emitSearch,
    'update:field': config.handleFieldUpdate,
    'update:keyword': (value: string) => config.updateKeyword(value),
    'update:selected-field-key': (key: string) => {
      config.selectedFieldKey.value = key as TFieldKey;
    },
    'update:sort-direction': ({
      index,
      value,
    }: {
      index: number;
      value: string | number | Array<string | number> | undefined;
    }) => config.updateSortDirection(index, value),
    'update:sort-field': ({
      index,
      value,
    }: {
      index: number;
      value: string | number | Array<string | number> | undefined;
    }) => config.updateSortField(index, value),
    'update:time-field': (payload: TTimeValue) => config.updateTimeField(payload),
  };
}

export function createSortDirection(value: string): SortDirection {
  return value === 'asc' ? 'asc' : 'desc';
}

export function buildAdvancedQueryTimeTag(label: string, range: string[], separator = ':') {
  if (!range.length) {
    return '';
  }

  return `${label}${separator}${separator === '：' ? '' : ' '}${range.filter(Boolean).join(' ~ ')}`;
}

function createSorterStateMutators<Field extends string, State extends { sorters: QuerySorter<Field>[] }>(config: {
  emit: (state: State) => void;
  fallbackDirection?: SortDirection;
  normalizeField: (value: string) => Field | '';
  sortOptions: () => Array<SortOption<Field>>;
  state: () => State;
}) {
  const fallbackDirection = config.fallbackDirection ?? 'desc';
  const normalizeDirection = (value: string) => createSortDirection(value);
  const emitUpdatedState = (state: State) => config.emit(state);

  return {
    addSorter: () => emitUpdatedState(appendSorterToState(config.state(), config.sortOptions())),
    moveSorterDown: (index: number) =>
      emitUpdatedState(moveSorterInState(config.state(), index, 1, config.sortOptions())),
    moveSorterUp: (index: number) =>
      emitUpdatedState(moveSorterInState(config.state(), index, -1, config.sortOptions())),
    removeSorter: (index: number) =>
      emitUpdatedState(removeSorterFromState(config.state(), index, config.sortOptions())),
    updateSortDirection: (index: number, value: SorterInputValue) =>
      emitUpdatedState(
        withSorterDirectionFromInput(config.state(), index, value, normalizeDirection, config.sortOptions()),
      ),
    updateSortField: (index: number, value: SorterInputValue) =>
      emitUpdatedState(
        withSorterFieldFromInput(
          config.state(),
          index,
          value,
          config.normalizeField,
          config.sortOptions(),
          fallbackDirection,
        ),
      ),
  };
}

type AdvancedQueryFilterBuilderFrameInput = Omit<AdvancedQueryFilterBuilderFrameState, 'sortFieldKey' | 'timeFieldKey'>;

function buildAdvancedQueryFilterBuilderFrame(
  config: AdvancedQueryFilterBuilderFrameInput,
): AdvancedQueryFilterBuilderFrameState {
  return {
    ...config,
    sortFieldKey: 'sorterBuilder',
    timeFieldKey: 'timeRange',
  };
}

function createAdvancedQueryFilterBuilderFrameState(config: {
  activePreset: () => string;
  fieldValues: () => Record<string, string | string[]>;
  fields: () => AdvancedQueryFilterFieldDefinition[];
  keyword: () => string;
  listeners: Record<string, (...args: never[]) => void>;
  loading: () => boolean | undefined;
  presets: () => AdvancedQueryFilterPreset[];
  selectedFieldKey: Ref<string>;
  sorterUi: ReturnType<typeof useAdvancedQuerySorterUiState<string>>;
  sortDirectionOptions: () => Array<{ label: string; value: string }>;
  tags: () => AdvancedQueryFilterTag[];
  timeFields: () => AdvancedQueryTimeRangeField[];
}) {
  return computed<AdvancedQueryFilterBuilderFrameState>(() =>
    buildAdvancedQueryFilterBuilderFrame({
      activePreset: config.activePreset(),
      fieldValues: config.fieldValues(),
      fields: config.fields(),
      keyword: config.keyword(),
      listeners: config.listeners,
      loading: config.loading(),
      presets: config.presets(),
      selectedFieldKey: config.selectedFieldKey.value,
      sortAddDisabled: config.sorterUi.sortAddDisabled.value,
      sortDirectionOptions: config.sortDirectionOptions(),
      sortFieldOptionsByIndex: config.sorterUi.sortFieldOptionsByIndex.value,
      sortMoveDownDisabled: config.sorterUi.sortMoveDownDisabled.value,
      sortMoveUpDisabled: config.sorterUi.sortMoveUpDisabled.value,
      sorters: config.sorterUi.normalizedSorters.value,
      tags: config.tags(),
      timeFields: config.timeFields(),
    }),
  );
}

export function createAdvancedQueryFilterBuilderFrameStateFromSource(config: {
  fieldValues: () => Record<string, string | string[]>;
  fields: () => AdvancedQueryFilterFieldDefinition[];
  listeners: Record<string, (...args: never[]) => void>;
  selectedFieldKey: Ref<string>;
  sorterUi: ReturnType<typeof useAdvancedQuerySorterUiState<string>>;
  sortDirectionOptions: () => Array<{ label: string; value: string }>;
  source: () => {
    activePreset: string;
    loading?: boolean;
    presets: AdvancedQueryFilterPreset[];
  };
  keyword: () => string;
  tags: () => AdvancedQueryFilterTag[];
  timeFields: () => AdvancedQueryTimeRangeField[];
}) {
  return createAdvancedQueryFilterBuilderFrameState({
    activePreset: () => config.source().activePreset,
    fieldValues: config.fieldValues,
    fields: config.fields,
    keyword: config.keyword,
    listeners: config.listeners,
    loading: () => config.source().loading,
    presets: () => config.source().presets,
    selectedFieldKey: config.selectedFieldKey,
    sorterUi: config.sorterUi,
    sortDirectionOptions: config.sortDirectionOptions,
    tags: config.tags,
    timeFields: config.timeFields,
  });
}

function buildFieldFilterTags<State extends Record<string, unknown>, Key extends string>(
  fields: AdvancedQueryFilterFieldDefinition[],
  state: State,
  separator = ':',
) {
  return fields.reduce<Array<AdvancedQueryFilterTag & { key: Key }>>((tags, definition) => {
    if (definition.kind === 'special') {
      return tags;
    }

    const rawValue = state[definition.key];
    const value = typeof rawValue === 'string' ? rawValue.trim() : rawValue;
    const listValue = Array.isArray(value) ? value.filter((item) => String(item).trim()) : null;
    if (!value || (listValue && listValue.length === 0)) {
      return tags;
    }

    const label = listValue
      ? listValue
          .map((item) =>
            definition.kind === 'select'
              ? definition.options?.find((option) => option.value === item)?.label || String(item)
              : String(item),
          )
          .join(', ')
      : definition.kind === 'select'
        ? definition.options?.find((option) => option.value === value)?.label || String(value)
        : String(value);
    const separatorGap = separator === '：' ? '' : ' ';
    tags.push({ key: definition.key as Key, label: `${definition.label}${separator}${separatorGap}${label}` });
    return tags;
  }, []);
}

export function buildAdvancedQueryActiveTags<
  State extends Record<string, unknown>,
  Key extends string,
  Field extends string,
>(config: {
  fields: AdvancedQueryFilterFieldDefinition[];
  filterState: State;
  fieldSeparator?: string;
  sortOptions: Array<SortOption<Field>>;
  sorterPrefix: string;
  sorters: QuerySorter<Field>[];
  timeTags?: AdvancedQueryFilterTag[];
}) {
  const filterTags = buildFieldFilterTags<State, Key>(config.fields, config.filterState, config.fieldSeparator ?? ':');

  return prependSorterTags(
    [...(config.timeTags ?? []), ...filterTags],
    config.sorters,
    config.sortOptions,
    config.sorterPrefix,
  );
}

export function updateAdvancedQueryFilterStateField<State extends Record<string, unknown>, Key extends keyof State>(
  state: State,
  key: Key,
  value: State[Key],
): State {
  return {
    ...state,
    [key]: typeof value === 'string' ? value.trim() : value,
  };
}
