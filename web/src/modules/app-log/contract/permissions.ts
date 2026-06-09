// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const APP_LOG_PERMISSION_CODE = {
  READ: 'app_log.read',
} as const;

export type AppLogPermissionCode = (typeof APP_LOG_PERMISSION_CODE)[keyof typeof APP_LOG_PERMISSION_CODE];
