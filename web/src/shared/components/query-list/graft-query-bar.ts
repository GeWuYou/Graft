export type GraftQueryFilterType = 'input' | 'select' | 'multi-select' | 'date-range' | 'number-range' | 'boolean';

export type GraftQueryFilterOption = {
  label: string;
  value: string | number;
};

export type GraftQueryFilterValue = boolean | number | string | Array<number | string> | undefined;

export type GraftQueryFilterDefinition = {
  key: string;
  label: string;
  type: GraftQueryFilterType;
  options?: GraftQueryFilterOption[];
  placeholder?: string;
  disabled?: boolean;
};

export type GraftQueryQuickFilter = {
  key: string;
  label: string;
  patch: Record<string, GraftQueryFilterValue>;
};

export type GraftQueryBarConfig = {
  placeholder?: string;
  filters?: GraftQueryFilterDefinition[];
  quickFilters?: GraftQueryQuickFilter[];
};

export type GraftQueryState = {
  keyword: string;
  filters: Record<string, GraftQueryFilterValue>;
  page: number;
  pageSize: number;
};
