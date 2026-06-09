// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export type JsonRecord = Record<string, unknown>;

export function parseJsonValue(value?: string | null): unknown {
  const trimmed = value?.trim();
  if (!trimmed) {
    return undefined;
  }

  try {
    return JSON.parse(trimmed);
  } catch {
    return undefined;
  }
}

export function parseJsonRecord(value?: string | null): JsonRecord {
  const parsed = parseJsonValue(value);
  return isJsonRecord(parsed) ? parsed : {};
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

export function formatJsonValue(value: unknown) {
  if (value === undefined) {
    return '';
  }

  return JSON.stringify(value, null, 2);
}

export function valuePreview(value: unknown, noneText: string, booleanLabel: (value: boolean) => string) {
  if (typeof value === 'boolean') {
    return booleanLabel(value);
  }
  if (value === undefined || value === null || value === '') {
    return noneText;
  }
  if (typeof value === 'object') {
    return JSON.stringify(value);
  }
  return String(value);
}
