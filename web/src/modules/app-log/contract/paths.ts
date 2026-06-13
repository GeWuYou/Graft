// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const APP_LOG_ROUTE_PATH = {
  ROOT: '/logs',
  LIST: '/logs/app',
  DETAIL: '/logs/app/:id',
} as const;

export const APP_LOG_API_PATH = {
  LIST: '/api/app-log',
  DETAIL: '/api/app-log/{id}',
  BATCH_DELETE: '/api/app-log/batch-delete',
} as const;
