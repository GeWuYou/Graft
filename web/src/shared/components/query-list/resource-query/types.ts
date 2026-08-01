export type ResourceQueryFilterType = 'input' | 'select' | 'multi-select' | 'date-range' | 'number-range' | 'boolean';

export type ResourceQueryFilterOption = {
  label: string;
  value: number | string;
};

export type ResourceQueryFilterValue = boolean | number | string | Array<number | string> | undefined;

export type ResourceQueryFilterDefinition = {
  key: string;
  label: string;
  type: ResourceQueryFilterType;
  options?: ResourceQueryFilterOption[];
  placeholder?: string;
  disabled?: boolean;
};

export type ResourceQueryQuickFilter = {
  key: string;
  label: string;
  patch: Record<string, ResourceQueryFilterValue>;
};

/** 资源列表查询的最小稳定状态；页面可在此基础上保留领域专属筛选状态。 */
export type ResourceQueryState = {
  keyword: string;
  filters: Record<string, ResourceQueryFilterValue>;
  page: number;
  pageSize: number;
};

export type ResourceQueryConfig = {
  resource: string;
  placeholder?: string;
  search?: boolean;
  filterBuilder?: { enabled: boolean };
  filters?: ResourceQueryFilterDefinition[];
  savedView?: boolean;
  quickFilters?: ResourceQueryQuickFilter[];
  timeRange?: boolean;
  sorting?: boolean;
};
