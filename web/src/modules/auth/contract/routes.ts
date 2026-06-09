// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const AUTH_ROUTE_NAME = {
  LOGIN: 'login',
  RESTRICTED_SESSION: 'RestrictedSession',
} as const;

export const AUTH_ROUTE_PATH = {
  LOGIN: '/login',
  RESTRICTED_SESSION: '/auth/restricted-session',
} as const;
