// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const SYSTEM_CONFIG_PERMISSION_CODE = {
  READ: 'system-config.read',
  WRITE: 'system-config.write',
} as const;

export type SystemConfigPermissionCode =
  (typeof SYSTEM_CONFIG_PERMISSION_CODE)[keyof typeof SYSTEM_CONFIG_PERMISSION_CODE];
