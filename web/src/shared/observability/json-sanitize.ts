// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

const MASKED_VALUE = '******';
const SENSITIVE_KEYS = new Set([
  'password',
  'passwd',
  'secret',
  'token',
  'access_token',
  'refresh_token',
  'api_key',
  'private_key',
  'authorization',
  'cookie',
]);

function isSensitiveJsonKey(key: string) {
  const normalized = key.trim().toLowerCase();
  return SENSITIVE_KEYS.has(normalized);
}

export function maskSensitiveJson<T>(value: T): T {
  return maskValue(value) as T;
}

function maskValue(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map((item) => maskValue(item));
  }

  if (!value || typeof value !== 'object') {
    return value;
  }

  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>).map(([key, childValue]) => [
      key,
      isSensitiveJsonKey(key) ? MASKED_VALUE : maskValue(childValue),
    ]),
  );
}
