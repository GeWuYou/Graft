import type { QuerySorter, SortOption } from '@/shared/observability/sorters';

import type {
  AdvancedQueryFilterFieldDefinition,
  AdvancedQueryFilterPreset,
  AdvancedQueryFilterTag,
  AdvancedQueryTimeRangeField,
} from './query-filter-builder';

export type AdvancedQueryFilterBuilderFrameState = {
  activePreset: string;
  fieldValues: Record<string, string | string[]>;
  fields: AdvancedQueryFilterFieldDefinition[];
  keyword: string;
  listeners: Record<string, (...args: never[]) => void>;
  loading?: boolean;
  presets: AdvancedQueryFilterPreset[];
  selectedFieldKey: string;
  sortAddDisabled: boolean;
  sortDirectionOptions: Array<{ label: string; value: string }>;
  sortFieldKey: string;
  sortFieldOptionsByIndex: Array<Array<SortOption<string>>>;
  sortMoveDownDisabled: boolean[];
  sortMoveUpDisabled: boolean[];
  sorters: QuerySorter<string>[];
  showSorterBuilder?: boolean;
  tags: AdvancedQueryFilterTag[];
  timeFieldKey: string;
  timeFields: AdvancedQueryTimeRangeField[];
};
