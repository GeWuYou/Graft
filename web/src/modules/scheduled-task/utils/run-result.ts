// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { isJsonRecord, type JsonRecord, parseJsonRecord } from './json';

export type ScheduledTaskRunResult = {
  summary?: string;
  stage?: string;
  affected_resource?: string;
  metrics?: JsonRecord;
  details?: JsonRecord;
  warnings?: string[];
};

export function parseRunResult(value?: string | null): ScheduledTaskRunResult {
  const parsed = parseJsonRecord(value);
  const nestedResult = isJsonRecord(parsed.result) ? parsed.result : {};
  const metrics = isJsonRecord(parsed.metrics)
    ? parsed.metrics
    : isJsonRecord(nestedResult.metrics)
      ? nestedResult.metrics
      : undefined;
  const details = isJsonRecord(parsed.details)
    ? parsed.details
    : isJsonRecord(nestedResult.details)
      ? nestedResult.details
      : undefined;
  const warnings = Array.isArray(parsed.warnings)
    ? parsed.warnings
    : Array.isArray(nestedResult.warnings)
      ? nestedResult.warnings
      : undefined;
  return {
    summary:
      typeof parsed.summary === 'string'
        ? parsed.summary
        : typeof nestedResult.summary === 'string'
          ? nestedResult.summary
          : undefined,
    stage:
      typeof parsed.stage === 'string'
        ? parsed.stage
        : typeof nestedResult.stage === 'string'
          ? nestedResult.stage
          : undefined,
    affected_resource:
      typeof parsed.affected_resource === 'string'
        ? parsed.affected_resource
        : typeof nestedResult.affected_resource === 'string'
          ? nestedResult.affected_resource
          : undefined,
    metrics,
    details,
    warnings: warnings?.filter((warning): warning is string => typeof warning === 'string'),
  };
}

export function runResultMetricNumber(result: ScheduledTaskRunResult, key: string): number | undefined {
  const value = result.metrics?.[key];
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : undefined;
  }
  return undefined;
}
