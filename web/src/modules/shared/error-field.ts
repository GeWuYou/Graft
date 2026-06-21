export function readErrorField(payload: unknown): string | null {
  if (!payload || typeof payload !== 'object' || !('data' in payload)) {
    return null;
  }

  const data = (payload as { data?: unknown }).data;
  if (!data || typeof data !== 'object' || !('field' in data)) {
    return null;
  }

  const field = (data as { field?: unknown }).field;
  return typeof field === 'string' && field.trim() !== '' ? field.trim() : null;
}
