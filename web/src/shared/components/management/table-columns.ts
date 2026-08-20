import type { TdBaseTableProps } from 'tdesign-vue-next';

type ColumnAlign = 'left' | 'center' | 'right';

type ColumnConfig = {
  align?: ColumnAlign;
  ellipsis?: boolean;
  fixed?: 'left' | 'right';
  minWidth?: number;
  width?: number;
};

type TableColumn = NonNullable<TdBaseTableProps['columns']>[number];

export type ManagedTableWidthPolicy = {
  contentWidth: number;
  mode: 'fill' | 'scroll';
  tableContentWidth?: string;
};

const DEFAULT_ELLIPSIS = { theme: 'default', placement: 'top-left' } as const;

function withCommonColumnOptions(column: TableColumn, config: ColumnConfig = {}) {
  return {
    align: config.align ?? 'left',
    ellipsis: config.ellipsis ?? DEFAULT_ELLIPSIS,
    ...column,
    ...(config.fixed ? { fixed: config.fixed } : {}),
    ...(config.width ? { width: config.width } : {}),
    ...(config.minWidth ? { minWidth: config.minWidth } : {}),
  } as TableColumn;
}

export function createTextColumn(
  title: string,
  colKey: string,
  config: Omit<ColumnConfig, 'align'> & { align?: ColumnAlign } = {},
) {
  return withCommonColumnOptions(
    {
      title,
      colKey,
    },
    config,
  );
}

export function createStatusColumn(title: string, colKey: string, width = 112) {
  return withCommonColumnOptions(
    {
      title,
      colKey,
    },
    {
      align: 'center',
      width,
      ellipsis: false,
    },
  );
}

export function createCountColumn(title: string, colKey: string, width = 108, align: ColumnAlign = 'center') {
  return withCommonColumnOptions(
    {
      title,
      colKey,
    },
    {
      align,
      width,
      ellipsis: false,
    },
  );
}

export function createTimeColumn(title: string, colKey: string, width = 168) {
  return withCommonColumnOptions(
    {
      title,
      colKey,
    },
    {
      width,
      align: 'center',
    },
  );
}

export function createMainTextColumn(title: string, colKey: string, minWidth = 360) {
  return createTextColumn(title, colKey, {
    minWidth,
  });
}

export function createIdentifierColumn(title: string, colKey: string, width = 180) {
  return createTextColumn(title, colKey, {
    width,
  });
}

export function createTechnicalColumn(title: string, colKey: string, width = 240) {
  return createTextColumn(title, colKey, {
    width,
  });
}

export function createActionColumn(title: string, width = 108, align: ColumnAlign = 'center', colKey = 'operation') {
  return withCommonColumnOptions(
    {
      title,
      colKey,
    },
    {
      width,
      align,
      fixed: 'right',
      ellipsis: false,
    },
  );
}

export type ManagedColumnKey = string;

export type ManagedColumnMeta = {
  defaultVisible: boolean;
  detailOnly?: boolean;
  key: ManagedColumnKey;
  label: string;
};

/** 将列选择限制在当前支持的键中，并在选择被清空时按回退顺序保留最少列数。 */
export function normalizeManagedColumnKeys(
  selectedKeys: string[],
  supportedKeys: string[],
  fallbackKeys: string[] = supportedKeys,
  minimumSelected = 1,
) {
  const supportedKeySet = new Set(supportedKeys);
  const nextKeys = Array.from(new Set(selectedKeys.filter((key) => supportedKeySet.has(key))));
  const requiredCount = Math.min(Math.max(1, minimumSelected), supportedKeys.length);

  for (const key of [...fallbackKeys, ...supportedKeys]) {
    if (nextKeys.length >= requiredCount) break;
    if (supportedKeySet.has(key) && !nextKeys.includes(key)) nextKeys.push(key);
  }

  return nextKeys;
}

export function buildVisibleColumns(
  columns: TdBaseTableProps['columns'],
  visibleKeys: string[],
  alwaysVisibleKeys: string[] = [],
) {
  const sourceColumns = columns ?? [];
  const sourceColumnKeys = sourceColumns.map((column) => String(column?.colKey));
  const visibleKeySet = new Set(
    normalizeManagedColumnKeys([...visibleKeys, ...alwaysVisibleKeys], sourceColumnKeys, alwaysVisibleKeys),
  );
  const visibleColumns = sourceColumns.filter((column) => visibleKeySet.has(String(column?.colKey)));
  return visibleColumns;
}

export function resolveManagedColumns(
  columns: TdBaseTableProps['columns'],
  visibleKeys?: string[],
  alwaysVisibleKeys: string[] = [],
) {
  if (!visibleKeys?.length) {
    return columns;
  }

  return buildVisibleColumns(columns, visibleKeys, alwaysVisibleKeys);
}

/**
 * 为表格空态生成随宿主宽度伸缩的列定义。
 *
 * 空态没有需要横向比对的行，保留宽表的固定列配置会触发 TDesign 的异步溢出空态布局；数据返回后仍使用原始列定义。
 */
export function resolveEmptyManagedColumns(columns: TdBaseTableProps['columns']) {
  return columns?.map((column) => {
    const emptyColumn = { ...column };
    delete emptyColumn.fixed;
    delete emptyColumn.minWidth;
    delete emptyColumn.width;

    if (emptyColumn.children?.length) {
      emptyColumn.children = resolveEmptyManagedColumns(emptyColumn.children);
    }

    return emptyColumn;
  });
}

function calculateVisibleColumnWidth(columns: TdBaseTableProps['columns']) {
  return (columns ?? []).reduce((sum, column) => {
    if (typeof column?.width === 'number') {
      return sum + column.width;
    }

    if (typeof column?.minWidth === 'number') {
      return sum + column.minWidth;
    }

    return sum + 160;
  }, 0);
}

export function resolveTableWidthPolicy(
  columns: TdBaseTableProps['columns'],
  containerWidth = 0,
): ManagedTableWidthPolicy {
  const contentWidth = calculateVisibleColumnWidth(columns);
  const mode = containerWidth > 0 && contentWidth > containerWidth ? 'scroll' : 'fill';

  return {
    contentWidth,
    mode,
    tableContentWidth: mode === 'scroll' ? `${contentWidth}px` : undefined,
  };
}

export type TextColumnSpec = {
  config?: ColumnConfig;
  key: string;
  kind?: 'identifier' | 'main' | 'technical' | 'text';
  title: string;
};

export type TimeColumnSpec = {
  key: string;
  kind: 'time';
  title: string;
  width?: number;
};

export type ConfiguredColumnSpec = TextColumnSpec | TimeColumnSpec;

export function createConfiguredColumns(specs: ConfiguredColumnSpec[]) {
  return specs.map((spec) => {
    if (spec.kind === 'time') {
      return createTimeColumn(spec.title, spec.key, spec.width);
    }

    if (spec.kind === 'main') {
      return createMainTextColumn(spec.title, spec.key, spec.config?.minWidth);
    }

    if (spec.kind === 'identifier') {
      return createIdentifierColumn(spec.title, spec.key, spec.config?.width);
    }

    if (spec.kind === 'technical') {
      return createTechnicalColumn(spec.title, spec.key, spec.config?.width);
    }

    return createTextColumn(spec.title, spec.key, spec.config);
  });
}
