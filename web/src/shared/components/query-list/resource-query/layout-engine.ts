import {
  type QueryFieldLayout,
  type QueryFieldSize,
  resolveResourceQueryField,
  type ResourceQueryFilterDefinition,
} from './types';

export type QueryPanelTier = 'wide' | 'medium' | 'narrow' | 'compact';

export type QueryLayoutItem = {
  field: ResourceQueryFilterDefinition;
  row: number;
  span: number;
};

export type QueryLayoutResult = {
  columns: number;
  commandBar: 'inline' | 'split' | 'stacked' | 'compact';
  fields: QueryLayoutItem[];
  tier: QueryPanelTier;
};

/** Query Panel 的主题化布局阈值；数值不在 renderer 内重复声明。 */
const QUERY_PANEL_LAYOUT_THEME = {
  compactMax: 767,
  mediumMin: 992,
  wideMin: 1200,
} as const;

const SIZE_SPANS: Record<QueryFieldSize, Record<QueryPanelTier, number>> = {
  sm: { wide: 2, medium: 3, narrow: 3, compact: 1 },
  md: { wide: 3, medium: 4, narrow: 3, compact: 1 },
  lg: { wide: 4, medium: 6, narrow: 6, compact: 1 },
  xl: { wide: 6, medium: 6, narrow: 6, compact: 1 },
  full: { wide: 12, medium: 12, narrow: 6, compact: 1 },
};

const TYPE_SIZE: Record<ReturnType<typeof resolveResourceQueryField>['type'], QueryFieldSize> = {
  boolean: 'sm',
  select: 'md',
  'multi-select': 'lg',
  input: 'lg',
  'date-range': 'xl',
  'number-range': 'xl',
};

function resolveQueryPanelTier(width: number): QueryPanelTier {
  if (width < QUERY_PANEL_LAYOUT_THEME.compactMax + 1) return 'compact';
  if (width < QUERY_PANEL_LAYOUT_THEME.mediumMin) return 'narrow';
  if (width < QUERY_PANEL_LAYOUT_THEME.wideMin) return 'medium';
  return 'wide';
}

export function resolveQueryLayout(fields: ResourceQueryFilterDefinition[], width: number): QueryLayoutResult {
  const tier = resolveQueryPanelTier(width);
  const columns = tier === 'compact' ? 1 : tier === 'narrow' ? 6 : 12;
  let usedColumns = 0;
  let row = 1;
  const items = [...fields]
    .sort((left, right) => (left.layout?.order ?? 0) - (right.layout?.order ?? 0))
    .map((field) => {
      const span = resolveFieldSpan(field.layout, resolveResourceQueryField(field).type, tier, columns);
      if (usedColumns + span > columns) {
        row += 1;
        usedColumns = 0;
      }
      usedColumns += span;
      return { field, row, span };
    });

  return {
    columns,
    commandBar: tier === 'wide' ? 'inline' : tier === 'medium' ? 'split' : tier === 'narrow' ? 'stacked' : 'compact',
    fields: items,
    tier,
  };
}

function resolveFieldSpan(
  layout: QueryFieldLayout | undefined,
  type: ReturnType<typeof resolveResourceQueryField>['type'],
  tier: QueryPanelTier,
  columns: number,
) {
  const size = layout?.size ?? TYPE_SIZE[type];
  return Math.min(layout?.span?.[tier] ?? SIZE_SPANS[size][tier], columns);
}
