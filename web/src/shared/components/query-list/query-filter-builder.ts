// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export type AdvancedQueryFilterOption = {
  label: string;
  value: string;
};

export type AdvancedQueryFilterFieldKind = 'text' | 'select' | 'multi-select' | 'tag-input' | 'special';

export type AdvancedQueryFilterFieldDefinition = {
  key: string;
  label: string;
  kind: AdvancedQueryFilterFieldKind;
  placeholder?: string;
  options?: AdvancedQueryFilterOption[];
  disabled?: boolean;
};

export type AdvancedQueryFilterTag = {
  key: string;
  label: string;
  closable?: boolean;
};

export type AdvancedQueryTimeRangeField = {
  key: string;
  label: string;
  value: string[];
  placeholder: [string, string];
};

export type AdvancedQuerySortItem = {
  field: string;
  direction?: 'asc' | 'desc';
};

export type AdvancedQuerySortOption = {
  label: string;
  value: string;
};

export type AdvancedQueryFilterPreset = {
  key: string;
  title: string;
};

export type AdvancedQuerySorterUiState = {
  sortAddDisabled: boolean;
  sortFieldOptionsByIndex: AdvancedQuerySortOption[][];
  sortMoveDownDisabled: boolean[];
  sortMoveUpDisabled: boolean[];
  sorters: AdvancedQuerySortItem[];
};
