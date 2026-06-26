export type LogStream = 'stdout' | 'stderr';

export type StructuredLogEntry = Readonly<{
  line: string;
  occurredAt: string;
  stream: LogStream;
}>;

/**
 * 规范化单条结构化日志，丢弃空行并为缺失字段补安全默认值。
 *
 * @param value - 待规范化的日志值
 * @returns 规范化后的结构化日志；无效时返回 `null`
 */
export function normalizeStructuredLogEntry(value: unknown): StructuredLogEntry | null {
  if (!value || typeof value !== 'object') {
    return null;
  }

  const candidate = value as {
    line?: unknown;
    occurredAt?: unknown;
    occurred_at?: unknown;
    stream?: unknown;
  };

  const line = typeof candidate.line === 'string' ? candidate.line : '';
  if (!line.length) {
    return null;
  }

  const occurredAt =
    typeof candidate.occurredAt === 'string'
      ? candidate.occurredAt
      : typeof candidate.occurred_at === 'string'
        ? candidate.occurred_at
        : '';
  const stream = candidate.stream === 'stderr' ? 'stderr' : 'stdout';

  return Object.freeze({
    line,
    occurredAt,
    stream,
  });
}
