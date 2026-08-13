import { createLogger } from '@/utils/logger';

import { isDebugFlagEnabled } from './runtime';

export const FRONTEND_INVESTIGATION_MARKER = 'FRONTEND-INVESTIGATION';
export const FRONTEND_INVESTIGATION_SCHEMA_VERSION = 1;

export type InvestigationPhase =
  | 'USER_ACTION'
  | 'UI_EVENT'
  | 'STATE_CHANGE'
  | 'WATCHER_TRIGGER'
  | 'ROUTE_NAVIGATION'
  | 'LIFECYCLE'
  | 'STORE_ACTION'
  | 'QUERY'
  | 'API_REQUEST'
  | 'ASYNC_BOUNDARY'
  | 'ERROR';

export type AsyncBoundary = 'sync' | 'microtask' | 'timer' | 'network' | 'unknown';

export interface BehaviorInvestigationEventInput {
  phase: InvestigationPhase;
  event: string;
  source: string;
  parentEventId?: string;
  asyncBoundary?: AsyncBoundary;
  component?: string;
  route?: string;
  stateSummary?: unknown;
  requestSummary?: unknown;
  queryKeySummary?: unknown;
  payloadSummary?: unknown;
}

export interface BehaviorInvestigationEvent {
  marker: typeof FRONTEND_INVESTIGATION_MARKER;
  schemaVersion: typeof FRONTEND_INVESTIGATION_SCHEMA_VERSION;
  investigationId: string;
  eventId: string;
  parentEventId?: string;
  seq: number;
  timestamp: string;
  elapsedMs: number;
  phase: InvestigationPhase;
  event: string;
  source: string;
  asyncBoundary: AsyncBoundary;
  component?: string;
  route?: string;
  stateSummary?: unknown;
  requestSummary?: unknown;
  queryKeySummary?: unknown;
  payloadSummary?: unknown;
}

export interface BehaviorInvestigationOptions {
  investigationId?: string;
  /** 显式 gate 可在测试或案件级临时范围内覆盖默认的 foundation debug flag。 */
  isEnabled?: () => boolean;
  maxEvents?: number;
  /** Additional explicitly approved summary keys for a bounded investigation case. */
  allowedSummaryKeys?: readonly string[];
}

export interface BehaviorInvestigationSession {
  readonly investigationId: string;
  emit(input: BehaviorInvestigationEventInput): BehaviorInvestigationEvent | null;
  events(): readonly BehaviorInvestigationEvent[];
  close(): void;
  isClosed(): boolean;
}

const logger = createLogger('debug.frontend-investigation');
const SECRET_KEY = /token|password|secret|cookie|authorization|credential|private.?key/i;
const DEFAULT_ALLOWED_SUMMARY_KEYS = new Set([
  'action',
  'count',
  'durationMs',
  'errorCode',
  'id',
  'method',
  'mutationType',
  'name',
  'path',
  'queryKey',
  'route',
  'status',
  'success',
  'visible',
]);
let sequence = 0;

function createId(prefix: string): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  sequence += 1;
  return `${prefix}-${Date.now().toString(36)}-${sequence.toString(36)}`;
}

function sanitize(value: unknown, allowedKeys: ReadonlySet<string>, depth = 0, key?: string): unknown {
  if (key && SECRET_KEY.test(key)) {
    return '[REDACTED]';
  }
  if (value === null || value === undefined || typeof value === 'boolean' || typeof value === 'number') {
    return value;
  }
  if (typeof value === 'bigint') {
    return String(value);
  }
  if (typeof value === 'string') {
    return value.length > 240 ? `${value.slice(0, 240)}...[truncated]` : value;
  }
  if (depth >= 3) {
    return '[Truncated]';
  }
  if (Array.isArray(value)) {
    return value.slice(0, 20).map((item) => sanitize(item, allowedKeys, depth + 1));
  }
  if (value instanceof Date) {
    return value.toISOString();
  }
  if (typeof value === 'object') {
    const result: Record<string, unknown> = {};
    Object.entries(value as Record<string, unknown>)
      .slice(0, 20)
      .forEach(([entryKey, entryValue]) => {
        if (!allowedKeys.has(entryKey) && !SECRET_KEY.test(entryKey)) {
          return;
        }
        result[entryKey] = sanitize(entryValue, allowedKeys, depth + 1, entryKey);
      });
    return result;
  }
  return String(value);
}

/**
 * 创建默认关闭且有界的前端行为取证会话。
 *
 * 会话只保留最近的结构化事件；调用方负责选择调查范围、传递父事件并在取证完成后关闭会话。
 */
export function createBehaviorInvestigation(options: BehaviorInvestigationOptions = {}): BehaviorInvestigationSession {
  const investigationId = options.investigationId ?? createId('frontend');
  const isEnabled = options.isEnabled ?? (() => isDebugFlagEnabled('frontend-investigation'));
  const maxEvents = Math.max(1, options.maxEvents ?? 500);
  const allowedSummaryKeys = new Set([...DEFAULT_ALLOWED_SUMMARY_KEYS, ...(options.allowedSummaryKeys ?? [])]);
  const startedAt = Date.now();
  const retainedEvents: BehaviorInvestigationEvent[] = [];
  let seq = 0;
  let closed = false;

  return {
    investigationId,
    emit(input) {
      if (closed || !isEnabled()) {
        return null;
      }
      seq += 1;
      const event: BehaviorInvestigationEvent = {
        marker: FRONTEND_INVESTIGATION_MARKER,
        schemaVersion: FRONTEND_INVESTIGATION_SCHEMA_VERSION,
        investigationId,
        eventId: `${investigationId}:${seq}`,
        ...(input.parentEventId ? { parentEventId: input.parentEventId } : {}),
        seq,
        timestamp: new Date().toISOString(),
        elapsedMs: Date.now() - startedAt,
        phase: input.phase,
        event: input.event,
        source: input.source,
        asyncBoundary: input.asyncBoundary ?? 'sync',
        ...(input.component ? { component: input.component } : {}),
        ...(input.route ? { route: input.route } : {}),
        ...(input.stateSummary !== undefined ? { stateSummary: sanitize(input.stateSummary, allowedSummaryKeys) } : {}),
        ...(input.requestSummary !== undefined
          ? { requestSummary: sanitize(input.requestSummary, allowedSummaryKeys) }
          : {}),
        ...(input.queryKeySummary !== undefined
          ? { queryKeySummary: sanitize(input.queryKeySummary, allowedSummaryKeys) }
          : {}),
        ...(input.payloadSummary !== undefined
          ? { payloadSummary: sanitize(input.payloadSummary, allowedSummaryKeys) }
          : {}),
      };
      retainedEvents.push(event);
      if (retainedEvents.length > maxEvents) {
        retainedEvents.splice(0, retainedEvents.length - maxEvents);
      }
      logger.debug(`${FRONTEND_INVESTIGATION_MARKER} ${event.event}`, event);
      return event;
    },
    events: () => retainedEvents.slice(),
    close: () => {
      closed = true;
    },
    isClosed: () => closed,
  };
}
