// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const SYSTEM_CONFIG_ROUTE_PATH = {
  LIST: '/server/system-config',
} as const;

export const SYSTEM_CONFIG_API_PATH = {
  LIST: '/api/system-configs',
  DETAIL: '/api/system-configs/{key}',
  RESET: '/api/system-configs/{key}/reset',
} as const;

export function buildSystemConfigDetailApiPath(key: string) {
  return SYSTEM_CONFIG_API_PATH.DETAIL.replace('{key}', encodeURIComponent(key));
}

export function buildSystemConfigResetApiPath(key: string) {
  return SYSTEM_CONFIG_API_PATH.RESET.replace('{key}', encodeURIComponent(key));
}
