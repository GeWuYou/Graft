// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { sanitizeTraceFieldsForDisplay } from './sanitize';

describe('sanitizeTraceFieldsForDisplay', () => {
  it('removes reserved trace fields from nested display payloads', () => {
    expect(
      sanitizeTraceFieldsForDisplay({
        request_id: 'req-1',
        trace_id: 'trace-1',
        nested: {
          traceId: 'trace-2',
          rows: [{ trace_id: 'trace-3', value: 'kept' }],
        },
      }),
    ).toEqual({
      request_id: 'req-1',
      nested: {
        rows: [{ value: 'kept' }],
      },
    });
  });
});
