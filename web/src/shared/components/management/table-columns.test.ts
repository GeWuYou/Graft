import { describe, expect, it } from 'vitest';

import {
  createCountColumn,
  createIdentifierColumn,
  createMainTextColumn,
  createStatusColumn,
  createTechnicalColumn,
  createTimeColumn,
  resolveTableWidthPolicy,
} from './table-columns';

describe('table column width policy', () => {
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
});
