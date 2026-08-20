import { describe, expect, it } from 'vitest';

import {
  buildVisibleColumns,
  createCountColumn,
  createIdentifierColumn,
  createMainTextColumn,
  createStatusColumn,
  createTechnicalColumn,
  createTimeColumn,
  normalizeManagedColumnKeys,
  resolveEmptyManagedColumns,
  resolveTableWidthPolicy,
} from './table-columns';

describe('table column width policy', () => {
  it('keeps the first column when a page attempts to hide every field', () => {
    const columns = [createIdentifierColumn('任务', 'task'), createStatusColumn('状态', 'status')];

    expect(buildVisibleColumns(columns, [])).toEqual([columns[0]]);
  });

  it('normalizes empty and stale column selections against supported keys', () => {
    expect(normalizeManagedColumnKeys([], ['name', 'status'])).toEqual(['name']);
    expect(normalizeManagedColumnKeys(['removed', 'status'], ['name', 'status'])).toEqual(['status']);
  });

  it('uses fill mode when visible columns fit the current table body', () => {
    const columns = [
      createTimeColumn('发生时间', 'occurred_at', 176),
      createStatusColumn('级别', 'severity', 104),
      createIdentifierColumn('组件', 'component', 184),
      createTechnicalColumn('事件 Key', 'operation', 196),
      createMainTextColumn('消息', 'message', 420),
    ];

    expect(resolveTableWidthPolicy(columns, 1280)).toEqual({
      contentWidth: 1080,
      mode: 'fill',
      tableContentWidth: undefined,
    });
  });

  it('uses internal scroll mode when visible columns exceed the current table body', () => {
    const columns = [
      createTimeColumn('发生时间', 'occurred_at', 176),
      createStatusColumn('级别', 'severity', 104),
      createIdentifierColumn('组件', 'component', 184),
      createTechnicalColumn('事件 Key', 'operation', 196),
      createMainTextColumn('消息', 'message', 420),
      createTechnicalColumn('请求 ID', 'request_id', 260),
      createCountColumn('字段数', 'fields', 92),
    ];

    expect(resolveTableWidthPolicy(columns, 1280)).toEqual({
      contentWidth: 1432,
      mode: 'scroll',
      tableContentWidth: '1432px',
    });
  });

  it('removes wide-table constraints from empty-state columns without mutating the data-state columns', () => {
    const columns = [
      createIdentifierColumn('任务', 'build_id', 180),
      {
        colKey: 'actions',
        title: '操作',
        width: 120,
        fixed: 'right' as const,
        children: [createTechnicalColumn('日志', 'logs', 260)],
      },
    ];

    expect(resolveEmptyManagedColumns(columns)).toEqual([
      { colKey: 'build_id', title: '任务', align: 'left', ellipsis: { theme: 'default', placement: 'top-left' } },
      {
        colKey: 'actions',
        title: '操作',
        children: [
          { colKey: 'logs', title: '日志', align: 'left', ellipsis: { theme: 'default', placement: 'top-left' } },
        ],
      },
    ]);
    expect(columns[0]?.width).toBe(180);
    expect(columns[1]?.fixed).toBe('right');
    expect(columns[1]?.children?.[0]?.width).toBe(260);
  });
});
