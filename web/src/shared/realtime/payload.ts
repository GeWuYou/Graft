export function isRealtimePayloadObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === 'object');
}

export function parseRealtimeEnvelopeData(raw: unknown): Record<string, unknown> | null {
  if (typeof raw !== 'string') {
    return null;
  }

  try {
    const parsed = JSON.parse(raw) as { data?: unknown };
    if (!isRealtimePayloadObject(parsed) || !isRealtimePayloadObject(parsed.data)) {
      return null;
    }
    return parsed.data;
  } catch {
    return null;
  }
}
