export type ResourceQueryFilterType = 'input' | 'select' | 'multi-select' | 'date-range' | 'number-range' | 'boolean';

export type ResourceQueryFilterOption = {
  label: string;
  value: number | string;
};

export type ResourceQueryFilterValue = boolean | number | string | Array<number | string> | undefined;

/** 字段语义独立于 Query Panel 的排版策略，供后续渲染器替换时稳定复用。 */
export type ResourceQueryFieldDefinition = {
  type: ResourceQueryFilterType;
  options?: ResourceQueryFilterOption[];
  placeholder?: string;
  disabled?: boolean;
};

export type QueryFieldSize = 'sm' | 'md' | 'lg' | 'xl' | 'full';

/** Grid-only 布局声明；不允许以像素宽度和 span 并行表达同一布局意图。 */
export type QueryFieldLayout = {
  size?: QueryFieldSize;
  span?: Partial<Record<'wide' | 'medium' | 'narrow' | 'compact', number>>;
  group?: string;
  order?: number;
};

export type ResourceQueryFilterDefinition = {
  key: string;
  label: string;
  field?: ResourceQueryFieldDefinition;
  layout?: QueryFieldLayout;
  /** 渐进迁移入口；renderer 始终先归一化为 field，新调用点使用 field。 */
  type?: ResourceQueryFilterType;
  options?: ResourceQueryFilterOption[];
  placeholder?: string;
  disabled?: boolean;
};

export function resolveResourceQueryField(definition: ResourceQueryFilterDefinition): ResourceQueryFieldDefinition {
  return (
    definition.field ?? {
      type: definition.type ?? 'input',
      options: definition.options,
      placeholder: definition.placeholder,
      disabled: definition.disabled,
    }
  );
}

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
  /** @deprecated Saved View 是 toolbar slot capability，不再由 Query Panel 解释。 */
  savedView?: boolean;
  quickFilters?: ResourceQueryQuickFilter[];
  timeRange?: boolean;
  sorting?: boolean;
};
