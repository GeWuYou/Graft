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
  return {
    summary: typeof parsed.summary === 'string' ? parsed.summary : undefined,
    stage: typeof parsed.stage === 'string' ? parsed.stage : undefined,
    affected_resource: typeof parsed.affected_resource === 'string' ? parsed.affected_resource : undefined,
    metrics: isJsonRecord(parsed.metrics) ? parsed.metrics : undefined,
    details: isJsonRecord(parsed.details) ? parsed.details : undefined,
    warnings: Array.isArray(parsed.warnings)
      ? parsed.warnings.filter((warning): warning is string => typeof warning === 'string')
      : undefined,
  };
}
