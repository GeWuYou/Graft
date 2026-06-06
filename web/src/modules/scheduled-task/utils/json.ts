export type JsonRecord = Record<string, unknown>;

export function parseJsonRecord(value?: string | null): JsonRecord {
  const trimmed = value?.trim();
  if (!trimmed) {
    return {};
  }

  try {
    const parsed: unknown = JSON.parse(trimmed);
    return isJsonRecord(parsed) ? parsed : {};
  } catch {
    return {};
  }
}

export function isJsonRecord(value: unknown): value is JsonRecord {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value));
}

export function formatJsonPreview(value?: string | null) {
  const trimmed = value?.trim();
  if (!trimmed) {
    return '';
  }

  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2);
  } catch {
    return trimmed;
  }
}
