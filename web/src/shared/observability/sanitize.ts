// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export function sanitizeTraceFieldsForDisplay(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map((item) => sanitizeTraceFieldsForDisplay(item));
  }
  if (!value || typeof value !== 'object') {
    return value;
  }

  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>)
      .filter(([key]) => key !== 'trace_id' && key !== 'traceId')
      .map(([key, item]) => [key, sanitizeTraceFieldsForDisplay(item)]),
  );
}
